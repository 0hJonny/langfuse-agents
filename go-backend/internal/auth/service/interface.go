package service

import "context"

type AuthService interface {
	Register(ctx context.Context, email, password, anonUserID string) (Token, error)
	Login(ctx context.Context, email, password string) (Token, error)
	ValidateToken(ctx context.Context, tokenstring string) (string, error)
	CreateAnonymous(ctx context.Context) (Token, error)
}
