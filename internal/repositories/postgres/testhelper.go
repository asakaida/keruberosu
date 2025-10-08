package postgres

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/asakaida/keruberosu/internal/infrastructure/config"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	_ "github.com/lib/pq"
)

// SetupTestDB creates a test database connection and runs migrations
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Initialize test config
	if err := config.InitConfig("test"); err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	pg, err := database.NewPostgres(&cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := pg.RunMigrations("../../../internal/infrastructure/database/migrations/postgres"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return pg.DB
}

// CleanupTestDB closes the database connection and cleans up test data
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clean up all tables
	tables := []string{"attributes", "relations", "schemas"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: Failed to clean up table %s: %v", table, err)
		}
	}

	if err := db.Close(); err != nil {
		t.Logf("Warning: Failed to close database: %v", err)
	}
}
