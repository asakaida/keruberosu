package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// SnapshotToken represents a PostgreSQL transaction snapshot.
// Format: "xmin:xmax:xip1,xip2,..." where xip is list of in-progress transactions.
type SnapshotToken struct {
	// Xmin is the earliest transaction ID that was still active
	Xmin int64

	// Xmax is the first transaction ID not yet assigned
	Xmax int64

	// Xip is the list of transaction IDs that were in progress
	Xip []int64
}

// String returns the snapshot token as a string
func (s *SnapshotToken) String() string {
	if len(s.Xip) == 0 {
		return fmt.Sprintf("%d:%d:", s.Xmin, s.Xmax)
	}

	xipStrs := make([]string, len(s.Xip))
	for i, xid := range s.Xip {
		xipStrs[i] = strconv.FormatInt(xid, 10)
	}

	return fmt.Sprintf("%d:%d:%s", s.Xmin, s.Xmax, strings.Join(xipStrs, ","))
}

// ParseSnapshotToken parses a snapshot token string
func ParseSnapshotToken(token string) (*SnapshotToken, error) {
	if token == "" {
		return nil, fmt.Errorf("empty snapshot token")
	}

	parts := strings.Split(token, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid snapshot token format: %s", token)
	}

	xmin, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid xmin in snapshot token: %w", err)
	}

	xmax, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid xmax in snapshot token: %w", err)
	}

	var xip []int64
	if len(parts) > 2 && parts[2] != "" {
		xipStrs := strings.Split(parts[2], ",")
		xip = make([]int64, 0, len(xipStrs))
		for _, xipStr := range xipStrs {
			xid, err := strconv.ParseInt(xipStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid xip in snapshot token: %w", err)
			}
			xip = append(xip, xid)
		}
	}

	return &SnapshotToken{
		Xmin: xmin,
		Xmax: xmax,
		Xip:  xip,
	}, nil
}

// SnapshotProvider provides snapshot tokens for MVCC-based caching.
// This interface allows dependency injection for different snapshot strategies.
type SnapshotProvider interface {
	GetCurrentSnapshotForRead(ctx context.Context) (*SnapshotToken, error)
}

// TokenGenerator generates snapshot tokens during write operations.
type TokenGenerator interface {
	// GenerateWriteToken generates a snapshot token during a write operation.
	// This is called within a transaction to capture the exact point of the write.
	// Returns the encoded token that clients should use for subsequent reads.
	GenerateWriteToken(ctx context.Context, tx *sql.Tx) (string, error)
}

// SnapshotManager manages PostgreSQL MVCC snapshots for caching.
type SnapshotManager struct {
	db *sql.DB
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(db *sql.DB) *SnapshotManager {
	return &SnapshotManager{db: db}
}

// GetCurrentSnapshotToken returns the current transaction ID as a snapshot token.
// This is used for cache invalidation - when data changes, the transaction ID changes.
func (m *SnapshotManager) GetCurrentSnapshotToken(ctx context.Context) (*SnapshotToken, error) {
	var txid int64
	err := m.db.QueryRowContext(ctx, "SELECT txid_current()").Scan(&txid)
	if err != nil {
		return nil, fmt.Errorf("failed to get current transaction ID: %w", err)
	}

	return &SnapshotToken{
		Xmin: txid,
		Xmax: txid,
		Xip:  nil,
	}, nil
}

// GetCurrentSnapshotForRead returns a snapshot token for consistent reads.
// This uses txid_current_snapshot() which returns the current snapshot.
func (m *SnapshotManager) GetCurrentSnapshotForRead(ctx context.Context) (*SnapshotToken, error) {
	var snapshotStr string
	err := m.db.QueryRowContext(ctx, "SELECT txid_current_snapshot()::text").Scan(&snapshotStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get current snapshot: %w", err)
	}

	return ParseSnapshotToken(snapshotStr)
}

// GenerateWriteToken generates a snapshot token during a write operation.
// This should be called within a transaction to capture the exact transaction ID.
// The returned token represents the state after this write is committed.
func (m *SnapshotManager) GenerateWriteToken(ctx context.Context, tx *sql.Tx) (string, error) {
	var txid int64
	err := tx.QueryRowContext(ctx, "SELECT txid_current()").Scan(&txid)
	if err != nil {
		return "", fmt.Errorf("failed to get current transaction ID: %w", err)
	}

	token := &SnapshotToken{
		Xmin: txid,
		Xmax: txid + 1,
		Xip:  nil,
	}

	return token.String(), nil
}

// GenerateWriteTokenWithDB generates a snapshot token using the database connection directly.
// This is a convenience method when a transaction is not available.
func (m *SnapshotManager) GenerateWriteTokenWithDB(ctx context.Context) (string, error) {
	var txid int64
	err := m.db.QueryRowContext(ctx, "SELECT txid_current()").Scan(&txid)
	if err != nil {
		return "", fmt.Errorf("failed to get current transaction ID: %w", err)
	}

	token := &SnapshotToken{
		Xmin: txid,
		Xmax: txid + 1,
		Xip:  nil,
	}

	return token.String(), nil
}

// IsTransactionVisible checks if a transaction is visible in the given snapshot.
// This is used for filtering query results based on MVCC.
func (s *SnapshotToken) IsTransactionVisible(xid int64) bool {
	// Transaction is too new
	if xid >= s.Xmax {
		return false
	}

	// Transaction is too old (committed before snapshot)
	if xid < s.Xmin {
		return true
	}

	// Transaction is in the in-progress list
	for _, inProgressXid := range s.Xip {
		if xid == inProgressXid {
			return false
		}
	}

	// Transaction is between xmin and xmax and not in progress
	return true
}

// BuildMVCCCondition builds SQL WHERE conditions for MVCC visibility.
// Returns SQL fragment for filtering rows based on transaction visibility.
func (s *SnapshotToken) BuildMVCCCondition() string {
	conditions := []string{
		// Row must be created before snapshot
		fmt.Sprintf("xmin::text::bigint < %d", s.Xmax),
		// Row must not be deleted, or deleted after snapshot
		fmt.Sprintf("(xmax = 0 OR xmax::text::bigint >= %d)", s.Xmax),
	}

	// If there are in-progress transactions, exclude rows created by them
	if len(s.Xip) > 0 {
		xipStrs := make([]string, len(s.Xip))
		for i, xid := range s.Xip {
			xipStrs[i] = strconv.FormatInt(xid, 10)
		}
		conditions = append(conditions, fmt.Sprintf("xmin::text::bigint NOT IN (%s)", strings.Join(xipStrs, ",")))
	}

	return strings.Join(conditions, " AND ")
}
