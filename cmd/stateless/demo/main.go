package main

import (
	"os"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	demo "github.com/JoeShih716/go-k8s-game-server/internal/app/game/stateless_demo"
	"github.com/JoeShih716/go-k8s-game-server/internal/config"
	"github.com/JoeShih716/go-k8s-game-server/internal/pkg/bootstrap"
)

func main() {
	gameConfig := bootstrap.GameServerConfig{
		ServiceName:     "stateless-demo",
		ServiceType:     proto.ServiceType_STATELESS,
		GameIDs:         []int32{10000},
		DefaultGrpcPort: config.DefaultGrpcPort,
	}

	host := os.Getenv("POD_IP")
	if host == "" {
		host = gameConfig.ServiceName
	}
	handler := demo.NewHandler(host)

	bootstrap.RunGameServer(gameConfig, handler)
}
