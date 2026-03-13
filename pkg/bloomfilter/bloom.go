package bloomfilter

import (
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

// Filter 布隆过滤器，用于缓存穿透防护：Test=false 表示 key 一定不存在，可跳过 cache+DB
type Filter struct {
	mu     sync.RWMutex
	filter *bloom.BloomFilter
	ready  bool   // 已从 DB 回填或尚未需要拦截，为 true 时才用 bloom 拦截
	n      uint   // 预期元素数量，Rebuild 时复用
	fp     float64 // 误判率，Rebuild 时复用
}

// New 创建布隆过滤器，n 为预期元素数量，fp 为可接受误判率（如 0.01）
func New(n uint, fp float64) *Filter {
	return &Filter{
		filter: bloom.NewWithEstimates(n, fp),
		n:      n,
		fp:     fp,
	}
}

// SetReady 标记过滤器已就绪（完成 DB 回填后调用），未调用前 Test 不拦截
func (f *Filter) SetReady() {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ready = true
}

// Add 将 key 加入过滤器（在 Create 成功后调用）
func (f *Filter) Add(key string) {
	if f == nil || f.filter == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filter.AddString(key)
}

// Test 判断 key 是否可能存在；false 且 ready 表示一定不存在
func (f *Filter) Test(key string) bool {
	if f == nil || f.filter == nil {
		return true
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.filter.TestString(key)
}

// ShouldReject 若 key 一定不存在则返回 true（用于穿透防护）
func (f *Filter) ShouldReject(key string) bool {
	if f == nil || f.filter == nil {
		return false
	}
	f.mu.RLock()
	ok := f.ready && !f.filter.TestString(key)
	f.mu.RUnlock()
	return ok
}

// Rebuild 用 keys 重建过滤器（用于定时任务，清除已删除数据；保持 ready 状态）
func (f *Filter) Rebuild(keys []string) {
	if f == nil {
		return
	}
	newFilter := bloom.NewWithEstimates(f.n, f.fp)
	for _, k := range keys {
		newFilter.AddString(k)
	}
	f.mu.Lock()
	f.filter = newFilter
	f.mu.Unlock()
}
