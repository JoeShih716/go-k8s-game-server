package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"time"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/internal/connector/session"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/router"
	grpcpkg "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
	"github.com/JoeShih716/go-k8s-game-server/pkg/wss"
)

// WebsocketHandler 實作 wss.Subscriber 介面，處理 WebSocket 事件
type WebsocketHandler struct {
	sessionMgr *session.Manager
	router     router.Router
	grpcPool   *grpcpkg.Pool
}

// NewWebsocketHandler 建立 WebSocket 事件處理器
func NewWebsocketHandler(mgr *session.Manager, r router.Router, pool *grpcpkg.Pool) *WebsocketHandler {
	return &WebsocketHandler{
		sessionMgr: mgr,
		router:     r,
		grpcPool:   pool,
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
	_ = conn.SendMessage("Welcome to Game Server! Please send JSON: {\"service_type\": 1, \"payload\": \"test\"}")
}

// OnDisconnect 當連線斷開時觸發
func (h *WebsocketHandler) OnDisconnect(conn wss.Client) {
	// 從管理器移除
	h.sessionMgr.Remove(conn.ID())

	slog.Info("Client disconnected", "id", conn.ID(), "online", h.sessionMgr.Count())
}

// OnMessage 當收到訊息時觸發
func (h *WebsocketHandler) OnMessage(conn wss.Client, msg []byte) {
	slog.Info("Received message", "id", conn.ID(), "payload", string(msg))

	// 1. 嘗試解析為 JSON (測試階段方便使用)
	var req struct {
		ServiceType int32  `json:"service_type"` // 1: Stateless, 2: Stateful
		Payload     string `json:"payload"`
	}

	if err := json.Unmarshal(msg, &req); err != nil {
		slog.Warn("Invalid JSON format", "error", err)
		_ = conn.SendMessage("Error: Invalid JSON format. Example: {\"service_type\": 1, \"payload\": \"hello\"}")
		return
	}

	// 2. 準備路由 Metadata
	metadata := &proto.RoutingMetadata{
		ServiceType: proto.ServiceType(req.ServiceType),
		UserId:      conn.ID(), // 暫用 ConnID 當 UserID
	}

	// 3. 呼叫 Router 決定目標
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targetAddr, err := h.router.Route(ctx, metadata)
	if err != nil {
		slog.Error("Routing failed", "error", err)
		_ = conn.SendMessage(fmt.Sprintf("Error: Service Unavailable (%s)", err.Error()))
		return
	}

	// 4.1 取得 gRPC 連線
	rpcConn, err := h.grpcPool.GetConnection(targetAddr)
	if err != nil {
		slog.Error("Failed to connect to backend", "target", targetAddr, "error", err)
		_ = conn.SendMessage(fmt.Sprintf("Error: Backend Connection Failed (%s)", err.Error()))
		return
	}

	// 4.2 建立 Client 並呼叫
	// 這裡目前只支援通用的 GameService
	client := proto.NewGameServiceClient(rpcConn)
	rpcReq := &proto.GameRequest{
		Header: &proto.PacketHeader{
			ReqId:     fmt.Sprintf("%d", time.Now().UnixNano()), // 簡單 ID
			UserId:    conn.ID(),
			Timestamp: time.Now().UnixMilli(),
		},
		Payload: []byte(req.Payload),
	}

	rpcResp, err := client.Call(ctx, rpcReq)
	if err != nil {
		slog.Error("RPC Call failed", "target", targetAddr, "error", err)
		_ = conn.SendMessage(fmt.Sprintf("Error: RPC Call Failed (%s)", err.Error()))
		return
	}

	// 5. 回傳結果
	successMsg := fmt.Sprintf("Response from %s: %s (Code: %d)", targetAddr, string(rpcResp.Payload), rpcResp.Code)
	slog.Info(successMsg)
	_ = conn.SendMessage(successMsg)
}
