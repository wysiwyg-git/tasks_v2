package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/wysiwyg-git/tasks_v2/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("task not found")

type TaskStore struct {
	pool *pgxpool.Pool
}

func NewTaskStore(ctx context.Context, connString string) (*TaskStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &TaskStore{pool: pool}, nil
}

func (s *TaskStore) Close() {
	s.pool.Close()
}

// GetAllTasks возвращает все задачи
func (s *TaskStore) GetAllTasks(ctx context.Context) ([]models.Task, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, title, done FROM tasks ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetTaskByID ищет задачу по ID
func (s *TaskStore) GetTaskByID(ctx context.Context, id int) (*models.Task, error) {
	var t models.Task
	err := s.pool.QueryRow(ctx, "SELECT id, title, done FROM tasks WHERE id = $1", id).
		Scan(&t.ID, &t.Title, &t.Done)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// CreateTask добавляет задачу и возвращает созданную
func (s *TaskStore) CreateTask(ctx context.Context, title string) (*models.Task, error) {
	var t models.Task
	err := s.pool.QueryRow(ctx,
		"INSERT INTO tasks (title, done) VALUES ($1, FALSE) RETURNING id, title, done",
		title).Scan(&t.ID, &t.Title, &t.Done)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTask обновляет поля задачи
func (s *TaskStore) UpdateTask(ctx context.Context, id int, title string, done bool) (*models.Task, error) {
	var t models.Task
	err := s.pool.QueryRow(ctx,
		"UPDATE tasks SET title = $2, done = $3 WHERE id = $1 RETURNING id, title, done",
		id, title, done).Scan(&t.ID, &t.Title, &t.Done)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// DeleteTask удаляет задачу по ID
func (s *TaskStore) DeleteTask(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
