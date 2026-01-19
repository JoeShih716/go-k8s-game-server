package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
	"github.com/JoeShih716/go-k8s-game-server/pkg/redis"
)

const (
	KeyTokenUserID = "token:%s"
	KeyUserID      = "user:%s"
)

type UserService struct {
	rds *redis.Client
}

// GetUserByID implements ports.UserService.
func (service *UserService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	userKey := fmt.Sprintf(KeyUserID, id)
	var data *domain.User
	err := service.rds.GetStruct(ctx, userKey, &data)
	if err != nil {
		if redis.IsNil(err) {
			return nil, ports.ErrUserNotFound
		}
		return nil, err
	}
	return data, nil
}

// GetUser implements ports.UserService.
func (service *UserService) GetUser(ctx context.Context, token string) (*domain.User, error) {
	tokenKey := fmt.Sprintf(KeyTokenUserID, token)
	userID, err := service.rds.Get(ctx, tokenKey)
	if err != nil {
		if redis.IsNil(err) {
			return nil, ports.ErrUserNotFound
		}
		return nil, err
	}
	return service.GetUserByID(ctx, userID)
}

// CreateGuestUser implements ports.UserService.
// 建立訪客遊玩
func (service *UserService) CreateGuestUser(ctx context.Context, token string, user *domain.User) error {
	tokenKey := fmt.Sprintf(KeyTokenUserID, token)
	err := service.rds.Set(ctx, tokenKey, user.ID, time.Hour)
	if err != nil {
		return err
	}
	userKey := fmt.Sprintf(KeyUserID, user.ID)
	// 只給使用一小時
	err = service.rds.SetStruct(ctx, userKey, user, time.Hour)
	if err != nil {
		return err
	}
	return nil
}

var _ ports.UserService = (*UserService)(nil)

func NewUserService(client *redis.Client) *UserService {
	service := &UserService{
		rds: client,
	}
	return service
}
