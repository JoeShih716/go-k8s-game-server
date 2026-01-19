package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/connector/protocol"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/connector/session"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	rpcsdk "github.com/JoeShih716/go-k8s-game-server/internal/pkg/client/central"
	pkggrpc "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
	"github.com/JoeShih716/go-k8s-game-server/pkg/wss"
)

// WebsocketHandler 實作 wss.Subscriber 介面，處理 WebSocket 事件
type WebsocketHandler struct {
	sessionMgr    *session.Manager
	grpcPool      *pkggrpc.Pool
	centralClient *rpcsdk.Client
	endpoint      string
}

// NewWebsocketHandler 建立 WebSocket 事件處理器
func NewWebsocketHandler(mgr *session.Manager, pool *pkggrpc.Pool, central *rpcsdk.Client, endpoint string) *WebsocketHandler {
	return &WebsocketHandler{
		sessionMgr:    mgr,
		grpcPool:      pool,
		centralClient: central,
		endpoint:      endpoint,
	}
}

// OnConnect 當新連線建立時觸發
func (h *WebsocketHandler) OnConnect(conn wss.Client) {
	// 1. 建立 Session 物件
	sess := domain.NewSession(conn)

	// 2. 加入管理器
	h.sessionMgr.Add(sess)

	slog.Info("Client connected", "id", conn.ID(), "online", h.sessionMgr.Count())

	// 設定 10 秒內必須登入，否則斷線
	loginTimer := time.AfterFunc(10*time.Second, func() {
		slog.Info("Login timeout, kicking client", "id", conn.ID())
		_ = conn.Kick("Login Timeout")
	})
	conn.SetTag("login_timer", loginTimer)
}

// OnDisconnect 當連線斷開時觸發
func (h *WebsocketHandler) OnDisconnect(conn wss.Client) {
	// 清理 Timer
	h.stopTimer(conn, "login_timer")
	h.stopTimer(conn, "enter_game_timer")

	// 若已在遊戲中，通知 Game Server 玩家離開
	var targetEndpoint string

	// 1. 嘗試取得固定路由 (Stateful)
	if target, ok := conn.GetTag("target_endpoint"); ok {
		if endpoint, ok := target.(string); ok && endpoint != "" {
			targetEndpoint = endpoint
		}
	}

	// 2. 若無固定路由，嘗試取得 GameID 進行動態路由 (Stateless)
	if targetEndpoint == "" {
		if gameIDStr, ok := conn.GetTag("current_game_id"); ok {
			var gameID int
			if _, err := fmt.Sscanf(gameIDStr.(string), "%d", &gameID); err == nil {
				// 嘗試向 Central 取得一個可用實例
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				ep, _, err := h.centralClient.GetRoute(ctx, int32(gameID))
				cancel()

				if err == nil && ep != "" {
					targetEndpoint = ep
					slog.Debug("Resolved stateless route for OnPlayerQuit", "gameID", gameID, "endpoint", ep)
				} else {
					slog.Warn("Failed to resolve route for OnPlayerQuit", "gameID", gameID, "error", err)
				}
			}
		}
	}

	// 3. 若有目標 Endpoint，發送信號
	if targetEndpoint != "" {
		// 非同步通知，避免阻塞斷線流程
		go func(ep string, uid string) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			rpcConn, err := h.grpcPool.GetConnection(ep)
			if err != nil {
				slog.Warn("Failed to get connection for OnPlayerQuit", "endpoint", ep, "error", err)
				return
			}
			client := gameRPC.NewGameRPCClient(rpcConn)
			_, err = client.OnPlayerQuit(ctx, &gameRPC.QuitReq{
				Header: &proto.PacketHeader{
					ReqId:     fmt.Sprintf("%d", time.Now().UnixNano()),
					UserId:    uid,
					SessionId: conn.ID(),
					Timestamp: time.Now().UnixMilli(),
				},
			})
			if err != nil {
				slog.Warn("OnPlayerQuit failed", "endpoint", ep, "error", err)
			} else {
				slog.Info("OnPlayerQuit sent", "uid", uid, "endpoint", ep)
			}
		}(targetEndpoint, h.getUserID(conn))
	} else {
		slog.Debug("OnPlayerQuit skipped: no target endpoint found", "id", conn.ID())
	}

	// 從管理器移除
	h.sessionMgr.Remove(conn.ID())

	slog.Info("Client disconnected", "id", conn.ID(), "online", h.sessionMgr.Count())
}

