package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"golangcourse/projects/taskapi/internal/domain"
	"golangcourse/projects/taskapi/internal/store"
)

// A fixed clock: tests that depend on time.Now() are flaky tests.
var fixedNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func newTestService(t *testing.T) *TaskService {
	t.Helper()
	// The real in-memory store is a better collaborator than a fake here: it
	// is fast, and it exercises the actual interface contract.
	return New(
		store.NewMemory(),
		slog.New(slog.NewTextHandler(io.Discard, nil)), // silence logs in tests
		WithClock(func() time.Time { return fixedNow }),
	)
}

func TestCreateValidation(t *testing.T) {
	future := fixedNow.Add(24 * time.Hour)
	past := fixedNow.Add(-24 * time.Hour)

	tests := []struct {
		name       string
		in         CreateTaskInput
		wantFields []string // which fields should be reported as invalid
	}{
		{
			name: "valid",
			in:   CreateTaskInput{Title: "write the service", Priority: domain.PriorityHigh, DueAt: &future},
		},
		{
			name: "defaults priority to medium",
			in:   CreateTaskInput{Title: "no priority given"},
		},
		{
			name:       "empty title",
			in:         CreateTaskInput{Title: "   "},
			wantFields: []string{"title"},
		},
		{
			name:       "bad priority",
			in:         CreateTaskInput{Title: "ok", Priority: "urgent"},
			wantFields: []string{"priority"},
		},
		{
			name:       "due date in the past",
			in:         CreateTaskInput{Title: "ok", DueAt: &past},
			wantFields: []string{"due_at"},
		},
		{
			name:       "several problems at once",
			in:         CreateTaskInput{Title: "", Priority: "nope", DueAt: &past},
			wantFields: []string{"title", "priority", "due_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newTestService(t)

			got, err := svc.Create(context.Background(), tt.in)

			if len(tt.wantFields) == 0 {
				if err != nil {
					t.Fatalf("Create(%+v) = unexpected error %v", tt.in, err)
				}
				if got.ID == "" {
					t.Error("created task has no ID")
				}
				if got.Priority == "" {
					t.Error("priority was not defaulted")
				}
				return
			}

			var fieldErrs domain.FieldErrors
			if !errors.As(err, &fieldErrs) {
				t.Fatalf("Create(%+v) = %v, want domain.FieldErrors", tt.in, err)
			}
			gotFields := map[string]bool{}
			for _, fe := range fieldErrs {
				gotFields[fe.Field] = true
			}
			for _, want := range tt.wantFields {
				if !gotFields[want] {
					t.Errorf("no validation error for %q; got %v", want, fieldErrs)
				}
			}
		})
	}
}

func TestCompleteLifecycle(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	created, err := svc.Create(ctx, CreateTaskInput{Title: "ship it"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	done, err := svc.Complete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !done.Done {
		t.Error("task is not marked done")
	}
	if !done.UpdatedAt.Equal(fixedNow) {
		t.Errorf("UpdatedAt = %v, want %v", done.UpdatedAt, fixedNow)
	}

	// Completing twice must be rejected by the domain rule.
	if _, err := svc.Complete(ctx, created.ID); !errors.Is(err, domain.ErrAlreadyDone) {
		t.Errorf("second Complete = %v, want domain.ErrAlreadyDone", err)
	}

	// The change was persisted, not just returned.
	reloaded, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reloaded.Done {
		t.Error("completion was not persisted")
	}
}

func TestGetAndDeleteMissing(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	// The sentinel survives being wrapped by both the store and the service.
	if _, err := svc.Get(ctx, "404"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get = %v, want domain.ErrNotFound", err)
	}
	if err := svc.Delete(ctx, "404"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete = %v, want domain.ErrNotFound", err)
	}
}

func TestListAppliesDefaultLimit(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	for range 25 {
		if _, err := svc.Create(ctx, CreateTaskInput{Title: "bulk"}); err != nil {
			t.Fatal(err)
		}
	}

	// Filter{} has Limit 0, which Normalise turns into 20.
	tasks, total, err := svc.List(ctx, domain.Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 20 {
		t.Errorf("page size = %d, want the default 20", len(tasks))
	}
	if total != 25 {
		t.Errorf("total = %d, want 25", total)
	}

	// And an absurd limit is capped rather than honoured.
	tasks, _, _ = svc.List(ctx, domain.Filter{Limit: 100000})
	if len(tasks) > 100 {
		t.Errorf("limit was not capped: got %d", len(tasks))
	}
}

func TestOverdueReport(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	soon := fixedNow.Add(time.Hour)
	if _, err := svc.Create(ctx, CreateTaskInput{Title: "future", DueAt: &soon}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateTaskInput{Title: "no due date"}); err != nil {
		t.Fatal(err)
	}

	// Nothing is overdue at fixedNow.
	report, err := svc.Overdue(ctx)
	if err != nil {
		t.Fatalf("Overdue: %v", err)
	}
	if len(report.Overdue) != 0 {
		t.Errorf("overdue = %d, want 0", len(report.Overdue))
	}

	// Advance the injected clock two days: now the due task is overdue.
	// This is why the clock is a dependency - no sleeping, no flakiness.
	svc.now = func() time.Time { return fixedNow.Add(48 * time.Hour) }
	report, err = svc.Overdue(ctx)
	if err != nil {
		t.Fatalf("Overdue: %v", err)
	}
	if len(report.Overdue) != 1 {
		t.Fatalf("overdue = %d, want 1", len(report.Overdue))
	}
	if report.Overdue[0].Title != "future" {
		t.Errorf("overdue task = %q", report.Overdue[0].Title)
	}
	if report.Total != 2 {
		t.Errorf("total = %d, want 2", report.Total)
	}
}

// --- A fake store, to show how you test failure paths that a real store
//     will not produce on demand ---

type failingStore struct {
	TaskStore // embed the interface: only override what this test needs
	err       error
}

func (f failingStore) Create(context.Context, domain.Task) (domain.Task, error) {
	return domain.Task{}, f.err
}

func TestCreateWrapsStoreError(t *testing.T) {
	boom := errors.New("database is on fire")
	svc := New(
		failingStore{err: boom},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithClock(func() time.Time { return fixedNow }),
	)

	_, err := svc.Create(context.Background(), CreateTaskInput{Title: "valid title"})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
	// And the wrapping added useful context rather than replacing the cause.
	if got := err.Error(); got == boom.Error() {
		t.Error("service did not add any context to the store error")
	}
}
