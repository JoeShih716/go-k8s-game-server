package central_sdk

import (
	"context"

	"google.golang.org/grpc"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
)

// Client 封裝了 Connector 對 Central 的 RPC 呼叫
// 提供 Login, GetRoute 等功能
type Client struct {
	rpcClient centralRPC.CentralRPCClient
}

// NewClient 建立 Connector RPC 客戶端
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{
		rpcClient: centralRPC.NewCentralRPCClient(conn),
	}
}

// Login 呼叫 Central 進行登入
func (c *Client) Login(ctx context.Context, token string) (*centralRPC.LoginResponse, error) {
	return c.rpcClient.Login(ctx, &centralRPC.LoginRequest{
		Token: token,
	})
}

// GetRoute 呼叫 Central 取得路由
func (c *Client) GetRoute(ctx context.Context, gameID int32) (string, proto.ServiceType, error) {
	resp, err := c.rpcClient.GetRoute(ctx, &centralRPC.GetRouteRequest{
		GameId: gameID,
	})
	if err != nil {
		return "", proto.ServiceType_UNKNOWN_SERVICE, err
	}
	return resp.TargetEndpoint, resp.Type, nil
}
