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
	"google.golang.org/grpc/reflection"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/internal/central/rpctool"
	"github.com/JoeShih716/go-k8s-game-server/internal/config"
	"github.com/JoeShih716/go-k8s-game-server/internal/stateless/demo"
)

func main() {
	// 1. 初始化 Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. 讀取 Config
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	cfg, err := config.Load(env)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
	}

	port := "9001"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	slog.Info("Starting Stateless Demo Service...", "port", port)

	// 3. 啟動 Service Registrar (自動註冊)
	// 此處 endpoint 不應該寫死 localhost，應該是外部可訪問的 IP (K8s Service DNS)
	// 在 Docker Compose 中，這台機器的名字叫 "stateless-demo"
	myEndpoint := fmt.Sprintf("stateless-demo:%s", port)

	// 從 config 取得 Central 地址
	coordAddr := cfg.Services["central"]
	if coordAddr == "" {
		coordAddr = "central:9003" // Fallback
	}

	registrar := rpctool.NewRegistrar(coordAddr, &proto.RegisterRequest{
		ServiceName: "slots-service-demo",
		Type:        proto.ServiceType_STATELESS,
		Endpoint:    myEndpoint,
	})

	// 在背景啟動註冊與心跳
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := registrar.Start(ctx); err != nil {
			slog.Error("Registrar failed", "error", err)
		}
	}()

	// 4. 建立 gRPC Server Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	// 5. 註冊 gRPC 服務
	grpcServer := grpc.NewServer()
	demoHandler := demo.NewHandler()
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
