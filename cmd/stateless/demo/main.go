package main

import (
	"os"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	demo "github.com/JoeShih716/go-k8s-game-server/internal/app/game/stateless_demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
)

func main() {
	config := bootstrap.GameServerConfig{
		ServiceName:     "stateless-demo",
		ServiceType:     proto.ServiceType_STATELESS,
		GameIDs:         []int32{10000},
		DefaultGrpcPort: 8090,
	}

	host := os.Getenv("POD_IP")
	if host == "" {
		host = config.ServiceName
	}
	handler := demo.NewHandler(host)

	bootstrap.RunGameServer(config, handler)
}
