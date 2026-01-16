package rpcsdk

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
)

// Registrar 封裝了後端服務對 Central 的註冊與心跳邏輯
type Registrar struct {
	cli      proto.CentralServiceClient
	config   *Config
	conn     *grpc.ClientConn
	leaseID  string
	stopChan chan struct{}
}

type Config struct {
	ServiceName string
	ServiceType proto.ServiceType
	Endpoint    string // Kubernetes Pod IP + Port
	CentralAddr string
	GameIDs     []int32 // [NEW] Supported Game IDs
}

// NewRegistrar 建立註冊器
func NewRegistrar(conn *grpc.ClientConn, cfg *Config) *Registrar {
	return &Registrar{
		cli:      proto.NewCentralServiceClient(conn),
		config:   cfg,
		conn:     conn,
		stopChan: make(chan struct{}),
	}
}

// Register 向 Central 註冊服務
func (r *Registrar) Register(ctx context.Context) error {
	return r.registerWithRetry(ctx)
}

// StartHeartbeat 啟動心跳 (Blocking)
func (r *Registrar) StartHeartbeat(ctx context.Context) {
	r.heartbeatLoop(ctx)
}

// Stop 停止心跳並登出
func (r *Registrar) Stop(ctx context.Context) {
	select {
	case <-r.stopChan:
		// Already closed
	default:
		close(r.stopChan)
	}

	if r.leaseID != "" {
		_, _ = r.cli.Deregister(ctx, &proto.DeregisterRequest{
			LeaseId: r.leaseID,
		})
	}
}

// Close 關閉連線
func (r *Registrar) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}

// registerWithRetry 嘗試註冊直到成功
func (r *Registrar) registerWithRetry(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			resp, err := r.cli.Register(ctx, &proto.RegisterRequest{
				ServiceName: r.config.ServiceName,
				Type:        r.config.ServiceType,
				Endpoint:    r.config.Endpoint,
				GameIds:     r.config.GameIDs,
			})
			if err == nil {
				r.leaseID = resp.LeaseId
				slog.Info("Service registered successfully", "lease_id", r.leaseID)
				return nil
			}
			slog.Error("Registration failed, retrying...", "error", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func (r *Registrar) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			slog.Info("Stopping heartbeat loop")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 發送心跳
			resp, err := r.cli.Heartbeat(ctx, &proto.HeartbeatRequest{
				LeaseId:     r.leaseID,
				CurrentLoad: 0, // TODO: 整合實際負載
			})

			if err != nil || (resp != nil && !resp.Success) {
				slog.Warn("Heartbeat failed, re-registering...", "error", err)
				// 重新註冊
				_ = r.registerWithRetry(ctx)
			}
		}
	}
}
