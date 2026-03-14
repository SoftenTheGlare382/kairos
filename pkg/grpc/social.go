package grpc

import (
	"context"
	"fmt"
	"time"

	socialpb "kairos/api/socialpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SocialClient 调用 Social 服务的 gRPC 客户端（供 IM 等使用）
type SocialClient struct {
	conn   *grpc.ClientConn
	client socialpb.SocialServiceClient
}

// NewSocialClient 创建 Social gRPC 客户端，addr 格式为 "host:port"（如 "127.0.0.1:9083"）
func NewSocialClient(addr string) (*SocialClient, error) {
	if addr == "" {
		addr = "127.0.0.1:9083"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("social grpc dial: %w", err)
	}
	return &SocialClient{
		conn:   conn,
		client: socialpb.NewSocialServiceClient(conn),
	}, nil
}

// Close 关闭连接
func (c *SocialClient) Close() error {
	return c.conn.Close()
}

// IsMutualFollow 是否互相关注（供 IM 校验私聊权限）
func (c *SocialClient) IsMutualFollow(ctx context.Context, userA, userB uint) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := c.client.IsMutualFollow(ctx, &socialpb.IsMutualFollowRequest{
		UserA: uint32(userA),
		UserB: uint32(userB),
	})
	if err != nil {
		return false, err
	}
	return resp.Mutual, nil
}
