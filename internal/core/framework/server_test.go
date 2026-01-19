package framework_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/framework"
	mock_ports "github.com/JoeShih716/go-k8s-game-server/test/mocks/core/ports"
	mock_framework "github.com/JoeShih716/go-k8s-game-server/test/mocks/framework"
)

// TestServer_OnPlayerJoin_Success 測試玩家成功加入 (Stateless)
func TestServer_OnPlayerJoin_Success(t *testing.T) {
	// 1. 初始化 Mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 2. 建立 Mock Objects
	mockHandler := mock_framework.NewMockGameHandler(ctrl)
	mockUserSvc := mock_ports.NewMockUserService(ctrl)
	mockWalletSvc := mock_ports.NewMockWalletService(ctrl)

	// 3. 設定預期行為 (Expectations)
	// 當呼叫 OnJoin 時，應該回傳 nil (表示成功)
	mockHandler.EXPECT().
		OnJoin(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// Expect User Service
	userID := "user-123"
	mockUser := &domain.User{ID: userID}
	mockUserSvc.EXPECT().GetUserByID(gomock.Any(), userID).Return(mockUser, nil)

	// Expect Wallet Service
	mockWalletSvc.EXPECT().GetBalance(gomock.Any(), userID).Return(decimal.NewFromInt(100), nil)

	// 4. 初始化 Server (Stateless 模式)
	server := framework.NewServer(mockHandler, nil, false, "test-service", mockUserSvc, mockWalletSvc)

	// 5. 執行測試
	req := &gameRPC.JoinReq{
		Header: &proto.PacketHeader{
			UserId:    userID,
			SessionId: "sess-abc",
		},
		ConnectorHost: "connector-1",
	}

	resp, err := server.OnPlayerJoin(context.Background(), req)

	// 6. 驗證結果
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Code != proto.ErrorCode_SUCCESS {
		t.Errorf("expected success code, got %v", resp.Code)
	}
}
