package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
	mysqlpkg "github.com/JoeShih716/go-k8s-game-server/pkg/mysql"
)

// ensure interface compliance
var _ ports.UserRepository = (*UserRepository)(nil)

// UserRepository 實作 ports.UserRepository
type UserRepository struct {
	client *mysqlpkg.Client
}

// NewUserRepository 建立 MySQL Repository
func NewUserRepository(client *mysqlpkg.Client) *UserRepository {
	return &UserRepository{
		client: client,
	}
}

// Create 建立使用者
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.client.DB().WithContext(ctx).Create(user).Error
}

// GetByID 根據 UserID 取得使用者
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	var user domain.User
	err := r.client.DB().WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新使用者資料
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	return r.client.DB().WithContext(ctx).Save(user).Error
}

// GetByUsername 根據使用者名稱取得使用者
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.client.DB().WithContext(ctx).Where("name = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
