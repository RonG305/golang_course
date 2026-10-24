package httpapi

import (
	"time"

	"golangcourse/projects/taskapi/internal/domain"
)

// The transport layer defines its own request and response types. Never
// serialise domain entities directly: a rename in the domain would silently
// break your API contract, and internal fields would leak to clients.

type createTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	DueAt       *time.Time `json:"due_at"`
}

type taskResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    string     `json:"priority"`
	Done        bool       `json:"done"`
	Overdue     bool       `json:"overdue"` // computed for the client
	DueAt       *time.Time `json:"due_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// toTaskResponse is the mapping layer between domain and wire format.
func toTaskResponse(t domain.Task, now time.Time) taskResponse {
	return taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Priority:    string(t.Priority),
		Done:        t.Done,
		Overdue:     t.Overdue(now),
		DueAt:       t.DueAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func toTaskResponses(tasks []domain.Task, now time.Time) []taskResponse {
	out := make([]taskResponse, 0, len(tasks)) // never nil: JSON [] not null
	for _, t := range tasks {
		out = append(out, toTaskResponse(t, now))
	}
	return out
}

type listTasksResponse struct {
	Tasks  []taskResponse `json:"tasks"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// errorResponse is the ONE error shape this API ever returns. A consistent
// error body is one of the kindest things you can do for API consumers.
type errorResponse struct {
	Error     string              `json:"error"`
	RequestID string              `json:"request_id,omitempty"`
	Fields    []domain.FieldError `json:"fields,omitempty"`
}
