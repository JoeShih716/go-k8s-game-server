package handler

import (
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/internal/connector/session"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/pkg/wss"
)

// WebsocketHandler 實作 wss.Subscriber 介面，處理 WebSocket 事件
type WebsocketHandler struct {
	sessionMgr *session.Manager
}

// NewWebsocketHandler 建立 WebSocket 事件處理器
func NewWebsocketHandler(mgr *session.Manager) *WebsocketHandler {
	return &WebsocketHandler{
		sessionMgr: mgr,
	}
}

// OnConnect 當新連線建立時觸發
func (h *WebsocketHandler) OnConnect(conn wss.Client) {
	// 1. 建立 Session 物件
	sess := domain.NewSession(conn)

	// 2. 加入管理器
	h.sessionMgr.Add(sess)

	slog.Info("Client connected", "id", conn.ID(), "online", h.sessionMgr.Count())

	// 發送歡迎訊息 (測試用)
	_ = conn.SendMessage("Welcome to Game Server!")
}

// OnDisconnect 當連線斷開時觸發
func (h *WebsocketHandler) OnDisconnect(conn wss.Client) {
	// 從管理器移除
	h.sessionMgr.Remove(conn.ID())

	slog.Info("Client disconnected", "id", conn.ID(), "online", h.sessionMgr.Count())
}

// OnMessage 當收到訊息時觸發
func (h *WebsocketHandler) OnMessage(conn wss.Client, msg []byte) {
	// 這裡未來會接 Router，目前先做 Echo
	slog.Info("Received message", "id", conn.ID(), "payload", string(msg))

	// Echo back
	_ = conn.SendMessage("Echo: " + string(msg))
}
