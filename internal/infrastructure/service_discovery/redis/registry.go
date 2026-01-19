package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
			return "", fmt.Errorf("failed to add to game set: %w", err)
		}

		// 3.1 儲存 GameID -> ServiceType 的映射 (Metadata)
		metaKey := fmt.Sprintf("game:%d:meta", gameID)
		_ = r.rds.Set(ctx, metaKey, int32(req.Type), 0)
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
func (r *Registry) SelectServiceByGame(ctx context.Context, gameID int32) (string, proto.ServiceType, error) {
	key := fmt.Sprintf(KeyGameSet, gameID)

	// 1. 隨機取出一個 Endpoint
	res, err := r.rds.SRandMember(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return "", proto.ServiceType_UNKNOWN_SERVICE, nil // Not found
		}
		return "", proto.ServiceType_UNKNOWN_SERVICE, err
	}

	// 2. 取得 ServiceType
	metaKey := fmt.Sprintf("game:%d:meta", gameID)
	val, err := r.rds.Get(ctx, metaKey)
	var sType int32
	if err == nil {
		_, _ = fmt.Sscanf(val, "%d", &sType)
	}

	return res, proto.ServiceType(sType), nil
}

// CleanupDeadServices 清理無效的服務節點 (Zombie Endpoints)
func (r *Registry) CleanupDeadServices(ctx context.Context) error {
	// 1. 取得所有活躍的 Leases
	// 注意: 在生產環境請使用 Scan 代替 Keys 避免阻塞
	leaseKeys, err := r.rds.Keys(ctx, "services:lease:*")
	if err != nil {
		return fmt.Errorf("failed to scan leases: %w", err)
	}

	validEndpoints := make(map[string]bool)
	for _, key := range leaseKeys {
		var data LeaseData
		// 如果 Lease 剛好過期，GetStruct 會失敗，那麼它就不是 Valid，正確
		if err := r.rds.GetStruct(ctx, key, &data); err == nil {
			validEndpoints[data.Endpoint] = true
		}
	}

	// 2. 掃描並清理 Service Sets (e.g., services:STATELESS)
	serviceKeys, err := r.rds.Keys(ctx, "services:*")
	if err != nil {
		slog.Warn("Failed to scan service sets", "error", err)
	} else {
		for _, key := range serviceKeys {
			// 排除 lease, metadata 等非 Set 的 Key
			if strings.Contains(key, ":lease:") {
				continue
			}

			members, err := r.rds.SMembers(ctx, key)
			if err != nil {
				continue
			}

			for _, member := range members {
				if !validEndpoints[member] {
					slog.Info("Removing zombie service endpoint", "key", key, "endpoint", member)
					_ = r.rds.SRem(ctx, key, member)
				}
			}
		}
	}

	// 3. 掃描並清理 Game Sets (e.g., game:10000)
	gameKeys, err := r.rds.Keys(ctx, "game:*")
	if err != nil {
		slog.Warn("Failed to scan game sets", "error", err)
	} else {
		for _, key := range gameKeys {
			// 排除 metadata
			if strings.Contains(key, ":meta") {
				continue
			}

			members, err := r.rds.SMembers(ctx, key)
			if err != nil {
				continue
			}

			for _, member := range members {
				if !validEndpoints[member] {
					slog.Info("Removing zombie game endpoint", "key", key, "endpoint", member)
					_ = r.rds.SRem(ctx, key, member)
				}
			}
		}
	}

	return nil
}
