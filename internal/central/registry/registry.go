package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/pkg/redis"
)

// Registry 負責管理所有活躍的遊戲服務
type Registry struct {
	rds *redis.Client
}

const (
	// Key Pattern: services:{ServiceType} -> Set of Endpoints
	KeyServiceSet = "services:%s"
	// Key Pattern: services:lease:{LeaseID} -> Service Metadata (JSON/Hash)
	KeyLease = "services:lease:%s"

	DefaultTTL = 10 * time.Second
)

// LeaseData 儲存於 Redis 的租約資訊
type LeaseData struct {
	Endpoint    string            `json:"endpoint"`
	ServiceType proto.ServiceType `json:"service_type"`
}

func NewRedisRegistry(rds *redis.Client) *Registry {
	return &Registry{rds: rds}
}

// Register 註冊一個新服務
func (r *Registry) Register(ctx context.Context, req *proto.RegisterRequest) (string, error) {
	leaseID := uuid.New().String()

	// 1. 儲存 Lease Metadata (包含 Endpoint 與 ServiceType)
	leaseKey := fmt.Sprintf(KeyLease, leaseID)
	data := &LeaseData{
		Endpoint:    req.Endpoint,
		ServiceType: req.Type,
	}

	err := r.rds.SetStruct(ctx, leaseKey, data, DefaultTTL)
	if err != nil {
		return "", fmt.Errorf("failed to set lease: %w", err)
	}

	// 2. 加入 Service Type 的集合 (方便做 Discovery)
	setKey := fmt.Sprintf(KeyServiceSet, req.Type.String())
	err = r.rds.SAdd(ctx, setKey, req.Endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to add to service set: %w", err)
	}

	return leaseID, nil
}

// Heartbeat 更新 Lease TTL
func (r *Registry) Heartbeat(ctx context.Context, leaseID string, load int32) error {
	leaseKey := fmt.Sprintf(KeyLease, leaseID)

	// 檢查 Key 是否存在 (如果 Central 重啟，Redis SET 還在，就可以續命)
	// 如果 Key 不見了 (例如過期)，回傳錯誤讓 Client 重新註冊
	exists, err := r.rds.Exists(ctx, leaseKey)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("lease not found")
	}

	// 續命
	return r.rds.Expire(ctx, leaseKey, DefaultTTL)
}

// Deregister 主動移除服務
func (r *Registry) Deregister(ctx context.Context, leaseID string) error {
	leaseKey := fmt.Sprintf(KeyLease, leaseID)

	// 1. 取得 LeaseData 以便得知是哪個 ServiceType
	var data LeaseData
	err := r.rds.GetStruct(ctx, leaseKey, &data)
	if err != nil {
		// Key 不存在 (可能已過期)，當作成功處理
		return nil
	}

	// 2. 刪除 Lease
	r.rds.Del(ctx, leaseKey)

	// 3. 從 Set 移除 Endpoint
	setKey := fmt.Sprintf(KeyServiceSet, data.ServiceType.String())
	return r.rds.SRem(ctx, setKey, data.Endpoint)
}

// GetServiceEndpoints 取得某類型的所有活躍地址
func (r *Registry) GetServiceEndpoints(ctx context.Context, serviceType proto.ServiceType) ([]string, error) {
	setKey := fmt.Sprintf(KeyServiceSet, serviceType.String())
	return r.rds.SMembers(ctx, setKey)
}
