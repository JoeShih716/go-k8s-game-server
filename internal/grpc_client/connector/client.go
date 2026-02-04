package connector_sdk

import (
	"context"

	"google.golang.org/grpc"

	"github.com/JoeShih716/go-k8s-game-server/api/proto/connectorRPC"
)

// Client wraps connectorRPC.ConnectorRPCClient
type Client struct {
	cli connectorRPC.ConnectorRPCClient
}

// NewClient creates a new Connector SDK client
func NewClient(cc grpc.ClientConnInterface) *Client {
	return &Client{
		cli: connectorRPC.NewConnectorRPCClient(cc),
	}
}

// Push sends a message to a specific session (player)
func (c *Client) Push(ctx context.Context, sessionID string, payload []byte) error {
	_, err := c.cli.SendMessage(ctx, &connectorRPC.SendMessageReq{
		SessionIds: []string{sessionID},
		Payload:    payload,
	})
	return err
}

// Broadcast sends a message to multiple sessions
func (c *Client) Broadcast(ctx context.Context, sessionIDs []string, payload []byte) error {
	_, err := c.cli.SendMessage(ctx, &connectorRPC.SendMessageReq{
		SessionIds: sessionIDs,
		Payload:    payload,
	})
	return err
}

// ForceKick kicks a player with a reason
func (c *Client) ForceKick(ctx context.Context, sessionID string, reason string) error {
	_, err := c.cli.Kick(ctx, &connectorRPC.KickReq{
		SessionId: sessionID,
		Reason:    reason,
	})
	return err
}
