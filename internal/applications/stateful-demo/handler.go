package statefuldemo

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
)

// Handler 實作 GameServiceServer 介面
type Handler struct {
	gameRPC.UnimplementedGameRPCServer
	host string
}

// NewHandler 建立一個新的 Demo Handler
func NewHandler(host string) *Handler {
	return &Handler{
		host: host,
	}
}

// Call 處理這來自 Connector 的請求
func (h *Handler) Call(ctx context.Context, req *gameRPC.GameRequest) (*gameRPC.GameResponse, error) {
	// 取得 Payload (假設內容是字串)
	payloadStr := string(req.Payload)

	slog.Info("Demo Handler Received",
		"user_id", req.Header.UserId,
		"req_id", req.Header.ReqId,
		"payload", payloadStr,
	)

	echo := echoResponse{
		Host:    h.host,
		Payload: "Hello Echo!!!!12345",
	}
	jsonBytes, err := json.Marshal(echo)
	if err != nil {
		return nil, err
	}
	return &gameRPC.GameResponse{
		Code:         proto.ErrorCode_SUCCESS,
		Payload:      jsonBytes,
		ErrorMessage: "",
	}, nil
}

type echoResponse struct {
	Host    string `json:"host"`
	Payload string `json:"payload"`
}
