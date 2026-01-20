package framework

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	connector_sdk "github.com/JoeShih716/go-k8s-game-server/internal/sdk/connector" // SDK
	grpcpkg "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
)

// Peer 代表一個玩家在 Game Server 上的連線狀態
// 包含網路 Session 資訊與業務 User 資訊
type Peer struct {
	User          *domain.User // 業務使用者資訊
	SessionID     string       // 網路層 Session ID (Connector 識別用)
	ConnectorHost string       // 來源 Connector
	rpcPool       *grpcpkg.Pool
}

// NewPeer 建立新的 Peer
func NewPeer(user *domain.User, sessionID, connectorHost string, pool *grpcpkg.Pool) *Peer {
	return &Peer{
		User:          user,
		SessionID:     sessionID,
		ConnectorHost: connectorHost,
		rpcPool:       pool,
	}
}

// Send 發送訊息給玩家 (透過 Connector)
func (p *Peer) Send(ctx context.Context, payload []byte) error {
	if p.rpcPool == nil {
		return nil
	}

	conn, err := p.rpcPool.GetConnection(p.ConnectorHost)
	if err != nil {
		return err
	}

	// 使用 SDK 封裝
	client := connector_sdk.NewClient(conn)

	// 如果傳入的 ctx 是 Background，建議給個 Timeout
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	return client.Push(ctx, p.SessionID, payload)
}

// Kick 踢除玩家
func (p *Peer) Kick(ctx context.Context, reason string) error {
	if p.rpcPool == nil {
		return nil
	}
	conn, err := p.rpcPool.GetConnection(p.ConnectorHost)
	if err != nil {
		return err
	}

	client := connector_sdk.NewClient(conn)

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	return client.ForceKick(ctx, p.SessionID, reason)
}

// PeerManager 管理 Stateful 服務的 Peers
type PeerManager struct {
	peers sync.Map // map[string]*Peer (key: SessionID !! 注意是用 SessionID 當 Key, 若要用UserID需考慮多開)
}

func NewPeerManager() *PeerManager {
	return &PeerManager{}
}

func (m *PeerManager) Add(p *Peer) {
	m.peers.Store(p.SessionID, p)
}

func (m *PeerManager) Remove(sessionID string) {
	m.peers.Delete(sessionID)
}

func (m *PeerManager) Get(sessionID string) *Peer {
	if v, ok := m.peers.Load(sessionID); ok {
		return v.(*Peer)
	}
	return nil
}

// Broadcast 廣播給該 Pod 上所有玩家
func (m *PeerManager) Broadcast(ctx context.Context, payload []byte) {
	m.peers.Range(func(key, value any) bool {
		p := value.(*Peer)
		go func(peer *Peer) {
			if err := peer.Send(ctx, payload); err != nil {
				slog.Warn("Broadcast failed", "session_id", peer.SessionID, "error", err)
			}
		}(p)
		return true
	})
}
