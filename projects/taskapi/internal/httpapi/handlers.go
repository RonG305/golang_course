package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golangcourse/projects/taskapi/internal/domain"
	"golangcourse/projects/taskapi/internal/service"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// Every handler follows the same four steps: decode, call the service, map
// the error, encode. When a handler grows beyond that, the extra logic
// belongs in the service.

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReady is the readiness probe: it should check the things the service
// cannot work without (database, cache) and fail if they are unavailable.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if _, _, err := s.tasks.List(r.Context(), domain.Filter{Limit: 1}); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "dependencies unavailable", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	task, err := s.tasks.Create(r.Context(), service.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    domain.Priority(req.Priority),
		DueAt:       req.DueAt,
	})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}

	// 201 plus a Location header is the correct answer to a POST that creates.
	w.Header().Set("Location", "/v1/tasks/"+task.ID)
	writeJSON(w, http.StatusCreated, toTaskResponse(task, s.now()))
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	filter, err := parseFilter(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	tasks, total, err := s.tasks.List(r.Context(), filter)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}

	filter = filter.Normalise()
	writeJSON(w, http.StatusOK, listTasksResponse{
		Tasks:  toTaskResponses(tasks, s.now()),
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.tasks.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskResponse(task, s.now()))
}

func (s *Server) handleCompleteTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.tasks.Complete(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskResponse(task, s.now()))
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if err := s.tasks.Delete(r.Context(), r.PathValue("id")); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	// 204: success, deliberately no body.
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleOverdueReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.tasks.Overdue(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"overdue":     toTaskResponses(report.Overdue, s.now()),
		"open_total":  report.Total,
		"overdue_num": len(report.Overdue),
	})
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound,
		fmt.Sprintf("no route for %s %s", r.Method, r.URL.Path), nil)
}

// --- Decoding helpers ---

// decodeJSON reads a bounded body, rejects unknown fields and turns the
// library's error messages into ones a client can act on.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" &&
		ct != "application/json" && ct != "application/json; charset=utf-8" {
		return fmt.Errorf("Content-Type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		var maxBytesErr *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("malformed JSON at byte %d", syntaxErr.Offset)
		case errors.As(err, &typeErr):
			return fmt.Errorf("field %q must be of type %s", typeErr.Field, typeErr.Type)
		case errors.Is(err, io.EOF):
			return errors.New("request body must not be empty")
		case errors.As(err, &maxBytesErr):
			return fmt.Errorf("request body must be smaller than %d bytes", maxBodyBytes)
		default:
			return fmt.Errorf("invalid request body: %w", err)
		}
	}

	// A second Decode must hit EOF, otherwise the body held two JSON values.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

// parseFilter builds a domain.Filter from the query string, rejecting bad
// input rather than silently ignoring it.
func parseFilter(r *http.Request) (domain.Filter, error) {
	q := r.URL.Query()
	var f domain.Filter

	if v := q.Get("done"); v != "" {
		done, err := strconv.ParseBool(v)
		if err != nil {
			return f, fmt.Errorf("done must be true or false, got %q", v)
		}
		f.Done = &done
	}
	if v := q.Get("priority"); v != "" {
		p := domain.Priority(v)
		if !p.Valid() {
			return f, fmt.Errorf("priority must be low, medium or high, got %q", v)
		}
		f.Priority = &p
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return f, fmt.Errorf("limit must be a non-negative integer, got %q", v)
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return f, fmt.Errorf("offset must be a non-negative integer, got %q", v)
		}
		f.Offset = n
	}
	return f, nil
}
