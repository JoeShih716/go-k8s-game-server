package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/JoeShih716/go-k8s-game-server/internal/central/rpcsdk"
	"github.com/JoeShih716/go-k8s-game-server/internal/config"
	"github.com/JoeShih716/go-k8s-game-server/internal/connector/handler"
	"github.com/JoeShih716/go-k8s-game-server/internal/connector/router"
	"github.com/JoeShih716/go-k8s-game-server/internal/connector/session"
	grpcpkg "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
	"github.com/JoeShih716/go-k8s-game-server/pkg/wss"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. 初始化 Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. 讀取設定
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	cfg, err := config.Load(env)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting Connector Service...", "env", env, "port", cfg.App.Port)

	// 3. 初始化 Session Manager
	sessionMgr := session.NewManager()

	// 4. 初始化 Central Client (The "Brain" Link)
	centralAddr := cfg.Services["central"] // From config/local.yaml
	if centralAddr == "" {
		centralAddr = "central:9003" // Fallback
	}

	// 建立 gRPC 連線
	centralConn, err := grpc.NewClient(centralAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("Failed to connect to Central", "error", err)
	} else {
		slog.Info("Connected to Central", "addr", centralAddr)
	}
	// 即使連線失敗，client 仍可建立 (gRPC 會自動重連，或在 Call 時報錯)
	centralClient := rpcsdk.NewClient(centralConn)

	// 5. 初始化 Discovery (Switch to CentralDiscovery)
	// discovery := router.NewStaticDiscovery(cfg.Services)
	discovery := router.NewCentralDiscovery(centralClient)
	smartRouter := router.NewSmartRouter(discovery)

	// 6. 初始化 gRPC Pool
	grpcPool := grpcpkg.NewPool()
	defer grpcPool.Close()

	// 7. 初始化 WebSocket Handler
	wsHandler := handler.NewWebsocketHandler(sessionMgr, smartRouter, grpcPool, centralClient)

	// 5. 初始化 WebSocket Server
	wsConfig := &wss.Config{
		AllowedOrigins:  cfg.WSS.AllowedOrigins,
		ReadBufferSize:  cfg.WSS.ReadBufferSize,
		WriteBufferSize: cfg.WSS.WriteBufferSize,
		WriteWait:       time.Duration(cfg.WSS.WriteWaitSec) * time.Second,
		PongWait:        time.Duration(cfg.WSS.PongWaitSec) * time.Second,
		MaxMessageSize:  cfg.WSS.MaxMessageSize,
	}
	// 需要傳入 Context 與 Logger
	wsServer := wss.NewServer(context.Background(), wsConfig, logger)

	// 註冊訂閱者 (監聽連線事件)
	wsServer.Register(wsHandler)

	// 6. 啟動 HTTP Server
	path := cfg.WSS.Path
	if path == "" {
		path = "/ws"
	}
	http.Handle(path, wsServer)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	slog.Info("Listening on", "addr", addr, "path", path)
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
