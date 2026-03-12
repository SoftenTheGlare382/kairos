package video

import (
	"context"
	"fmt"
	"time"

	accountpb "kairos/api/accountpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AccountClient 调用 Account 服务的 gRPC 客户端
type AccountClient struct {
	conn   *grpc.ClientConn
	client accountpb.AccountServiceClient
}

// NewAccountClient 创建 gRPC 客户端，addr 格式为 "host:port"（如 "127.0.0.1:9081"）
func NewAccountClient(addr string) (*AccountClient, error) {
	if addr == "" {
		addr = "127.0.0.1:9081"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}
	return &AccountClient{
		conn:   conn,
		client: accountpb.NewAccountServiceClient(conn),
	}, nil
}

// Close 关闭连接
func (c *AccountClient) Close() error {
	return c.conn.Close()
}

// UserInfo 用户信息
type UserInfo struct {
	ID       uint
	Username string
}

// GetByID 按 ID 获取用户（gRPC 调用 Account.FindByID）
func (c *AccountClient) GetByID(ctx context.Context, id uint) (*UserInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := c.client.FindByID(ctx, &accountpb.FindByIDRequest{Id: uint32(id)})
	if err != nil {
		return nil, err
	}
	return &UserInfo{
		ID:       uint(resp.Id),
		Username: resp.Username,
	}, nil
}
