package cache

import (
	"context"
	"time"
)

// Cache is the interface for caching check results.
// It provides Get, Set, and Delete operations with TTL support.
type Cache interface {
	// Get retrieves a value from cache.
	// Returns the value and true if found, or nil and false if not found.
	Get(ctx context.Context, key string) (interface{}, bool)

	// Set stores a value in cache with TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from cache.
	Delete(ctx context.Context, key string) error

	// Clear removes all entries from cache.
	Clear(ctx context.Context) error

	// Close releases resources held by the cache.
	Close() error

	// Metrics returns cache statistics.
	Metrics() *Metrics
}

// Metrics holds cache performance statistics.
type Metrics struct {
	// Hits is the number of cache hits
	Hits uint64

	// Misses is the number of cache misses
	Misses uint64

	// KeysAdded is the number of keys added to cache
	KeysAdded uint64

	// KeysEvicted is the number of keys evicted from cache
	KeysEvicted uint64

	// CostAdded is the total cost of added entries
	CostAdded uint64

	// CostEvicted is the total cost of evicted entries
	CostEvicted uint64
}

// HitRate returns the cache hit rate (0.0 to 1.0).
func (m *Metrics) HitRate() float64 {
	total := m.Hits + m.Misses
	if total == 0 {
		return 0.0
	}
	return float64(m.Hits) / float64(total)
}
