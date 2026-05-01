package server

import (
	"context"
	"net/http"
	"strings"

	"log/slog"

	"github.com/google/uuid"
	"github.com/wysiwyg-git/tasks_v2/internal/auth"
)

type contextKey string

const requestIDKey contextKey = "requestID"
const userClaimsKey = contextKey("userClaims")

// RequestIDMiddleware — middleware для chi, который внедряет Request ID.
func RequestIDMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" {
				id = uuid.New().String()
			}
			// Кладём в контекст
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			// Кладём в логгер, чтобы он был доступен обработчикам
			loggerWithID := logger.With(slog.String("request_id", id))
			ctx = context.WithValue(ctx, loggerKey{}, loggerWithID) // нужно определить loggerKey
			// Устанавливаем заголовок ответа
			w.Header().Set("X-Request-Id", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// loggerKey — тип для ключа контекста, чтобы избежать коллизий.
type loggerKey struct{}

// GetLogger возвращает логгер из контекста (если есть) или глобальный.
func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	// Fallback: можно вернуть глобальный дефолтный логгер
	return slog.Default()
}

func JWTAuthMiddleware(secret string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing or invalid token", http.StatusUnauthorized)
				return
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.ParseToken(tokenString, secret)
			if err != nil {
				logger.Warn("invalid token", "error", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Добавляем claims в контекст
			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
