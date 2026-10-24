// Lesson 29: graceful shutdown.
//
// When Kubernetes, Docker or systemd stops your service it sends SIGTERM and
// then waits. If you ignore it you drop in-flight requests, lose buffered
// writes and leave half-finished jobs. This is the startup/shutdown skeleton
// to use for every long-running Go program.
//
// This demo signals ITSELF after a moment so it finishes on its own.
//
// Run: go run ./lessons/29_graceful_shutdown
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const shutdownTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey { // stable output for the lesson
				return slog.Attr{}
			}
			return a
		},
	}))

	// run() returns an error instead of calling os.Exit, which keeps every
	// deferred cleanup working and makes the whole thing testable.
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
	logger.Info("exited cleanly")
}

func run(logger *slog.Logger) error {
	// --- 1. One context, cancelled by SIGINT or SIGTERM, for the whole app ---
	// signal.NotifyContext is the modern one-liner for this.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop() // restores the default signal behaviour

	// --- 2. Start the HTTP server ---
	mux := http.NewServeMux()
	mux.HandleFunc("GET /fast", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		// An in-flight request that must be allowed to finish.
		logger.Info("slow handler started")
		select {
		case <-time.After(1500 * time.Millisecond):
			fmt.Fprintln(w, "slow work done")
			logger.Info("slow handler finished")
		case <-r.Context().Done():
			// The CLIENT went away (or the server is force-closing).
			logger.Warn("slow handler aborted", "err", r.Context().Err())
		}
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// BaseContext ties every request's context to the app lifecycle.
		BaseContext: func(net.Listener) context.Context { return context.Background() },
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	addr := ln.Addr().String()
	logger.Info("server listening", "addr", addr)

	// srvErr carries a startup failure (e.g. port in use) back to run().
	srvErr := make(chan error, 1)
	go func() {
		// ErrServerClosed means Shutdown was called: an expected outcome.
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
			return
		}
		srvErr <- nil
	}()

	// --- 3. Start background workers that also watch ctx ---
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pollQueue(ctx, logger)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		flushMetrics(ctx, logger)
	}()

	// --- Demo scaffolding: fire a slow request, then SIGTERM ourselves ---
	go func() {
		time.Sleep(300 * time.Millisecond)
		go func() {
			// This request is in flight when the signal arrives. Watch the
			// logs: Shutdown waits for it instead of cutting it off.
			resp, err := http.Get("http://" + addr + "/slow")
			if err != nil {
				logger.Warn("slow request failed", "err", err)
				return
			}
			defer resp.Body.Close()
			logger.Info("slow request got a response", "status", resp.StatusCode)
		}()
		time.Sleep(300 * time.Millisecond)
		logger.Info("sending ourselves SIGTERM (this is what the orchestrator does)")
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}()

	// --- 4. Block until a signal arrives or the server dies ---
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-srvErr:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}

	// --- 5. Shut down in reverse order of startup, on a deadline ---
	// Stop listening for signals so a second Ctrl-C can still kill us hard.
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Shutdown: stop accepting new connections, wait for in-flight handlers.
	logger.Info("draining http server", "timeout", shutdownTimeout)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// The deadline passed with requests still running: force the issue.
		logger.Error("graceful shutdown failed, forcing close", "err", err)
		_ = srv.Close()
	} else {
		logger.Info("http server drained")
	}

	// Wait for the background workers (they saw ctx cancel in step 4).
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		logger.Info("workers stopped")
	case <-shutdownCtx.Done():
		logger.Warn("workers did not stop in time")
	}

	// Then release resources, innermost last: DB pools, caches, flushes.
	closeResources(logger)
	return nil
}

// pollQueue is the shape of every background loop: select on a ticker AND
// on ctx.Done(), and return from the goroutine on cancellation.
func pollQueue(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	processed := 0
	for {
		select {
		case <-ticker.C:
			processed++
			logger.Info("queue poll", "processed", processed)
		case <-ctx.Done():
			// Finish the current unit of work, then stop. Do NOT start new work.
			logger.Info("queue worker stopping", "total", processed)
			return
		}
	}
}

func flushMetrics(ctx context.Context, logger *slog.Logger) {
	<-ctx.Done()
	// A final flush gets a FRESH context: the app context is already cancelled,
	// so using it here would make every call fail immediately.
	flushCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	select {
	case <-time.After(50 * time.Millisecond):
		logger.Info("final metrics flushed")
	case <-flushCtx.Done():
		logger.Warn("metrics flush timed out")
	}
}

func closeResources(logger *slog.Logger) {
	// Close in reverse dependency order, and log failures rather than panicking.
	for _, r := range []string{"cache client", "database pool"} {
		logger.Info("closing", "resource", r)
	}
}
