package di

import (
	"context"

	infraRedis "github.com/JoeShih716/go-k8s-game-server/internal/infrastructure/redis"
	"github.com/JoeShih716/go-k8s-game-server/internal/kit/config"
)

// InitializeRedisProvider initializes the Redis provider with config
func InitializeRedisProvider(_ context.Context, cfg *config.Config) (*infraRedis.Provider, error) {
	// Note: NewProvider now accepts config.RedisGlobalConfig directly
	provider, err := infraRedis.NewProvider(cfg.Redis)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
