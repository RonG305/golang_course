// Package httpapi is the HTTP transport layer. Its only jobs are: route,
// decode, call the service, map errors, encode. No business logic lives here.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"golangcourse/projects/taskapi/internal/config"
	"golangcourse/projects/taskapi/internal/domain"
	"golangcourse/projects/taskapi/internal/service"
)

type Server struct {
	tasks  *service.TaskService
	logger *slog.Logger
	cfg    config.Config
	now    func() time.Time
	http   *http.Server
}

func NewServer(tasks *service.TaskService, logger *slog.Logger, cfg config.Config) *Server {
	s := &Server{tasks: tasks, logger: logger, cfg: cfg, now: time.Now}
	s.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
		// Hook the server's own errors into the structured logger.
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	return s
}

// Handler builds the routes and wraps them in middleware. It is exported so
// tests can drive the whole stack through httptest without binding a port.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Go 1.22+ patterns: method and path wildcards, no third-party router.
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("POST /v1/tasks", s.handleCreateTask)
	mux.HandleFunc("GET /v1/tasks", s.handleListTasks)
	mux.HandleFunc("GET /v1/tasks/{id}", s.handleGetTask)
	mux.HandleFunc("POST /v1/tasks/{id}/complete", s.handleCompleteTask)
	mux.HandleFunc("DELETE /v1/tasks/{id}", s.handleDeleteTask)
	mux.HandleFunc("GET /v1/reports/overdue", s.handleOverdueReport)
	mux.HandleFunc("/", s.handleNotFound)

	// Outermost first: a panic anywhere below is still recovered and logged
	// with its request ID.
	return chain(mux,
		requestID(),
		recoverPanics(s.logger),
		requestLogger(s.logger),
		timeout(30*time.Second),
	)
}

// Run serves until ctx is cancelled, then drains gracefully (lesson 29).
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.cfg.Addr, err)
	}
	s.logger.Info("server listening",
		slog.String("addr", ln.Addr().String()),
		slog.String("env", s.cfg.Env))

	serveErr := make(chan error, 1)
	go func() {
		if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		s.logger.Info("shutdown requested, draining connections",
			slog.Duration("timeout", s.cfg.ShutdownTimeout))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		_ = s.http.Close()
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	s.logger.Info("server drained")
	return nil
}

// --- Encoding helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Too late to change the status; the client will see a short read.
		return
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string, fields []domain.FieldError) {
	writeJSON(w, status, errorResponse{
		Error:     msg,
		RequestID: RequestIDFrom(r.Context()),
		Fields:    fields,
	})
}

// writeServiceError is the single place that maps domain errors to HTTP
// status codes. Handlers never choose a status for a failure themselves,
// so the mapping can never drift between endpoints.
func (s *Server) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var fieldErrs domain.FieldErrors
	switch {
	case errors.As(err, &fieldErrs):
		writeError(w, r, http.StatusUnprocessableEntity, "validation failed", fieldErrs)

	case errors.Is(err, domain.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "task not found", nil)

	case errors.Is(err, domain.ErrAlreadyDone):
		writeError(w, r, http.StatusConflict, "task is already done", nil)

	case errors.Is(err, domain.ErrNotDone):
		writeError(w, r, http.StatusConflict, "task is not done", nil)

	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, r, http.StatusGatewayTimeout, "request timed out", nil)

	case errors.Is(err, context.Canceled):
		// The client hung up; nobody is listening for a response.
		s.logger.InfoContext(r.Context(), "client cancelled the request")

	default:
		// An unexpected error: log the detail, return something generic.
		// Never leak internal errors (or SQL) to clients.
		s.logger.ErrorContext(r.Context(), "unhandled error",
			slog.String("err", err.Error()),
			slog.String("request_id", RequestIDFrom(r.Context())))
		writeError(w, r, http.StatusInternalServerError, "internal server error", nil)
	}
}
