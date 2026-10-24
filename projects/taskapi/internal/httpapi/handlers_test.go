package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golangcourse/projects/taskapi/internal/config"
	"golangcourse/projects/taskapi/internal/service"
	"golangcourse/projects/taskapi/internal/store"
)

var testNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// newTestServer wires the real stack with an in-memory store. These are
// integration tests over the whole HTTP surface, and they need no network:
// httptest.NewRecorder drives the handler directly.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(store.NewMemory(), logger,
		service.WithClock(func() time.Time { return testNow }))
	s := NewServer(svc, logger, config.Config{Env: "development", Addr: "127.0.0.1:0"})
	s.now = func() time.Time { return testNow }
	return s
}

// do sends a request through the FULL middleware chain and returns the result.
func do(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decoding %s: %v", rec.Body.String(), err)
	}
	return v
}

func TestHealthAndReady(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/healthz", "/readyz"} {
		rec := do(t, s, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

func TestCreateTask(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantField  string // expected validation field, if any
	}{
		{
			name:       "valid",
			body:       `{"title":"write the API","priority":"high"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "minimal body defaults the priority",
			body:       `{"title":"minimal"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing title",
			body:       `{"description":"no title here"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "title",
		},
		{
			name:       "bad priority",
			body:       `{"title":"ok","priority":"urgent"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "priority",
		},
		{
			name:       "due date in the past",
			body:       `{"title":"ok","due_at":"2020-01-01T00:00:00Z"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "due_at",
		},
		{
			name:       "malformed JSON",
			body:       `{"title":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown field",
			body:       `{"title":"ok","colour":"red"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong type",
			body:       `{"title":123}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "two JSON objects",
			body:       `{"title":"a"}{"title":"b"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestServer(t)

			rec := do(t, s, http.MethodPost, "/v1/tasks", tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if rec.Code == http.StatusCreated {
				if loc := rec.Header().Get("Location"); loc == "" {
					t.Error("201 response is missing a Location header")
				}
				got := decode[taskResponse](t, rec)
				if got.ID == "" {
					t.Error("created task has no id")
				}
				if got.Priority == "" {
					t.Error("priority was not defaulted")
				}
			}
			if tt.wantField != "" {
				got := decode[errorResponse](t, rec)
				found := false
				for _, f := range got.Fields {
					if f.Field == tt.wantField {
						found = true
					}
				}
				if !found {
					t.Errorf("no error for field %q; got %+v", tt.wantField, got.Fields)
				}
			}
			// Every response must carry the request ID back to the client.
			if rec.Header().Get("X-Request-Id") == "" {
				t.Error("missing X-Request-Id header")
			}
		})
	}
}

func TestTaskLifecycle(t *testing.T) {
	s := newTestServer(t)

	// Create
	rec := do(t, s, http.MethodPost, "/v1/tasks", `{"title":"ship it","priority":"high"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	created := decode[taskResponse](t, rec)

	// Read
	rec = do(t, s, http.MethodGet, "/v1/tasks/"+created.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get = %d", rec.Code)
	}
	if got := decode[taskResponse](t, rec); got.Title != "ship it" {
		t.Errorf("title = %q", got.Title)
	}

	// Complete
	rec = do(t, s, http.MethodPost, "/v1/tasks/"+created.ID+"/complete", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("complete = %d: %s", rec.Code, rec.Body)
	}
	if got := decode[taskResponse](t, rec); !got.Done {
		t.Error("task is not done after completing it")
	}

	// Completing twice is a conflict, not a 500 and not a silent success.
	rec = do(t, s, http.MethodPost, "/v1/tasks/"+created.ID+"/complete", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("second complete = %d, want 409", rec.Code)
	}

	// Delete
	rec = do(t, s, http.MethodDelete, "/v1/tasks/"+created.ID, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Error("204 response must have an empty body")
	}

	// Gone
	rec = do(t, s, http.MethodGet, "/v1/tasks/"+created.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", rec.Code)
	}
}

func TestListTasks(t *testing.T) {
	s := newTestServer(t)
	for _, body := range []string{
		`{"title":"a","priority":"high"}`,
		`{"title":"b","priority":"low"}`,
		`{"title":"c","priority":"high"}`,
	} {
		if rec := do(t, s, http.MethodPost, "/v1/tasks", body); rec.Code != http.StatusCreated {
			t.Fatalf("seeding failed: %s", rec.Body)
		}
	}
	// Complete one so the done filter has something to find.
	all := decode[listTasksResponse](t, do(t, s, http.MethodGet, "/v1/tasks", ""))
	do(t, s, http.MethodPost, "/v1/tasks/"+all.Tasks[0].ID+"/complete", "")

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantLen    int
	}{
		{"all", "", http.StatusOK, 3},
		{"done", "?done=true", http.StatusOK, 1},
		{"not done", "?done=false", http.StatusOK, 2},
		{"high priority", "?priority=high", http.StatusOK, 2},
		{"paged", "?limit=2", http.StatusOK, 2},
		{"second page", "?limit=2&offset=2", http.StatusOK, 1},
		{"bad done", "?done=maybe", http.StatusBadRequest, 0},
		{"bad priority", "?priority=urgent", http.StatusBadRequest, 0},
		{"negative limit", "?limit=-1", http.StatusBadRequest, 0},
		{"non-numeric limit", "?limit=ten", http.StatusBadRequest, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, s, http.MethodGet, "/v1/tasks"+tt.query, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			got := decode[listTasksResponse](t, rec)
			if len(got.Tasks) != tt.wantLen {
				t.Errorf("returned %d tasks, want %d", len(got.Tasks), tt.wantLen)
			}
			if got.Limit == 0 {
				t.Error("response does not echo the effective limit")
			}
		})
	}
}

func TestEmptyListIsAnArrayNotNull(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, http.MethodGet, "/v1/tasks", "")
	// A nil slice encodes as null, which breaks clients that expect to
	// iterate. The DTO layer guarantees [] instead.
	if !strings.Contains(rec.Body.String(), `"tasks":[]`) {
		t.Errorf("empty list should encode as []: %s", rec.Body)
	}
}

func TestRouting(t *testing.T) {
	s := newTestServer(t)
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/v1/tasks/missing", http.StatusNotFound},
		{http.MethodGet, "/nope", http.StatusNotFound},
		{http.MethodPut, "/v1/tasks", http.StatusNotFound}, // method not routed
		{http.MethodDelete, "/v1/tasks/missing", http.StatusNotFound},
	}
	for _, tt := range tests {
		rec := do(t, s, tt.method, tt.path, "")
		if rec.Code != tt.want {
			t.Errorf("%s %s = %d, want %d", tt.method, tt.path, rec.Code, tt.want)
		}
	}
}

func TestErrorResponseShape(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, http.MethodGet, "/v1/tasks/missing", "")

	got := decode[errorResponse](t, rec)
	if got.Error == "" {
		t.Error("error response has no message")
	}
	if got.RequestID == "" {
		t.Error("error response does not include the request id")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestOverdueReport(t *testing.T) {
	s := newTestServer(t)
	soon := testNow.Add(time.Hour).Format(time.RFC3339)
	if rec := do(t, s, http.MethodPost, "/v1/tasks",
		`{"title":"due soon","due_at":"`+soon+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("seeding: %s", rec.Body)
	}

	rec := do(t, s, http.MethodGet, "/v1/reports/overdue", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode[map[string]any](t, rec)
	if got["overdue_num"] != float64(0) {
		t.Errorf("overdue_num = %v, want 0", got["overdue_num"])
	}
}

// The panic middleware must turn a handler panic into a 500, not a crash.
func TestPanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("handler exploded")
	}), requestID(), recoverPanics(logger))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal server error") {
		t.Errorf("body = %s", rec.Body)
	}
	// And the panic detail must NOT leak to the client.
	if strings.Contains(rec.Body.String(), "exploded") {
		t.Error("panic message leaked into the response")
	}
}

func TestUpstreamRequestIDIsHonoured(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "trace-from-gateway")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != "trace-from-gateway" {
		t.Errorf("X-Request-Id = %q, want the upstream value", got)
	}
}

func BenchmarkCreateTask(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(store.NewMemory(), logger)
	s := NewServer(svc, logger, config.Config{Env: "development", Addr: "127.0.0.1:0"})
	h := s.Handler()
	body := `{"title":"benchmark","priority":"low"}`

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/v1/tasks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.now = time.Now
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}
