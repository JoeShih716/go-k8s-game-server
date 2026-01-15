package rpctool

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
)

// Registrar 負責處理服務註冊、心跳與登出的客戶端組件
type Registrar struct {
	centralAddr string
	conn        *grpc.ClientConn
	client      proto.CentralServiceClient

	serviceInfo *proto.RegisterRequest
	leaseID     string

	stopChan chan struct{}
}

func NewRegistrar(centralAddr string, info *proto.RegisterRequest) *Registrar {
	return &Registrar{
		centralAddr: centralAddr,
		serviceInfo: info,
		stopChan:    make(chan struct{}),
	}
}

// Start 啟動註冊流程 (包含無限重試直到成功)
// 此方法會阻塞，直到第一次註冊成功，建議在 Goroutine 中執行，或在 main 中等待
func (r *Registrar) Start(ctx context.Context) error {
	// 1. 建立 gRPC 連線
	conn, err := grpc.NewClient(r.centralAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to create grpc client: %w", err)
	}
	r.conn = conn
	r.client = proto.NewCentralServiceClient(conn)

	// 2. 初始註冊 (Retry Loop)
	if err := r.registerWithRetry(ctx); err != nil {
		return err
	}

	// 3. 啟動心跳 Loop (背景執行)
	go r.heartbeatLoop(ctx)

	return nil
}

// registerWithRetry 嘗試註冊直到成功
func (r *Registrar) registerWithRetry(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		slog.Info("Attempting to register to Central...", "addr", r.centralAddr)

		resp, err := r.client.Register(ctx, r.serviceInfo)
		if err == nil {
			r.leaseID = resp.LeaseId
			slog.Info("Service registered successfully", "lease_id", r.leaseID)
			return nil
		}

		slog.Warn("Registration failed, retrying in 2s...", "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue loop
		}
	}
}

// heartbeatLoop 定期發送心跳
func (r *Registrar) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // 假設 TTL 是 10s
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			slog.Info("Stopping heartbeat loop")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			req := &proto.HeartbeatRequest{
				LeaseId:     r.leaseID,
				CurrentLoad: 0, // TODO: 整合實際負載
			}
			resp, err := r.client.Heartbeat(ctx, req)

			// 如果失敗或 Central 說這 Lease 無效 (Success=false)
			if err != nil || (resp != nil && !resp.Success) {
				slog.Warn("Heartbeat failed, triggering re-registration...", "error", err)
				// 觸發重新註冊
				if err := r.registerWithRetry(ctx); err != nil {
					slog.Error("Re-registration failed, giving up heartbeat", "error", err)
					return
				}
			}
		}
	}
}

// Stop 停止心跳並發送登出請求 (Graceful Shutdown)
func (r *Registrar) Stop(ctx context.Context) {
	close(r.stopChan)

	if r.leaseID != "" && r.client != nil {
		slog.Info("Deregistering service...", "lease_id", r.leaseID)
		_, err := r.client.Deregister(ctx, &proto.DeregisterRequest{LeaseId: r.leaseID})
		if err != nil {
			slog.Warn("Deregister failed", "error", err)
		} else {
			slog.Info("Deregister success")
		}
	}

	if r.conn != nil {
		r.conn.Close()
	}
}
