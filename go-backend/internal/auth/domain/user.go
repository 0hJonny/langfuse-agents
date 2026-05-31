package domain

import (
	"regexp"
	"strings"
	"time"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type UserRole string

const (
	UserRoleAnonymous UserRole = "anonymous"
	UserRoleUser      UserRole = "user"
	UserRoleAdmin     UserRole = "admin"
)

type UserParams struct {
	ID           string
	Email        *string
	PasswordHash *string
	Role         UserRole
}

type User struct {
	CreatedAt    time.Time
	Email        *string
	PasswordHash *string
	ID           string
	Role         UserRole
}

func NewUser(params UserParams) (User, error) {
	var cleanEmail *string

	// 1. Проверяем email, только если он передан
	if params.Email != nil {
		emailStr := strings.ToLower(strings.TrimSpace(*params.Email))

		if !emailRegex.MatchString(emailStr) {
			return User{}, ErrInvalidEmail
		}
		cleanEmail = &emailStr
	}

	// 2. Дефолтная роль, если передана пустая строка
	role := params.Role
	if role == "" {
		role = UserRoleAnonymous
	}

	// 3. Бизнес-валидация: обычный пользователь ОБЯЗАН иметь email и пароль
	if role == UserRoleUser || role == UserRoleAdmin {
		if cleanEmail == nil || params.PasswordHash == nil || *params.PasswordHash == "" {
			return User{}, ErrInvalidCreds
		}
	}

	return User{
		ID:           params.ID,
		Email:        cleanEmail,
		PasswordHash: params.PasswordHash,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
	}, nil
}
