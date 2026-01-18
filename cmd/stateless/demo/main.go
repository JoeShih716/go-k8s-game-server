package main

import (
	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	demo "github.com/JoeShih716/go-k8s-game-server/internal/applications/stateless-demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
)

func main() {
	config := bootstrap.GameServerConfig{
		ServiceName: "stateless-demo",
		ServiceType: proto.ServiceType_STATELESS,
		GameIDs:     []int32{10000},
		DefaultPort: 9001,
	}

	handler := demo.NewHandler(config.ServiceName)

	bootstrap.RunGameServer(config, handler)
}
