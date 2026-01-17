package statefuldemo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/connector/protocol"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/stateful-demo/manager"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// Handler 實作 proto.GameServiceServer
type Handler struct {
	gameRPC.UnimplementedGameRPCServer
	roomMgr *manager.Manager
}

func NewHandler(mgr *manager.Manager) *Handler {
	return &Handler{
		roomMgr: mgr,
	}
}

// Call 是單一進入點，負責解包 Request
func (h *Handler) Call(ctx context.Context, req *gameRPC.GameRequest) (*gameRPC.GameResponse, error) {
	// 1. 解析信封 (Envelope)
	var envelope protocol.Envelope
	if err := json.Unmarshal(req.Payload, &envelope); err != nil {
		slog.Error("Invalid envelope", "err", err)
		return nil, fmt.Errorf("invalid envelope")
	}

	userID := req.Header.UserId

	// 使用 RoomID 作為 State 的隔離單位
	// 在這個設計中，我們假設 Client 傳來的 Action 裡應該包含 RoomID
	// 或者，更穩健的做法是：在 Central/Session 綁定 User <-> RoomID，
	// 然後在這裡查詢 User 在哪個 Room。
	//
	// 為了 Demo 簡單起見，我們假設前端的 EnterGameReq 裡包含 GameID，
	// 並且我們用一個固定的規則產生 RoomID (例如 "room_1", "room_2")。
	// 或者直接把 GameID 當作 RoomID (如果是一對一或者單房)。

	slog.Info("Handler Call", "userID", userID, "action", envelope.Action)

	switch envelope.Action {
	case "enter_game":
		return h.handleEnterGame(ctx, userID, envelope.Payload)
	case "game_action": // 自定義動作
		return h.handleGameAction(ctx, userID, envelope.Payload)
	default:
		slog.Warn("Unknown Action", "action", envelope.Action)
		return &gameRPC.GameResponse{
			Payload: []byte(`{"error": "unknown action"}`),
		}, nil
	}
}

func (h *Handler) handleEnterGame(ctx context.Context, userID string, payload []byte) (*gameRPC.GameResponse, error) {
	// 解析 Payload 取得 GameID 或設定
	var req protocol.EnterGameReq
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	// 這裡的 GameID 在 Stateful Service 中通常對應到「房間ID」或者「桌號」
	// 為了 Demo，我們直接把 int32 的 GameID 轉成 string RoomID
	roomID := fmt.Sprintf("room_%d", req.GameID)

	user := &domain.User{
		ID:   userID,
		Name: fmt.Sprintf("User-%s", userID),
	}

	// 加入房間
	if err := h.roomMgr.JoinRoom(roomID, user); err != nil {
		slog.Error("Failed to join room", "roomID", roomID, "err", err)
		return nil, err
	}

	// 回傳成功
	resp := protocol.EnterGameResp{
		Success: true,
		GameID:  req.GameID,
	}
	bytes, _ := json.Marshal(resp)

	return &gameRPC.GameResponse{
		Payload: bytes,
	}, nil
}

func (h *Handler) handleGameAction(ctx context.Context, userID string, payload []byte) (*gameRPC.GameResponse, error) {
	// 在真實場景，我們需要知道 User 在哪個 Room。
	// 由於 Stateful Service 是 Stateful 的，我們理論上可以在 Memory 裡查表。
	// 但這裡沒有傳入 RoomID，我們暫時 Hack 一下：假設所有人都去 room_2
	roomID := "room_2" // Harcode for demo

	r, err := h.roomMgr.GetRoom(roomID)
	if err != nil {
		return nil, err
	}

	if err := r.OnAction(userID, payload); err != nil {
		return nil, err
	}

	return &gameRPC.GameResponse{
		Payload: []byte(`{"status":"ok"}`),
	}, nil
}
