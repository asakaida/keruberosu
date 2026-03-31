package database

import (
	"sync"
	"time"
)

// WriteTracker tracks recent writes per tenant to prevent stale reads
// from read replicas during replication lag.
type WriteTracker struct {
	mu       sync.RWMutex
	writes   map[string]time.Time
	window   time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewWriteTracker creates a new WriteTracker.
// windowSeconds specifies how long after a write to keep routing reads to primary.
// If windowSeconds <= 0, defaults to 1 second.
func NewWriteTracker(windowSeconds int) *WriteTracker {
	if windowSeconds <= 0 {
		windowSeconds = 1
	}
	return &WriteTracker{
		writes: make(map[string]time.Time),
		window: time.Duration(windowSeconds) * time.Second,
		stopCh: make(chan struct{}),
	}
}

// RecordWrite records a write for the given tenant.
func (w *WriteTracker) RecordWrite(tenantID string) {
	w.mu.Lock()
	w.writes[tenantID] = time.Now()
	w.mu.Unlock()
}

// HasRecentWrite returns true if the tenant had a write within the tracking window.
func (w *WriteTracker) HasRecentWrite(tenantID string) bool {
	w.mu.RLock()
	t, ok := w.writes[tenantID]
	w.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Since(t) < w.window
}

// Start begins the background cleanup goroutine that removes expired entries.
func (w *WriteTracker) Start() {
	go w.cleanupLoop()
}

// Stop stops the background cleanup goroutine.
func (w *WriteTracker) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
}

func (w *WriteTracker) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.cleanup()
		case <-w.stopCh:
			return
		}
	}
}

func (w *WriteTracker) cleanup() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for tenantID, t := range w.writes {
		if time.Since(t) >= w.window {
			delete(w.writes, tenantID)
		}
	}
}
