package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/lib/pq"
)

// RetryConfig configures retry behavior for transient DB errors.
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   2 * time.Second,
	}
}

// ResilientDB wraps *sql.DB with automatic retry for transient errors.
// It implements the DBTX interface.
type ResilientDB struct {
	db     *sql.DB
	config RetryConfig
}

// NewResilientDB creates a new ResilientDB wrapping the given *sql.DB.
func NewResilientDB(db *sql.DB, cfg RetryConfig) *ResilientDB {
	return &ResilientDB{db: db, config: cfg}
}

// DB returns the underlying *sql.DB.
func (r *ResilientDB) DB() *sql.DB {
	return r.db
}

// ExecContext executes a query with retry on transient errors.
func (r *ResilientDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	err := r.retry(ctx, func() error {
		var execErr error
		result, execErr = r.db.ExecContext(ctx, query, args...)
		return execErr
	})
	return result, err
}

// QueryContext executes a query returning rows with retry on transient errors.
func (r *ResilientDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	err := r.retry(ctx, func() error {
		var queryErr error
		rows, queryErr = r.db.QueryContext(ctx, query, args...)
		return queryErr
	})
	return rows, err
}

// QueryRowContext executes a query returning a single row.
// This is passed through without retry since the error is deferred until Scan.
func (r *ResilientDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return r.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction with retry on transient errors.
func (r *ResilientDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	var tx *sql.Tx
	err := r.retry(ctx, func() error {
		var beginErr error
		tx, beginErr = r.db.BeginTx(ctx, opts)
		return beginErr
	})
	return tx, err
}

// PingContext verifies the database connection.
func (r *ResilientDB) PingContext(ctx context.Context) error {
	return r.retry(ctx, func() error {
		return r.db.PingContext(ctx)
	})
}

func (r *ResilientDB) retry(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isTransientError(lastErr) {
			return lastErr
		}
		if attempt == r.config.MaxRetries {
			break
		}
		delay := r.backoffDelay(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

func (r *ResilientDB) backoffDelay(attempt int) time.Duration {
	delay := float64(r.config.BaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}
	// Add jitter: 50-100% of calculated delay
	jitter := delay * (0.5 + rand.Float64()*0.5)
	return time.Duration(jitter)
}

// isTransientError checks if an error is transient and worth retrying.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch {
		case strings.HasPrefix(string(pqErr.Code), "08"): // connection exceptions
			return true
		case strings.HasPrefix(string(pqErr.Code), "53"): // insufficient resources
			return true
		case string(pqErr.Code) == "57P01": // admin shutdown
			return true
		}
	}
	msg := err.Error()
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") {
		return true
	}
	return false
}
