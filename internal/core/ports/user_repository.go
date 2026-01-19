package ports

import (
	"context"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// UserRepository 定義使用者資料的存取介面
//
//go:generate mockgen -destination=../../../test/mocks/core/ports/mock_user_repository.go -package=mock_ports github.com/JoeShih716/go-k8s-game-server/internal/core/ports UserRepository
type UserRepository interface {
	// Create 建立使用者
	Create(ctx context.Context, user *domain.User) error

	// GetByID 根據 UserID 取得使用者
	GetByID(ctx context.Context, userID string) (*domain.User, error)

	// Update 更新使用者資料
	Update(ctx context.Context, user *domain.User) error

	// GetByUsername 根據使用者名稱取得使用者 (Optional, for login)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}
