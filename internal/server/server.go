package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/wysiwyg-git/tasks_v2/internal/models"
	"github.com/wysiwyg-git/tasks_v2/internal/store"
)

type TaskStore interface {
	GetAllTasks(ctx context.Context) ([]models.Task, error)
	GetTaskByID(ctx context.Context, id int) (*models.Task, error)
	CreateTask(ctx context.Context, title string) (*models.Task, error)
	UpdateTask(ctx context.Context, id int, title string, done bool) (*models.Task, error)
	DeleteTask(ctx context.Context, id int) error
	CreateUser(ctx context.Context, username, passwordHash string) (*store.User, error)
	GetUserByUsername(ctx context.Context, username string) (*store.User, error)
}

type Server struct {
	Store  TaskStore
	Config struct {
		JWTSecret   string
		JWTTokenTTL time.Duration
	}
	logger *slog.Logger
}

func NewServer(store TaskStore, jwtSecret string, tokenTTL time.Duration, logger *slog.Logger) *Server {
	return &Server{
		Store: store,
		Config: struct {
			JWTSecret   string
			JWTTokenTTL time.Duration
		}{jwtSecret, tokenTTL},
		logger: logger,
	}
}
