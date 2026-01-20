package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
)

// CentralService 負責中央核心業務 (使用者管理、登入、遊戲路由)
// 它組裝了各種 Repository 和 Service (Domain Ports)
type CentralService struct {
	userSvc   ports.UserService
	walletSvc ports.WalletService
	registry  ports.RegistryService
	logger    *slog.Logger
	counterID int
}

// NewCentralService 建立 Central Service
func NewCentralService(userRepo ports.UserService, walletSvc ports.WalletService, registry ports.RegistryService, logger *slog.Logger) *CentralService {
	return &CentralService{
		userSvc:   userRepo,
		walletSvc: walletSvc,
		registry:  registry,
		logger:    logger,
		counterID: 100000,
	}
}

// ---------------------------------------------------------
// Service Discovery Methods (Delegation)
// ---------------------------------------------------------

func (s *CentralService) RegisterService(ctx context.Context, req *centralRPC.RegisterRequest) (string, error) {
	return s.registry.Register(ctx, req)
}

func (s *CentralService) Heartbeat(ctx context.Context, leaseID string, load int32) error {
	return s.registry.Heartbeat(ctx, leaseID, load)
}

func (s *CentralService) DeregisterService(ctx context.Context, leaseID string) error {
	return s.registry.Deregister(ctx, leaseID)
}

func (s *CentralService) GetGameServerEndpoint(ctx context.Context, gameID int32) (string, proto.ServiceType, error) {
	return s.registry.SelectServiceByGame(ctx, gameID)
}

// ---------------------------------------------------------
// User Logic
// ---------------------------------------------------------

// Login 處理玩家登入邏輯
// 回傳 User 實體與可能發生的錯誤
func (s *CentralService) Login(ctx context.Context, token string) (*domain.User, error) {
	// 1. 驗證 Token (這裡暫時模擬，實際應呼叫 Auth Service 或 JWT verify)
	if token == "" {
		return nil, domain.ErrInvalidToken
	}
	var user *domain.User
	// 2. 查找使用者
	user, err := s.userSvc.GetUser(ctx, token)
	if err != nil {
		// 3. 如果使用者不存在，自動註冊 (Auto-Register)
		if err == ports.ErrUserNotFound {
			// 3.1. 建立使用者
			userID := fmt.Sprintf("%d", s.counterID)
			userName := fmt.Sprintf("guest-%d", s.counterID)
			s.counterID++
			user = domain.NewUser(userID, userName)
			// 3.2. 建立使用者
			err = s.userSvc.CreateGuestUser(ctx, token, user)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// 4. 更新餘額快照 (Wallet Service -> User Entity)
	// 這是一個 "Anti-Corruption Layer" 的行為，將 Wallet 的狀態同步到 User Cache
	balance, err := s.walletSvc.GetBalance(ctx, user.ID)
	if err == nil {
		user.Balance = balance
	} else {
		s.logger.Warn("Failed to fetch balance", "user_id", user.ID, "error", err)
	}

	// 5. 登入成功
	s.logger.Info("User logged in", "user_id", user.ID, "balance", user.Balance)
	return user, nil
}
