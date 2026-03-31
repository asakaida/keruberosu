package memorycache

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asakaida/keruberosu/pkg/cache"
)

// entry represents a cache entry with value and metadata
type entry struct {
	key       string
	value     interface{}
	expiresAt time.Time
	size      int64 // Approximate memory size in bytes
}

// Cache implements an LRU cache with TTL support.
// This cache is simple, predictable, and maintainable.
type Cache struct {
	mu sync.RWMutex

	// LRU tracking
	items     map[string]*list.Element // key -> list element
	evictList *list.List               // LRU list (front = most recent, back = least recent)

	// Configuration
	maxSize int64 // Maximum total size in bytes
	ttl     time.Duration

	// Current state
	currentSize int64

	// Metrics
	metrics *cacheMetrics
}

type cacheMetrics struct {
	hits        atomic.Uint64
	misses      atomic.Uint64
	keysAdded   atomic.Uint64
	keysEvicted atomic.Uint64
}

// Config holds configuration for the memory cache.
type Config struct {
	// MaxSizeBytes is the maximum total size of cached items in bytes.
	// When this limit is exceeded, least recently used items are evicted.
	MaxSizeBytes int64

	// DefaultTTL is the default time-to-live for cached items.
	// Items expire after this duration.
	DefaultTTL time.Duration

	// EnableMetrics enables collection of cache metrics.
	EnableMetrics bool
}

// New creates a new memory cache with the given configuration.
func New(config *Config) (*Cache, error) {
	maxSize := config.MaxSizeBytes
	if maxSize <= 0 {
		maxSize = 100 * 1024 * 1024 // default 100MB
	}

	c := &Cache{
		items:     make(map[string]*list.Element),
		evictList: list.New(),
		maxSize:   maxSize,
		ttl:       config.DefaultTTL,
	}

	if config.EnableMetrics {
		c.metrics = &cacheMetrics{}
	}

	return c, nil
}

// Get retrieves a value from cache.
func (c *Cache) Get(ctx context.Context, key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.items[key]
	if !exists {
		if c.metrics != nil {
			c.metrics.misses.Add(1)
		}
		return nil, false
	}

	ent := elem.Value.(*entry)

	// Check if expired
	if time.Now().After(ent.expiresAt) {
		c.removeElement(elem)
		if c.metrics != nil {
			c.metrics.misses.Add(1)
		}
		return nil, false
	}

	// Cache hit - move to front for LRU
	c.evictList.MoveToFront(elem)
	if c.metrics != nil {
		c.metrics.hits.Add(1)
	}

	return ent.value, true
}

// Set stores a value in cache with the specified TTL.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Estimate size (rough approximation: 100 bytes per entry + key length)
	size := int64(100 + len(key))

	// Check if key already exists
	if elem, exists := c.items[key]; exists {
		// Update existing entry
		ent := elem.Value.(*entry)
		oldSize := ent.size
		ent.value = value
		ent.expiresAt = time.Now().Add(ttl)
		ent.size = size
		c.currentSize += (size - oldSize)
		c.evictList.MoveToFront(elem)
		return nil
	}

	// Add new entry
	ent := &entry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
		size:      size,
	}

	elem := c.evictList.PushFront(ent)
	c.items[key] = elem
	c.currentSize += size

	if c.metrics != nil {
		c.metrics.keysAdded.Add(1)
	}

	// Evict LRU items if over capacity
	for c.currentSize > c.maxSize && c.evictList.Len() > 0 {
		oldest := c.evictList.Back()
		if oldest != nil {
			c.removeElement(oldest)
			if c.metrics != nil {
				c.metrics.keysEvicted.Add(1)
			}
		}
	}

	return nil
}

// Delete removes a value from cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		c.removeElement(elem)
	}

	return nil
}

// Clear removes all entries from cache.
func (c *Cache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.evictList.Init()
	c.currentSize = 0

	return nil
}

// Close releases resources (no-op for memory cache).
func (c *Cache) Close() error {
	return nil
}

// Metrics returns cache statistics.
func (c *Cache) Metrics() *cache.Metrics {
	if c.metrics == nil {
		return &cache.Metrics{}
	}

	return &cache.Metrics{
		Hits:        c.metrics.hits.Load(),
		Misses:      c.metrics.misses.Load(),
		KeysAdded:   c.metrics.keysAdded.Load(),
		KeysEvicted: c.metrics.keysEvicted.Load(),
	}
}

// ResetMetrics resets cache statistics.
func (c *Cache) ResetMetrics() {
	if c.metrics == nil {
		return
	}

	c.metrics.hits.Store(0)
	c.metrics.misses.Store(0)
	c.metrics.keysAdded.Store(0)
	c.metrics.keysEvicted.Store(0)
}

// removeElement removes an element from cache (must be called with lock held).
func (c *Cache) removeElement(elem *list.Element) {
	c.evictList.Remove(elem)
	ent := elem.Value.(*entry)
	delete(c.items, ent.key)
	c.currentSize -= ent.size
}

// Len returns the current number of items in cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.evictList.Len()
}

// Size returns the current total size in bytes.
func (c *Cache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}
