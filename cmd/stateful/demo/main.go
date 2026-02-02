package main

import (
	"os"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	statefuldemo "github.com/JoeShih716/go-k8s-game-server/internal/app/game/stateful_demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/config"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
)

func main() {
	gameConfig := bootstrap.GameServerConfig{
		ServiceName:     "stateful-demo",
		ServiceType:     proto.ServiceType_STATEFUL,
		GameIDs:         []int32{20000},
		DefaultGrpcPort: config.DefaultGrpcPort,
	}

	host := os.Getenv("POD_IP")

	if host == "" {
		host = gameConfig.ServiceName
	}

	handler := statefuldemo.NewHandler(host)

	bootstrap.RunGameServer(gameConfig, handler)
}
