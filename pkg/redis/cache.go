package redis

import (
	"context"
	"time"
)

// GetBytes 获取字节值
func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	return c.rdb.Get(ctx, key).Bytes()
}

// SetBytes 设置字节值
func (c *Client) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Del 删除键
func (c *Client) Del(ctx context.Context, key string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, key).Err()
}
