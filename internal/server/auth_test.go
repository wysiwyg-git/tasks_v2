package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wysiwyg-git/tasks_v2/internal/auth"
	"github.com/wysiwyg-git/tasks_v2/internal/models"
	"github.com/wysiwyg-git/tasks_v2/internal/store"
)

// -------------------------------------------------------
// Mock Store для тестов аутентификации
// -------------------------------------------------------

type authMockStore struct {
	users map[string]*store.User
}

func newAuthMockStore() *authMockStore {
	return &authMockStore{
		users: make(map[string]*store.User),
	}
}

var errUserExists = errors.New("user already exists")

func (m *authMockStore) CreateUser(ctx context.Context, username, passwordHash string) (*store.User, error) {
	if _, exists := m.users[username]; exists {
		return nil, errUserExists
	}
	user := &store.User{
		ID:           len(m.users) + 1,
		Username:     username,
		PasswordHash: passwordHash,
	}
	m.users[username] = user
	return user, nil
}

func (m *authMockStore) GetUserByUsername(ctx context.Context, username string) (*store.User, error) {
	user, ok := m.users[username]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *authMockStore) GetAllTasks(ctx context.Context) ([]models.Task, error) {
	return nil, nil
}

func (m *authMockStore) GetTaskByID(ctx context.Context, id int) (*models.Task, error) {
	return nil, store.ErrNotFound
}

func (m *authMockStore) CreateTask(ctx context.Context, title string) (*models.Task, error) {
	return nil, errors.New("not implemented")
}

func (m *authMockStore) UpdateTask(ctx context.Context, id int, title string, done bool) (*models.Task, error) {
	return nil, errors.New("not implemented")
}

func (m *authMockStore) DeleteTask(ctx context.Context, id int) error {
	return errors.New("not implemented")
}

// -------------------------------------------------------
// Вспомогательные конструкции для тестов аутентификации
// -------------------------------------------------------

// newAuthTestServer создаёт сервер с тестовым JWT-секретом и новым mockStore.
func newAuthTestServer() (*Server, *authMockStore) {
	ms := newAuthMockStore()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	srv := NewServer(ms, "test-secret", time.Hour, logger)
	return srv, ms
}

// newAuthRouter создаёт chi-роутер только с эндпоинтами Register и Login.
func newAuthRouter(srv *Server) http.Handler {
	r := chi.NewRouter()
	r.Post("/register", srv.Register)
	r.Post("/login", srv.Login)
	return r
}

// -------------------------------------------------------
// Тесты Register
// -------------------------------------------------------

func TestRegister(t *testing.T) {
	srv, _ := newAuthTestServer()
	router := newAuthRouter(srv)

	tests := []struct {
		name         string
		body         string
		expectedCode int
		checkToken   bool
	}{
		{
			name:         "successful registration",
			body:         `{"username":"alice","password":"pass123"}`,
			expectedCode: http.StatusOK,
			checkToken:   true,
		},
		{
			name:         "empty username",
			body:         `{"username":"","password":"pass"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "empty password",
			body:         `{"username":"bob","password":""}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid json",
			body:         `invalid`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			assert.Equal(t, tc.expectedCode, w.Code)

			if tc.checkToken {
				var resp AuthResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Token)

				// дополнительно проверим, что токен валиден
				claims, err := auth.ParseToken(resp.Token, "test-secret")
				require.NoError(t, err)
				assert.Equal(t, "alice", claims.Username)
			}
		})
	}
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	srv, ms := newAuthTestServer()
	// заранее добавляем пользователя в хранилище
	hash, err := auth.HashPassword("pass")
	require.NoError(t, err)
	ms.users["alice"] = &store.User{ID: 1, Username: "alice", PasswordHash: hash}

	router := newAuthRouter(srv)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"alice","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// -------------------------------------------------------
// Тесты Login
// -------------------------------------------------------

func TestLogin(t *testing.T) {
	srv, ms := newAuthTestServer()
	// подготавливаем пользователя
	hash, err := auth.HashPassword("pass123")
	require.NoError(t, err)
	ms.users["alice"] = &store.User{ID: 1, Username: "alice", PasswordHash: hash}

	router := newAuthRouter(srv)

	tests := []struct {
		name         string
		body         string
		expectedCode int
	}{
		{
			name:         "successful login",
			body:         `{"username":"alice","password":"pass123"}`,
			expectedCode: http.StatusOK,
		},
		{
			name:         "wrong password",
			body:         `{"username":"alice","password":"wrong"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "user not found",
			body:         `{"username":"bob","password":"pass"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "empty username",
			body:         `{"username":"","password":"pass"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "empty password",
			body:         `{"username":"alice","password":""}`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			assert.Equal(t, tc.expectedCode, w.Code)

			if tc.expectedCode == http.StatusOK {
				var resp AuthResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Token)
			}
		})
	}
}
