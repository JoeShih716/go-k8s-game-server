package router

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/internal/central/rpcsdk"
)

// ErrServiceNotFound 表示找不到請求的服務
var ErrServiceNotFound = errors.New("service not found")

// Discovery 定義服務發現介面
// 負責根據服務名稱和類型查找可用的服務實例地址
type Discovery interface {
	// GetServiceAddr 根據服務名稱取得地址 (Metadata 可選，用於進階路由如 Consistent Hash or Central GetRoute)
	GetServiceAddr(ctx context.Context, serviceName string, metadata *proto.RoutingMetadata) (string, error)
}

// StaticDiscovery 靜態服務發現實作 (讀取 Config)
type StaticDiscovery struct {
	services map[string]string
	mu       sync.RWMutex
}

// NewStaticDiscovery 建立一個基於靜態設定的 Discovery
func NewStaticDiscovery(services map[string]string) *StaticDiscovery {
	return &StaticDiscovery{
		services: services,
	}
}

// GetServiceAddr 實作 Discovery 介面
func (d *StaticDiscovery) GetServiceAddr(ctx context.Context, serviceName string, metadata *proto.RoutingMetadata) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	addr, ok := d.services[serviceName]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrServiceNotFound, serviceName)
	}
	return addr, nil
}

// CentralDiscovery 基於 Central Service 的動態發現實作
type CentralDiscovery struct {
	client *rpcsdk.Client
}

func NewCentralDiscovery(client *rpcsdk.Client) *CentralDiscovery {
	return &CentralDiscovery{
		client: client,
	}
}

func (d *CentralDiscovery) GetServiceAddr(ctx context.Context, serviceName string, metadata *proto.RoutingMetadata) (string, error) {
	// 嘗試從 Metadata 取得 GameID (如果有的話)
	// 假設 Tags["game_id"] 存放 GameID
	if metadata != nil && metadata.Tags != nil {
		if gameIDStr, ok := metadata.Tags["game_id"]; ok {
			gameID, err := strconv.Atoi(gameIDStr)
			if err == nil {
				// 使用 Central GetRoute
				// UserID 假設在 metadata.UserId
				return d.client.GetRoute(ctx, metadata.UserId, int32(gameID))
			}
		}
	}

	// Fallback: 如果沒有 GameID，雖然 Central GetRoute 需要 GameID，
	// 但如果只是純粹找 "serviceName" (e.g. internal calls)，這裡可能暫時無法處理，
	// 或者我們可以實作另一個 RPC GetService(serviceName)
	// 目前 Assume Dynamic Routing 都是基於 GameID

	return "", fmt.Errorf("central discovery requires game_id in metadata")
}
