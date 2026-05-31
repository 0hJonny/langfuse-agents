package config

import "github.com/0hJonny/langfuse-agents/pkg/config"

type Config struct {
	ServerPort   string `env:"PORT" env-default:"8080"`
	AuthGRPCAddr string `env:"AUTH_GRPC_ADDR" env-default:"localhost:50051"`

	// HTTP Апстримы для проксирования
	AuthHTTPAddr   string `env:"AUTH_HTTP_ADDR" env-default:"http://localhost:8081"`
	AgentsHTTPAddr string `env:"AGENTS_HTTP_ADDR" env-default:"http://localhost:8082"`
	ChatsHTTPAddr  string `env:"CHATS_HTTP_ADDR" env-default:"http://localhost:8083"`

	AllowedOrigins string `env:"ALLOWED_ORIGINS" env-default:"http://localhost:3000,http://localhost:5173"`
	// Будущая конфигурация Redis (пока закомментируем или оставим на дефолтах)
	RedisAddr string `env:"REDIS_ADDR" env-default:"localhost:6379"`
}

func Load() *Config {
	var cfg Config
	config.LoadEnvAndParse("internal/gateway/.env", &cfg)
	return &cfg
}
