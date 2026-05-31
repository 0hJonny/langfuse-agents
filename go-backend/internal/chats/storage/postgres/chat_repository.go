package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/0hJonny/langfuse-agents/internal/chats/domain"
	"github.com/0hJonny/langfuse-agents/pkg/postgres"
)

var jsonEmptyObject = []byte("{}")

var _ domain.ChatRepository = (*ChatRepositoryImpl)(nil)

type ChatRepositoryImpl struct {
	queries *Queries
}

func NewChatRepository(queries *Queries) *ChatRepositoryImpl {
	return &ChatRepositoryImpl{queries: queries}
}

func (r *ChatRepositoryImpl) getQueries(ctx context.Context) *Queries {
	if tx := postgres.GetTxFromContext(ctx); tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

// ==========================================
// СЕССИИ (THREADS)
// ==========================================

func (r *ChatRepositoryImpl) CreateSession(ctx context.Context, userID, title string) (domain.Session, error) {
	q := r.getQueries(ctx)

	var dbUserID pgtype.UUID
	if err := dbUserID.Scan(userID); err != nil {
		return domain.Session{}, fmt.Errorf("failed to parse user uuid: %w", err)
	}

	dbSession, err := q.CreateSession(ctx, CreateSessionParams{
		UserID: dbUserID,
		Title:  title,
	})
	if err != nil {
		return domain.Session{}, err
	}

	return toDomainSession(&dbSession), nil
}

func (r *ChatRepositoryImpl) GetSessionByID(ctx context.Context, sessionID string) (domain.Session, error) {
	q := r.getQueries(ctx)

	var dbSessionID pgtype.UUID
	if err := dbSessionID.Scan(sessionID); err != nil {
		return domain.Session{}, fmt.Errorf("failed to parse session uuid: %w", err)
	}

	dbSession, err := q.GetSessionByID(ctx, dbSessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}

	return toDomainSession(&dbSession), nil
}

func (r *ChatRepositoryImpl) GetActiveSessionsByUserID(ctx context.Context, userID string) ([]domain.Session, error) {
	q := r.getQueries(ctx)

	var dbUserID pgtype.UUID
	if err := dbUserID.Scan(userID); err != nil {
		return nil, fmt.Errorf("failed to parse user uuid: %w", err)
	}

	dbSessions, err := q.GetUserSessions(ctx, dbUserID)
	if err != nil {
		return nil, err
	}

	sessions := make([]domain.Session, len(dbSessions))
	for i := range dbSessions {
		sessions[i] = toDomainSession(&dbSessions[i])
	}
	return sessions, nil
}

func (r *ChatRepositoryImpl) UpdateSessionTitle(ctx context.Context, sessionID, title string) error {
	q := r.getQueries(ctx)

	var dbSessionID pgtype.UUID
	if err := dbSessionID.Scan(sessionID); err != nil {
		return fmt.Errorf("failed to parse session uuid: %w", err)
	}

	return q.UpdateSessionTitle(ctx, UpdateSessionTitleParams{
		ID:    dbSessionID,
		Title: title,
	})
}

func (r *ChatRepositoryImpl) SoftDeleteSession(ctx context.Context, sessionID string) error {
	q := r.getQueries(ctx)

	var dbSessionID pgtype.UUID
	if err := dbSessionID.Scan(sessionID); err != nil {
		return fmt.Errorf("failed to parse session uuid: %w", err)
	}

	return q.SoftDeleteSession(ctx, dbSessionID)
}

// ==========================================
// СООБЩЕНИЯ
// ==========================================

func (r *ChatRepositoryImpl) AppendMessage(
	ctx context.Context,
	sessionID string,
	parentID *string,
	role domain.MessageRole,
	content string,
	traceID *string,
	meta domain.MessageMeta,
) (domain.Message, error) {
	q := r.getQueries(ctx)

	var dbSessionID pgtype.UUID
	if err := dbSessionID.Scan(sessionID); err != nil {
		return domain.Message{}, fmt.Errorf("failed to parse session uuid: %w", err)
	}

	var dbParentID pgtype.UUID
	if parentID != nil && *parentID != "" {
		if err := dbParentID.Scan(*parentID); err != nil {
			return domain.Message{}, fmt.Errorf("failed to parse parent message uuid: %w", err)
		}
	}

	jsonMeta, err := json.Marshal(meta)
	if err != nil {
		jsonMeta = jsonEmptyObject
	}

	dbMsg, err := q.AppendMessage(ctx, AppendMessageParams{
		SessionID: dbSessionID,
		ParentID:  dbParentID,
		Role:      role,
		Content:   content,
		TraceID:   traceID,
		MetaData:  jsonMeta,
	})
	if err != nil {
		return domain.Message{}, err
	}

	return toDomainMessage(&dbMsg), nil
}

func (r *ChatRepositoryImpl) GetMessagesBySessionID(ctx context.Context, sessionID string) ([]domain.Message, error) {
	q := r.getQueries(ctx)

	var dbSessionID pgtype.UUID
	if err := dbSessionID.Scan(sessionID); err != nil {
		return nil, fmt.Errorf("failed to parse session uuid: %w", err)
	}

	dbMsgs, err := q.GetSessionMessages(ctx, dbSessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]domain.Message, len(dbMsgs))
	for i := range dbMsgs {
		messages[i] = toDomainMessage(&dbMsgs[i])
	}
	return messages, nil
}

