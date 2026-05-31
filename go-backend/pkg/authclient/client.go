package authclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/0hJonny/langfuse-agents/pkg/authclient/pb"
)

func NewAuthClient(addr string) (pb.AuthServiceClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return pb.NewAuthServiceClient(conn), nil
}
