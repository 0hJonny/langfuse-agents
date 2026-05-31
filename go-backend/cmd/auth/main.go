package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/0hJonny/langfuse-agents/internal/auth/config"
	"github.com/0hJonny/langfuse-agents/internal/auth/service"
	pgStorage "github.com/0hJonny/langfuse-agents/internal/auth/storage/postgres"
	authGrpc "github.com/0hJonny/langfuse-agents/internal/auth/transport/grpc"
	authHttp "github.com/0hJonny/langfuse-agents/internal/auth/transport/http"
	"github.com/0hJonny/langfuse-agents/pkg/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Auth Service initialization...")

	cfg := config.Load()

	grpcLis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("Failed to listen gRPC port", slog.String("port", cfg.GRPCPort), slog.String("error", err.Error()))
		os.Exit(1)
	}

	dbPool, err := pgxpool.New(context.Background(), cfg.DBUrl)
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		_ = grpcLis.Close()
		os.Exit(1)
	}

	defer dbPool.Close()

	// Инициализация менеджера транзакций и репозитория Postgres
	txManager := postgres.NewPostgresTxManager(dbPool)
	queries := pgStorage.New(dbPool)
	repo := pgStorage.NewPostgresRepository(queries)

	// Сборка слоев
	authService := service.NewAuthService(txManager, repo, cfg.JWTSecret)
	authHandler := authHttp.NewHandler(authService, logger)
	router := authHandler.RegisterRoutes()

	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
	}

	grpcServerImpl := authGrpc.NewServer(authService)
	grpcServer := grpcServerImpl.RegisterServices()

	go func() {
		logger.Info("Auth HTTP Service is listening", slog.String("port", cfg.ServerPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP Server crashed", slog.String("error", err.Error()))
		}
	}()

	go func() {
		logger.Info("Auth gRPC Service is listening", slog.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(grpcLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("gRPC Server crashed", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("Received shutdown signal, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP Server forced to shutdown", slog.String("error", err.Error()))
	}

	grpcServer.GracefulStop()
	logger.Info("gRPC Server stopped gracefully")

	logger.Info("Auth Service stopped successfully")
}
