// Lesson 25: building an HTTP server with net/http alone.
//
// You do not need a framework. net/http plus the Go 1.22+ routing patterns
// covers the great majority of real services.
//
// This program starts a server on a random free port, calls itself a few
// times so you can see the responses, then shuts down cleanly.
//
// Run: go run ./lessons/25_http_server
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Domain ---

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// store is an in-memory repository, guarded by a mutex.
type store struct {
	mu     sync.RWMutex
	tasks  map[int]Task
	nextID int
}

func newStore() *store { return &store{tasks: map[int]Task{}, nextID: 1} }

func (s *store) list() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out
}

func (s *store) get(id int) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	return t, ok
}

func (s *store) create(title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := Task{ID: s.nextID, Title: title}
	s.tasks[t.ID] = t
	s.nextID++
	return t
}

// --- Handlers ---
//
// A handler is any func(http.ResponseWriter, *http.Request). Keeping handlers
// as methods on a server struct is how you inject dependencies without globals.

type server struct {
	store  *store
	logger *log.Logger
}

// routes builds the mux. Go 1.22+ patterns support methods and wildcards:
//
//	"GET /tasks/{id}"   -> r.PathValue("id")
//	"POST /tasks"
//	"/"                 -> catch-all
func (s *server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /tasks", s.handleListTasks)
	mux.HandleFunc("POST /tasks", s.handleCreateTask)
	mux.HandleFunc("GET /tasks/{id}", s.handleGetTask)
	mux.HandleFunc("/", s.handleNotFound)

	// Middleware wraps the whole mux. The order here is outermost-first:
	// recoverPanic sees everything, then logging, then the handler.
	return recoverPanic(s.logger)(logging(s.logger)(mux))
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	// Query parameters: /tasks?done=true
	tasks := s.store.list()
	if v := r.URL.Query().Get("done"); v != "" {
		want, err := strconv.ParseBool(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "done must be a boolean")
			return
		}
		filtered := tasks[:0:0]
		for _, t := range tasks {
			if t.Done == want {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks, "count": len(tasks)})
}

func (s *server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id")) // wildcard from the route pattern
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	t, ok := s.store.get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	// ALWAYS bound the body you are willing to read.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

	var req struct {
		Title string `json:"title"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	t := s.store.create(req.Title)
	w.Header().Set("Location", fmt.Sprintf("/tasks/%d", t.ID))
	writeJSON(w, http.StatusCreated, t)
}

func (s *server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "no route for "+r.Method+" "+r.URL.Path)
}

// --- Response helpers: write these once and use them everywhere ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	// Set headers BEFORE WriteHeader; after that they are ignored.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status is already sent, so all you can do is log.
		log.Println("encode response:", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Middleware: a func(http.Handler) http.Handler ---

func logging(l *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r) // call the next handler in the chain
			l.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status,
				time.Since(start).Round(time.Microsecond))
		})
	}
}

// A panic in a handler would otherwise kill the connection (net/http recovers
// the process, but the client gets nothing useful). Convert it to a 500.
func recoverPanic(l *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					l.Printf("PANIC %s %s: %v", r.Method, r.URL.Path, rec)
					writeError(w, http.StatusInternalServerError, "internal error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder captures the status code, which the ResponseWriter does not
// expose. Wrapping the writer like this is a standard middleware technique.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// base is the server's root URL, set once the listener has a port.
var base string

func main() {
	logger := log.New(log.Writer(), "http: ", 0)

	s := &server{store: newStore(), logger: logger}

	// --- Configure the server explicitly. The zero-value http.Server has NO
	//     timeouts, which is how real services get taken down by slow clients. ---
	srv := &http.Server{
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	// Listen on a random free port so this lesson never clashes with anything.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	base = "http://" + ln.Addr().String()
	fmt.Println("listening on", base)

	go func() {
		// Serve returns ErrServerClosed after a graceful Shutdown: not an error.
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// --- Exercise the API against itself ---
	client := &http.Client{Timeout: 3 * time.Second}
	call(client, "GET", base+"/health", "")
	call(client, "POST", base+"/tasks", `{"title":"learn net/http"}`)
	call(client, "POST", base+"/tasks", `{"title":"write tests"}`)
	call(client, "POST", base+"/tasks", `{"title":""}`)
	call(client, "POST", base+"/tasks", `{"nope":1}`)
	call(client, "GET", base+"/tasks", "")
	call(client, "GET", base+"/tasks/1", "")
	call(client, "GET", base+"/tasks/99", "")
	call(client, "GET", base+"/tasks/abc", "")
	call(client, "GET", base+"/nothing-here", "")

	// --- Graceful shutdown: stop accepting, let in-flight requests finish ---
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("shutdown:", err)
	}
	fmt.Println("server stopped cleanly")
}

func call(c *http.Client, method, url, body string) {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		log.Println(err)
		return
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	// Closing the body is mandatory - otherwise you leak the connection.
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	fmt.Printf("  %d %s %s -> %s", resp.StatusCode, method, strings.TrimPrefix(url, base), b)
}
