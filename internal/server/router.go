package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(srv *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(RequestIDMiddleware(srv.logger))

	// Публичные эндпоинты
	r.Post("/register", srv.Register)
	r.Post("/login", srv.Login)

	// Защищённые
	r.Group(func(r chi.Router) {
		r.Use(JWTAuthMiddleware(srv.Config.JWTSecret, srv.logger))
		r.Get("/tasks", srv.GetAllTasks)
		r.Post("/tasks", srv.CreateTask)
		r.Get("/tasks/{id}", srv.GetTaskByID)
		r.Put("/tasks/{id}", srv.UpdateTaskByID)
		r.Delete("/tasks/{id}", srv.DeleteTaskByID)
	})

	r.Get("/health", srv.HealthCheck)
	return r
}
