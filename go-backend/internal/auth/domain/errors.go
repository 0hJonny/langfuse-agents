package domain

import "errors"

var (
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotFound           = errors.New("user not found")
	ErrInvalidCreds       = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
