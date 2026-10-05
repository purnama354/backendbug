// Package repository menyediakan akses data task.
// Implementasi saat ini memakai memory store, tetapi interface
// TaskRepository dibuat agar mudah diganti dengan database sungguhan.
package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"taskapi/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, t *domain.Task) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	List(ctx context.Context, status string, q string, offset, limit int) ([]*domain.Task, int, error)
	Update(ctx context.Context, t *domain.Task) error
	Delete(ctx context.Context, id int64) error
}

type MemoryTaskRepo struct {
	mu     sync.RWMutex
	tasks  map[int64]*domain.Task
	nextID int64
}

func NewMemoryTaskRepo() *MemoryTaskRepo {
	return &MemoryTaskRepo{
		tasks: make(map[int64]*domain.Task),
	}
}

func (r *MemoryTaskRepo) Create(ctx context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	t.ID = r.nextID
	t.CreatedAt = time.Now()
	t.UpdatedAt = t.CreatedAt
	r.tasks[t.ID] = t
	return nil
}

func (r *MemoryTaskRepo) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("get task %d: %w", id, domain.ErrNotFound)
	}
	cp := *t
	return &cp, nil
}

func (r *MemoryTaskRepo) List(ctx context.Context, status string, q string, offset, limit int) ([]*domain.Task, int, error) {
	r.mu.RLock()
	all := make([]*domain.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		cp := *t
		all = append(all, &cp)
	}
	r.mu.RUnlock()

	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})

	filtered := make([]*domain.Task, 0, len(all))
	for _, t := range all {
		if status != "" && !strings.EqualFold(t.Status, status) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(t.Title), strings.ToLower(q)) {
			continue
		}
		filtered = append(filtered, t)
	}

	total := len(filtered)
	if offset > total {
		return []*domain.Task{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (r *MemoryTaskRepo) Update(ctx context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	old, ok := r.tasks[t.ID]
	if !ok {
		return fmt.Errorf("update task %d: %w", t.ID, domain.ErrNotFound)
	}
	t.CreatedAt = old.CreatedAt
	t.UpdatedAt = time.Now()
	r.tasks[t.ID] = t
	return nil
}

func (r *MemoryTaskRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok {
		return fmt.Errorf("delete task %d: %w", id, domain.ErrNotFound)
	}
	delete(r.tasks, id)
	return nil
}
