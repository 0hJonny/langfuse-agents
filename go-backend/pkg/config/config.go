package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

func LoadEnvAndParse(envPath string, target any) {
	// Пытаемся загрузить .env файл по переданному пути (например, "internal/auth/.env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Info: No .env file found at %s, reading directly from system env\n", envPath)
	}

	// Читаем переменные окружения прямо в структуру через теги
	if err := cleanenv.ReadEnv(target); err != nil {
		log.Fatalf("Critical: failed to parse environment variables: %v", err)
	}
}
