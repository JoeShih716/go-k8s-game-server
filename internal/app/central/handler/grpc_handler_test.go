package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/shopspring/decimal"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/central/service"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	mock_ports "github.com/JoeShih716/go-k8s-game-server/test/mocks/core/ports"
)

// Helper to setup dependencies
func setupDependencies(t *testing.T) (*GRPCHandler, *mock_ports.MockUserService, *mock_ports.MockWalletService, *mock_ports.MockRegistryService) {
	ctrl := gomock.NewController(t)
	// Note: In newer gomock/Go versions, Verify() is called automatically if Reporter is set (which passing t does).
	// But defer ctrl.Finish() is explicit.
	t.Cleanup(ctrl.Finish)

	mockUserSvc := mock_ports.NewMockUserService(ctrl)
	mockWalletSvc := mock_ports.NewMockWalletService(ctrl)
	mockRegistry := mock_ports.NewMockRegistryService(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := service.NewCentralService(mockUserSvc, mockWalletSvc, mockRegistry, logger)
	handler := NewGRPCHandler(svc)

	return handler, mockUserSvc, mockWalletSvc, mockRegistry
}

func TestGRPCHandler_Login_Success(t *testing.T) {
	h, mockUserSvc, mockWalletSvc, _ := setupDependencies(t)
	ctx := context.Background()
	req := &centralRPC.LoginRequest{Token: "valid-token"}
	userID := "user-123"

	// 1. Mock Service Login flow
	// Service.Login calls:
	// - UserSvc.GetUser
	// - WalletSvc.GetBalance (if user found)

	mockUserSvc.EXPECT().GetUser(ctx, req.Token).Return(&domain.User{
		ID:      userID,
		Name:    "TestUser",
		Balance: decimal.NewFromInt(100),
	}, nil)

	mockWalletSvc.EXPECT().GetBalance(ctx, userID).Return(decimal.NewFromInt(500), nil)

	// Call Handler
	resp, err := h.Login(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, userID, resp.UserId)
	assert.Equal(t, "500", resp.Balance) // Balance updated from wallet
}

func TestGRPCHandler_Login_Failed(t *testing.T) {
	h, mockUserSvc, _, _ := setupDependencies(t)
	ctx := context.Background()
	req := &centralRPC.LoginRequest{Token: "invalid-token"}

	// Mock Service Login flow fails at GetUser
	mockUserSvc.EXPECT().GetUser(ctx, req.Token).Return(nil, domain.ErrInvalidToken)

	resp, err := h.Login(ctx, req)

	// Implementation note: Handler returns nil error but success=false in response?
	// Let's check handler code:
	// if err != nil { return &LoginResponse{Success: false, ...}, nil }
	assert.NoError(t, err) // GRPC call itself succeeded
	assert.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, domain.ErrInvalidToken.Error())
}

func TestGRPCHandler_Register_Success(t *testing.T) {
	h, _, _, mockRegistry := setupDependencies(t)
	ctx := context.Background()
	req := &centralRPC.RegisterRequest{
		ServiceName: "game-1",
		Endpoint:    "localhost:9000",
		Type:        proto.ServiceType_STATELESS,
	}

	mockRegistry.EXPECT().Register(ctx, req).Return("lease-123", nil)

	resp, err := h.Register(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "lease-123", resp.LeaseId)
}

func TestGRPCHandler_Heartbeat(t *testing.T) {
	h, _, _, mockRegistry := setupDependencies(t)
	ctx := context.Background()
	req := &centralRPC.HeartbeatRequest{
		LeaseId:     "lease-123",
		CurrentLoad: 50,
	}

	mockRegistry.EXPECT().Heartbeat(ctx, req.LeaseId, req.CurrentLoad).Return(nil)

	resp, err := h.Heartbeat(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestGRPCHandler_Heartbeat_Fail(t *testing.T) {
	h, _, _, mockRegistry := setupDependencies(t)
	ctx := context.Background()
	req := &centralRPC.HeartbeatRequest{
		LeaseId: "lease-expired",
	}

	mockRegistry.EXPECT().Heartbeat(ctx, req.LeaseId, req.CurrentLoad).Return(fmt.Errorf("lease not found"))

	resp, err := h.Heartbeat(ctx, req)
	// Handler swallows error and returns success=false
	assert.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestGRPCHandler_GetRoute_Success(t *testing.T) {
	h, _, _, mockRegistry := setupDependencies(t)
	ctx := context.Background()
	req := &centralRPC.GetRouteRequest{GameId: 1001}

	mockRegistry.EXPECT().SelectServiceByGame(ctx, req.GameId).Return("localhost:8081", proto.ServiceType_STATELESS, nil)

	resp, err := h.GetRoute(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:8081", resp.TargetEndpoint)
	assert.Equal(t, proto.ServiceType_STATELESS, resp.Type)
}
