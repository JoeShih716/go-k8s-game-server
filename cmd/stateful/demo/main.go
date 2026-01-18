package main

import (
	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	statefuldemo "github.com/JoeShih716/go-k8s-game-server/internal/applications/stateful-demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
)

func main() {
	config := bootstrap.GameServerConfig{
		ServiceName: "stateful-demo",
		ServiceType: proto.ServiceType_STATEFUL,
		GameIDs:     []int32{20000},
		DefaultPort: 9002,
	}

	// 這裡改為傳入實現了 framework.GameHandler 的實例
	handler := statefuldemo.NewHandler(config.ServiceName)

	bootstrap.RunGameServer(config, handler)
}
