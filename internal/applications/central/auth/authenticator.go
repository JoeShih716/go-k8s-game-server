package auth

import (
	"context"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// Authenticator 定義登入驗證策略
type Authenticator interface {
	// Verify 驗證 Token 並回傳領域層 User 物件
	Verify(ctx context.Context, token string) (*domain.User, error)
}
