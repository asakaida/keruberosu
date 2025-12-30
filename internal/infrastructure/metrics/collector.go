package metrics

import (
	"sync"
	"sync/atomic"

	"github.com/asakaida/keruberosu/pkg/cache"
	"github.com/asakaida/keruberosu/pkg/cache/memorycache"
)

// Collector collects and aggregates metrics for the application.
type Collector struct {
	// API metrics
	apiRequests sync.Map // map[string]*uint64 - method -> count
	apiErrors   sync.Map // map[string]*uint64 - method -> error count
	apiDuration sync.Map // map[string]*durationValue - method -> total duration in seconds

	// Cache reference (optional, for querying cache-specific metrics)
	cache cache.Cache
}

// durationValue holds duration with mutex for thread-safe updates.
type durationValue struct {
	mu           sync.Mutex
	totalSeconds float64
}

// CacheMetrics holds cache performance metrics.
type CacheMetrics struct {
	Hits        uint64
	Misses      uint64
	HitRate     float64
	KeysCurrent int64
	MemoryBytes int64
	Evictions   uint64
}

// APIMetrics holds API request metrics.
type APIMetrics struct {
	RequestCounts        map[string]uint64
	ErrorCounts          map[string]uint64
	TotalDurationSeconds map[string]float64
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{}
}

// SetCache sets the cache instance for collecting cache metrics.
func (c *Collector) SetCache(cache cache.Cache) {
	c.cache = cache
}

// RecordRequest records an API request.
func (c *Collector) RecordRequest(method string) {
	counter := c.getOrCreateCounter(&c.apiRequests, method)
	atomic.AddUint64(counter, 1)
}

// RecordError records an API error.
func (c *Collector) RecordError(method string) {
	counter := c.getOrCreateCounter(&c.apiErrors, method)
	atomic.AddUint64(counter, 1)
}

// RecordDuration records the duration of an API call in seconds.
func (c *Collector) RecordDuration(method string, durationSeconds float64) {
	val, _ := c.apiDuration.LoadOrStore(method, &durationValue{})
	dv := val.(*durationValue)

	dv.mu.Lock()
	dv.totalSeconds += durationSeconds
	dv.mu.Unlock()
}

// GetCacheMetrics returns current cache metrics.
func (c *Collector) GetCacheMetrics() *CacheMetrics {
	if c.cache == nil {
		return &CacheMetrics{}
	}

	metrics := c.cache.Metrics()
	if metrics == nil {
		return &CacheMetrics{}
	}

	result := &CacheMetrics{
		Hits:      metrics.Hits,
		Misses:    metrics.Misses,
		HitRate:   metrics.HitRate(),
		Evictions: metrics.KeysEvicted,
	}

	// Get current keys and memory if available
	if memCache, ok := c.cache.(*memorycache.Cache); ok {
		result.KeysCurrent = int64(memCache.Len())
		result.MemoryBytes = memCache.Size()
	}

	return result
}

// GetAPIMetrics returns current API metrics.
func (c *Collector) GetAPIMetrics() *APIMetrics {
	result := &APIMetrics{
		RequestCounts:        make(map[string]uint64),
		ErrorCounts:          make(map[string]uint64),
		TotalDurationSeconds: make(map[string]float64),
	}

	// Collect request counts
	c.apiRequests.Range(func(key, value interface{}) bool {
		method := key.(string)
		count := atomic.LoadUint64(value.(*uint64))
		result.RequestCounts[method] = count
		return true
	})

	// Collect error counts
	c.apiErrors.Range(func(key, value interface{}) bool {
		method := key.(string)
		count := atomic.LoadUint64(value.(*uint64))
		result.ErrorCounts[method] = count
		return true
	})

	// Collect duration totals
	c.apiDuration.Range(func(key, value interface{}) bool {
		method := key.(string)
		dv := value.(*durationValue)
		dv.mu.Lock()
		result.TotalDurationSeconds[method] = dv.totalSeconds
		dv.mu.Unlock()
		return true
	})

	return result
}

// getOrCreateCounter gets or creates a counter for the given key.
func (c *Collector) getOrCreateCounter(m *sync.Map, key string) *uint64 {
	val, _ := m.LoadOrStore(key, new(uint64))
	return val.(*uint64)
}
