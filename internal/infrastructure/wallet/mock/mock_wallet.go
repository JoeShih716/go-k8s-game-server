package mock

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
	return decimal.NewFromInt(1000000), nil
}

func (m *MockWallet) Deposit(ctx context.Context, userID string, amount decimal.Decimal, reason string) error {
	return nil
}

func (m *MockWallet) Withdraw(ctx context.Context, userID string, amount decimal.Decimal, reason string) error {
	return nil
}
