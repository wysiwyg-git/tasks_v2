package server

import (
	"encoding/json"
	"net/http"

	"github.com/wysiwyg-git/tasks_v2/internal/auth"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	user, err := s.Store.CreateUser(r.Context(), req.Username, hash)
	if err != nil {
		// Если пользователь существует — 409 Conflict
		if err.Error() == "user already exists" { // Тут лучше использовать errors.Is, но пока так
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		s.logger.Error("register: create user", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	token, err := auth.GenerateToken(user.ID, user.Username, s.Config.JWTSecret, s.Config.JWTTokenTTL)
	if err != nil {
		s.logger.Error("register: generate token", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	user, err := s.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		s.logger.Error("login: get user", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if user == nil || !auth.CheckPassword(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	token, err := auth.GenerateToken(user.ID, user.Username, s.Config.JWTSecret, s.Config.JWTTokenTTL)
	if err != nil {
		s.logger.Error("login: generate token", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}
