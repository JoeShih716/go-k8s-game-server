package statelessdemo

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/framework"
)

// Handler 實作 framework.GameHandler 介面
type Handler struct {
	framework.BaseHandler
	host string
}

// NewHandler 建立一個新的 Demo Handler
func NewHandler(host string) *Handler {
	return &Handler{
		host: host,
	}
}

// OnMessage 處理這來自 Connector 的請求
func (h *Handler) OnMessage(ctx context.Context, session *framework.Session, payload []byte) ([]byte, error) {
	// 取得 Payload (假設內容是字串)
	payloadStr := string(payload)

	slog.Info("Stateless-Demo Service Received",
		"user_id", session.UserID,
		"req_id", session.SessionID, // 注意: 這裡 SessionID 可能就是 ReqID (視 Connector 實作而定) 或者就是 SessionID
		"payload", payloadStr,
	)

	// [Optional] 如果需要主動發送 (Push)，可以呼叫 `session.Send(...)`
	// 但 Stateless 通常是 Request-Response 模型，直接回傳即可。
	// 若要實現使用者說的 "額外多發封包給他"，可以這樣做：
	//
	// if err := session.Send(ctx, []byte("Extra Push Message")); err != nil {
	//     slog.Warn("Failed to push extra message", "error", err)
	// }

	echo := echoResponse{
		Host:    h.host,
		Payload: "Hello Echo from Stateless!!!! " + payloadStr,
	}
	jsonBytes, err := json.Marshal(echo)
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

// OnJoin 處理玩家進入
func (h *Handler) OnJoin(ctx context.Context, session *framework.Session) error {
	slog.Info("Player Joined Stateless Service", "user_id", session.UserID, "connector", session.ConnectorHost)
	return nil
}

// OnQuit 處理玩家離開
func (h *Handler) OnQuit(ctx context.Context, session *framework.Session) error {
	slog.Info("Player Quit Stateless Service", "user_id", session.UserID)
	return nil
}

type echoResponse struct {
	Host    string `json:"host"`
	Payload string `json:"payload"`
}
