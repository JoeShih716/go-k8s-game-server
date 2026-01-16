package wallet

import (
	"context"

	"github.com/shopspring/decimal"
)

// Wallet 定義錢包服務介面
type Wallet interface {
	// GetBalance 取得玩家餘額
	GetBalance(ctx context.Context, userID string) (decimal.Decimal, error)
}
