package feed

import (
	"context"
	"fmt"
	"time"

	videopb "kairos/api/videopb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// VideoClient Feed 调用 Video 服务的 gRPC 客户端
type VideoClient struct {
	conn   *grpc.ClientConn
	client videopb.VideoServiceClient
}

// NewVideoClient 创建 Video gRPC 客户端，addr 格式 "host:port"
func NewVideoClient(addr string) (*VideoClient, error) {
	if addr == "" {
		addr = "127.0.0.1:9082"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("video grpc dial: %w", err)
	}
	return &VideoClient{
		conn:   conn,
		client: videopb.NewVideoServiceClient(conn),
	}, nil
}

// Close 关闭连接
func (c *VideoClient) Close() error {
	return c.conn.Close()
}

// ListLatest 最新视频列表
func (c *VideoClient) ListLatest(ctx context.Context, limit, offset int32) ([]*videopb.VideoInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := c.client.ListLatest(ctx, &videopb.ListLatestRequest{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return resp.Videos, nil
}

// ListByAuthorIDs 按作者 ID 列表获取视频
func (c *VideoClient) ListByAuthorIDs(ctx context.Context, authorIDs []uint32) ([]*videopb.VideoInfo, error) {
	if len(authorIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := c.client.ListByAuthorIDs(ctx, &videopb.ListByAuthorIDsRequest{AuthorIds: authorIDs})
	if err != nil {
		return nil, err
	}
	return resp.Videos, nil
}

// ListByPopularity 按热度排序视频列表
func (c *VideoClient) ListByPopularity(ctx context.Context, limit, offset int32) ([]*videopb.VideoInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := c.client.ListByPopularity(ctx, &videopb.ListByPopularityRequest{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return resp.Videos, nil
}

// BatchIsLiked 批量查询用户是否已点赞
func (c *VideoClient) BatchIsLiked(ctx context.Context, videoIDs []uint32, accountID uint32) (map[uint32]bool, error) {
	if len(videoIDs) == 0 || accountID == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := c.client.BatchIsLiked(ctx, &videopb.BatchIsLikedRequest{VideoIds: videoIDs, AccountId: accountID})
	if err != nil {
		return nil, err
	}
	return resp.Liked, nil
}