func (r *ChatRepositoryImpl) GetMessageByID(ctx context.Context, messageID string) (domain.Message, error) {
	q := r.getQueries(ctx)

	var dbMessageID pgtype.UUID
	if err := dbMessageID.Scan(messageID); err != nil {
		return domain.Message{}, fmt.Errorf("failed to parse message uuid: %w", err)
	}

	dbMsg, err := q.GetMessageByID(ctx, dbMessageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, domain.ErrMessageNotFound
		}
		return domain.Message{}, err
	}

	return toDomainMessage(&dbMsg), nil
}

// ==========================================
// ФИДБЕК
// ==========================================

func (r *ChatRepositoryImpl) SetFeedback(ctx context.Context, messageID string, rating domain.FeedbackRating, comment *string) (domain.Feedback, error) {
	q := r.getQueries(ctx)

	var dbMessageID pgtype.UUID
	if err := dbMessageID.Scan(messageID); err != nil {
		return domain.Feedback{}, fmt.Errorf("failed to parse message uuid: %w", err)
	}

	dbFeedback, err := q.SetFeedback(ctx, SetFeedbackParams{
		MessageID: dbMessageID,
		Rating:    rating,
		Comment:   comment,
	})
	if err != nil {
		return domain.Feedback{}, err
	}

	return domain.Feedback{
		ID:        dbFeedback.ID.String(),
		MessageID: dbFeedback.MessageID.String(),
		Rating:    dbFeedback.Rating,
		Comment:   dbFeedback.Comment,
		CreatedAt: dbFeedback.CreatedAt.Time,
	}, nil
}

// ==========================================
// ФУНКЦИИ МАППИНГА
// ==========================================

func toDomainSession(s *ChatsSessions) domain.Session {
	var deletedAt *time.Time
	if s.DeletedAt.Valid {
		t := s.DeletedAt.Time
		deletedAt = &t
	}

	return domain.Session{
		ID:        s.ID.String(),
		UserID:    s.UserID.String(),
		Title:     s.Title,
		CreatedAt: s.CreatedAt.Time,
		UpdatedAt: s.UpdatedAt.Time,
		DeletedAt: deletedAt,
	}
}

func toDomainMessage(m *ChatsMessages) domain.Message {
	var parentID *string
	if m.ParentID.Valid {
		pStr := m.ParentID.String()
		parentID = &pStr
	}

	var meta domain.MessageMeta
	if len(m.MetaData) > 0 && string(m.MetaData) != "{}" {
		_ = json.Unmarshal(m.MetaData, &meta)
	}

	return domain.Message{
		ID:        m.ID.String(),
		SessionID: m.SessionID.String(),
		ParentID:  parentID,
		Role:      m.Role,
		Content:   m.Content,
		TraceID:   m.TraceID,
		MetaData:  meta,
		CreatedAt: m.CreatedAt.Time,
	}
}
