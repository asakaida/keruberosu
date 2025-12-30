package cache

import (
	"context"
	"testing"
	"time"
)

func TestSnapshotManager_SetToken(t *testing.T) {
	// Create a SnapshotManager without DB connection (testing mode)
	mgr := &SnapshotManager{
		db:         nil,
		refreshTTL: 5 * time.Minute,
		stopCh:     make(chan struct{}),
	}

	// Set token manually
	mgr.SetToken("test-token-123")

	// Get current token should return the set token
	token, err := mgr.GetCurrentToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != "test-token-123" {
		t.Errorf("expected token 'test-token-123', got '%s'", token)
	}
}

func TestSnapshotManager_GetCurrentSnapshotForRead(t *testing.T) {
	// Create a SnapshotManager without DB connection (testing mode)
	mgr := &SnapshotManager{
		db:         nil,
		refreshTTL: 5 * time.Minute,
		stopCh:     make(chan struct{}),
	}

	// Set a numeric token
	mgr.SetToken("12345")

	// Get snapshot for read
	snapshot, err := mgr.GetCurrentSnapshotForRead(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot == nil {
		t.Fatal("expected snapshot, got nil")
	}

	if snapshot.Xmin != 12345 {
		t.Errorf("expected Xmin 12345, got %d", snapshot.Xmin)
	}

	if snapshot.Xmax != 12345 {
		t.Errorf("expected Xmax 12345, got %d", snapshot.Xmax)
	}
}

func TestSnapshotManager_GetCurrentSnapshotForRead_EmptyToken(t *testing.T) {
	// Create a SnapshotManager without DB connection (testing mode)
	mgr := &SnapshotManager{
		db:         nil,
		refreshTTL: 5 * time.Minute,
		stopCh:     make(chan struct{}),
	}

	// Don't set any token - should return empty snapshot
	snapshot, err := mgr.GetCurrentSnapshotForRead(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot == nil {
		t.Fatal("expected snapshot, got nil")
	}

	// Empty token should result in zero values
	if snapshot.Xmin != 0 {
		t.Errorf("expected Xmin 0, got %d", snapshot.Xmin)
	}
}

func TestSnapshotManager_Stop(t *testing.T) {
	// Create a SnapshotManager without DB connection (testing mode)
	mgr := &SnapshotManager{
		db:         nil,
		refreshTTL: 5 * time.Minute,
		stopCh:     make(chan struct{}),
	}

	// Stop should not panic
	err := mgr.Stop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second stop should also not panic
	err = mgr.Stop()
	if err != nil {
		t.Fatalf("unexpected error on second stop: %v", err)
	}
}

func TestSnapshotManager_TokenRefresh(t *testing.T) {
	// Create a SnapshotManager with very short TTL
	mgr := &SnapshotManager{
		db:         nil, // No DB means no actual refresh, but we can test the logic
		refreshTTL: 1 * time.Millisecond,
		stopCh:     make(chan struct{}),
	}

	// Set initial token
	mgr.SetToken("initial-token")

	// Wait for TTL to expire
	time.Sleep(5 * time.Millisecond)

	// In testing mode (db == nil), GetCurrentToken should still return the current token
	// even when TTL has expired
	token, err := mgr.GetCurrentToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != "initial-token" {
		t.Errorf("expected 'initial-token', got '%s'", token)
	}
}
