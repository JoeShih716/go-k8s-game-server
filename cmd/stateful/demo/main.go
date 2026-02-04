package main

import (
	"os"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	statefuldemo "github.com/JoeShih716/go-k8s-game-server/internal/app/game/stateful_demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/engine"
	"github.com/JoeShih716/go-k8s-game-server/internal/kit/config"
)

func main() {
	gameConfig := engine.GameServerConfig{
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

	engine.RunGameServer(gameConfig, handler)
}
