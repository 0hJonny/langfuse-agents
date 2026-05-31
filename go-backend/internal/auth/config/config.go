package config

import "github.com/0hJonny/langfuse-agents/pkg/config"

type Config struct {
	DBUrl      string `env:"DATABASE_URL" env-required:"true"`
	JWTSecret  string `env:"JWT_SECRET" env-required:"true"`
	ServerPort string `env:"PORT" env-default:"8081"`
	GRPCPort   string `env:"GRPC_PORT" env-default:"50051"`
}

func Load() *Config {
	var cfg Config

	config.LoadEnvAndParse("internal/auth/.env", &cfg)

	return &cfg
}
