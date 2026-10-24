// Lesson 26: calling HTTP APIs properly.
//
// The default http.Get is fine for a script and dangerous in a service: no
// timeout means a hung server can block your goroutine forever. This lesson
// builds a client the way you should for production.
//
// It spins up a local test server (net/http/httptest) so everything here runs
// offline and deterministically.
//
// Run: go run ./lessons/26_http_client
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"
)

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
}

// Retryable tells the retry loop whether another attempt could help.
func (e *APIError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// --- A typed client. This is the shape to copy for any API wrapper. ---

type Client struct {
	baseURL string
	http    *http.Client
	token   string
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		// Build the transport explicitly: connection pooling settings matter
		// a lot under load, and the defaults are conservative.
		http: &http.Client{
			Timeout: 10 * time.Second, // covers dial + write + read + redirects
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10, // default is 2: raise it for busy APIs
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ForceAttemptHTTP2:     true,
			},
		},
	}
}

// do performs the request and decodes JSON into out (which may be nil).
// Every method goes through here, so auth, error mapping and body handling
// are written exactly once.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reader = strings.NewReader(string(b))
	}

	// NewRequestWithContext, always: this is what makes the call cancellable.
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	// Close the body on EVERY path. Draining it first lets the connection be
	// reused instead of being thrown away.
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(msg))}
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) GetTask(ctx context.Context, id int) (Task, error) {
	var t Task
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/tasks/%d", id), nil, &t)
	return t, err
}

func (c *Client) CreateTask(ctx context.Context, title string) (Task, error) {
	var t Task
	err := c.do(ctx, http.MethodPost, "/tasks", map[string]string{"title": title}, &t)
	return t, err
}

// ListTasks shows query-string construction. Use url.Values, never fmt.Sprintf
// with raw user input - it needs escaping.
func (c *Client) ListTasks(ctx context.Context, search string, limit int) ([]Task, error) {
	q := url.Values{}
	if search != "" {
		q.Set("q", search)
	}
	q.Set("limit", fmt.Sprint(limit))

	var resp struct {
		Tasks []Task `json:"tasks"`
	}
	err := c.do(ctx, http.MethodGet, "/tasks?"+q.Encode(), nil, &resp)
	return resp.Tasks, err
}

// GetTaskWithRetry retries only on errors that could plausibly succeed later.
func (c *Client) GetTaskWithRetry(ctx context.Context, id, attempts int) (Task, error) {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		t, err := c.GetTask(ctx, id)
		if err == nil {
			return t, nil
		}
		lastErr = err

		var apiErr *APIError
		if errors.As(err, &apiErr) && !apiErr.Retryable() {
			return Task{}, err // a 404 or 400 will never fix itself
		}
		if attempt == attempts {
			break
		}

		backoff := time.Duration(1<<(attempt-1)) * 10 * time.Millisecond
		jitter := time.Duration(rand.Int63n(int64(backoff) + 1))
		fmt.Printf("    attempt %d failed (%v), retrying in ~%v\n", attempt, err, backoff)
		select {
		case <-time.After(backoff + jitter):
		case <-ctx.Done():
			return Task{}, ctx.Err()
		}
	}
	return Task{}, fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}

func main() {
	srv := newFakeAPI()
	defer srv.Close()

	c := NewClient(srv.URL, "secret-token")
	ctx := context.Background()

	// --- Happy path ---
	t, err := c.CreateTask(ctx, "ship the service")
	fmt.Println("created:", t, "err:", err)

	t, err = c.GetTask(ctx, 1)
	fmt.Println("fetched:", t, "err:", err)

	tasks, err := c.ListTasks(ctx, "ship it", 10)
	fmt.Println("listed:", len(tasks), "tasks, err:", err)

	// --- Error handling: inspect the typed error ---
	_, err = c.GetTask(ctx, 404)
	fmt.Println("not found:", err)
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		fmt.Println("  status:", apiErr.StatusCode, "retryable:", apiErr.Retryable())
	}

	// --- Retry on a flaky endpoint (fails twice, then succeeds) ---
	fmt.Println("retrying a flaky endpoint:")
	t, err = c.GetTaskWithRetry(ctx, 500, 4)
	fmt.Println("  result:", t, "err:", err)

	// --- Timeout via context: the caller sets the budget ---
	slowCtx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	_, err = c.GetTask(slowCtx, 999) // the fake server sleeps 200ms
	fmt.Println("context timeout:", err)
	fmt.Println("  is DeadlineExceeded:", errors.Is(err, context.DeadlineExceeded))

	// --- Cancelling in flight ---
	cancelCtx, cancelNow := context.WithCancel(ctx)
	go func() { time.Sleep(20 * time.Millisecond); cancelNow() }()
	_, err = c.GetTask(cancelCtx, 999)
	fmt.Println("cancelled:", errors.Is(err, context.Canceled))

	// --- Concurrent requests, bounded by a semaphore ---
	fmt.Println("concurrent fetches:", concurrentFetch(ctx, c, []int{1, 2, 3, 4, 5}, 2))

	// --- Streaming a large response instead of buffering it ---
	fmt.Println("streamed bytes:", streamDownload(ctx, srv.URL+"/big"))
}

// newFakeAPI stands in for a real service. httptest.Server is also exactly
// how you test YOUR client code - no network, no mocks of http internals.
func newFakeAPI() *httptest.Server {
	var flakyHits int
	mux := http.NewServeMux()

	mux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Task{ID: 1, Title: body["title"]})
	})

	mux.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		// Show that the query string arrived correctly escaped.
		fmt.Printf("    server saw q=%q limit=%q\n",
			r.URL.Query().Get("q"), r.URL.Query().Get("limit"))
		_ = json.NewEncoder(w).Encode(map[string][]Task{
			"tasks": {{ID: 1, Title: "a"}, {ID: 2, Title: "b"}},
		})
	})

	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "404":
			http.Error(w, "task not found", http.StatusNotFound)
		case "500":
			flakyHits++
			if flakyHits < 3 { // fail twice, then succeed
				http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
				return
			}
			_ = json.NewEncoder(w).Encode(Task{ID: 500, Title: "recovered"})
		case "999":
			time.Sleep(200 * time.Millisecond) // too slow on purpose
			_ = json.NewEncoder(w).Encode(Task{ID: 999})
		default:
			_ = json.NewEncoder(w).Encode(Task{ID: 1, Title: "ship the service"})
		}
	})

	mux.HandleFunc("GET /big", func(w http.ResponseWriter, r *http.Request) {
		for range 100 {
			_, _ = w.Write([]byte(strings.Repeat("x", 100)))
		}
	})

	return httptest.NewServer(mux)
}

func concurrentFetch(ctx context.Context, c *Client, ids []int, limit int) int {
	sem := make(chan struct{}, limit)
	results := make(chan Task, len(ids))
	for _, id := range ids {
		go func(id int) {
			sem <- struct{}{}
			defer func() { <-sem }()
			t, err := c.GetTask(ctx, id)
			if err != nil {
				log.Println("fetch:", err)
				results <- Task{}
				return
			}
			results <- t
		}(id)
	}
	got := 0
	for range ids {
		if (<-results).ID != 0 {
			got++
		}
	}
	return got
}

// streamDownload reads the body in chunks. Never io.ReadAll an unbounded
// response: a large or hostile body will exhaust your memory.
func streamDownload(ctx context.Context, url string) int64 {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return 0
	}
	defer resp.Body.Close()
	n, _ := io.Copy(io.Discard, resp.Body) // in real code: a file, a hash, a parser
	return n
}
