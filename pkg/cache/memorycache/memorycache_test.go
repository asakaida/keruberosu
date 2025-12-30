package memorycache

import (
	"context"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	cache, err := New(&Config{
		MaxSizeBytes:  1024 * 1024, // 1MB
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()

	// Set a value
	err = cache.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	// Get the value
	value, found := cache.Get(ctx, "key1")
	if !found {
		t.Error("expected to find key1")
	}
	if value != "value1" {
		t.Errorf("expected value1, got %v", value)
	}

	// Get non-existent key
	_, found = cache.Get(ctx, "nonexistent")
	if found {
		t.Error("expected not to find nonexistent key")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	cache, err := New(&Config{
		MaxSizeBytes:  1024 * 1024,
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()

	// Set a value with short TTL
	err = cache.Set(ctx, "key1", "value1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	// Should find it immediately
	_, found := cache.Get(ctx, "key1")
	if !found {
		t.Error("expected to find key1 before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should not find it after expiration
	_, found = cache.Get(ctx, "key1")
	if found {
		t.Error("expected not to find key1 after expiration")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	// Create a cache with very small capacity
	cache, err := New(&Config{
		MaxSizeBytes:  200, // Very small
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()

	// Add multiple items
	for i := 0; i < 10; i++ {
		key := string(rune('a' + i))
		err = cache.Set(ctx, key, i, time.Minute)
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}
	}

	// Cache should have evicted older items
	if cache.Len() >= 10 {
		t.Errorf("expected less than 10 items due to eviction, got %d", cache.Len())
	}

	// Most recent items should still be present
	_, found := cache.Get(ctx, "j") // last item
	if !found {
		t.Error("expected to find most recent item 'j'")
	}
}

func TestCache_Delete(t *testing.T) {
	cache, err := New(&Config{
		MaxSizeBytes:  1024 * 1024,
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()

	// Set and verify
	cache.Set(ctx, "key1", "value1", time.Minute)
	_, found := cache.Get(ctx, "key1")
	if !found {
		t.Error("expected to find key1")
	}

	// Delete
	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Should not find it
	_, found = cache.Get(ctx, "key1")
	if found {
		t.Error("expected not to find key1 after deletion")
	}

	// Delete non-existent key should not error
	err = cache.Delete(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("delete of non-existent key should not error: %v", err)
	}
}

func TestCache_Clear(t *testing.T) {
	cache, err := New(&Config{
		MaxSizeBytes:  1024 * 1024,
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()

	// Add multiple items
	cache.Set(ctx, "key1", "value1", time.Minute)
	cache.Set(ctx, "key2", "value2", time.Minute)
	cache.Set(ctx, "key3", "value3", time.Minute)

	if cache.Len() != 3 {
		t.Errorf("expected 3 items, got %d", cache.Len())
	}

	// Clear
	err = cache.Clear(ctx)
	if err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	if cache.Len() != 0 {
		t.Errorf("expected 0 items after clear, got %d", cache.Len())
	}
}

func TestCache_Metrics(t *testing.T) {
	cache, err := New(&Config{
		MaxSizeBytes:  1024 * 1024,
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()

	// Initially no hits or misses
	metrics := cache.Metrics()
	if metrics.Hits != 0 || metrics.Misses != 0 {
		t.Errorf("expected 0 hits and misses initially, got %d hits and %d misses", metrics.Hits, metrics.Misses)
	}

	// Set a value
	cache.Set(ctx, "key1", "value1", time.Minute)

	// Get should be a hit
	cache.Get(ctx, "key1")
	metrics = cache.Metrics()
	if metrics.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", metrics.Hits)
	}

	// Get non-existent should be a miss
	cache.Get(ctx, "nonexistent")
	metrics = cache.Metrics()
	if metrics.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", metrics.Misses)
	}

	// Verify hit rate
	expectedHitRate := 0.5 // 1 hit, 1 miss
	if metrics.HitRate() != expectedHitRate {
		t.Errorf("expected hit rate %f, got %f", expectedHitRate, metrics.HitRate())
	}
}

func TestCache_UpdateExisting(t *testing.T) {
	cache, err := New(&Config{
		MaxSizeBytes:  1024 * 1024,
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()

	// Set initial value
	cache.Set(ctx, "key1", "value1", time.Minute)

	// Update value
	cache.Set(ctx, "key1", "value2", time.Minute)

	// Get updated value
	value, found := cache.Get(ctx, "key1")
	if !found {
		t.Error("expected to find key1")
	}
	if value != "value2" {
		t.Errorf("expected value2, got %v", value)
	}

	// Should still be only 1 item
	if cache.Len() != 1 {
		t.Errorf("expected 1 item, got %d", cache.Len())
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache, err := New(&Config{
		MaxSizeBytes:  1024 * 1024,
		DefaultTTL:    time.Minute,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a' + id))
				cache.Set(ctx, key, j, time.Minute)
			}
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a' + id))
				cache.Get(ctx, key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Just verify no panics occurred
	t.Log("concurrent access test passed without panics")
}
