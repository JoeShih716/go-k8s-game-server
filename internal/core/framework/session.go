package framework

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/connectorRPC"
	grpcpkg "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
)

// Session 代表一個玩家在 Game Server 上的連線狀態 (邏輯上的)
type Session struct {
	UserID        string
	SessionID     string
	ConnectorHost string
	rpcPool       *grpcpkg.Pool
}

// NewSession 建立新的 Session
func NewSession(userID, sessionID, connectorHost string, pool *grpcpkg.Pool) *Session {
	return &Session{
		UserID:        userID,
		SessionID:     sessionID,
		ConnectorHost: connectorHost,
		rpcPool:       pool,
	}
}

// Send 發送訊息給玩家 (透過 Connector)
func (s *Session) Send(ctx context.Context, payload []byte) error {
	if s.rpcPool == nil {
		return nil // 或者 return error
	}

	conn, err := s.rpcPool.GetConnection(s.ConnectorHost)
	if err != nil {
		return err
	}

	client := connectorRPC.NewConnectorRPCClient(conn)

	// 如果傳入的 ctx 是 Background，建議給個 Timeout
	// 若 ctx 已經有 Deadline 則沿用
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	_, err = client.SendMessage(ctx, &connectorRPC.SendMessageReq{
		SessionIds: []string{s.SessionID},
		Payload:    payload,
	})
	return err
}

// Kick 踢除玩家
func (s *Session) Kick(ctx context.Context, reason string) error {
	if s.rpcPool == nil {
		return nil
	}
	conn, err := s.rpcPool.GetConnection(s.ConnectorHost)
	if err != nil {
		return err
	}
	client := connectorRPC.NewConnectorRPCClient(conn)

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	_, err = client.Kick(ctx, &connectorRPC.KickReq{
		SessionId: s.SessionID,
		Reason:    reason,
	})
	return err
}

// SessionManager 管理 Stateful 服務的 Sessions
type SessionManager struct {
	sessions sync.Map // map[string]*Session (key: SessionID)
}

func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

func (m *SessionManager) Add(s *Session) {
	m.sessions.Store(s.SessionID, s)
}

func (m *SessionManager) Remove(sessionID string) {
	m.sessions.Delete(sessionID)
}

func (m *SessionManager) Get(sessionID string) *Session {
	if v, ok := m.sessions.Load(sessionID); ok {
		return v.(*Session)
	}
	return nil
}

// Broadcast 廣播給該 Pod 上所有玩家 (注意：這不是全服廣播，僅限於此 Pod 管理的玩家)
func (m *SessionManager) Broadcast(ctx context.Context, payload []byte) {
	m.sessions.Range(func(key, value any) bool {
		s := value.(*Session)
		go func(sess *Session) {
			if err := sess.Send(ctx, payload); err != nil {
				slog.Warn("Broadcast failed", "session_id", sess.SessionID, "error", err)
			}
		}(s)
		return true
	})
}
