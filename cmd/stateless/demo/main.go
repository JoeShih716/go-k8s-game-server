package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/internal/applications/central/rpcsdk"
	demo "github.com/JoeShih716/go-k8s-game-server/internal/applications/stateless-demo"
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

	port := "9001"
	if p := os.Getenv(config.EnvPort); p != "" {
		port = p
	}

	slog.Info("Starting Stateless Demo Service...", "port", port)

	// 3. 啟動 Service Registrar (自動註冊)
	// 優先使用 POD_IP (K8s Downward API)，若無則 fallback 到 DNS 名稱 (Docker Compose)
	host := "stateless-demo"
	if podIP := os.Getenv(config.EnvPodIP); podIP != "" {
		host = podIP
	}
	myEndpoint := fmt.Sprintf("%s:%s", host, port)

	// 從 config 取得 Central 地址
	central := cfg.Services["central"]
	if central == "" {
		central = "central:9003" // Fallback
	}

	// 建立 gRPC 連線到 Central
	centralConn, err := grpc.NewClient(central,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("Failed to connect to Central", "addr", central, "error", err)
		// 繼續執行，Registrar 會嘗試重連 (如果我們把 conn 傳進去的話需確保 conn 是有效的嗎? grpc.NewClient 是 non-blocking)
	}

	registrar := rpcsdk.NewRegistrar(centralConn, &rpcsdk.Config{
		ServiceName: "stateless-demo", // 與 docker-compose service name 一致 (或 logic name)
		ServiceType: proto.ServiceType_STATELESS,
		Endpoint:    myEndpoint,
		GameIDs:     []int32{10000}, // 假設這個服務負責 Game ID 10000
	})

	// 在背景啟動註冊與心跳
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 啟動註冊
	go func() {
		if err := registrar.Register(ctx); err != nil {
			slog.Error("Registrar failed to register", "error", err)
			return // 註冊失敗是否要 Exit?
		}
		// 註冊成功後啟動心跳
		registrar.StartHeartbeat(ctx)
	}()

	// 4. 建立 gRPC Server Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	// 5. 註冊 gRPC 服務
	grpcServer := grpc.NewServer()
	demoHandler := demo.NewHandler(host)
	proto.RegisterGameServiceServer(grpcServer, demoHandler)

	// 啟用 gRPC Reflection
	reflection.Register(grpcServer)

	// 6. Graceful Shutdown
	go func() {
		slog.Info("Server listening on " + lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve", "error", err)
			os.Exit(1)
		}
	}()

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
