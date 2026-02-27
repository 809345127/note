package memory

import (
	"context"
	"sync"

	"ddd/examples/minimal-service/domain"
)

type TaskRepository struct {
	mu    sync.RWMutex
	tasks []*domain.Task
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{tasks: make([]*domain.Task, 0)}
}

func (r *TaskRepository) Save(ctx context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks = append(r.tasks, task)
	return nil
}

func (r *TaskRepository) List(ctx context.Context) ([]*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.Task, len(r.tasks))
	copy(result, r.tasks)
	return result, nil
}

var _ domain.Repository = (*TaskRepository)(nil)
