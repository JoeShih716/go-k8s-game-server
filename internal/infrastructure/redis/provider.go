package redis

import (
	"fmt"
	"log/slog"

	"github.com/JoeShih716/go-k8s-game-server/internal/kit/config"
	pkgRedis "github.com/JoeShih716/go-k8s-game-server/pkg/redis"
)

type DBName string

const (
	DBNameUser    DBName = "user"
	DBNameCentral DBName = "central"
	// Future: DBNameGame, DBNameJackpot...
)

// DBSupplier defines the interface for retrieving specific Redis DB clients
type DBSupplier interface {
	GetUser() *pkgRedis.Client
	GetCentral() *pkgRedis.Client
	Close() error
}

type Provider struct {
	databases map[DBName]*pkgRedis.Client
}

// NewProvider creates clients for all configured redis databases
func NewProvider(globalCfg config.RedisGlobalConfig) (*Provider, error) {
	clients := make(map[DBName]*pkgRedis.Client)

	// Iterate over the configured databases (e.g., "central", "user")
	for dbKey, dbConfig := range globalCfg.DB {
		// Combine global settings (Addr, Password) with specific DB index
		client, err := pkgRedis.NewClient(pkgRedis.Config{
			Addr:     globalCfg.Addr,
			Password: globalCfg.Password,
			DB:       dbConfig.Name, // "name" maps to DB Index as per user request
		})
		if err != nil {
			// If one fails, close already created ones and return error
			for _, c := range clients {
				c.Close()
			}
			return nil, fmt.Errorf("failed to init redis db '%s': %w", dbKey, err)
		}

		clients[DBName(dbKey)] = client
	}

	return &Provider{databases: clients}, nil
}

func (p *Provider) GetUser() *pkgRedis.Client {
	if client, ok := p.databases[DBNameUser]; ok {
		return client
	}
	// Fallback or panic? For now, avoid nil pointer if possible, or return nil
	slog.Warn("Redis User DB not found in config")
	return nil
}

func (p *Provider) GetCentral() *pkgRedis.Client {
	if client, ok := p.databases[DBNameCentral]; ok {
		return client
	}
	slog.Warn("Redis Central DB not found in config")
	return nil
}

func (p *Provider) Close() error {
	for _, client := range p.databases {
		client.Close()
	}
	return nil
}

// ProvideRedisDB is to satisfy potential interface requirements
func (p *Provider) ProvideRedisDB() DBSupplier {
	return p
}
