package engine

import (
	"context"
	"errors"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
	grpcpkg "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
)

// GameHandler 使用者需實作的業務邏輯介面
//
//go:generate mockgen -destination=../../test/mocks/engine/mock_game_handler.go -package=mock_engine -source=server.go -self_package github.com/JoeShih716/go-k8s-game-server/test/mocks/engine GameHandler
type GameHandler interface {
	OnJoin(ctx context.Context, peer *Peer) error
	OnQuit(ctx context.Context, peer *Peer) error
	OnMessage(ctx context.Context, peer *Peer, payload []byte) ([]byte, error)
}

// BaseHandler 提供 GameHandler 的預設空實作 (Optional)
type BaseHandler struct{}

func (_ *BaseHandler) OnJoin(_ context.Context, _ *Peer) error { return nil }
func (_ *BaseHandler) OnQuit(_ context.Context, _ *Peer) error { return nil }
func (_ *BaseHandler) OnMessage(_ context.Context, _ *Peer, _ []byte) ([]byte, error) {
	return nil, nil
}

// Server 實作 gameRPC.GameRPCServer，負責 Peer 生命週期管理
type Server struct {
	gameRPC.UnimplementedGameRPCServer
	handler     GameHandler
	peerMgr     *PeerManager // Renamed from sessMgr
	grpcPool    *grpcpkg.Pool
	isStateful  bool
	serviceName string
	// Injected Services
	userSvc   ports.UserService
	walletSvc ports.WalletService
}

// NewServer 建立 Framework Server
// Dependencies: UserService, WalletService
func NewServer(
	handler GameHandler,
	pool *grpcpkg.Pool,
	isStateful bool,
	serviceName string,
	userSvc ports.UserService,
	walletSvc ports.WalletService,
) *Server {
	var mgr *PeerManager
	if isStateful {
		mgr = NewPeerManager()
	}

	return &Server{
		handler:     handler,
		peerMgr:     mgr,
		grpcPool:    pool,
		isStateful:  isStateful,
		serviceName: serviceName,
		userSvc:     userSvc,
		walletSvc:   walletSvc,
	}
}

// PeerManager 回傳 PeerManager
func (s *Server) PeerManager() *PeerManager {
	return s.peerMgr
}

// OnPlayerJoin 玩家進入
func (s *Server) OnPlayerJoin(ctx context.Context, req *gameRPC.JoinReq) (*gameRPC.JoinResp, error) {
	userID := req.Header.UserId
	sessID := req.Header.SessionId
	connHost := req.ConnectorHost

	slog.Info("OnPlayerJoin", "service", s.serviceName, "user_id", userID, "session_id", sessID)

	// 1. Fetch User Data (using UserService)
	user, err := s.userSvc.GetUserByID(ctx, userID)
	if err != nil {
		slog.Error("Failed to get user info", "user_id", userID, "error", err)
		return &gameRPC.JoinResp{Code: proto.ErrorCode_SERVER_ERROR}, err
	}

	// 2. Refresh Balance (Synch with Wallet) -> Populate User.Balance
	balance, err := s.walletSvc.GetBalance(ctx, userID)
	if err == nil {
		user.Balance = balance
	} else {
		slog.Warn("Failed to fetch balance in OnPlayerJoin", "user_id", userID, "error", err)
	}

	peer := NewPeer(user, sessID, connHost, s.grpcPool)

	if s.isStateful {
		s.peerMgr.Add(peer)
	}

	// 呼叫業務邏輯
	if err := s.handler.OnJoin(ctx, peer); err != nil {
		slog.Error("Handler.OnJoin failed", "error", err)
		if s.isStateful {
			s.peerMgr.Remove(sessID)
		}
		return nil, err
	}

	return &gameRPC.JoinResp{Code: proto.ErrorCode_SUCCESS}, nil
}

// OnPlayerQuit 玩家離開
func (s *Server) OnPlayerQuit(ctx context.Context, req *gameRPC.QuitReq) (*gameRPC.QuitResp, error) {
	sessID := req.Header.SessionId
	slog.Info("OnPlayerQuit", "service", s.serviceName, "session_id", sessID)

	var peer *Peer

	if s.isStateful {
		peer = s.peerMgr.Get(sessID)
		if peer == nil {
			slog.Warn("Peer not found during Quit", "session_id", sessID)
			return &gameRPC.QuitResp{Code: proto.ErrorCode_SUCCESS}, nil
		}
	} else {
		// Stateless: 建立 Partial Peer
		mockUser := &domain.User{ID: req.Header.UserId}
		peer = NewPeer(mockUser, sessID, "", s.grpcPool)
	}

	// 呼叫業務邏輯
	if err := s.handler.OnQuit(ctx, peer); err != nil {
		slog.Error("Handler.OnQuit failed", "error", err)
	}

	if s.isStateful {
		s.peerMgr.Remove(sessID)
	}

	return &gameRPC.QuitResp{Code: proto.ErrorCode_SUCCESS}, nil
}

// OnMessage 處理訊息
func (s *Server) OnMessage(ctx context.Context, req *gameRPC.MsgReq) (*gameRPC.MsgResp, error) {
	sessID := req.Header.SessionId

	var peer *Peer

	if s.isStateful {
		peer = s.peerMgr.Get(sessID)
		if peer == nil {
			return &gameRPC.MsgResp{
				Code:         proto.ErrorCode_SERVER_ERROR,
				ErrorMessage: "Peer not found",
			}, errors.New("peer not found")
		}
	} else {
		// Stateless: 建立 Partial Peer
		mockUser := &domain.User{ID: req.Header.UserId}
		peer = NewPeer(mockUser, sessID, "", s.grpcPool)
	}

	respPayload, err := s.handler.OnMessage(ctx, peer, req.Payload)
	if err != nil {
		slog.Error("Handler.OnMessage failed", "error", err)
		return &gameRPC.MsgResp{
			Code:         proto.ErrorCode_SERVER_ERROR,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &gameRPC.MsgResp{
		Code:    proto.ErrorCode_SUCCESS,
		Payload: respPayload,
	}, nil
}