// OnMessage 當收到訊息時觸發
func (h *WebsocketHandler) OnMessage(conn wss.Client, msg []byte) {
	// 1. 解析基礎封包
	var envelope protocol.Envelope
	if err := json.Unmarshal(msg, &envelope); err != nil {
		slog.Warn("Invalid JSON envelope", "error", err)
		h.sendError(conn, "unknown", "Invalid JSON format")
		return
	}

	ctx := context.Background()

	// 2. 本地指令攔截 (Local Intercept)
	// 即使已在遊戲中，這些指令也必須由 Connector 本地處理，不能轉發
	switch envelope.Action {
	case protocol.ActionLogin:
		h.handleLogin(ctx, conn, envelope.Payload)
		return
	case protocol.ActionEnterGame:
		h.handleEnterGame(ctx, conn, envelope.Payload)
		return
	}

	// 3. 轉發邏輯 (Forwarding)
	// A. Sticky Routing (Stateful): 若已有固定路由，直接轉發
	if target, ok := conn.GetTag("target_endpoint"); ok {
		if endpoint, ok := target.(string); ok && endpoint != "" {
			h.forwardToBackend(ctx, conn, endpoint, msg)
			return
		}
	}

	// B. Round-Robin Routing (Stateless): 若無固定路由，但已在遊戲中，重新查詢
	if gameIDStr, ok := conn.GetTag("current_game_id"); ok {
		// 這裡可以做進一步檢查 service_type，但基本上只要沒 target_endpoint 且有 gameID 就是 Stateless
		// 重新向 Central 詢問路由 (實現 Packet-Level Load Balancing)
		// 注意: 這會增加 Central 的負載，生產環境可用本地快取列表優化

		var gameID int
		if _, err := fmt.Sscanf(gameIDStr.(string), "%d", &gameID); err != nil {
			slog.Error("Failed to parse gameID from session", "game_id_str", gameIDStr)
			return
		}

		// 快速 GetRoute
		routeCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		endpoint, _, err := h.centralClient.GetRoute(routeCtx, int32(gameID))
		cancel()

		if err == nil && endpoint != "" {
			h.forwardToBackend(ctx, conn, endpoint, msg)
			return
		}
		slog.Warn("Failed to resolve route for stateless game", "error", err)
	}

	// 4. 未知指令或未入桌
	slog.Warn("Unknown Action and No Route", "action", envelope.Action)
	h.sendError(conn, envelope.Action, "Unknown Action or Not In Game")
}

// -------------------------------------------------------------
// Handlers
// -------------------------------------------------------------

func (h *WebsocketHandler) handleLogin(ctx context.Context, conn wss.Client, payload []byte) {
	// 檢查是否重複登入
	if h.getUserID(conn) != "" {
		h.sendError(conn, protocol.ActionLogin, "Already Logged In")
		return
	}

	// 停止 Login Timer
	h.stopTimer(conn, "login_timer")

	var req protocol.LoginReq
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(conn, protocol.ActionLogin, "Invalid Login Payload")
		_ = conn.Kick("Invalid Protocol")
		return
	}

	// 呼叫 Central 進行登入
	loginCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := h.centralClient.Login(loginCtx, req.Token)
	if err != nil {
		slog.Error("Login failed", "error", err)
		h.sendError(conn, protocol.ActionLogin, "Authentication Failed")
		// 驗證失敗斷線
		time.AfterFunc(100*time.Millisecond, func() { _ = conn.Kick("Auth Failed") })
		return
	}

	// 登入成功，綁定 Session
	conn.SetTag("user_id", resp.UserId)
	slog.Info("User Logged In", "userID", resp.UserId)

	balance, _ := decimal.NewFromString(resp.Balance)
	h.sendResponse(conn, protocol.ActionLogin, protocol.LoginResp{
		Success:  true,
		UserID:   resp.UserId,
		Nickname: resp.Nickname,
		Balance:  balance,
	})

	// 啟動 Enter Game Timer (3分鐘)
	enterGameTimer := time.AfterFunc(3*time.Minute, func() {
		slog.Info("Enter Game timeout, kicking client", "id", conn.ID())
		_ = conn.Kick("Enter Game Timeout")
	})
	conn.SetTag("enter_game_timer", enterGameTimer)
}

