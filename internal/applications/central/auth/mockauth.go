package auth

import (
	"context"
	"strconv"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// MockAuthenticator 是一個測試用的驗證器
type MockAuthenticator struct {
	CountID int `json:"count_id"`
}

func NewMockAuthenticator() *MockAuthenticator {
	return &MockAuthenticator{
		CountID: 1000000,
	}
}

func (m *MockAuthenticator) Verify(ctx context.Context, token string) (*domain.User, error) {
	// 模擬驗證: 只要 Token 不為空且不是 "invalid" 就過
	if token == "" || token == "invalid" {
		return nil, domain.ErrInvalidToken
	}

	// 回傳 Mock User
	// 注意: 這裡使用了 domain.User，符合依賴反轉原則
	m.CountID++
	strID := strconv.Itoa(m.CountID)
	return &domain.User{
		ID:   strID,
		Name: "MockPlayer-" + strID,
	}, nil
}
