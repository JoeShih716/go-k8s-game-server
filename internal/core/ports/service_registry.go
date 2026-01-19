package ports

import (
	"context"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
)

// ServiceRegistry 定義服務註冊與發現的介面
//
//go:generate mockgen -destination=../../../test/mocks/core/ports/mock_service_registry.go -package=mock_ports github.com/JoeShih716/go-k8s-game-server/internal/core/ports ServiceRegistry
type ServiceRegistry interface {
	// Register 註冊一個服務實例
	// 回傳 LeaseID (string)
	Register(ctx context.Context, req *centralRPC.RegisterRequest) (string, error)

	// Heartbeat 更新服務心跳 (Renew Lease)
	Heartbeat(ctx context.Context, leaseID string, load int32) error

	// Deregister 註銷服務
	Deregister(ctx context.Context, leaseID string) error

	// SelectServiceByGame 根據 GameID 選擇一個合適的服務實例 (負載均衡)
	// 回傳 Endpoint (host:port) 與 ServiceType
	SelectServiceByGame(ctx context.Context, gameID int32) (string, proto.ServiceType, error)

	// CleanupDeadServices 清理無效的服務節點 (Zombie Endpoints)
	CleanupDeadServices(ctx context.Context) error
}
