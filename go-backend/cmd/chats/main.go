package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Предполагаем, что у вас есть пакет config

	pgStorage "github.com/0hJonny/langfuse-agents/internal/chats/storage/postgres"
	chatHttp "github.com/0hJonny/langfuse-agents/internal/chats/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0hJonny/langfuse-agents/internal/chats/config"
	"github.com/0hJonny/langfuse-agents/internal/chats/service"
	"github.com/0hJonny/langfuse-agents/pkg/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Chats Microservice initialization...")

	cfg := config.Load()

	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer poolCancel()

	pool, err := pgxpool.New(poolCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to connect to Postgres pool", slog.String("error", err.Error()))
		poolCancel()
		return
	}
	defer pool.Close()

	if err := pool.Ping(poolCtx); err != nil {
		logger.Error("Postgres pool ping failed", slog.String("error", err.Error()))
		return
	}
	logger.Info("Connected to Postgres successfully")

	// 1. Инициализируем xменеджер транзакций
	txManager := postgres.NewPostgresTxManager(pool)
	queries := pgStorage.New(pool)
	repo := pgStorage.NewChatRepository(queries)

	chatService := service.NewChatService(txManager, repo)
	handler := chatHttp.NewChatHandler(chatService, logger)

	chiRouter := handler.RegisterRoutes()

	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           chiRouter,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		logger.Info("Chats service is listening", slog.String("port", cfg.ServerPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Chats service crashed", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("Received shutdown signal, shutting down Chats service gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Chats service forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("Chats microservice stopped successfully")
}
