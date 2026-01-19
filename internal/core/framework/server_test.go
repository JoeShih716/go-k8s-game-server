package framework_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/framework"
	mock_framework "github.com/JoeShih716/go-k8s-game-server/test/mocks/framework"
)

// TestServer_OnPlayerJoin_Success 測試玩家成功加入 (Stateless)
func TestServer_OnPlayerJoin_Success(t *testing.T) {
	// 1. 初始化 Mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 2. 建立 Mock Object
	mockHandler := mock_framework.NewMockGameHandler(ctrl)

	// 3. 設定預期行為 (Expectations)
	// 當呼叫 OnJoin 時，應該回傳 nil (表示成功)
	// gomock.Any() 表示不檢查參數內容，若需檢查可用 gomock.Eq() 或自訂 Matcher
	mockHandler.EXPECT().
		OnJoin(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// 4. 初始化 Server (Stateless 模式)
	server := framework.NewServer(mockHandler, nil, false, "test-service")

	// 5. 執行測試
	req := &gameRPC.JoinReq{
		Header: &proto.PacketHeader{
			UserId:    "user-123",
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
