package wallet

import (
	"context"

	"github.com/shopspring/decimal"
)

// MockWallet 模擬錢包
type MockWallet struct{}

func NewMockWallet() *MockWallet {
	return &MockWallet{}
}

func (m *MockWallet) GetBalance(ctx context.Context, userID string) (decimal.Decimal, error) {
	// 模擬: 這是一種"富豪"錢包，每個人都有 1,000,000
	return decimal.NewFromInt(1000000), nil
}
