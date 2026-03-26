package registry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceType 区分 HTTP / gRPC
type ServiceType string

const (
	ServiceHTTP ServiceType = "http"
	ServiceGRPC ServiceType = "grpc"
)

// EtcdRegistry 基于 etcd 的服务注册与发现。
// 约定 key：{prefix}/{service}/{type}/{instanceID} => {address}
type EtcdRegistry struct {
	cli    *clientv3.Client
	prefix string
	ttl    int64

	mu     sync.RWMutex
	cache  map[string][]string // "{service}/{type}" -> []addr
	closed chan struct{}
}

type Config struct {
	Endpoints []string
	Prefix    string
	TTL       int // seconds
}

func NewEtcd(cfg Config) (*EtcdRegistry, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, errors.New("etcd endpoints empty")
	}
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = "/kairos/services"
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 10
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	// 快速探活：避免 etcd 不可用时阻塞业务启动
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	_, err = cli.Status(ctx, cfg.Endpoints[0])
	cancel()
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("etcd status failed: %w", err)
	}

	r := &EtcdRegistry{
		cli:    cli,
		prefix: strings.TrimRight(prefix, "/"),
		ttl:    int64(ttl),
		cache:  make(map[string][]string),
		closed: make(chan struct{}),
	}
	return r, nil
}

func (r *EtcdRegistry) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	if r.cli != nil {
		return r.cli.Close()
	}
	return nil
}

// Register 注册一个实例（带 lease keepalive）。
// - service：如 "account"
// - st：ServiceHTTP / ServiceGRPC
// - instanceID：如 "host:port-pid"（需唯一）
// - addr：对外可访问地址（如 "127.0.0.1:8081" 或 "http://127.0.0.1:8081"）
func (r *EtcdRegistry) Register(ctx context.Context, service string, st ServiceType, instanceID string, addr string) (context.CancelFunc, error) {
	if r == nil || r.cli == nil {
		return nil, errors.New("etcd registry not initialized")
	}
	service = strings.Trim(service, "/")
	instanceID = strings.TrimSpace(instanceID)
	if service == "" || instanceID == "" || addr == "" {
		return nil, errors.New("service/instanceID/addr required")
	}

	key := fmt.Sprintf("%s/%s/%s/%s", r.prefix, service, st, instanceID)
	lease, err := r.cli.Grant(ctx, r.ttl)
	if err != nil {
		return nil, err
	}
	if _, err := r.cli.Put(ctx, key, addr, clientv3.WithLease(lease.ID)); err != nil {
		return nil, err
	}

	kctx, cancel := context.WithCancel(context.Background())
	ch, err := r.cli.KeepAlive(kctx, lease.ID)
	if err != nil {
		cancel()
		return nil, err
	}
	go func() {
		for {
			select {
			case <-kctx.Done():
				return
			case <-r.closed:
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
			}
		}
	}()
	return cancel, nil
}

// StartPollingDiscovery 定时拉取 service/type 下的实例列表，写入本地缓存。
// 发现失败时保留旧缓存，不影响调用方继续使用（最终一致）。
func (r *EtcdRegistry) StartPollingDiscovery(ctx context.Context, service string, st ServiceType, interval time.Duration) {
	if r == nil || r.cli == nil {
		return
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	service = strings.Trim(service, "/")
	cacheKey := fmt.Sprintf("%s/%s", service, st)
	prefix := fmt.Sprintf("%s/%s/%s/", r.prefix, service, st)

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.closed:
				return
			case <-ticker.C:
				cctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
				resp, err := r.cli.Get(cctx, prefix, clientv3.WithPrefix())
				cancel()
				if err != nil {
					log.Printf("etcd discovery get failed (%s): %v", prefix, err)
					continue
				}
				addrs := make([]string, 0, len(resp.Kvs))
				for _, kv := range resp.Kvs {
					v := strings.TrimSpace(string(kv.Value))
					if v != "" {
						addrs = append(addrs, v)
					}
				}
				r.mu.Lock()
				r.cache[cacheKey] = addrs
				r.mu.Unlock()
			}
		}
	}()
}

// Pick 从缓存中随机选一个实例地址。
func (r *EtcdRegistry) Pick(service string, st ServiceType) (string, bool) {
	if r == nil {
		return "", false
	}
	cacheKey := fmt.Sprintf("%s/%s", strings.Trim(service, "/"), st)
	r.mu.RLock()
	list := r.cache[cacheKey]
	r.mu.RUnlock()
	if len(list) == 0 {
		return "", false
	}
	return list[rand.Intn(len(list))], true
}