func (h *WebsocketHandler) handleEnterGame(ctx context.Context, conn wss.Client, payload []byte) {
	// 檢查是否已經在遊戲中
	if _, ok := conn.GetTag("current_game_id"); ok {
		h.sendError(conn, protocol.ActionEnterGame, "Already In Game")
		return
	}

	var req protocol.EnterGameReq
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(conn, protocol.ActionEnterGame, "Invalid EnterGame Payload")
		return
	}

	// 檢查是否已登入
	userID := h.getUserID(conn)
	if userID == "" {
		h.sendError(conn, protocol.ActionEnterGame, "Not Logged In")
		_ = conn.Kick("Not Logged In")
		return
	}

	// 停止 Enter Game Timer
	h.stopTimer(conn, "enter_game_timer")

	// 呼叫 Central 取得路由
	routeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	endpoint, serviceType, err := h.centralClient.GetRoute(routeCtx, req.GameID)
	// Central 會處理 10000 邏輯，若 error 代表不合法或 demo 以外
	if err != nil {
		slog.Error("GetRoute failed", "game_id", req.GameID, "error", err)
		h.sendError(conn, protocol.ActionEnterGame, "Game Service Unavailable or Invalid ID")
		return
	}

	// ---------------------------------------------------------
	// 新增: 通知 Game Server (OnPlayerJoin)
	// ---------------------------------------------------------
	// 建立與 Game Server 的連線
	rpcConn, err := h.grpcPool.GetConnection(endpoint)
	if err != nil {
		slog.Error("Connect to Game Server failed", "endpoint", endpoint, "error", err)
		h.sendError(conn, protocol.ActionEnterGame, "Game Server Unavailable")
		return
	}
	client := gameRPC.NewGameRPCClient(rpcConn)

	joinCtx, joinCancel := context.WithTimeout(ctx, 3*time.Second)
	defer joinCancel()

	joinResp, err := client.OnPlayerJoin(joinCtx, &gameRPC.JoinReq{
		Header: &proto.PacketHeader{
			ReqId:     fmt.Sprintf("%d", time.Now().UnixNano()),
			UserId:    userID,
			SessionId: conn.ID(),
			Timestamp: time.Now().UnixMilli(),
		},
		ConnectorHost: h.endpoint,
	})

	if err != nil {
		slog.Error("OnPlayerJoin failed", "endpoint", endpoint, "error", err)
		h.sendError(conn, protocol.ActionEnterGame, "Join Game Failed")
		return
	}

	if joinResp.Code != proto.ErrorCode_SUCCESS {
		slog.Error("OnPlayerJoin refused", "code", joinResp.Code, "msg", joinResp.ErrorMessage)
		h.sendError(conn, protocol.ActionEnterGame, "Join Game Refused: "+joinResp.ErrorMessage)
		return
	}
	// ---------------------------------------------------------

	// 緩存路由資訊到 Session (Tag)
	conn.SetTag("current_game_id", fmt.Sprintf("%d", req.GameID))
	conn.SetTag("service_type", serviceType) // 記錄服務類型

	// 我們先針對 Stateful 記錄 Endpoint。
	if serviceType == proto.ServiceType_STATEFUL {
		conn.SetTag("target_endpoint", endpoint)
	}

	slog.Info("Enter Game Success", "userID", userID, "gameID", req.GameID, "target", endpoint, "type", serviceType)

	h.sendResponse(conn, protocol.ActionEnterGame, protocol.EnterGameResp{
		Success: true,
		GameID:  req.GameID,
	})
}

// forwardToBackend 將訊息直接透傳給後端
func (h *WebsocketHandler) forwardToBackend(ctx context.Context, conn wss.Client, targetAddr string, msg []byte) {
	// 準備 gRPC 請求
	rpcConn, err := h.grpcPool.GetConnection(targetAddr)
	if err != nil {
		h.sendError(conn, "forward", "Backend Connection Failed")
		return
	}

	client := gameRPC.NewGameRPCClient(rpcConn)
	rpcReq := &gameRPC.MsgReq{
		Header: &proto.PacketHeader{
			ReqId:     fmt.Sprintf("%d", time.Now().UnixNano()),
			UserId:    h.getUserID(conn),
			SessionId: conn.ID(),
			Timestamp: time.Now().UnixMilli(),
		},
		Payload: msg, // 直接透傳原始 JSON bytes
	}

	// 呼叫後端
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rpcResp, err := client.OnMessage(callCtx, rpcReq)
	if err != nil {
		slog.Error("RPC OnMessage failed", "target", targetAddr, "error", err)
		h.sendError(conn, "forward", "Game Server Error: "+err.Error())
		return
	}

	// 轉發回應給前端 (假設後端回傳的就是完整 JSON 封包)
	_ = conn.SendMessage(string(rpcResp.Payload))
}

// -------------------------------------------------------------
// Helpers
// -------------------------------------------------------------

func (h *WebsocketHandler) stopTimer(conn wss.Client, tagKey string) {
	if v, ok := conn.GetTag(tagKey); ok {
		if t, ok := v.(*time.Timer); ok {
			t.Stop()
		}
	}
}

func (h *WebsocketHandler) getUserID(conn wss.Client) string {
	if v, ok := conn.GetTag("user_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (h *WebsocketHandler) sendError(conn wss.Client, action protocol.ConnectorProtocol, msg string) {
	resp := protocol.Response{
		Action: action,
		Error:  msg,
	}
	bytes, _ := json.Marshal(resp)
	_ = conn.SendMessage(string(bytes))
}

func (h *WebsocketHandler) sendResponse(conn wss.Client, action protocol.ConnectorProtocol, data any) {
	resp := protocol.Response{
		Action: action,
		Data:   data,
	}
	bytes, _ := json.Marshal(resp)
	_ = conn.SendMessage(string(bytes))
}
