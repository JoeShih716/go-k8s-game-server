package demo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
)

// Handler 實作 GameServiceServer 介面
type Handler struct {
	proto.UnimplementedGameServiceServer
}

// NewHandler 建立一個新的 Demo Handler
func NewHandler() *Handler {
	return &Handler{}
}

// Call 處理這來自 Connector 的請求
func (h *Handler) Call(ctx context.Context, req *proto.GameRequest) (*proto.GameResponse, error) {
	// 取得 Payload (假設內容是字串)
	payloadStr := string(req.Payload)

	slog.Info("Demo Handler Received",
		"user_id", req.Header.UserId,
		"req_id", req.Header.ReqId,
		"payload", payloadStr,
	)

	// 簡單的 Echo 邏輯 (加上 ack 後綴)
	responsePayload := fmt.Sprintf("%s-ack", payloadStr)

	return &proto.GameResponse{
		Code:         proto.ErrorCode_SUCCESS,
		Payload:      []byte(responsePayload),
		ErrorMessage: "",
	}, nil
}
