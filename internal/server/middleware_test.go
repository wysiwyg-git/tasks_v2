package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wysiwyg-git/tasks_v2/internal/auth"
	"github.com/wysiwyg-git/tasks_v2/internal/server"
)

func TestRequestIDMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := r.Context().Value(server.RequestIDKey).(string)
		require.NotEmpty(t, id)
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(server.RequestIDMiddleware(logger)(handler))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL, nil)
	resp, _ := ts.Client().Do(req)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-Id"))
}

func TestJWTAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validToken, _ := auth.GenerateToken(1, "alice", secret, time.Hour)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{"No header", "", http.StatusUnauthorized},
		{"Malformed", "Bearer ", http.StatusUnauthorized},
		{"Invalid token", "Bearer invalid", http.StatusUnauthorized},
		{"Valid token", "Bearer " + validToken, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := server.JWTAuthMiddleware(secret, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, _ := r.Context().Value(server.UserClaimsKey).(*auth.Claims)
				if tc.expectedStatus == http.StatusOK {
					require.NotNil(t, claims)
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
