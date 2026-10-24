// Package service holds the use cases: it orchestrates the domain and the
// store, and it is where transactions, authorisation and cross-entity rules
// would live.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golangcourse/projects/taskapi/internal/domain"
)

// TaskStore is declared HERE, by the consumer, listing only what this service
// actually uses. The store package does not know this interface exists, which
// is exactly the point: service has no dependency on any database.
type TaskStore interface {
	Create(ctx context.Context, t domain.Task) (domain.Task, error)
	ByID(ctx context.Context, id string) (domain.Task, error)
	Update(ctx context.Context, t domain.Task) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f domain.Filter) ([]domain.Task, int, error)
}

// Clock is injected so tests control time instead of sleeping. Any dependency
// on the real world - time, randomness, the network - belongs behind a seam.
type Clock func() time.Time

type TaskService struct {
	store  TaskStore
	logger *slog.Logger
	now    Clock
}

// Option configures a TaskService. The options pattern keeps the constructor
// stable while still allowing extras - it is very common in Go libraries.
type Option func(*TaskService)

func WithClock(c Clock) Option {
	return func(s *TaskService) { s.now = c }
}

func New(store TaskStore, logger *slog.Logger, opts ...Option) *TaskService {
	s := &TaskService{store: store, logger: logger, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CreateTaskInput is the service's own input type. Deliberately not the HTTP
// request struct and not the domain entity: each layer owns its own shapes.
type CreateTaskInput struct {
	Title       string
	Description string
	Priority    domain.Priority
	DueAt       *time.Time
}

func (s *TaskService) Create(ctx context.Context, in CreateTaskInput) (domain.Task, error) {
	// Validation belongs to the domain, not to the service or the handler.
	task, err := domain.NewTask(in.Title, in.Description, in.Priority, in.DueAt, s.now())
	if err != nil {
		return domain.Task{}, err // FieldErrors flow up unchanged
	}

	created, err := s.store.Create(ctx, task)
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}

	s.logger.InfoContext(ctx, "task created",
		slog.String("task_id", created.ID),
		slog.String("priority", string(created.Priority)))
	return created, nil
}

func (s *TaskService) Get(ctx context.Context, id string) (domain.Task, error) {
	task, err := s.store.ByID(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task %s: %w", id, err)
	}
	return task, nil
}

func (s *TaskService) List(ctx context.Context, f domain.Filter) ([]domain.Task, int, error) {
	tasks, total, err := s.store.List(ctx, f.Normalise())
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, total, nil
}

// Complete is a read-modify-write use case. In a SQL store this would run in
// a transaction, or use optimistic locking on a version column.
func (s *TaskService) Complete(ctx context.Context, id string) (domain.Task, error) {
	task, err := s.store.ByID(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("complete task %s: %w", id, err)
	}

	// The rule lives on the entity; the service just applies it.
	if err := task.Complete(s.now()); err != nil {
		return domain.Task{}, fmt.Errorf("complete task %s: %w", id, err)
	}

	if err := s.store.Update(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("complete task %s: %w", id, err)
	}

	s.logger.InfoContext(ctx, "task completed", slog.String("task_id", id))
	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task %s: %w", id, err)
	}
	s.logger.InfoContext(ctx, "task deleted", slog.String("task_id", id))
	return nil
}

// OverdueReport shows a use case that spans several records - the kind of
// logic that has no business being in an HTTP handler.
type OverdueReport struct {
	Total   int
	Overdue []domain.Task
}

func (s *TaskService) Overdue(ctx context.Context) (OverdueReport, error) {
	notDone := false
	tasks, total, err := s.store.List(ctx, domain.Filter{Done: &notDone, Limit: 100})
	if err != nil {
		return OverdueReport{}, fmt.Errorf("overdue report: %w", err)
	}

	now := s.now()
	report := OverdueReport{Total: total}
	for _, t := range tasks {
		if t.Overdue(now) {
			report.Overdue = append(report.Overdue, t)
		}
	}
	return report, nil
}
