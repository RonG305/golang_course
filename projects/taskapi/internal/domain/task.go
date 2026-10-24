// Package domain holds the entities and business rules of the application.
//
// It is the innermost layer, so it imports NOTHING from the other layers and
// nothing framework-specific - no HTTP, no SQL, no JSON concerns. That is what
// keeps the rules testable and lets you swap transports and databases.
package domain

import (
	"strings"
	"time"
)

// Priority is a domain enum.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

// Task is the central entity.
type Task struct {
	ID          string
	Title       string
	Description string
	Priority    Priority
	Done        bool
	DueAt       *time.Time // nil means "no due date"
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTask builds a valid Task or explains why it cannot. Constructing through
// a function like this means an invalid Task can never exist in the system.
func NewTask(title, description string, priority Priority, dueAt *time.Time, now time.Time) (Task, error) {
	title = strings.TrimSpace(title)

	// Collect every problem, so the API can report them all at once.
	var errs FieldErrors
	if title == "" {
		errs = errs.Add("title", "is required")
	}
	if len(title) > 200 {
		errs = errs.Add("title", "must be at most 200 characters")
	}
	if len(description) > 2000 {
		errs = errs.Add("description", "must be at most 2000 characters")
	}
	if priority == "" {
		priority = PriorityMedium // a sensible default, not an error
	}
	if !priority.Valid() {
		errs = errs.Add("priority", "must be one of low, medium, high")
	}
	if dueAt != nil && dueAt.Before(now) {
		errs = errs.Add("due_at", "must be in the future")
	}
	if len(errs) > 0 {
		return Task{}, errs
	}

	return Task{
		Title:       title,
		Description: strings.TrimSpace(description),
		Priority:    priority,
		DueAt:       dueAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Complete marks the task done. Business rules live on the entity, which is
// why this is a method and not a field assignment in a handler.
func (t *Task) Complete(now time.Time) error {
	if t.Done {
		return ErrAlreadyDone
	}
	t.Done = true
	t.UpdatedAt = now
	return nil
}

// Reopen is the inverse operation.
func (t *Task) Reopen(now time.Time) error {
	if !t.Done {
		return ErrNotDone
	}
	t.Done = false
	t.UpdatedAt = now
	return nil
}

// Overdue reports whether the task missed its due date.
func (t Task) Overdue(now time.Time) bool {
	return !t.Done && t.DueAt != nil && t.DueAt.Before(now)
}

// Filter describes a query. Pointer fields distinguish "not filtering on this"
// from "filtering on the zero value".
type Filter struct {
	Done     *bool
	Priority *Priority
	Limit    int
	Offset   int
}

// Normalise applies safe defaults and caps, so a client cannot ask for a
// million rows.
func (f Filter) Normalise() Filter {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return f
}
