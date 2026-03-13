package feed

import (
	"context"
	"fmt"
	"time"

	socialpb "kairos/api/socialpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SocialClient Feed 调用 Social 服务的 gRPC 客户端
type SocialClient struct {
	conn   *grpc.ClientConn
	client socialpb.SocialServiceClient
}

// NewSocialClient 创建 Social gRPC 客户端，addr 格式 "host:port"
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

// GetFollowingIDs 获取某用户关注的用户 ID 列表
func (c *SocialClient) GetFollowingIDs(ctx context.Context, followerID uint32) ([]uint32, error) {
	if followerID == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := c.client.GetFollowingIDs(ctx, &socialpb.GetFollowingIDsRequest{FollowerId: followerID})
	if err != nil {
		return nil, err
	}
	return resp.FollowingIds, nil
}
