package statefuldemo

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/JoeShih716/go-k8s-game-server/internal/engine"
)

// Handler 實作 engine.GameHandler 介面
type Handler struct {
	engine.BaseHandler
	host string
}

// NewHandler 建立一個新的 Demo Handler
func NewHandler(host string) *Handler {
	return &Handler{
		host: host,
	}
}

// OnMessage 處理這來自 Connector 的請求
func (h *Handler) OnMessage(_ context.Context, peer *engine.Peer, payload []byte) ([]byte, error) {
	// 取得 Payload (假設內容是字串)
	payloadStr := string(payload)

	slog.Info("Stateful-Demo Service Received",
		"user_id", peer.User.ID,
		"session_id", peer.SessionID,
		"payload", payloadStr,
	)

	echo := echoResponse{
		Host:    h.host,
		Payload: "Hello Echo from Stateful!!!! " + payloadStr,
	}
	jsonBytes, err := json.Marshal(echo)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

// OnJoin 處理玩家進入
func (_ *Handler) OnJoin(_ context.Context, peer *engine.Peer) error {
	slog.Info("Player Joined Stateful Service",
		"user_id", peer.User.ID,
		"session_id", peer.SessionID,
		"connector", peer.ConnectorHost,
	)

	// 一秒後送給他message
	go func() {
		time.Sleep(time.Second)

		// 使用 peer.Send 發送訊息
		ctx := context.Background()
		userID := peer.User.ID
		userName := peer.User.Name
		balance := peer.User.Balance
		err := peer.Send(ctx, []byte("Welcome! This is Stateful Service"+"\n"+"User ID: "+userID+"\n"+"User Name: "+userName+"\n"+"Balance: "+balance.String()))
		if err != nil {
			slog.Warn("PlayerJoinedStatefulService: SendMessage failed", "error", err)
		}
	}()

	return nil
}

// OnQuit 處理玩家離開
func (_ *Handler) OnQuit(_ context.Context, peer *engine.Peer) error {
	slog.Info("Player Quit Stateful Service",
		"user_id", peer.User.ID,
		"session_id", peer.SessionID,
	)
	return nil
}

type echoResponse struct {
	Host    string `json:"host"`
	Payload string `json:"payload"`
}
