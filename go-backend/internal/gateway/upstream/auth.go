package upstream

import (
	"context"

	"github.com/0hJonny/langfuse-agents/pkg/authclient/pb"
)

type AuthServiceClientAdapter struct {
	client pb.AuthServiceClient
	// Сюда в будущем добавится: redisClient *redis.Client
}

func NewAuthServiceClientAdapter(client pb.AuthServiceClient) *AuthServiceClientAdapter {
	return &AuthServiceClientAdapter{client: client}
}

// ValidateToken реализует интерфейс http.TokenValidator
func (a *AuthServiceClientAdapter) ValidateToken(ctx context.Context, token string) (string, error) {
	// Будущая логика Redis Blacklist:
	// isBlacklisted, _ := a.redis.Exists(ctx, "blacklist:"+token).Result()
	// if isBlacklisted { return "", errors.New("token blacklisted") }

	resp, err := a.client.ValidateToken(ctx, &pb.ValidateTokenRequest{Token: token})
	if err != nil {
		return "", err
	}

	return resp.GetUserId(), nil
}
