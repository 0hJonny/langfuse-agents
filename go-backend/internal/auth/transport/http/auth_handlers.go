package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/0hJonny/langfuse-agents/internal/auth/domain"
)

func (h *Handler) HandleAnonymousAuth(w http.ResponseWriter, r *http.Request) {
	token, err := h.service.CreateAnonymous(r.Context())
	if err != nil {
		h.log.Error("failed to create anonymous user", "error", err)
		h.respondWithError(w, http.StatusInternalServerError, "failed to authenticate anonymously")
		return
	}

	h.respondWithJSON(w, http.StatusCreated, TokenResponse{
		Token:     token.Value,
		ExpiresAt: token.ExpiresAt,
	})
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Email == "" || req.Password == "" {
		h.respondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	var anonUserID string
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		if id, err := h.service.ValidateToken(r.Context(), tokenStr); err == nil {
			anonUserID = id
		}
	}

	token, err := h.service.Register(r.Context(), req.Email, req.Password, anonUserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidEmail):
			h.respondWithError(w, http.StatusBadRequest, "invalid email format")

		case errors.Is(err, domain.ErrUserAlreadyExists):
			h.respondWithError(w, http.StatusConflict, "email already registered")

		default:
			h.log.Error("registration failed", "email", req.Email, "error", err)
			h.respondWithError(w, http.StatusInternalServerError, "failed to register user")
		}
		return
	}

	h.respondWithJSON(w, http.StatusCreated, TokenResponse{Token: token.Value, ExpiresAt: token.ExpiresAt})
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCreds) {
			h.respondWithError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		h.log.Error("login failed", "email", req.Email, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.respondWithJSON(w, http.StatusOK, TokenResponse{Token: token.Value, ExpiresAt: token.ExpiresAt})
}
