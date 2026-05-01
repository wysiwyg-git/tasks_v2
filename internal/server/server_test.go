package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wysiwyg-git/tasks_v2/internal/models"
	"github.com/wysiwyg-git/tasks_v2/internal/server"
	"github.com/wysiwyg-git/tasks_v2/internal/store"

	"github.com/go-chi/chi/v5"
)

type mockStore struct {
	tasks      []models.Task
	nextID     int
	failGetAll bool
	failCreate bool
	failUpdate bool
	failDelete bool
}

func (m *mockStore) GetAllTasks(ctx context.Context) ([]models.Task, error) {
	if m.failGetAll {
		return nil, errors.New("fail")
	}
	return m.tasks, nil
}

func (m *mockStore) GetTaskByID(ctx context.Context, id int) (*models.Task, error) {
	for _, t := range m.tasks {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *mockStore) CreateTask(ctx context.Context, title string) (*models.Task, error) {
	if m.failCreate {
		return nil, errors.New("fail")
	}
	if title == "" {
		return nil, errors.New("Title is required")
	}
	m.nextID++
	t := models.Task{ID: m.nextID, Title: title, Done: false}
	m.tasks = append(m.tasks, t)
	return &t, nil
}

func (m *mockStore) UpdateTask(ctx context.Context, id int, title string, done bool) (*models.Task, error) {
	if m.failUpdate {
		return nil, errors.New("fail")
	}
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks[i].Title = title
			m.tasks[i].Done = done
			return &m.tasks[i], nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *mockStore) DeleteTask(ctx context.Context, id int) error {
	if m.failDelete {
		return errors.New("fail")
	}
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}

func setupTestServer() http.Handler {
	ms := &mockStore{
		tasks: []models.Task{
			{ID: 1, Title: "Выучить Go", Done: false},
			{ID: 2, Title: "Написать REST API", Done: false},
		},
		nextID: 2,
	}
	srv := &server.Server{Store: ms}
	r := chi.NewRouter()
	r.Get("/tasks", srv.GetAllTasks)
	r.Post("/tasks", srv.CreateTask)
	r.Get("/tasks/{id}", srv.GetTaskByID)
	r.Put("/tasks/{id}", srv.UpdateTaskByID)
	r.Delete("/tasks/{id}", srv.DeleteTaskByID)
	return r
}

// Table-driven tests for GET /tasks
func TestGetTasksTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func() *mockStore
		expectedStatus int
		expectedLen    int
	}{
		{
			name: "success - returns tasks",
			setupMock: func() *mockStore {
				return &mockStore{
					tasks: []models.Task{
						{ID: 1, Title: "Task 1", Done: false},
						{ID: 2, Title: "Task 2", Done: true},
					},
				}
			},
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name: "success - empty list",
			setupMock: func() *mockStore {
				return &mockStore{tasks: []models.Task{}}
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name: "failure - store error",
			setupMock: func() *mockStore {
				return &mockStore{failGetAll: true}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &server.Server{Store: tt.setupMock()}
			r := chi.NewRouter()
			r.Get("/tasks", srv.GetAllTasks)

			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedLen >= 0 {
				var tasks []models.Task
				if err := json.NewDecoder(w.Body).Decode(&tasks); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(tasks) != tt.expectedLen {
					t.Errorf("expected %d tasks, got %d", tt.expectedLen, len(tasks))
				}
			}
		})
	}
}

// Table-driven tests for POST /tasks
func TestCreateTaskTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		taskTitle      string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success - create task",
			taskTitle:      "New Task",
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var created models.Task
				if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if created.ID == 0 {
					t.Error("expected non-zero ID")
				}
				if created.Title != "New Task" {
					t.Errorf("expected title 'New Task', got '%s'", created.Title)
				}
			},
		},
		{
			name:           "failure - empty title",
			taskTitle:      "",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
		{
			name:           "failure - invalid JSON",
			taskTitle:      "invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestServer()

			var body []byte
			if tt.taskTitle != "invalid" {
				newTask := models.Task{Title: tt.taskTitle}
				body, _ = json.Marshal(newTask)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// Table-driven tests for GET /tasks/{id}
func TestGetTaskByIDTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		taskID         string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success - get existing task",
			taskID:         "1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var task models.Task
				if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if task.ID != 1 {
					t.Errorf("expected task ID 1, got %d", task.ID)
				}
			},
		},
		{
			name:           "failure - task not found",
			taskID:         "999",
			expectedStatus: http.StatusNotFound,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
		{
			name:           "failure - invalid ID",
			taskID:         "abc",
			expectedStatus: http.StatusNotFound,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestServer()

			req := httptest.NewRequest(http.MethodGet, "/tasks/"+tt.taskID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// Table-driven tests for PUT /tasks/{id}
func TestUpdateTaskByIDTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		taskID         string
		taskTitle      string
		taskDone       bool
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success - update task",
			taskID:         "1",
			taskTitle:      "Updated Task",
			taskDone:       true,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var task models.Task
				if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if task.Title != "Updated Task" {
					t.Errorf("expected title 'Updated Task', got '%s'", task.Title)
				}
				if !task.Done {
					t.Error("expected Done=true")
				}
			},
		},
		{
			name:           "failure - task not found",
			taskID:         "999",
			taskTitle:      "Updated Task",
			taskDone:       false,
			expectedStatus: http.StatusNotFound,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
		{
			name:           "failure - empty title",
			taskID:         "1",
			taskTitle:      "",
			taskDone:       false,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
		{
			name:           "failure - invalid JSON",
			taskID:         "1",
			taskTitle:      "invalid",
			taskDone:       false,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestServer()

			var body []byte
			if tt.taskTitle != "invalid" {
				newTask := models.Task{Title: tt.taskTitle, Done: tt.taskDone}
				body, _ = json.Marshal(newTask)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest(http.MethodPut, "/tasks/"+tt.taskID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// Table-driven tests for DELETE /tasks/{id}
func TestDeleteTaskByIDTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		taskID         string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success - delete task",
			taskID:         "1",
			expectedStatus: http.StatusNoContent,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
		{
			name:           "failure - task not found",
			taskID:         "999",
			expectedStatus: http.StatusNotFound,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
		{
			name:           "failure - invalid ID",
			taskID:         "abc",
			expectedStatus: http.StatusNotFound,
			checkResponse:  func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestServer()

			req := httptest.NewRequest(http.MethodDelete, "/tasks/"+tt.taskID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// Test for createTask store error
func TestCreateTaskStoreError(t *testing.T) {
	ms := &mockStore{
		tasks:      []models.Task{},
		nextID:     0,
		failCreate: true,
	}
	srv := &server.Server{Store: ms}
	r := chi.NewRouter()
	r.Post("/tasks", srv.CreateTask)

	newTask := models.Task{Title: "Test task"}
	body, _ := json.Marshal(newTask)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Test for updateTaskByID with store error (non-404)
func TestUpdateTaskByIDStoreError(t *testing.T) {
	ms := &mockStore{
		tasks: []models.Task{
			{ID: 1, Title: "Task 1", Done: false},
		},
		nextID:     1,
		failUpdate: true,
	}
	srv := &server.Server{Store: ms}
	r := chi.NewRouter()
	r.Put("/tasks/{id}", srv.UpdateTaskByID)

	newTask := models.Task{Title: "Updated", Done: true}
	body, _ := json.Marshal(newTask)
	req := httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
