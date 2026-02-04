package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/connectorRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/connector/handler"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/connector/session"
	central_sdk "github.com/JoeShih716/go-k8s-game-server/internal/grpc_client/central"
	"github.com/JoeShih716/go-k8s-game-server/internal/kit/bootstrap"
	grpcpkg "github.com/JoeShih716/go-k8s-game-server/pkg/grpc"
	"github.com/JoeShih716/go-k8s-game-server/pkg/wss"
)

func main() {
	// 1. Bootstrap App (Logs, Config)
	app := bootstrap.NewApp("connector")

	// 2. 初始化核心組件
	sessionMgr := session.NewManager()

	// 2. Connect to Central Service
	centralAddr := app.Config.Services["central"]
	// 建立 gRPC 連線
	centralConn, err := grpc.NewClient(centralAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to central", "error", err)
		os.Exit(1)
	} else {
		slog.Info("Connected to Central", "addr", centralAddr)
	}
	centralClient := central_sdk.NewClient(centralConn)

	// 4. gRPC Pool
	grpcPool := grpcpkg.NewPool()

	// 5. WebSocket Handler
	podIP := app.Config.App.PodIP
	if podIP == "" {
		podIP = "127.0.0.1"
		slog.Warn("POD_IP not set, using default", "ip", podIP)
	}
	// gRPC Port
	grpcPort := app.Config.App.GrpcPort
	myRPCPoint := fmt.Sprintf("%s:%d", podIP, grpcPort)
	slog.Info("Connector.. ", "myEndpoint", myRPCPoint)

	wsHandler := handler.NewWebsocketHandler(sessionMgr, grpcPool, centralClient, myRPCPoint)

	// 6. WebSocket Server
	wsConfig := &wss.Config{
		AllowedOrigins:  app.Config.WSS.AllowedOrigins,
		ReadBufferSize:  app.Config.WSS.ReadBufferSize,
		WriteBufferSize: app.Config.WSS.WriteBufferSize,
		WriteWait:       time.Duration(app.Config.WSS.WriteWaitSec) * time.Second,
		PongWait:        time.Duration(app.Config.WSS.PongWaitSec) * time.Second,
		MaxMessageSize:  app.Config.WSS.MaxMessageSize,
	}
	wsServer := wss.NewServer(context.Background(), wsConfig, app.Logger)
	wsServer.Register(wsHandler)

	// 7. HTTP Route
	path := app.Config.WSS.Path
	if path == "" {
		path = "/ws"
	}
	http.Handle(path, wsServer)

	// 8. 啟動服務 (Run)
	var grpcServer *grpc.Server // Declare outside to access in cleanup
	app.Run(func() error {
		// 8.1 啟動 gRPC Server (Background)
		go func() {
			slog.Info("Attempting to start gRPC Server", "port", grpcPort)
			lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
			if err != nil {
				// 這裡如果失敗必須 Panic 讓 Pod 重啟，因為這是關鍵服務
				panic(fmt.Sprintf("Failed to listen gRPC on port %d: %v", grpcPort, err))
			}

			grpcServer = grpc.NewServer(
				grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
					MinTime:             5 * time.Second,
					PermitWithoutStream: true,
				}),
			)
			connectorRPC.RegisterConnectorRPCServer(grpcServer, handler.NewGrpcHandler(sessionMgr))

			slog.Info("ConnectorRPC Listening", "port", grpcPort)
			if err := grpcServer.Serve(lis); err != nil {
				slog.Error("Failed to serve gRPC", "error", err)
			}
		}()

		// 8.2 啟動 HTTP Server (Blocking)
		addr := fmt.Sprintf(":%d", app.Config.App.Port)
		slog.Info("Listening on", "addr", addr, "path", path)
		return http.ListenAndServe(addr, nil)
	}, func() {
		// Cleanup Logic (Graceful Shutdown)
		slog.Info("Starting Cleanup...")

		// 1. WebSocket Server Shutdown (Kick all players)
		wsServer.Shutdown()

		// 2. gRPC Server Graceful Stop (Finish current requests)
		if grpcServer != nil {
			slog.Info("Stopping gRPC Server...")
			grpcServer.GracefulStop()
			slog.Info("gRPC Server stopped")
		}

		// 3. Wait for Async Handlers (e.g. OnPlayerQuit)
		slog.Info("Waiting for async handlers...")
		wsHandler.Close()

		// Cleanup Resources
		grpcPool.Close()
	})
}
