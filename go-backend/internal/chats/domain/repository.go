package domain

import "context"

type ChatRepository interface {
	// Сессии
	CreateSession(ctx context.Context, userID string, title string) (Session, error)
	GetSessionByID(ctx context.Context, sessionID string) (Session, error)
	GetActiveSessionsByUserID(ctx context.Context, userID string) ([]Session, error)
	UpdateSessionTitle(ctx context.Context, sessionID string, title string) error
	SoftDeleteSession(ctx context.Context, sessionID string) error

	// Сообщения
	AppendMessage(ctx context.Context, sessionID string, parentID *string, role MessageRole, content string, traceID *string, meta MessageMeta) (Message, error)
	GetMessagesBySessionID(ctx context.Context, sessionID string) ([]Message, error)
	GetMessageByID(ctx context.Context, messageID string) (Message, error)

	// Фидбек
	SetFeedback(ctx context.Context, messageID string, rating FeedbackRating, comment *string) (Feedback, error)
}
