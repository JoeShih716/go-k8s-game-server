package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/auth"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/registry"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/service"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/wallet"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
	"github.com/JoeShih716/go-k8s-game-server/pkg/mysql"
	"github.com/JoeShih716/go-k8s-game-server/pkg/redis"
)

func main() {
	// 1. 初始化 App
	app := bootstrap.NewApp("central")

	// 2. 初始化 Redis
	rds, err := redis.NewClient(redis.Config{
		Addr:     app.Config.Redis.Addr,
		Password: app.Config.Redis.Password,
		DB:       app.Config.Redis.DB,
	})
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	// 3. 初始化 MySQL
	db, err := mysql.NewClient(mysql.Config{
		User:     app.Config.MySQL.User,
		Password: app.Config.MySQL.Password,
		Host:     app.Config.MySQL.Host,
		Port:     app.Config.MySQL.Port,
		DBName:   app.Config.MySQL.DBName,
		LogLevel: "error", // Default log level
	})
	if err != nil {
		slog.Error("Failed to connect to MySQL", "error", err)
		os.Exit(1)
	}
	slog.Info("Database connected", "db", app.Config.MySQL.DBName)

	// 4. 初始化核心組件
	reg := registry.NewRedisRegistry(rds)
	authenticator := auth.NewMockAuthenticator()
	mockWallet := wallet.NewMockWallet()
	svc := service.NewService(reg, db, authenticator, mockWallet)

	// 5. 啟動服務
	// Central 的預設 Port 是 9003
	// 若 config/config.yaml 或環境變數 (PORT) 有設定，則使用該設定
	port := 9003
	if p := app.Config.App.Port; p != 0 {
		port = p
	}

	app.Run(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}

		grpcServer := grpc.NewServer(
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             5 * time.Second,
				PermitWithoutStream: true,
			}),
		)
		centralRPC.RegisterCentralRPCServer(grpcServer, svc)
		reflection.Register(grpcServer)

		slog.Info("Central Service listening", "port", port)
		return grpcServer.Serve(lis)
	}, func() {
		// Cleanup
		rds.Close()
		// db.Close() // GORM db generic interface might need type assertion or sqlDB.Close(), skip for now or add helper
	})
}
