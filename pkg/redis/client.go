package redis

import (
	"context"
	"strconv"

	redis "github.com/redis/go-redis/v9"
)

// Config Redis 连接配置
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// Client Redis 客户端
type Client struct {
	rdb *redis.Client
}

// New 创建 Redis 客户端
func New(cfg Config) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &Client{rdb: rdb}
}

// Close 关闭连接
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// Ping 检查连接
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Ping(ctx).Err()
}
