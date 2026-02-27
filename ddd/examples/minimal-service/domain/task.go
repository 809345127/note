package domain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrTaskTitleEmpty = errors.New("task title cannot be empty")

type Task struct {
	id        string
	title     string
	createdAt time.Time
}

func NewTask(title string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrTaskTitleEmpty
	}

	return &Task{
		id:        uuid.NewString(),
		title:     title,
		createdAt: time.Now(),
	}, nil
}

func (t *Task) ID() string {
	return t.id
}

func (t *Task) Title() string {
	return t.title
}

func (t *Task) CreatedAt() time.Time {
	return t.createdAt
}

type Repository interface {
	Save(ctx context.Context, task *Task) error
	List(ctx context.Context) ([]*Task, error)
}
