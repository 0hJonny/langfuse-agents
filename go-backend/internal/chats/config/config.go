package config

import "github.com/0hJonny/langfuse-agents/pkg/config"

type Config struct {
	DatabaseURL string `env:"DATABASE_URL" env-required:"true"`

	ServerPort string `env:"PORT" env-default:"8082"`
}

func Load() *Config {
	var cfg Config
	config.LoadEnvAndParse("internal/chats/.env", &cfg)
	return &cfg
}
