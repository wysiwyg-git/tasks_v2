package server

import (
	"context"

	"github.com/wysiwyg-git/tasks_v2/internal/models"
)

type TaskStore interface {
	GetAllTasks(ctx context.Context) ([]models.Task, error)
	GetTaskByID(ctx context.Context, id int) (*models.Task, error)
	CreateTask(ctx context.Context, title string) (*models.Task, error)
	UpdateTask(ctx context.Context, id int, title string, done bool) (*models.Task, error)
	DeleteTask(ctx context.Context, id int) error
}

type Server struct {
	Store TaskStore
}

func NewServer(ts TaskStore) *Server {
	return &Server{
		Store: ts,
	}
}
