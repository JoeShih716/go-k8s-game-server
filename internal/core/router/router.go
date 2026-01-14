package router

import (
	"context"

	proto "github.com/JoeShih716/go-k8s-game-server/api/proto"
)

// Router 定義了請求路由的策略介面。
// 負責決定一個請求應該被轉發到哪個後端服務實例 (Pod)。
type Router interface {
	// Route根據路由元數據計算目標地址
	//
	// 參數:
	//
	//	ctx: context.Context - 上下文
	//	metadata: *proto.RoutingMetadata - 路由決策所需的資訊 (ServiceType, RoomID, UserID 等)
	//
	// 回傳值:
	//
	//	string: 目標服務地址 (例如: "fishing-service-0.fishing-service.default.svc.cluster.local:8080")
	//	error: 若找不到合適的路由目標則回傳錯誤
	Route(ctx context.Context, metadata *proto.RoutingMetadata) (string, error)
}
