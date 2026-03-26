package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"kairos/pkg/config"
	"kairos/pkg/middleware"
	"kairos/pkg/redis"
	"kairos/pkg/registry"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnvFromSearchPaths(true)
	cfg := config.Load()

	gin.SetMode(cfg.Server.GinMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Redis：用于鉴权（token store）与限流计数
	rdb := redis.New(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	{
		ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		err := rdb.Ping(ctx)
		cancel()
		if err != nil {
		log.Printf("gateway: redis ping failed (auth/ratelimit may fail): %v", err)
	}
	}
	defer rdb.Close()

	// 上游服务（按路径前缀路由）
	accountUpstream := mustParseURL(fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.AccountPort))
	videoUpstream := mustParseURL(fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.VideoPort))
	socialUpstream := mustParseURL(fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.SocialPort))
	feedUpstream := mustParseURL(fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.FeedPort))
	imUpstream := mustParseURL(fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.IMPort))

	// etcd：网关也注册自己，并从 etcd 发现上游（可选；失败回退固定端口）
	var etcdReg *registry.EtcdRegistry
	if len(cfg.Etcd.Endpoints) > 0 {
		r, err := registry.NewEtcd(registry.Config{Endpoints: cfg.Etcd.Endpoints, Prefix: cfg.Etcd.Prefix, TTL: cfg.Etcd.TTL})
		if err != nil {
			log.Printf("gateway: etcd disabled: %v", err)
		} else {
			etcdReg = r
			instanceID := fmt.Sprintf("gateway-%d", os.Getpid())
			httpPort := cfg.Server.GatewayPort
			if httpPort == 0 {
				httpPort = 8080
			}
			_, _ = etcdReg.Register(context.Background(), "gateway", registry.ServiceHTTP, instanceID, fmt.Sprintf("127.0.0.1:%d", httpPort))

			dctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			etcdReg.StartPollingDiscovery(dctx, "account", registry.ServiceHTTP, 2*time.Second)
			etcdReg.StartPollingDiscovery(dctx, "video", registry.ServiceHTTP, 2*time.Second)
			etcdReg.StartPollingDiscovery(dctx, "social", registry.ServiceHTTP, 2*time.Second)
			etcdReg.StartPollingDiscovery(dctx, "feed", registry.ServiceHTTP, 2*time.Second)
			etcdReg.StartPollingDiscovery(dctx, "im", registry.ServiceHTTP, 2*time.Second)
		}
	}
	defer func() { _ = etcdReg.Close() }()

	proxies := map[string]*httputil.ReverseProxy{
		"account": newProxy(accountUpstream),
		"video":   newProxy(videoUpstream),
		"social":  newProxy(socialUpstream),
		"feed":    newProxy(feedUpstream),
		"im":      newProxy(imUpstream),
	}

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	// 对外不暴露 Account 内部接口
	r.Any("/internal/*any", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	// 公开接口白名单
	public := func(path string) bool {
		switch {
		case path == "/account/register" || path == "/account/login":
			return true
		case path == "/video/listByAuthorID" || path == "/video/getDetail" || path == "/video/search":
			return true
		case path == "/comment/listAll":
			return true
		case path == "/im/ws":
			// WebSocket 鉴权在 IM 服务首帧完成
			return true
		case strings.HasPrefix(path, "/static/"):
			// Video 本地存储静态文件
			return true
		default:
			return false
		}
	}

	// 全局：令牌桶（IP 级）限流（默认 200 rps，burst=400）
	r.Use(rateLimitTokenBucket(rdb, 400, 200, func(c *gin.Context) string {
		return "ip:" + clientIP(c)
	}))

	// 对非公开接口启用鉴权
	r.Use(func(c *gin.Context) {
		if public(c.Request.URL.Path) {
			c.Next()
			return
		}
		middleware.JWTAuth(rdb, cfg.Jwt)(c)
	})

	// 用户级令牌桶限流（默认 60 rps，burst=120；未登录退化到 IP）
	r.Use(rateLimitTokenBucket(rdb, 120, 60, func(c *gin.Context) string {
		if v, ok := c.Get("accountID"); ok {
			if id, ok := v.(uint); ok && id > 0 {
				return fmt.Sprintf("uid:%d", id)
			}
		}
		return "ip:" + clientIP(c)
	}))

	// 路由转发：使用 NoRoute 避免与静态路由冲突
	r.NoRoute(func(c *gin.Context) {
		upstream := pickUpstream(c.Request.URL.Path)
		if upstream == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown route"})
			return
		}
		// etcd 动态发现：命中则临时改写 proxy target
		if etcdReg != nil {
			if addr, ok := etcdReg.Pick(upstream, registry.ServiceHTTP); ok {
				target := mustParseURL("http://" + addr)
				proxies[upstream] = newProxy(target)
			}
		}
		p := proxies[upstream]
		if p == nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream not configured"})
			return
		}
		// 透传真实客户端信息
		c.Request.Header.Set("X-Forwarded-For", clientIP(c))
		p.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	})

	port := cfg.Server.GatewayPort
	if port == 0 {
		port = 8080
	}
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Gateway listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("gateway serve: %v", err)
	}
}

func newProxy(target *url.URL) *httputil.ReverseProxy {
	p := httputil.NewSingleHostReverseProxy(target)
	p.FlushInterval = 50 * time.Millisecond
	return p
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func pickUpstream(path string) string {
	switch {
	case strings.HasPrefix(path, "/account"):
		return "account"
	case strings.HasPrefix(path, "/video") || strings.HasPrefix(path, "/comment") || strings.HasPrefix(path, "/like") || strings.HasPrefix(path, "/static/"):
		return "video"
	case strings.HasPrefix(path, "/social"):
		return "social"
	case strings.HasPrefix(path, "/feed"):
		return "feed"
	case strings.HasPrefix(path, "/im"):
		return "im"
	default:
		return ""
	}
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		return "unknown"
	}
	return ip
}

// rateLimitTokenBucket Redis 令牌桶：capacity 为桶容量（burst），refillPerSec 为每秒补充令牌数（rps）。
// 每个请求消耗 1 个令牌；不足则返回 429。
func rateLimitTokenBucket(rdb *redis.Client, capacity int64, refillPerSec int64, keyFn func(*gin.Context) string) gin.HandlerFunc {
	script := `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_per_ms = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])

local data = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
if tokens == nil then tokens = capacity end
if ts == nil then ts = now_ms end

local delta = now_ms - ts
if delta < 0 then delta = 0 end
tokens = math.min(capacity, tokens + delta * refill_per_ms)

local allowed = 0
if tokens >= 1 then
  allowed = 1
  tokens = tokens - 1
end

redis.call("HMSET", key, "tokens", tokens, "ts", now_ms)
-- 过期时间：保证一段时间不访问后自动清理
local ttl = math.ceil((capacity / (refill_per_ms * 1000)) * 2)
if ttl < 10 then ttl = 10 end
redis.call("EXPIRE", key, ttl)
return allowed
`

	refillPerMs := float64(refillPerSec) / 1000.0

	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}
		k := keyFn(c)
		if k == "" {
			k = "unknown"
		}
		redisKey := fmt.Sprintf("gateway:tb:%s", k)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()

		allowed, err := rdb.EvalInt(ctx, script, []string{redisKey},
			capacity,
			refillPerMs,
			time.Now().UnixMilli(),
		)
		if err != nil {
			// Redis 异常时保守放行，避免网关成为单点（也可改为 fail-closed）
			c.Next()
			return
		}
		if allowed != 1 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

