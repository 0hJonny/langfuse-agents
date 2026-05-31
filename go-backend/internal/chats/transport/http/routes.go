package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/0hJonny/langfuse-agents/internal/chats/domain"
)

func (h *ChatHandler) getUserID(ctx context.Context) string {
	if val, ok := ctx.Value(userIDKey).(string); ok {
		return val
	}
	return ""
}

func (h *ChatHandler) RegisterRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	r.Route("/api/v1/chats", func(r chi.Router) {
		r.Use(h.FromGatewayHeader)
		r.Post("/sessions", h.HandleCreateChat)
		r.Get("/sessions", h.HandleGetUserChats)
		r.Put("/sessions/{id}/title", h.HandleRenameChat)
		r.Delete("/sessions/{id}", h.HandleDeleteChat)
		r.Get("/sessions/{id}/messages", h.HandleGetChatHistory)
		r.Post("/messages", h.HandleAppendMessage)
		r.Post("/feedback", h.HandleSubmitFeedback)
	})
	return r
}

func (h *ChatHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *ChatHandler) respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *ChatHandler) mapError(w http.ResponseWriter, err error, defaultMessage, userID string) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		h.log.Warn("unauthorized access attempt", "user_id", userID, "error", err)
		h.respondWithError(w, http.StatusForbidden, "you don't have access to this resource")
	case errors.Is(err, domain.ErrSessionNotFound):
		h.respondWithError(w, http.StatusNotFound, "chat session not found")
	case errors.Is(err, domain.ErrMessageNotFound):
		h.respondWithError(w, http.StatusNotFound, "message not found")
	default:
		h.log.Error("operation failed", "user_id", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, defaultMessage)
	}
}
