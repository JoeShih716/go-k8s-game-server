package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
)

// CentralService 負責中央核心業務 (使用者管理、登入、遊戲路由)
// 它組裝了各種 Repository 和 Service (Domain Ports)
type CentralService struct {
	userRepo  ports.UserRepository
	walletSvc ports.WalletService
	registry  ports.ServiceRegistry
	logger    *slog.Logger
}

// NewCentralService 建立 Central Service
func NewCentralService(userRepo ports.UserRepository, walletSvc ports.WalletService, registry ports.ServiceRegistry, logger *slog.Logger) *CentralService {
	return &CentralService{
		userRepo:  userRepo,
		walletSvc: walletSvc,
		registry:  registry,
		logger:    logger,
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

func (s *CentralService) GetGameServerEndpoint(ctx context.Context, gameID int32) (string, error) {
	return s.registry.SelectServiceByGame(ctx, gameID)
}

// ---------------------------------------------------------
// User Logic
// ---------------------------------------------------------

// Login 處理玩家登入邏輯
// 回傳 User 實體與可能發生的錯誤
func (s *CentralService) Login(ctx context.Context, token string) (*domain.User, error) {
	// 1. 驗證 Token (這裡暫時模擬，實際應呼叫 Auth Service 或 JWT verify)
	if token == "invalid-token" {
		return nil, domain.ErrInvalidToken
	}

	// 範例：假設 Token 就是 UserID 或 Username，這裡做簡單模擬
	// 在真實場景中，應該從 Token 解析出 UserID
	userID := token // Mock

	// 2. 查找使用者
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 3. 如果使用者不存在，自動註冊 (Auto-Register)
	if user == nil {
		s.logger.Info("User not found, registering new user", "user_id", userID)
		newUser := domain.NewUser(userID, "Guest-"+userID) // 預設名稱
		// 給點初始錢
		// 注意: domain.User.Balance 是 int64 in cents, WalletService 使用 Decimal
		// 這裡假設 Create 後，WalletService 會負責初始金額，或者我們在此呼叫 Deposit

		if err := s.userRepo.Create(ctx, newUser); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		user = newUser
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
