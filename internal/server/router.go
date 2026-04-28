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

	r.Route("/tasks", func(r chi.Router) {
		r.Get("/", srv.GetTasks)
		r.Post("/", srv.CreateTask)
		r.Get("/{id}", srv.GetTaskByID)
		r.Put("/{id}", srv.UpdateTaskByID)
		r.Delete("/{id}", srv.DeleteTaskByID)
	})
	r.Get("/health", srv.HealthCheck)

	return r
}
