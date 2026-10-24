package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// Domain tests are pure: no database, no HTTP, no clock. They run in
// microseconds, which means you can have thousands of them.

func TestNewTaskTrimsAndDefaults(t *testing.T) {
	task, err := NewTask("  spaced out  ", "  body  ", "", nil, now)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	if task.Title != "spaced out" {
		t.Errorf("Title = %q, want it trimmed", task.Title)
	}
	if task.Description != "body" {
		t.Errorf("Description = %q, want it trimmed", task.Description)
	}
	if task.Priority != PriorityMedium {
		t.Errorf("Priority = %q, want the default %q", task.Priority, PriorityMedium)
	}
	if !task.CreatedAt.Equal(now) || !task.UpdatedAt.Equal(now) {
		t.Error("timestamps were not set from the injected time")
	}
}

func TestNewTaskValidation(t *testing.T) {
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name        string
		title       string
		description string
		priority    Priority
		dueAt       *time.Time
		wantFields  []string
	}{
		{name: "valid", title: "ok", priority: PriorityHigh, dueAt: &future},
		{name: "blank title", title: "   ", wantFields: []string{"title"}},
		{name: "title too long", title: strings.Repeat("x", 201), wantFields: []string{"title"}},
		{name: "description too long", title: "ok", description: strings.Repeat("x", 2001),
			wantFields: []string{"description"}},
		{name: "unknown priority", title: "ok", priority: "critical",
			wantFields: []string{"priority"}},
		{name: "due in the past", title: "ok", dueAt: &past, wantFields: []string{"due_at"}},
		{name: "all at once", title: "", priority: "nope", dueAt: &past,
			wantFields: []string{"title", "priority", "due_at"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewTask(tt.title, tt.description, tt.priority, tt.dueAt, now)

			if len(tt.wantFields) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			var fieldErrs FieldErrors
			if !errors.As(err, &fieldErrs) {
				t.Fatalf("err = %v, want FieldErrors", err)
			}
			got := map[string]bool{}
			for _, fe := range fieldErrs {
				got[fe.Field] = true
			}
			for _, want := range tt.wantFields {
				if !got[want] {
					t.Errorf("missing error for %q; got %v", want, fieldErrs)
				}
			}
		})
	}
}

func TestCompleteAndReopen(t *testing.T) {
	task, err := NewTask("cycle me", "", PriorityLow, nil, now)
	if err != nil {
		t.Fatal(err)
	}

	later := now.Add(time.Hour)
	if err := task.Complete(later); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !task.Done {
		t.Error("Done is false after Complete")
	}
	if !task.UpdatedAt.Equal(later) {
		t.Errorf("UpdatedAt = %v, want %v", task.UpdatedAt, later)
	}

	if err := task.Complete(later); !errors.Is(err, ErrAlreadyDone) {
		t.Errorf("Complete twice = %v, want ErrAlreadyDone", err)
	}

	if err := task.Reopen(later); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if task.Done {
		t.Error("Done is true after Reopen")
	}
	if err := task.Reopen(later); !errors.Is(err, ErrNotDone) {
		t.Errorf("Reopen twice = %v, want ErrNotDone", err)
	}
}

func TestOverdue(t *testing.T) {
	past, future := now.Add(-time.Hour), now.Add(time.Hour)

	tests := []struct {
		name  string
		dueAt *time.Time
		done  bool
		want  bool
	}{
		{"no due date", nil, false, false},
		{"due in the future", &future, false, false},
		{"due in the past", &past, false, true},
		{"past due but completed", &past, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{DueAt: tt.dueAt, Done: tt.done}
			if got := task.Overdue(now); got != tt.want {
				t.Errorf("Overdue() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestFilterNormalise(t *testing.T) {
	tests := []struct {
		name                  string
		in                    Filter
		wantLimit, wantOffset int
	}{
		{"zero gets the default", Filter{}, 20, 0},
		{"negative limit", Filter{Limit: -5}, 20, 0},
		{"huge limit is capped", Filter{Limit: 10_000}, 100, 0},
		{"negative offset", Filter{Limit: 10, Offset: -3}, 10, 0},
		{"valid values pass through", Filter{Limit: 50, Offset: 100}, 50, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Normalise()
			if got.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", got.Offset, tt.wantOffset)
			}
		})
	}
}

func TestPriorityValid(t *testing.T) {
	for _, p := range []Priority{PriorityLow, PriorityMedium, PriorityHigh} {
		if !p.Valid() {
			t.Errorf("%q should be valid", p)
		}
	}
	for _, p := range []Priority{"", "URGENT", "Low", "critical"} {
		if p.Valid() {
			t.Errorf("%q should not be valid", p)
		}
	}
}

func TestFieldErrorsMessage(t *testing.T) {
	errs := FieldErrors{}.Add("title", "is required").Add("age", "must be positive")
	msg := errs.Error()
	for _, want := range []string{"title is required", "age must be positive"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, want it to contain %q", msg, want)
		}
	}
}
