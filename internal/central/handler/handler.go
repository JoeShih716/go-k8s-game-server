package handler

import (
	"context"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/internal/central/registry"
	"github.com/JoeShih716/go-k8s-game-server/pkg/mysql"
)

type Handler struct {
	proto.UnimplementedCentralServiceServer
	registry *registry.Registry
	db       *mysql.Client
}

func NewHandler(reg *registry.Registry, db *mysql.Client) *Handler {
	return &Handler{
		registry: reg,
		db:       db,
	}
}

// Register 處理服務註冊
func (h *Handler) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
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
func (h *Handler) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	// slog.Debug("Heartbeat received", "lease", req.LeaseId) // Debug level

	err := h.registry.Heartbeat(ctx, req.LeaseId, req.CurrentLoad)
	if err != nil {
		slog.Warn("Heartbeat failed (re-register needed)", "lease", req.LeaseId, "error", err)
		return &proto.HeartbeatResponse{Success: false}, nil
	}

	return &proto.HeartbeatResponse{Success: true}, nil
}

// Deregister 處理登出
func (h *Handler) Deregister(ctx context.Context, req *proto.DeregisterRequest) (*proto.DeregisterResponse, error) {
	slog.Info("Deregistering service", "lease", req.LeaseId)

	err := h.registry.Deregister(ctx, req.LeaseId)
	if err != nil {
		return &proto.DeregisterResponse{Success: false}, nil
	}
	return &proto.DeregisterResponse{Success: true}, nil
}

// Login 玩家登入 (Placeholder)
func (h *Handler) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	// TODO: 驗證 Token, 查詢 DB
	// 目前先無條件成功
	return &proto.LoginResponse{
		UserId:       "user-123", // Mock ID
		Success:      true,
		ErrorMessage: "",
	}, nil
}

// GetRoute 取得路由 (Placeholder)
func (h *Handler) GetRoute(ctx context.Context, req *proto.GetRouteRequest) (*proto.GetRouteResponse, error) {
	// TODO: 真正的動態路由邏輯
	// 1. 根據 GameID 判斷是 Stateless 還是 Stateful
	// 2. 如果是 Stateless -> 從 Registry 拿所有 stateless-service -> Round Robin -> 回傳 Endpoint
	// 3. 如果是 Stateful -> Consistent Hash / Sticky Session -> 回傳 Endpoint

	// 目前先回傳 Mock
	return &proto.GetRouteResponse{
		TargetEndpoint: "stateless-demo:9001",
		Type:           proto.ServiceType_STATELESS,
	}, nil
}
