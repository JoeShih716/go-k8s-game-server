package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/central/handler"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/central/service"
	"github.com/JoeShih716/go-k8s-game-server/internal/di"
	infraRedis "github.com/JoeShih716/go-k8s-game-server/internal/infrastructure/redis"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
)

func main() {
	// 1. 初始化 App (載入 Config, Logger)
	app := bootstrap.NewApp("central")
	ctx := context.Background()

	slog.InfoContext(ctx, "Initializing dependencies concurrently...")

	// 2. 並行初始化資源 (Redis, DB...)
	// 參考 slot-go 的 Concurrent Init 模式
	redisChan := make(chan *infraRedis.Provider, 1)
	errChan := make(chan error, 2) // Buffer size = number of concurrent tasks

	// Task A: Init Redis
	go func() {
		provider, err := di.InitializeRedisProvider(ctx, app.Config)
		if err != nil {
			errChan <- fmt.Errorf("redis init failed: %w", err)
			return
		}
		redisChan <- provider
	}()

	// (Future) Task B: Init MySQL
	// go func() { ... }()

	// 3. 收集初始化結果
	var redisProvider *infraRedis.Provider

	// 這裡只有 1 個任務，若是多個可用 loop + select
	// 為了擴充性，這裡寫成 loop 形式 (雖然只有 1 iteration)
	const numTasks = 1
	for i := 0; i < numTasks; i++ {
		select {
		case provider := <-redisChan:
			redisProvider = provider
			slog.Info("Redis initialized")
		case err := <-errChan:
			slog.Error("Dependency initialization failed", "error", err)
			os.Exit(1)
		}
	}

	// 確保資源釋放
	defer func() {
		if redisProvider != nil {
			redisProvider.Close()
		}
	}()

	// 4. 初始化 Services (Wiring)
	// 使用 Generic DI Providers 取得各個單一職責的 Service
	userService := di.ProvideUserService(app.Config, redisProvider)
	walletService := di.ProvideWalletService(app.Config, redisProvider)
	svcRegistry := di.ProvideRegistry(app.Config, redisProvider)

	// 5. 組裝 Central Service (Application Service)
	centralSvc := service.NewCentralService(
		userService,
		walletService,
		svcRegistry,
		app.Logger,
	)

	// 任務: 定期清理 Zombie Services (每 30 秒)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := svcRegistry.CleanupDeadServices(context.Background()); err != nil {
				slog.Warn("CleanupDeadServices failed", "error", err)
			}
		}
	}()

	// Handler Layer
	grpcHandler := handler.NewGRPCHandler(centralSvc)

	// 5. 啟動服務
	port := 8090 // Default internal gRPC port
	if p := app.Config.App.GrpcPort; p != 0 {
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

		centralRPC.RegisterCentralRPCServer(grpcServer, grpcHandler)
		reflection.Register(grpcServer)

		slog.Info("Central Service listening", "port", port)
		return grpcServer.Serve(lis)
	}, func() {
		// Cleanup handled by defer above
	})
}
