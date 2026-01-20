package di

import (
	"context"

	"github.com/JoeShih716/go-k8s-game-server/internal/config"
	infraRedis "github.com/JoeShih716/go-k8s-game-server/internal/infrastructure/redis"
)

// InitializeRedisProvider initializes the Redis provider with config
func InitializeRedisProvider(ctx context.Context, cfg *config.Config) (*infraRedis.Provider, error) {
	// Note: NewProvider now accepts config.RedisGlobalConfig directly
	provider, err := infraRedis.NewProvider(cfg.Redis)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
