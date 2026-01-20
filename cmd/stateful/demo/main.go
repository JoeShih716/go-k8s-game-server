package main

import (
	"os"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	statefuldemo "github.com/JoeShih716/go-k8s-game-server/internal/app/game/stateful_demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
)

func main() {
	config := bootstrap.GameServerConfig{
		ServiceName:     "stateful-demo",
		ServiceType:     proto.ServiceType_STATEFUL,
		GameIDs:         []int32{20000},
		DefaultGrpcPort: 8090,
	}

	host := os.Getenv("POD_IP")
	if host == "" {
		host = config.ServiceName
	}

	handler := statefuldemo.NewHandler(host)

	bootstrap.RunGameServer(config, handler)
}
