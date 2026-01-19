package ports

import (
	"context"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// UserService 定義使用者服務的介面
//
//go:generate mockgen -destination=../../../test/mocks/core/ports/mock_user_service.go -package=mock_ports github.com/JoeShih716/go-k8s-game-server/internal/core/ports UserService
type UserService interface {

	// GetUser 根據 Token 取得使用者
	GetUser(ctx context.Context, token string) (*domain.User, error)
	// CreateGuestUser 建立訪客使用者
	CreateGuestUser(ctx context.Context, token string, user *domain.User) error
	// GetUserByID 根據 ID 取得使用者
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
}
