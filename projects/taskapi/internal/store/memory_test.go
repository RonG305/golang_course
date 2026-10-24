package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"golangcourse/projects/taskapi/internal/domain"
)

func mustCreate(t *testing.T, s *Memory, title string, done bool, p domain.Priority) domain.Task {
	t.Helper()
	task, err := s.Create(context.Background(), domain.Task{
		Title: title, Done: done, Priority: p, CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create(%q): %v", title, err)
	}
	return task
}

func TestCreateAssignsID(t *testing.T) {
	s := NewMemory()
	a := mustCreate(t, s, "first", false, domain.PriorityLow)
	b := mustCreate(t, s, "second", false, domain.PriorityLow)

	if a.ID == "" || b.ID == "" {
		t.Fatal("Create did not assign IDs")
	}
	if a.ID == b.ID {
		t.Errorf("duplicate IDs: %s", a.ID)
	}
}

func TestByIDNotFound(t *testing.T) {
	s := NewMemory()
	_, err := s.ByID(context.Background(), "nope")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("ByID on missing task = %v, want domain.ErrNotFound", err)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()
	task := mustCreate(t, s, "original", false, domain.PriorityLow)

	task.Title = "changed"
	if err := s.Update(ctx, task); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := s.ByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Title != "changed" {
		t.Errorf("Title = %q, want %q", got.Title, "changed")
	}

	if err := s.Delete(ctx, task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.ByID(ctx, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("task still present after Delete")
	}
	if err := s.Delete(ctx, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second Delete = %v, want domain.ErrNotFound", err)
	}
}

func TestListFiltersAndPages(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()
	mustCreate(t, s, "a", false, domain.PriorityHigh)
	mustCreate(t, s, "b", true, domain.PriorityHigh)
	mustCreate(t, s, "c", false, domain.PriorityLow)

	done := true
	high := domain.PriorityHigh

	tests := []struct {
		name      string
		filter    domain.Filter
		wantLen   int
		wantTotal int
	}{
		{"no filter", domain.Filter{Limit: 10}, 3, 3},
		{"only done", domain.Filter{Done: &done, Limit: 10}, 1, 1},
		{"only high", domain.Filter{Priority: &high, Limit: 10}, 2, 2},
		{"first page", domain.Filter{Limit: 2}, 2, 3},
		{"second page", domain.Filter{Limit: 2, Offset: 2}, 1, 3},
		{"offset past end", domain.Filter{Limit: 2, Offset: 99}, 0, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := s.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestListIsDeterministic(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()
	for _, title := range []string{"a", "b", "c", "d"} {
		mustCreate(t, s, title, false, domain.PriorityLow)
	}
	first, _, _ := s.List(ctx, domain.Filter{Limit: 10})
	for range 20 {
		again, _, _ := s.List(ctx, domain.Filter{Limit: 10})
		for i := range first {
			if first[i].ID != again[i].ID {
				t.Fatal("List order is not stable across calls")
			}
		}
	}
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewMemory()
	if _, err := s.Create(ctx, domain.Task{Title: "x"}); !errors.Is(err, context.Canceled) {
		t.Errorf("Create with cancelled ctx = %v, want context.Canceled", err)
	}
}

// Run this with -race: it is the test that proves the mutex is doing its job.
func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task, err := s.Create(ctx, domain.Task{Title: "t", CreatedAt: time.Now()})
			if err != nil {
				t.Errorf("Create: %v", err)
				return
			}
			if _, err := s.ByID(ctx, task.ID); err != nil {
				t.Errorf("ByID: %v", err)
			}
			if _, _, err := s.List(ctx, domain.Filter{Limit: 5}); err != nil {
				t.Errorf("List: %v", err)
			}
		}(i)
	}
	wg.Wait()

	_, total, _ := s.List(ctx, domain.Filter{Limit: 1})
	if total != 50 {
		t.Errorf("total = %d, want 50 (lost writes mean a broken lock)", total)
	}
}
