package redis

import (
	"context"
	"math/rand"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// TTLWithJitter 返回 base + [0, base*jitterRatio) 的随机偏移，缓解缓存雪崩
func TTLWithJitter(base time.Duration, jitterRatio float64) time.Duration {
	if jitterRatio <= 0 || jitterRatio > 1 {
		return base
	}
	jitter := time.Duration(float64(base) * jitterRatio * rand.Float64())
	return base + jitter
}

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

// ZIncrBy 对有序集合中成员的分数加上增量，member 为 string（如 video_id）
func (c *Client) ZIncrBy(ctx context.Context, key string, increment float64, member string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.ZIncrBy(ctx, key, increment, member).Err()
}

// ZAdd 向有序集合添加成员，用于全量同步
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.ZAdd(ctx, key, members...).Err()
}

// ZRevRange 按分数降序返回成员（start, stop 为索引，含边界）
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	return c.rdb.ZRevRange(ctx, key, start, stop).Result()
}

// SAdd 向集合添加成员
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.SAdd(ctx, key, members...).Err()
}

// SRem 从集合移除成员
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.SRem(ctx, key, members...).Err()
}

// SMembers 返回集合所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	return c.rdb.SMembers(ctx, key).Result()
}

// SIsMember 判断 member 是否在集合中
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, nil
	}
	return c.rdb.SIsMember(ctx, key, member).Result()
}
