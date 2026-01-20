package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/central/service"
)

// GRPCHandler 負責將 gRPC 請求轉換為業務調用
type GRPCHandler struct {
	centralRPC.UnimplementedCentralRPCServer
	svc *service.CentralService
}

// NewGRPCHandler 建立 gRPC Handler
func NewGRPCHandler(svc *service.CentralService) *GRPCHandler {
	return &GRPCHandler{
		svc: svc,
	}
}

// Register 處理服務註冊
func (h *GRPCHandler) Register(ctx context.Context, req *centralRPC.RegisterRequest) (*centralRPC.RegisterResponse, error) {
	slog.Info("Registering service", "name", req.ServiceName, "type", req.Type, "endpoint", req.Endpoint)

	leaseID, err := h.svc.RegisterService(ctx, req)
	if err != nil {
		slog.Error("Failed to register service", "error", err)
		return nil, err
	}

	return &centralRPC.RegisterResponse{
		LeaseId:    leaseID,
		TtlSeconds: 10,
	}, nil
}

// Heartbeat 處理心跳
func (h *GRPCHandler) Heartbeat(ctx context.Context, req *centralRPC.HeartbeatRequest) (*centralRPC.HeartbeatResponse, error) {
	err := h.svc.Heartbeat(ctx, req.LeaseId, req.CurrentLoad)
	if err != nil {
		slog.Warn("Heartbeat failed (re-register needed)", "lease", req.LeaseId, "error", err)
		return &centralRPC.HeartbeatResponse{Success: false}, nil
	}

	return &centralRPC.HeartbeatResponse{Success: true}, nil
}

// Deregister 處理登出
func (h *GRPCHandler) Deregister(ctx context.Context, req *centralRPC.DeregisterRequest) (*centralRPC.DeregisterResponse, error) {
	slog.Info("Deregistering service", "lease", req.LeaseId)

	err := h.svc.DeregisterService(ctx, req.LeaseId)
	if err != nil {
		return &centralRPC.DeregisterResponse{Success: false}, nil
	}
	return &centralRPC.DeregisterResponse{Success: true}, nil
}

// Login 處理玩家登入
func (h *GRPCHandler) Login(ctx context.Context, req *centralRPC.LoginRequest) (*centralRPC.LoginResponse, error) {
	user, err := h.svc.Login(ctx, req.Token)
	if err != nil {
		return &centralRPC.LoginResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &centralRPC.LoginResponse{
		Success:  true,
		UserId:   user.ID,
		Nickname: user.Name,
		Balance:  user.Balance.String(),
	}, nil
}

// GetRoute 取得路由
func (h *GRPCHandler) GetRoute(ctx context.Context, req *centralRPC.GetRouteRequest) (*centralRPC.GetRouteResponse, error) {
	endpoint, sType, err := h.svc.GetGameServerEndpoint(ctx, req.GameId)
	if err != nil {
		slog.Error("Failed to lookup service for game", "game_id", req.GameId, "error", err)
		return nil, fmt.Errorf("internal server error")
	}

	if endpoint == "" {
		slog.Warn("No service found for game", "game_id", req.GameId)
		return nil, fmt.Errorf("service not found for game %d", req.GameId)
	}

	return &centralRPC.GetRouteResponse{
		TargetEndpoint: endpoint,
		Type:           sType,
	}, nil
}
