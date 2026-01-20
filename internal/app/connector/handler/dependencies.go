package handler

import (
	"context"

	"google.golang.org/grpc"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
)

// CentralClient 定義了與 Central Service 互動的介面
//
//go:generate mockgen -destination=../../../../test/mocks/handlers/mock_central_client.go -package=mock_handlers . CentralClient
type CentralClient interface {
	Login(ctx context.Context, token string) (*centralRPC.LoginResponse, error)
	GetRoute(ctx context.Context, gameID int32) (string, proto.ServiceType, error)
}

// GRPCPool 定義了取得 gRPC 連線的介面
//
//go:generate mockgen -destination=../../../../test/mocks/handlers/mock_grpc_pool.go -package=mock_handlers . GRPCPool
type GRPCPool interface {
	GetConnection(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}
