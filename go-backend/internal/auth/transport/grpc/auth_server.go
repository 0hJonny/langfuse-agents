package grpc

import (
	"google.golang.org/grpc"

	"github.com/0hJonny/langfuse-agents/pkg/authclient/pb"
)

type Server struct {
	grpcServer *grpc.Server
	handler    *GRPCHandler
}

func NewServer(service TokenValidator) *Server {
	s := grpc.NewServer()

	return &Server{
		grpcServer: s,
		handler:    NewGRPCHandler(service),
	}
}

func (s *Server) RegisterServices() *grpc.Server {
	// Регистрируем наш сервис в gRPC движке
	pb.RegisterAuthServiceServer(s.grpcServer, s.handler)

	return s.grpcServer
}
