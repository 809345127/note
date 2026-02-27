package application

import (
	"context"
	"time"

	"ddd/examples/minimal-service/domain"
)

type TaskService struct {
	repo domain.Repository
}

func NewTaskService(repo domain.Repository) *TaskService {
	return &TaskService{repo: repo}
}

type CreateTaskRequest struct {
	Title string `json:"title" binding:"required"`
}

type TaskResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *TaskService) CreateTask(ctx context.Context, req CreateTaskRequest) (*TaskResponse, error) {
	task, err := domain.NewTask(req.Title)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, task); err != nil {
		return nil, err
	}

	return toResponse(task), nil
}

func (s *TaskService) ListTasks(ctx context.Context) ([]*TaskResponse, error) {
	tasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*TaskResponse, len(tasks))
	for i, task := range tasks {
		result[i] = toResponse(task)
	}

	return result, nil
}

func toResponse(task *domain.Task) *TaskResponse {
	return &TaskResponse{
		ID:        task.ID(),
		Title:     task.Title(),
		CreatedAt: task.CreatedAt(),
	}
}
