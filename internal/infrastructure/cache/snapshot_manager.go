package cache

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/asakaida/keruberosu/internal/repositories/postgres"
	"github.com/lib/pq"
)

// SnapshotManager manages snapshot tokens for cache consistency across distributed instances.
// It uses PostgreSQL LISTEN/NOTIFY for instant synchronization when data changes.
type SnapshotManager struct {
	mu           sync.RWMutex
	currentToken string
	db           *sql.DB
	refreshTTL   time.Duration
	lastRefresh  time.Time
	listener     *pq.Listener
	connStr      string
	stopCh       chan struct{}
	stopped      bool
}

// NewSnapshotManager creates a new SnapshotManager.
// connStr is the PostgreSQL connection string for LISTEN/NOTIFY.
// refreshTTL is the fallback interval for refreshing the token from DB.
func NewSnapshotManager(db *sql.DB, connStr string, refreshTTL time.Duration) *SnapshotManager {
	return &SnapshotManager{
		db:         db,
		connStr:    connStr,
		refreshTTL: refreshTTL,
		stopCh:     make(chan struct{}),
	}
}

// Start initializes the SnapshotManager by fetching the initial token
// and starting the LISTEN/NOTIFY listener.
func (m *SnapshotManager) Start(ctx context.Context) error {
	// Fetch initial token
	token, err := m.fetchLatestToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch initial token: %w", err)
	}

	m.mu.Lock()
	m.currentToken = token
	m.lastRefresh = time.Now()
	m.mu.Unlock()

	// Start listener for NOTIFY events
	if err := m.startListener(); err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	return nil
}

// Stop stops the SnapshotManager and cleans up resources.
func (m *SnapshotManager) Stop() error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil
	}
	m.stopped = true
	close(m.stopCh)
	m.mu.Unlock()

	if m.listener != nil {
		return m.listener.Close()
	}
	return nil
}

// GetCurrentToken returns the current snapshot token.
// If the token is stale (older than refreshTTL), it refreshes from the database.
func (m *SnapshotManager) GetCurrentToken(ctx context.Context) (string, error) {
	m.mu.RLock()
	token := m.currentToken
	needsRefresh := time.Since(m.lastRefresh) > m.refreshTTL
	m.mu.RUnlock()

	// If db is nil (testing mode), just return the current token
	if m.db == nil {
		return token, nil
	}

	if needsRefresh || token == "" {
		return m.refreshFromDB(ctx)
	}

	return token, nil
}

// refreshFromDB fetches the latest token from the database and updates the cache.
func (m *SnapshotManager) refreshFromDB(ctx context.Context) (string, error) {
	token, err := m.fetchLatestToken(ctx)
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	m.currentToken = token
	m.lastRefresh = time.Now()
	m.mu.Unlock()

	return token, nil
}

// fetchLatestToken fetches the latest transaction ID from the database.
func (m *SnapshotManager) fetchLatestToken(ctx context.Context) (string, error) {
	var token string
	err := m.db.QueryRowContext(ctx, `
		SELECT COALESCE(id::text, '')
		FROM transactions
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&token)

	if err == sql.ErrNoRows {
		// No transactions yet, return empty token
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest token: %w", err)
	}

	return token, nil
}

// startListener starts the PostgreSQL LISTEN/NOTIFY listener.
func (m *SnapshotManager) startListener() error {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			// Log error but don't fail - we have TTL fallback
			fmt.Printf("SnapshotManager listener error: %v\n", err)
		}
	}

	m.listener = pq.NewListener(m.connStr, 10*time.Second, time.Minute, reportProblem)

	err := m.listener.Listen("snapshot_changed")
	if err != nil {
		return fmt.Errorf("failed to listen on snapshot_changed: %w", err)
	}

	// Start goroutine to handle notifications
	go m.handleNotifications()

	return nil
}

// handleNotifications processes incoming NOTIFY events.
func (m *SnapshotManager) handleNotifications() {
	for {
		select {
		case <-m.stopCh:
			return
		case notification := <-m.listener.Notify:
			if notification == nil {
				// Connection lost, listener will reconnect automatically
				continue
			}

			// Update token from notification payload
			m.mu.Lock()
			m.currentToken = notification.Extra
			m.lastRefresh = time.Now()
			m.mu.Unlock()
		case <-time.After(90 * time.Second):
			// Periodic ping to keep connection alive
			go func() {
				if err := m.listener.Ping(); err != nil {
					fmt.Printf("SnapshotManager ping error: %v\n", err)
				}
			}()
		}
	}
}

// SetToken manually sets the current token.
// This is primarily used for testing.
func (m *SnapshotManager) SetToken(token string) {
	m.mu.Lock()
	m.currentToken = token
	m.lastRefresh = time.Now()
	m.mu.Unlock()
}

// GetCurrentSnapshotForRead implements postgres.SnapshotProvider interface.
// This allows SnapshotManager to be used directly with existing Checker code.
// The token value is used as both Xmin and Xmax for compatibility.
func (m *SnapshotManager) GetCurrentSnapshotForRead(ctx context.Context) (*postgres.SnapshotToken, error) {
	token, err := m.GetCurrentToken(ctx)
	if err != nil {
		return nil, err
	}

	// Parse token as int64, default to 0 if empty or invalid
	var tokenValue int64
	if token != "" {
		fmt.Sscanf(token, "%d", &tokenValue)
	}

	return &postgres.SnapshotToken{
		Xmin: tokenValue,
		Xmax: tokenValue,
		Xip:  nil,
	}, nil
}
