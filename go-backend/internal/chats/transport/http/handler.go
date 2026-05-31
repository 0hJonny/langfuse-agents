package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/0hJonny/langfuse-agents/internal/chats/service"
)

type contextKey string

const userIDKey contextKey = "userID"

type ChatHandler struct {
	service service.ChatService
	log     *slog.Logger
}

func NewChatHandler(srv service.ChatService, log *slog.Logger) *ChatHandler {
	return &ChatHandler{service: srv, log: log}
}

func (h *ChatHandler) HandleCreateChat(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r.Context())
	var req struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	session, err := h.service.CreateNewChat(r.Context(), userID, req.Title)
	if err != nil {
		h.log.Error("failed to create chat", "error", err, "user_id", userID)
		h.respondWithError(w, http.StatusInternalServerError, "failed to create chat session")
		return
	}
	h.respondWithJSON(w, http.StatusCreated, session)
}

func (h *ChatHandler) HandleGetUserChats(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r.Context())
	sessions, err := h.service.GetUserChats(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get user chats", "error", err, "user_id", userID)
		h.respondWithError(w, http.StatusInternalServerError, "failed to fetch chats")
		return
	}
	h.respondWithJSON(w, http.StatusOK, sessions)
}

func (h *ChatHandler) HandleGetChatHistory(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r.Context())
	sessionID := chi.URLParam(r, "id")

	history, err := h.service.GetChatHistory(r.Context(), userID, sessionID)
	if err != nil {
		h.mapError(w, err, "failed to fetch chat history", userID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, history)
}

func (h *ChatHandler) HandleAppendMessage(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r.Context())
	var req service.SendMessageDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SessionID == "" || req.Content == "" || req.Role == "" {
		h.respondWithError(w, http.StatusBadRequest, "session_id, role, and content are required")
		return
	}

	msg, err := h.service.SaveMessage(r.Context(), userID, &req)
	if err != nil {
		h.mapError(w, err, "failed to save message", userID)
		return
	}
	h.respondWithJSON(w, http.StatusCreated, msg)
}

func (h *ChatHandler) HandleRenameChat(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r.Context())
	sessionID := chi.URLParam(r, "id")

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		h.respondWithError(w, http.StatusBadRequest, "invalid or empty title")
		return
	}

	if err := h.service.RenameChat(r.Context(), userID, sessionID, req.Title); err != nil {
		h.mapError(w, err, "failed to rename chat", userID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *ChatHandler) HandleDeleteChat(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r.Context())
	sessionID := chi.URLParam(r, "id")

	if err := h.service.DeleteChat(r.Context(), userID, sessionID); err != nil {
		h.mapError(w, err, "failed to delete chat", userID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *ChatHandler) HandleSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r.Context())
	var req service.SetFeedbackDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	feedback, err := h.service.SubmitFeedback(r.Context(), userID, &req)
	if err != nil {
		h.mapError(w, err, "failed to submit feedback", userID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, feedback)
}
