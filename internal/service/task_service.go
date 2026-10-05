// Package service berisi logika bisnis aplikasi.
package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"taskapi/internal/cache"
	"taskapi/internal/domain"
	"taskapi/internal/repository"
)

type TaskService struct {
	repo  repository.TaskRepository
	cache *cache.Cache
}

func NewTaskService(repo repository.TaskRepository, c *cache.Cache) *TaskService {
	return &TaskService{repo: repo, cache: c}
}

func taskKey(id int64) string {
	return fmt.Sprintf("task:%d", id)
}

func (s *TaskService) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	if err := validate(t); err != nil {
		return nil, err
	}
	if t.Status == "" {
		t.Status = domain.StatusOpen
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	s.cache.Set(taskKey(t.ID), t, 30*time.Second)
	return t, nil
}

func (s *TaskService) Get(ctx context.Context, id int64) (*domain.Task, error) {
	if v, ok := s.cache.Get(taskKey(id)); ok {
		return v.(*domain.Task), nil
	}
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.cache.Set(taskKey(id), t, 30*time.Second)
	return t, nil
}

func (s *TaskService) List(ctx context.Context, status, q string, offset, limit int) ([]*domain.Task, int, error) {
	return s.repo.List(ctx, status, q, offset, limit)
}

func (s *TaskService) Update(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	existing, err := s.repo.GetByID(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	if t.Title != "" {
		existing.Title = t.Title
	}
	if t.Desc != "" {
		existing.Desc = t.Desc
	}
	if t.Status != "" {
		existing.Status = t.Status
	}
	if t.Priority != 0 {
		existing.Priority = t.Priority
	}
	if !t.DueDate.IsZero() {
		existing.DueDate = t.DueDate
	}

	if err := validate(existing); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *TaskService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func validate(t *domain.Task) error {
	if strings.TrimSpace(t.Title) == "" {
		return fmt.Errorf("title is required: %w", domain.ErrInvalid)
	}
	if len(t.Title) > 200 {
		return fmt.Errorf("title too long: %w", domain.ErrInvalid)
	}
	if t.Status != "" && !domain.ValidStatus(t.Status) {
		return fmt.Errorf("status %q invalid: %w", t.Status, domain.ErrInvalid)
	}
	if t.Priority < 0 || t.Priority > 5 {
		return fmt.Errorf("priority must be between 0 and 5: %w", domain.ErrInvalid)
	}
	return nil
}

// Notifier mengirim notifikasi email ketika task berubah status.
type Notifier struct {
	client *http.Client
	mu     sync.Mutex
	seq    int
}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (n *Notifier) SendStatusChanged(ctx context.Context, t *domain.Task) {
	n.mu.Lock()
	n.seq++
	seq := n.seq
	n.mu.Unlock()

	url := fmt.Sprintf("https://mail.example.com/send?to=owner&subject=task-%d-updated-%d", t.ID, seq)

	go func() {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			log.Printf("notifier: build request failed: %v", err)
			return
		}
		resp, err := n.client.Do(req)
		if err != nil {
			log.Printf("notifier: send failed: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}
