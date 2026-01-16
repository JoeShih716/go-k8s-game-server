package rpcsdk

import (
	"context"
	"fmt"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"google.golang.org/grpc"
)

// Client 封裝了 Connector 對 Central 的 RPC 呼叫
// 提供 Login, GetRoute 等功能
type Client struct {
	cli proto.CentralServiceClient
}

// NewClient 建立 Connector RPC 客戶端
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{
		cli: proto.NewCentralServiceClient(conn),
	}
}

// Login 呼叫 Central 進行登入
func (c *Client) Login(ctx context.Context, token string) (string, error) {
	resp, err := c.cli.Login(ctx, &proto.LoginRequest{
		Token: token,
	})
	if err != nil {
		return "", err
	}
	if !resp.Success {
		return "", fmt.Errorf("login failed: %s", resp.ErrorMessage)
	}
	return resp.UserId, nil
}

// GetRoute 呼叫 Central 取得路由
func (c *Client) GetRoute(ctx context.Context, userID string, gameID int32) (string, error) {
	resp, err := c.cli.GetRoute(ctx, &proto.GetRouteRequest{
		UserId: userID,
		GameId: gameID,
	})
	if err != nil {
		return "", err
	}
	return resp.TargetEndpoint, nil
}
