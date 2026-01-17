package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
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
	// Key Pattern: game:{GameID} -> Set of Endpoints
	KeyGameSet = "game:%d"

	DefaultTTL = 10 * time.Second
)

// LeaseData 儲存於 Redis 的租約資訊
type LeaseData struct {
	Endpoint    string            `json:"endpoint"`
	ServiceType proto.ServiceType `json:"service_type"`
	GameIDs     []int32           `json:"game_ids"`
}

func NewRedisRegistry(rds *redis.Client) *Registry {
	return &Registry{rds: rds}
}

// Register 註冊一個新服務
func (r *Registry) Register(ctx context.Context, req *centralRPC.RegisterRequest) (string, error) {
	leaseID := uuid.New().String()

	// 1. 儲存 Lease Metadata (包含 Endpoint 與 ServiceType)
	leaseKey := fmt.Sprintf(KeyLease, leaseID)
	data := &LeaseData{
		Endpoint:    req.Endpoint,
		ServiceType: req.Type,
		GameIDs:     req.GameIds,
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

	// 3. 加入 Game ID 的集合 (方便做 Game Routing)
	for _, gameID := range req.GameIds {
		gameKey := fmt.Sprintf(KeyGameSet, gameID)
		err = r.rds.SAdd(ctx, gameKey, req.Endpoint)
		if err != nil {
			// 盡最大努力寫入，不因為單一失敗中斷流程，但建議 Log (這裡直接回傳錯)
			return "", fmt.Errorf("failed to add to game set: %w", err)
		}
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
	_ = r.rds.Del(ctx, leaseKey)

	// 3. 從 Set 移除 Endpoint
	setKey := fmt.Sprintf(KeyServiceSet, data.ServiceType.String())
	_ = r.rds.SRem(ctx, setKey, data.Endpoint)

	// 4. 從 Game ID Set 移除 Endpoint
	for _, gameID := range data.GameIDs {
		gameKey := fmt.Sprintf(KeyGameSet, gameID)
		_ = r.rds.SRem(ctx, gameKey, data.Endpoint)
	}

	return nil
}

// GetServiceEndpoints 取得某類型的所有活躍地址
func (r *Registry) GetServiceEndpoints(ctx context.Context, serviceType proto.ServiceType) ([]string, error) {
	setKey := fmt.Sprintf(KeyServiceSet, serviceType.String())
	return r.rds.SMembers(ctx, setKey)
}

// SelectService 隨機挑選一個健康的服務實例 (Simple Load Balancing)
func (r *Registry) SelectService(ctx context.Context, serviceType proto.ServiceType) (string, error) {
	// SrandMemberCommand (go-redis v9 uses SrandMemberN or SrandMember)
	// For single random member:
	setKey := fmt.Sprintf(KeyServiceSet, serviceType.String())

	res, err := r.rds.SRandMember(ctx, setKey)
	if err != nil {
		return "", err
	}

	return res, nil
}

// SelectServiceByGame 根據 GameID 隨機挑選一個服務實例
func (r *Registry) SelectServiceByGame(ctx context.Context, gameID int32) (string, error) {
	key := fmt.Sprintf(KeyGameSet, gameID)

	// 隨機取出一個
	res, err := r.rds.SRandMember(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", nil // Not found
		}
		return "", err
	}

	return res, nil
}
