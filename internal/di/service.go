package di

import (
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/internal/config"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
	infraRedis "github.com/JoeShih716/go-k8s-game-server/internal/infrastructure/redis"
	registry "github.com/JoeShih716/go-k8s-game-server/internal/infrastructure/service_discovery/redis"
	user "github.com/JoeShih716/go-k8s-game-server/internal/infrastructure/user/redis"
	wallet "github.com/JoeShih716/go-k8s-game-server/internal/infrastructure/wallet/mock"
)

// ProvideUserService creates a UserService using the 'user' Redis DB
func ProvideUserService(cfg *config.Config, redisProvider *infraRedis.Provider) ports.UserService {
	switch cfg.App.Env {
	case "prod":
		return user.NewUserService(redisProvider.GetUser())
	default:
		userRedisClient := redisProvider.GetUser()
		if userRedisClient == nil {
			panic("Redis User DB (key: 'user') not found in config")
		}
		return user.NewUserService(userRedisClient)
	}
}

// ProvideRegistry creates a ServiceRegistry using the 'central' Redis DB
func ProvideRegistry(cfg *config.Config, redisProvider *infraRedis.Provider) ports.RegistryService {
	centralRedisClient := redisProvider.GetCentral()
	if centralRedisClient == nil {
		panic("Redis Central DB (key: 'central') not found in config")
	}
	return registry.NewRedisRegistry(centralRedisClient)
}

// ProvideWalletService selects implementation based on Environment
func ProvideWalletService(cfg *config.Config, redisProvider *infraRedis.Provider) ports.WalletService {
	if cfg.App.Env == "prod" {
		slog.Warn("Using Mock Wallet in PROD (Not implemented yet)")
		return wallet.NewMockWallet()
	}
	slog.Info("Using Mock Wallet Service")
	return wallet.NewMockWallet()
}
