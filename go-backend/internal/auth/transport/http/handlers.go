package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/0hJonny/langfuse-agents/internal/auth/service"
)

type Handler struct {
	service service.AuthService
	log     *slog.Logger
}

func NewHandler(svc service.AuthService, log *slog.Logger) *Handler {
	return &Handler{
		service: svc,
		log:     log,
	}
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, ErrorResponse{Error: message})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.log.Error("failed to encode json response", "error", err)
	}
}
