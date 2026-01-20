package handler

import (
	"context"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/connectorRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/connector/session"
)

// GrpcHandler 實作 connectorRPC.ConnectorRPCServer 介面
type GrpcHandler struct {
	connectorRPC.UnimplementedConnectorRPCServer
	sessionMgr *session.Manager
}

// NewGrpcHandler 建立 gRPC Handler
func NewGrpcHandler(mgr *session.Manager) *GrpcHandler {
	return &GrpcHandler{
		sessionMgr: mgr,
	}
}

// SendMessage 發送訊息給指定玩家 (單播/多播)
func (h *GrpcHandler) SendMessage(ctx context.Context, req *connectorRPC.SendMessageReq) (*connectorRPC.SendMessageResp, error) {
	slog.Info("ConnectorRPC Receive SendMessage", "count", len(req.SessionIds), "payload_len", len(req.Payload))

	// 遍歷所有目標 SessionID
	for _, sessID := range req.SessionIds {
		client, ok := h.sessionMgr.Get(sessID)
		if !ok {
			// 若找不到玩家，紀錄 Warn 但不中斷其他發送
			slog.Warn("SendMessage: Session not found", "session_id", sessID)
			continue
		}

		// 直接發送原始 Bytes
		if err := client.Send(string(req.Payload)); err != nil {
			slog.Error("SendMessage: Failed to write", "session_id", sessID, "error", err)
		}
	}

	return &connectorRPC.SendMessageResp{
		Code: proto.ErrorCode_SUCCESS,
	}, nil
}

// Kick 強制踢除玩家
func (h *GrpcHandler) Kick(ctx context.Context, req *connectorRPC.KickReq) (*connectorRPC.KickResp, error) {
	slog.Info("ConnectorRPC Receive Kick", "session_id", req.SessionId, "reason", req.Reason)

	client, ok := h.sessionMgr.Get(req.SessionId)
	if !ok {
		slog.Warn("Kick: Session not found", "session_id", req.SessionId)
		return &connectorRPC.KickResp{
			Code: proto.ErrorCode_SUCCESS, // 即使找不到也回傳成功，因為目標已斷線
		}, nil
	}

	// 執行踢除
	if err := client.Kick(req.Reason); err != nil {
		slog.Error("Kick: Failed to close connection", "session_id", req.SessionId, "error", err)
	}

	return &connectorRPC.KickResp{
		Code: proto.ErrorCode_SUCCESS,
	}, nil
}
