package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/0hJonny/langfuse-agents/internal/auth/domain"
	"github.com/0hJonny/langfuse-agents/pkg/postgres"
)

var _ AuthService = (*AuthServiceImpl)(nil)

type AuthServiceImpl struct {
	txManager postgres.TxManager
	repo      domain.UserRepository
	secret    []byte
}

func NewAuthService(txManager postgres.TxManager, repo domain.UserRepository, secret string) *AuthServiceImpl {
	return &AuthServiceImpl{
		txManager: txManager,
		repo:      repo,
		secret:    []byte(secret),
	}
}

// Изменили сигнатуру: добавили anonUserID
func (s *AuthServiceImpl) Register(ctx context.Context, email, password, anonUserID string) (Token, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Token{}, fmt.Errorf("failed to hash password: %w", err)
	}
	hashStr := string(hash)

	tx, txCtx, err := s.txManager.Begin(ctx)
	if err != nil {
		return Token{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(txCtx)
	}()

	var user domain.User

	if anonUserID != "" {
		// Сценарий 1: Апгрейд существующего анонима
		domainUser, err := domain.NewUser(domain.UserParams{
			ID:           anonUserID,
			Email:        &email,
			PasswordHash: &hashStr,
			Role:         domain.UserRoleUser,
		})
		if err != nil {
			return Token{}, err
		}

		user, err = s.repo.UpdateUser(txCtx, &domainUser)
		if err != nil {
			return Token{}, err
		}
	} else {
		// Сценарий 2: Чистая регистрация с нуля
		domainUser, err := domain.NewUser(domain.UserParams{
			Email:        &email,
			PasswordHash: &hashStr,
			Role:         domain.UserRoleUser,
		})
		if err != nil {
			return Token{}, err
		}

		user, err = s.repo.CreateUser(txCtx, &domainUser)
		if err != nil {
			return Token{}, err
		}
	}

	if err := tx.Commit(txCtx); err != nil {
		return Token{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.generateToken(user.ID, user.Role)
}

// Новый метод для генерации анонимного профиля
func (s *AuthServiceImpl) CreateAnonymous(ctx context.Context) (Token, error) {
	tx, txCtx, err := s.txManager.Begin(ctx)
	if err != nil {
		return Token{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(txCtx)
	}()

	// Домен сам выставит роль UserRoleAnonymous и обнулит поля
	domainUser, err := domain.NewUser(domain.UserParams{})
	if err != nil {
		return Token{}, err
	}

	user, err := s.repo.CreateUser(txCtx, &domainUser)
	if err != nil {
		return Token{}, fmt.Errorf("failed to create anonymous user: %w", err)
	}

	if err := tx.Commit(txCtx); err != nil {
		return Token{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.generateToken(user.ID, user.Role)
}

func (s *AuthServiceImpl) Login(ctx context.Context, email, password string) (Token, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return Token{}, domain.ErrInvalidCreds
	}

	// Безопасность: анонимы без пароля не пройдут
	if user.PasswordHash == nil {
		return Token{}, domain.ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return Token{}, domain.ErrInvalidCreds
	}

	return s.generateToken(user.ID, user.Role)
}

func (s *AuthServiceImpl) ValidateToken(ctx context.Context, tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil || token == nil || !token.Valid {
		return "", domain.ErrInvalidToken
	}

	userID, err := token.Claims.GetSubject()
	if err != nil || userID == "" {
		return "", domain.ErrInvalidToken
	}

	return userID, nil
}

// Принимает строгий тип domain.UserRole вместо string
func (s *AuthServiceImpl) generateToken(userID string, role domain.UserRole) (Token, error) {
	now := time.Now().UTC()
	expirationTime := now.Add(24 * time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": string(role),
		"exp":  expirationTime.Unix(),
		"iat":  now.Unix(),
	})

	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return Token{}, err
	}

	return Token{
		Value:     tokenString,
		ExpiresAt: expirationTime.Unix(),
	}, nil
}
