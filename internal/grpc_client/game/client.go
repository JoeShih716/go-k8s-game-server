package game_sdk

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/gameRPC"
)

// Client wraps gameRPC.GameRPCClient
type Client struct {
	cli gameRPC.GameRPCClient
}

// NewClient creates a new Game SDK client
func NewClient(cc grpc.ClientConnInterface) *Client {
	return &Client{
		cli: gameRPC.NewGameRPCClient(cc),
	}
}

// Join sends OnPlayerJoin request to Game Server
func (c *Client) Join(ctx context.Context, userID, sessionID, connectorHost string) (*gameRPC.JoinResp, error) {
	return c.cli.OnPlayerJoin(ctx, &gameRPC.JoinReq{
		Header:        c.newHeader(userID, sessionID),
		ConnectorHost: connectorHost,
	})
}

// Quit sends OnPlayerQuit request to Game Server
func (c *Client) Quit(ctx context.Context, userID, sessionID string) (*gameRPC.QuitResp, error) {
	return c.cli.OnPlayerQuit(ctx, &gameRPC.QuitReq{
		Header: c.newHeader(userID, sessionID),
	})
}

// SendMessage sends a message (payload) to Game Server
func (c *Client) SendMessage(ctx context.Context, userID, sessionID string, payload []byte) (*gameRPC.MsgResp, error) {
	return c.cli.OnMessage(ctx, &gameRPC.MsgReq{
		Header:  c.newHeader(userID, sessionID),
		Payload: payload,
	})
}

// newHeader creates a new packet header with current timestamp
func (c *Client) newHeader(userID, sessionID string) *proto.PacketHeader {
	return &proto.PacketHeader{
		ReqId:     fmt.Sprintf("%d", time.Now().UnixNano()),
		UserId:    userID,
		SessionId: sessionID,
		Timestamp: time.Now().UnixMilli(),
	}
}
