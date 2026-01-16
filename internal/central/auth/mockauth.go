package auth

import (
	"context"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// MockAuthenticator 是一個測試用的驗證器
type MockAuthenticator struct{}

func NewMockAuthenticator() *MockAuthenticator {
	return &MockAuthenticator{}
}

func (m *MockAuthenticator) Verify(ctx context.Context, token string) (*domain.User, error) {
	// 模擬驗證: 只要 Token 不為空且不是 "invalid" 就過
	if token == "" || token == "invalid" {
		return nil, domain.ErrInvalidToken
	}

	// 回傳 Mock User
	// 注意: 這裡使用了 domain.User，符合依賴反轉原則
	return &domain.User{
		ID:      "user-" + token[:min(len(token), 8)],
		Name:    "MockPlayer",
		Balance: 10000,
		Avatar:  "http://example.com/avatar.png",
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
