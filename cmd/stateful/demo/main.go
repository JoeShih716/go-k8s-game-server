package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/rpcsdk"
	statefuldemo "github.com/JoeShih716/go-k8s-game-server/internal/applications/stateful-demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/config"
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
	}

	port := "9002"
	if p := os.Getenv(config.EnvPort); p != "" {
		port = p
	}

	slog.Info("Starting Stateful Demo Service...", "port", port)

	// 3. 準備 Service Registrar
	// Stateful Service 必須註冊自己的 POD_IP 與支援的 GameID
	host := "stateful-demo"
	if podIP := os.Getenv(config.EnvPodIP); podIP != "" {
		host = podIP
	}

	centralAddr := cfg.Services["central"]
	if centralAddr == "" {
		centralAddr = "central:9003"
	}
	if addr := os.Getenv(config.EnvCentralAddr); addr != "" {
		centralAddr = addr
	}

	// 建立 Central 連線
	conn, err := grpc.NewClient(centralAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to central", "error", err)
		// 繼續執行，Registrar 會嘗試重連
	}

	registrar := rpcsdk.NewRegistrar(conn, &rpcsdk.Config{
		ServiceName: "stateful-demo",
		ServiceType: proto.ServiceType_STATEFUL,
		Endpoint:    fmt.Sprintf("%s:%s", host, port),
		CentralAddr: centralAddr,
		// GameIDs: Stateful Demo 負責 GameID 20000
		GameIDs: []int32{20000},
	})
	// 在背景啟動註冊與心跳
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 啟動 Registrar (背景執行)
	go func() {
		// 給 gRPC server 一點時間啟動
		time.Sleep(1 * time.Second)
		if err := registrar.Register(ctx); err != nil {
			slog.Error("Failed to register service", "error", err)
			return // 註冊失敗，依賴 Heartbeat 或重啟
		}
		// 註冊成功後啟動心跳
		registrar.StartHeartbeat(ctx)
	}()

	// 5. 建立 gRPC Server Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	// 6. 註冊 gRPC 服務
	grpcServer := grpc.NewServer()
	demoHandler := statefuldemo.NewHandler(host)
	gameRPC.RegisterGameRPCServer(grpcServer, demoHandler)

	// 啟用 gRPC Reflection
	reflection.Register(grpcServer)

	// 7. 啟動 Server
	go func() {
		slog.Info("Stateful Demo Service listening on " + lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// 8. Graceful Shutdown
	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// 先停止註冊 (發送 Deregister)
	registrar.Stop(context.Background())

	grpcServer.GracefulStop()
	slog.Info("Server exited")
}
