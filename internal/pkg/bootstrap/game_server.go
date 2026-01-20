package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/framework"
	"github.com/JoeShih716/go-k8s-game-server/internal/di"
	rpcsdk "github.com/JoeShih716/go-k8s-game-server/internal/pkg/client/central"
	grpcpkg "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
)

// GameServerConfig 定義 Game Server 的專屬配置
type GameServerConfig struct {
	ServiceName string
	ServiceType proto.ServiceType
	GameIDs     []int32
	DefaultPort int
}

// RunGameServer 啟動通用的 Game Server 流程
// 接受 framework.GameHandler (業務邏輯) 而非底層 gRPC 註冊回呼
func RunGameServer(cfg GameServerConfig, handler framework.GameHandler) {
	// 1. 初始化 App
	app := NewApp(cfg.ServiceName)

	port := cfg.DefaultPort
	if p := app.Config.App.Port; p != 0 {
		port = p
	}

	// 2. 決定 Host
	host := cfg.ServiceName
	if app.Config.App.PodIP != "" {
		host = app.Config.App.PodIP
	}

	// 3. 連線 Central
	centralAddr := app.Config.Services["central"]
	if centralAddr == "" {
		centralAddr = "central:9003"
	}

	conn, err := grpc.NewClient(centralAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to central", "error", err)
	}

	// 4. 初始化 Registrar
	registrar := rpcsdk.NewRegistrar(conn, &rpcsdk.Config{
		ServiceName: cfg.ServiceName,
		ServiceType: cfg.ServiceType,
		Endpoint:    fmt.Sprintf("%s:%d", host, port),
		CentralAddr: centralAddr,
		GameIDs:     cfg.GameIDs,
	})

	// 5. gRPC Pool (共用組件)
	grpcPool := grpcpkg.NewPool()

	// 5.1 Initialize Redis Provider (Use DI)
	// 使用共用的 DI 初始化邏輯
	// 注意: Game Server 通常需要 User DB (for UserRepo) 和 Central DB (若有需要)
	// InitializeRedisProvider 會檢查 Config 並建立連線
	redisProvider, err := di.InitializeRedisProvider(context.Background(), app.Config)
	if err != nil {
		slog.Error("Failed to initialize Redis provider", "error", err)
		return
	}
	defer redisProvider.Close()

	// 5.2 Initialize Services
	// 使用 Generic DI Providers
	userSvc := di.ProvideUserService(app.Config, redisProvider)
	walletSvc := di.ProvideWalletService(app.Config, redisProvider)

	// 6. Framework Server Setup
	// 判斷是否為 Stateful (根據 ServiceType)
	isStateful := cfg.ServiceType == proto.ServiceType_STATEFUL
	gameServer := framework.NewServer(handler, grpcPool, isStateful, cfg.ServiceName, userSvc, walletSvc)

	// 7. gRPC Server Setup
	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	// 註冊 Framework Server 到 gRPC
	gameRPC.RegisterGameRPCServer(grpcServer, gameServer)
	reflection.Register(grpcServer)

	// 8. 執行
	app.Run(func() error {
		// 8.1 Listener
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}

		// 8.2 Background Registrar
		go func() {
			time.Sleep(1 * time.Second) // 等待 Server Ready

			// 無限重試
			ctx := context.Background()
			if err := registrar.Register(ctx); err != nil {
				slog.Error("Failed to register service", "error", err)
				return
			}
			registrar.StartHeartbeat(ctx)
		}()

		slog.Info("Game Service listening", "service", cfg.ServiceName, "port", port, "stateful", isStateful)
		return grpcServer.Serve(lis)
	}, func() {
		// Cleanup
		registrar.Stop(context.Background())
		grpcPool.Close()
		grpcServer.GracefulStop()
	})
}
