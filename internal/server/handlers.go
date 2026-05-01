package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/wysiwyg-git/tasks_v2/internal/models"
	"github.com/wysiwyg-git/tasks_v2/internal/store"

	"github.com/go-chi/chi/v5"
)

func (s *Server) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())

	tasks, err := s.Store.GetAllTasks(r.Context())
	if err != nil {
		logger.Error("failed to get tasks", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())

	id, err := parseIDFromURL(r)
	if err != nil {
		logger.Error("failed to parse task ID", "error", err)
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	task, err := s.Store.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			logger.Error("failed to get task by id", "error", err)
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) CreateTask(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())

	var newTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if newTask.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	task, err := s.Store.CreateTask(r.Context(), newTask.Title)
	if err != nil {
		logger.Error("failed to create task", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/tasks/%d", task.ID))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (s *Server) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())

	id, err := parseIDFromURL(r)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	var newTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if newTask.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	task, err := s.Store.UpdateTask(r.Context(), id, newTask.Title, newTask.Done)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			logger.Error("failed to get task by id", "error", err)
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		logger.Error("failed to update task", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/tasks/%d", task.ID))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

func (s *Server) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	logger := GetLogger(r.Context())

	id, err := parseIDFromURL(r)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	err = s.Store.DeleteTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		logger.Error("failed to delete task", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func parseIDFromURL(r *http.Request) (int, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("Invalid task ID")
	}
	return id, nil
}
