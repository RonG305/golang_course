// Package store holds persistence adapters. This one keeps everything in
// memory: it is the implementation used by tests and by `go run` without a
// database, and it documents exactly what a Postgres version must do.
package store

import (
	"context"
	"sort"
	"strconv"
	"sync"

	"golangcourse/projects/taskapi/internal/domain"
)

// Memory is a concurrency-safe in-memory task store.
type Memory struct {
	mu     sync.RWMutex // guards everything below
	tasks  map[string]domain.Task
	nextID int
}

func NewMemory() *Memory {
	return &Memory{tasks: make(map[string]domain.Task), nextID: 1}
}

// Create stores a new task and returns it with its assigned ID.
func (m *Memory) Create(ctx context.Context, t domain.Task) (domain.Task, error) {
	// Even an in-memory store respects cancellation: the real one will too,
	// and the service layer should not behave differently between them.
	if err := ctx.Err(); err != nil {
		return domain.Task{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	t.ID = strconv.Itoa(m.nextID)
	m.nextID++
	m.tasks[t.ID] = t
	return t, nil
}

// ByID returns one task, or domain.ErrNotFound.
func (m *Memory) ByID(ctx context.Context, id string) (domain.Task, error) {
	if err := ctx.Err(); err != nil {
		return domain.Task{}, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.tasks[id]
	if !ok {
		// Wrap the sentinel so the caller gets context AND can still match it.
		return domain.Task{}, domain.ErrNotFound
	}
	return t, nil
}

// Update replaces an existing task.
func (m *Memory) Update(ctx context.Context, t domain.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tasks[t.ID]; !ok {
		return domain.ErrNotFound
	}
	m.tasks[t.ID] = t
	return nil
}

func (m *Memory) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tasks[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}

// List applies the filter and returns a page of tasks plus the total count
// of matches (before paging), which clients need to render pagination.
func (m *Memory) List(ctx context.Context, f domain.Filter) ([]domain.Task, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	matched := make([]domain.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		if f.Done != nil && t.Done != *f.Done {
			continue
		}
		if f.Priority != nil && t.Priority != *f.Priority {
			continue
		}
		matched = append(matched, t)
	}

	// Map iteration is random, so impose a deterministic order: newest first,
	// ID as the tiebreak. An API without a stable sort has broken pagination.
	sort.Slice(matched, func(i, j int) bool {
		if !matched[i].CreatedAt.Equal(matched[j].CreatedAt) {
			return matched[i].CreatedAt.After(matched[j].CreatedAt)
		}
		return matched[i].ID < matched[j].ID
	})

	total := len(matched)
	if f.Offset >= total {
		return []domain.Task{}, total, nil
	}
	end := min(f.Offset+f.Limit, total)
	return matched[f.Offset:end], total, nil
}
