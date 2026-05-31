package service

import (
	"context"
	"fmt"

	"github.com/0hJonny/langfuse-agents/internal/chats/domain"
	"github.com/0hJonny/langfuse-agents/pkg/postgres"
)

var _ ChatService = (*ChatServiceImpl)(nil)

type ChatServiceImpl struct {
	txManager postgres.TxManager
	repo      domain.ChatRepository
}

func NewChatService(txManager postgres.TxManager, repo domain.ChatRepository) *ChatServiceImpl {
	return &ChatServiceImpl{
		txManager: txManager,
		repo:      repo,
	}
}

// 1. Создание нового чата (Обычно транзакция не нужна, но если в будущем захотите сразу писать первое системное сообщение — она пригодится)
func (s *ChatServiceImpl) CreateNewChat(ctx context.Context, userID, title string) (domain.Session, error) {
	if title == "" {
		title = "New diolog"
	}
	return s.repo.CreateSession(ctx, userID, title)
}

// 2. Получение списка чатов конкретного пользователя
func (s *ChatServiceImpl) GetUserChats(ctx context.Context, userID string) ([]domain.Session, error) {
	return s.repo.GetActiveSessionsByUserID(ctx, userID)
}

// 3. Переименование чата
func (s *ChatServiceImpl) RenameChat(ctx context.Context, userID, sessionID, newTitle string) error {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.UserID != userID {
		return domain.ErrUnauthorized
	}

	if newTitle == "" {
		newTitle = "Без названия"
	}

	return s.repo.UpdateSessionTitle(ctx, sessionID, newTitle)
}

// 4. Мягкое удаление чата
func (s *ChatServiceImpl) DeleteChat(ctx context.Context, userID, sessionID string) error {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.UserID != userID {
		return domain.ErrUnauthorized
	}

	return s.repo.SoftDeleteSession(ctx, sessionID)
}

// 5. Выгрузка истории сообщений
func (s *ChatServiceImpl) GetChatHistory(ctx context.Context, userID, sessionID string) ([]domain.Message, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	return s.repo.GetMessagesBySessionID(ctx, sessionID)
}

// 6. АТОМАРНОЕ сохранение сообщения с обновлением времени треда
func (s *ChatServiceImpl) SaveMessage(ctx context.Context, userID string, dto *SendMessageDTO) (domain.Message, error) {
	// Открываем транзакцию
	tx, txCtx, err := s.txManager.Begin(ctx)
	if err != nil {
		return domain.Message{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	// Все проверки и чтение делаем ВНУТРИ транзакции (используем txCtx)
	session, err := s.repo.GetSessionByID(txCtx, dto.SessionID)
	if err != nil {
		return domain.Message{}, err
	}
	if session.UserID != userID {
		return domain.Message{}, domain.ErrUnauthorized
	}

	// Шаг A: Записываем само сообщение
	msg, err := s.repo.AppendMessage(
		txCtx, // Передаем контекст транзакции
		dto.SessionID,
		dto.ParentID,
		dto.Role,
		dto.Content,
		dto.TraceID,
		dto.MetaData,
	)
	if err != nil {
		return domain.Message{}, err
	}

	// Шаг B: Обновляем updated_at у сессии, чтобы чат поднялся в топ Списка
	// Для этого вызываем существующий метод UpdateSessionTitle, передавая текущее имя (или пишем отдельный метод UpdateSessionTimestamp)
	if err := s.repo.UpdateSessionTitle(txCtx, dto.SessionID, session.Title); err != nil {
		return domain.Message{}, fmt.Errorf("failed to update session timestamp: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(txCtx); err != nil {
		return domain.Message{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return msg, nil
}

// 7. Сохранение лайка/дизлайка
func (s *ChatServiceImpl) SubmitFeedback(ctx context.Context, userID string, dto *SetFeedbackDTO) (domain.Feedback, error) {
	tx, txCtx, err := s.txManager.Begin(ctx)
	if err != nil {
		return domain.Feedback{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	message, err := s.repo.GetMessageByID(txCtx, dto.MessageID)
	if err != nil {
		return domain.Feedback{}, err
	}

	session, err := s.repo.GetSessionByID(txCtx, message.SessionID)
	if err != nil {
		return domain.Feedback{}, err
	}
	if session.UserID != userID {
		return domain.Feedback{}, domain.ErrUnauthorized
	}

	feedback, err := s.repo.SetFeedback(txCtx, dto.MessageID, dto.Rating, dto.Comment)
	if err != nil {
		return domain.Feedback{}, err
	}

	if err := tx.Commit(txCtx); err != nil {
		return domain.Feedback{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return feedback, nil
}
