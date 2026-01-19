package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

// WalletService 定義錢包相關的業務邏輯介面
//
//go:generate mockgen -destination=../../../test/mocks/core/ports/mock_wallet_service.go -package=mock_ports github.com/JoeShih716/go-k8s-game-server/internal/core/ports WalletService
type WalletService interface {
	// GetBalance 取得餘額
	GetBalance(ctx context.Context, userID string) (decimal.Decimal, error)

	// Deposit 存款 (增加餘額)
	Deposit(ctx context.Context, userID string, amount decimal.Decimal, reason string) error

	// Withdraw 提款 (扣除餘額)
	Withdraw(ctx context.Context, userID string, amount decimal.Decimal, reason string) error
}
