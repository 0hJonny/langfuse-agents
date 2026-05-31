package domain

import (
	"errors"
	"time"
)

var (
	ErrSessionNotFound = errors.New("chat session not found")
	ErrMessageNotFound = errors.New("message not found")
	ErrUnauthorized    = errors.New("access denied: user does not own this session")
)

// Кастомные типы для ENUM из базы данных
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

type FeedbackRating string

const (
	RatingLike    FeedbackRating = "like"
	RatingDislike FeedbackRating = "dislike"
)

// Сущность сессии чата (Thread)
type Session struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	ID        string
	UserID    string
	Title     string
}

type MessageMeta struct {
	Temperature  *float64 `json:"temperature,omitempty"`
	Model        string   `json:"model,omitempty"`
	FinishReason string   `json:"finish_reason,omitempty"`
	PromptTokens int32    `json:"prompt_tokens,omitempty"`
	ComplTokens  int32    `json:"compl_tokens,omitempty"`
}

type Message struct {
	CreatedAt time.Time
	ParentID  *string
	TraceID   *string
	ID        string
	SessionID string
	Role      MessageRole
	Content   string
	MetaData  MessageMeta
}

// Сущность фидбека к сообщению
type Feedback struct {
	CreatedAt time.Time
	Comment   *string
	ID        string
	MessageID string
	Rating    FeedbackRating
}
