package router

import (
	"context"
	"fmt"

	proto "github.com/JoeShih716/go-k8s-game-server/api/proto"
)

// SmartRouter 實作 core.Router 介面
// 負責整合 Discovery 模組，根據路由規則分發流量
type SmartRouter struct {
	discovery Discovery
}

// NewSmartRouter 建立一個路由器
func NewSmartRouter(d Discovery) *SmartRouter {
	return &SmartRouter{
		discovery: d,
	}
}

// Route 根據 Metadata 決定目標地址
func (r *SmartRouter) Route(ctx context.Context, metadata *proto.RoutingMetadata) (string, error) {
	// 簡單路由策略：
	// 1. 如果是 COORDINATOR，轉發到 coordinator 服務
	// 2. 如果是 STATELESS (e.g. SLOTS)，根據 ServiceName 轉發
	// 3. 如果是 STATEFUL (e.g. FISHING)，也是根據 ServiceName 轉發 (未來需加入 Sticky Session)

	// 目前只處理 ServiceName 的映射
	// 在 routing.proto 中，我們還沒有把 ServiceName 放入 Metadata，
	// 假設 Metadata 中的 Action 或其他欄位能暗示服務，
	// 或者我們先用一個簡單的 mapping: ServiceType -> ServiceName

	// 暫時邏輯：依據 ServiceType 決定服務名稱
	var serviceName string
	// 實際場景：
	// Client 不會直接傳 ServiceType，而是傳送 CmdID 或 GameID。
	// SmartRouter 需要：
	// 1. 查詢 Coordinator (或是本地 Cache) 得知該 CmdID/GameID 屬於哪個服務類型。
	// 2. 獲取該服務的具體地址。

	// 目前測試階段暫時依賴前端傳來的 ServiceType
	switch metadata.ServiceType {
	case proto.ServiceType_STATELESS:
		// 模擬：如果是 Stateless，先假設是 Slots Service
		serviceName = "slots-service"
	case proto.ServiceType_STATEFUL:
		serviceName = "fishing-service"
	default:
		return "", fmt.Errorf("unknown service type: %v", metadata.ServiceType)
	}

	// 查詢 Discovery
	addr, _, err := r.discovery.GetServiceAddr(ctx, serviceName, metadata)
	if err != nil {
		return "", err
	}

	return addr, nil
}
