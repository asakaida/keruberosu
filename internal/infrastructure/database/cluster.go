package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/asakaida/keruberosu/internal/infrastructure/config"
)

// DBCluster manages primary and optional read replica database connections
// with write tracking for replica consistency.
type DBCluster struct {
	primary      *ResilientDB
	replica      *ResilientDB // nil if no replica configured
	writeTracker *WriteTracker
}

// NewDBCluster creates a new DBCluster from configuration.
// If no replica is configured, all reads go to the primary.
func NewDBCluster(cfg *config.DatabaseConfig) (*DBCluster, error) {
	primaryPg, err := newPostgresDB(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary: %w", err)
	}

	primary := NewResilientDB(primaryPg, DefaultRetryConfig())

	var replica *ResilientDB
	if cfg.HasReplica() {
		replicaPg, err := newPostgresDB(cfg.ReplicaConnectionString())
		if err != nil {
			primaryPg.Close()
			return nil, fmt.Errorf("failed to connect to replica: %w", err)
		}
		replica = NewResilientDB(replicaPg, DefaultRetryConfig())
	}

	writeTracker := NewWriteTracker(cfg.WriteTrackerWindowSeconds)

	return &DBCluster{
		primary:      primary,
		replica:      replica,
		writeTracker: writeTracker,
	}, nil
}

// Writer returns the primary database for write operations.
func (c *DBCluster) Writer() DBTX {
	return c.primary
}

// ReaderFor returns the appropriate database for read operations.
// Returns primary if no replica is configured or if the tenant had a recent write.
func (c *DBCluster) ReaderFor(tenantID string) DBTX {
	if c.replica == nil {
		return c.primary
	}
	if c.writeTracker.HasRecentWrite(tenantID) {
		return c.primary
	}
	return c.replica
}

// RecordWrite records a write for the given tenant.
func (c *DBCluster) RecordWrite(tenantID string) {
	c.writeTracker.RecordWrite(tenantID)
}

// PrimaryDB returns the underlying *sql.DB of the primary.
// Used for migrations, health checks, snapshot management, and transactions.
func (c *DBCluster) PrimaryDB() *sql.DB {
	return c.primary.DB()
}

// Start begins background processes (WriteTracker cleanup).
func (c *DBCluster) Start() {
	c.writeTracker.Start()
}

// Stop stops background processes.
func (c *DBCluster) Stop() {
	c.writeTracker.Stop()
}

// Close closes all database connections and stops background processes.
func (c *DBCluster) Close() error {
	c.writeTracker.Stop()
	var firstErr error
	if err := c.primary.DB().Close(); err != nil {
		firstErr = fmt.Errorf("failed to close primary: %w", err)
	}
	if c.replica != nil {
		if err := c.replica.DB().Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close replica: %w", err)
		}
	}
	return firstErr
}

// HealthCheck verifies all database connections are healthy.
func (c *DBCluster) HealthCheck() error {
	if err := c.primary.DB().Ping(); err != nil {
		return fmt.Errorf("primary health check failed: %w", err)
	}
	if c.replica != nil {
		if err := c.replica.DB().Ping(); err != nil {
			return fmt.Errorf("replica health check failed: %w", err)
		}
	}
	return nil
}

// newPostgresDB creates a *sql.DB with standard pool settings.
func newPostgresDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
