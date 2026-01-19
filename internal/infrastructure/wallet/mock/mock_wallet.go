package mock

import (
	"context"

	"github.com/shopspring/decimal"
)

// MockWallet 模擬錢包
type MockWallet struct{
	userBalances map[string]decimal.Decimal
}

func NewMockWallet() *MockWallet {
	return &MockWallet{
		userBalances: make(map[string]decimal.Decimal),
	}
}

func (m *MockWallet) GetBalance(ctx context.Context, userID string) (decimal.Decimal, error) {
	if balance, exists := m.userBalances[userID]; exists {
		return balance, nil
	}
	m.userBalances[userID] = decimal.NewFromInt(1000000)
	return m.userBalances[userID], nil
}

func (m *MockWallet) Deposit(ctx context.Context, userID string, amount decimal.Decimal, reason string) error {
	m.userBalances[userID] = m.userBalances[userID].Add(amount)
	return nil
}

func (m *MockWallet) Withdraw(ctx context.Context, userID string, amount decimal.Decimal, reason string) error {
	m.userBalances[userID] = m.userBalances[userID].Sub(amount)
	return nil
}
