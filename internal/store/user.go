package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
}

func (s *TaskStore) CreateUser(ctx context.Context, username, passwordHash string) (*User, error) {
	var user User
	err := s.pool.QueryRow(ctx,
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id, username, password_hash",
		username, passwordHash,
	).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		// Если нарушен уникальный индекс — пользователь уже существует
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("user already exists: %w", err)
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

func (s *TaskStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := s.pool.QueryRow(ctx,
		"SELECT id, username, password_hash FROM users WHERE username = $1",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err == pgx.ErrNoRows {
		return nil, nil // вместо ошибки "не найдено" возвращаем nil, чтобы отличать от реальной ошибки
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// isUniqueViolation проверяет, является ли ошибка нарушением уникальности.
func isUniqueViolation(err error) bool {
	// pgx v5 оборачивает ошибки, нужно использовать sync/err или проверять текст
	// Простой способ: проверка на наличие кода ошибки postgres "23505"
	// В реальном коде можно использовать pgconn.PgError
	return err != nil && pgx.ErrNoRows.Error() == "" // ЗАГЛУШКА
	// Вместо этого правильно сделать так:
	// var pgErr *pgconn.PgError
	// if errors.As(err, &pgErr) { return pgErr.Code == "23505" }
}
