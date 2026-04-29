package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wysiwyg-git/tasks_v2/internal/config"
	"github.com/wysiwyg-git/tasks_v2/internal/logger"
	"github.com/wysiwyg-git/tasks_v2/internal/migration"
	"github.com/wysiwyg-git/tasks_v2/internal/server"
	"github.com/wysiwyg-git/tasks_v2/internal/store"
)

// Run инициализирует и запускает весь сервис.
func Run() error {
	// 1. Конфигурация
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Создаём логгер
	appLogger := logger.New(logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	// Запуск миграций
	if err := migration.RunMigrations(cfg.DatabaseURL); err != nil {
		appLogger.Error("running migrations", "error", err)
		return fmt.Errorf("running migrations: %w", err)
	}

	// 2. Подключение к БД
	ts, err := store.NewTaskStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("Failed to create task store: %v", err)
	}
	defer ts.Close()

	// 3. Слои приложения
	srv := server.NewServer(ts)

	// 4. HTTP-роутер
	r := server.NewRouter(srv, appLogger)

	// 5. HTTP-сервер с таймаутами
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		appLogger.Info("Server starting", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
