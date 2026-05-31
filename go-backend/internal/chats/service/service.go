package service

import (
	"context"

	"github.com/0hJonny/langfuse-agents/internal/chats/domain"
)

// Структуры для передачи параметров (DTO), чтобы не раздувать аргументы методов
type SendMessageDTO struct {
	ParentID  *string            `json:"parent_id"`
	TraceID   *string            `json:"trace_id"`
	SessionID string             `json:"session_id"`
	Role      domain.MessageRole `json:"role"`
	Content   string             `json:"content"`
	MetaData  domain.MessageMeta `json:"meta_data"`
}

type SetFeedbackDTO struct {
	Comment   *string               `json:"comment"`
	MessageID string                `json:"message_id"`
	Rating    domain.FeedbackRating `json:"rating"`
}

type ChatService interface {
	// Управление сессиями (чатами)
	CreateNewChat(ctx context.Context, userID string, title string) (domain.Session, error)
	GetUserChats(ctx context.Context, userID string) ([]domain.Session, error)
	RenameChat(ctx context.Context, userID string, sessionID string, newTitle string) error
	DeleteChat(ctx context.Context, userID string, sessionID string) error

	// Управление сообщениями (история диалога)
	// Метод принимает userID для валидации прав (проверяет, принадлежит ли чат этому пользователю)
	GetChatHistory(ctx context.Context, userID string, sessionID string) ([]domain.Message, error)

	// Основной метод, который будет дергать Python-сервис для атомарного сохранения сообщений
	SaveMessage(ctx context.Context, userID string, dto *SendMessageDTO) (domain.Message, error)

	// Оценка ответов ИИ
	SubmitFeedback(ctx context.Context, userID string, dto *SetFeedbackDTO) (domain.Feedback, error)
}
