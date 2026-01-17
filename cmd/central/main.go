package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/auth"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/registry"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/service"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/wallet"
	"github.com/JoeShih716/go-k8s-game-server/internal/config"
	"github.com/JoeShih716/go-k8s-game-server/pkg/mysql"
	"github.com/JoeShih716/go-k8s-game-server/pkg/redis"
)

func main() {
	// 1. 初始化 Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. 讀取 Config
	env := os.Getenv(config.EnvAppEnv)
	if env == "" {
		env = "local"
	}
	cfg, err := config.Load(env)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("Config loaded", "env", env)

	// 3. 初始化基礎設施 (Redis & MySQL)
	rds, err := redis.NewClient(redis.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rds.Close()

	// 目前還沒用到 MySQL，先註解保留擴充性
	db, err := mysql.NewClient(mysql.Config{
		User:     cfg.MySQL.User,
		Password: cfg.MySQL.Password,
		Host:     cfg.MySQL.Host,
		Port:     cfg.MySQL.Port,
		DBName:   cfg.MySQL.DBName,
		LogLevel: "error", // Default log level
	})
	if err != nil {
		slog.Error("Failed to connect to MySQL", "error", err)
		os.Exit(1)
	}
	slog.Info("Database connected", "db", cfg.MySQL.DBName)

	// 4. 初始化核心組件
	// 4.1 Service Registry (Redis Based)
	reg := registry.NewRedisRegistry(rds)

	// 4.2 Central Service (gRPC 實作)
	// 初始化 Mock Authenticator & Mock Wallet
	authenticator := auth.NewMockAuthenticator()
	mockWallet := wallet.NewMockWallet()
	h := service.NewService(reg, db, authenticator, mockWallet)

	// 5. 啟動 gRPC Server
	// Central 固定跑在 9003 (參考 config/local.yaml)
	// 在 K8s 中通常也是開這 port
	port := "9003"
	if p := os.Getenv(config.EnvPort); p != "" {
		port = p
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	centralRPC.RegisterCentralRPCServer(grpcServer, h)
	reflection.Register(grpcServer)

	// 6. Graceful Shutdown
	go func() {
		slog.Info("Central Service listening on " + lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down Central Service...")
	grpcServer.GracefulStop()
	slog.Info("Central Service exited")
}
