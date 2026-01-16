package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/auth"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/registry"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/wallet"
	"github.com/JoeShih716/go-k8s-game-server/pkg/mysql"
)

type Service struct {
	proto.UnimplementedCentralServiceServer
	registry      *registry.Registry
	db            *mysql.Client
	authenticator auth.Authenticator
	wallet        wallet.Wallet
}

func NewService(r *registry.Registry, db *mysql.Client, auth auth.Authenticator, wallet wallet.Wallet) *Service {
	return &Service{
		registry:      r,
		db:            db,
		authenticator: auth,
		wallet:        wallet,
	}
}

// Register 處理服務註冊
func (h *Service) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	slog.Info("Registering service", "name", req.ServiceName, "type", req.Type, "endpoint", req.Endpoint)

	leaseID, err := h.registry.Register(ctx, req)
	if err != nil {
		slog.Error("Failed to register service", "error", err)
		return nil, err
	}

	return &proto.RegisterResponse{
		LeaseId:    leaseID,
		TtlSeconds: 10, // 與 Registry DefaultTTL 一致
	}, nil
}

// Heartbeat 處理心跳
func (h *Service) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	// slog.Debug("Heartbeat received", "lease", req.LeaseId) // Debug level

	err := h.registry.Heartbeat(ctx, req.LeaseId, req.CurrentLoad)
	if err != nil {
		slog.Warn("Heartbeat failed (re-register needed)", "lease", req.LeaseId, "error", err)
		return &proto.HeartbeatResponse{Success: false}, nil
	}

	return &proto.HeartbeatResponse{Success: true}, nil
}

// Deregister 處理登出
func (h *Service) Deregister(ctx context.Context, req *proto.DeregisterRequest) (*proto.DeregisterResponse, error) {
	slog.Info("Deregistering service", "lease", req.LeaseId)

	err := h.registry.Deregister(ctx, req.LeaseId)
	if err != nil {
		return &proto.DeregisterResponse{Success: false}, nil
	}
	return &proto.DeregisterResponse{Success: true}, nil
}

func (h *Service) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	// 1. 使用 Authenticator 驗證 Token
	user, err := h.authenticator.Verify(ctx, req.Token)
	if err != nil {
		// 驗證失敗 (例如 Token 無效)
		return &proto.LoginResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 2. 從 Wallet 取得餘額
	balance, err := h.wallet.GetBalance(ctx, user.ID)
	if err != nil {
		slog.Error("Failed to get balance", "user_id", user.ID, "error", err)
		// 餘額取得失敗，是否要中止登入？ 暫定: 允許登入但餘額為 0 或回傳錯誤
		// 這裡選擇回傳錯誤
		return &proto.LoginResponse{
			Success:      false,
			ErrorMessage: "Failed to retrieve wallet balance",
		}, nil
	}

	return &proto.LoginResponse{
		Success:      true,
		UserId:       user.ID,
		Nickname:     user.Name,
		Balance:      balance.String(),
		ErrorMessage: "",
	}, nil
}

// GetRoute 取得路由
func (h *Service) GetRoute(ctx context.Context, req *proto.GetRouteRequest) (*proto.GetRouteResponse, error) {
	// 根據 GameID 尋找負責的服務 Endpoint
	endpoint, err := h.registry.SelectServiceByGame(ctx, req.GameId)
	if err != nil {
		slog.Error("Failed to lookup service for game", "game_id", req.GameId, "error", err)
		return nil, fmt.Errorf("internal server error")
	}

	if endpoint == "" {
		slog.Warn("No service found for game", "game_id", req.GameId)
		return nil, fmt.Errorf("service not found for game %d", req.GameId)
	}

	// 這裡假設所有由 GameID 查到的服務都是 STATELESS 或 STATEFUL
	// 如果需要區分，可能需要在 Redis 存更多 Metadata 或從 Endpoint 反查 Lease
	// 目前簡化假設 Game 服務都是 STATELESS (Demo 階段)
	return &proto.GetRouteResponse{
		TargetEndpoint: endpoint,
		Type:           proto.ServiceType_STATELESS, // 暫定
	}, nil
}
