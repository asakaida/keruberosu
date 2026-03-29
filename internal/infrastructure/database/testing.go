package database

import "database/sql"

// NewSingleNodeCluster creates a DBCluster wrapping a single *sql.DB (no replica).
// Intended for testing only.
func NewSingleNodeCluster(db *sql.DB) *DBCluster {
	resilient := NewResilientDB(db, DefaultRetryConfig())
	return &DBCluster{
		primary:      resilient,
		replica:      nil,
		writeTracker: NewWriteTracker(1),
	}
}
