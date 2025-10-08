package database

import (
	"testing"

	"github.com/asakaida/keruberosu/internal/infrastructure/config"
)

func TestPostgres_Close(t *testing.T) {
	tests := []struct {
		name    string
		pg      *Postgres
		wantErr bool
	}{
		{
			name:    "nil DB",
			pg:      &Postgres{DB: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pg.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("Postgres.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewPostgres_InvalidConfig(t *testing.T) {
	// Test with invalid configuration that should fail to connect
	cfg := &config.DatabaseConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     99999,
		User:     "invalid",
		Password: "invalid",
		Database: "invalid",
		SSLMode:  "disable",
	}

	pg, err := NewPostgres(cfg)
	if err == nil {
		if pg != nil && pg.DB != nil {
			pg.Close()
		}
		t.Error("NewPostgres() with invalid config should return error")
	}
}

func TestDatabaseConfig_Integration(t *testing.T) {
	// This is an integration test that requires a real database
	// It will only run if DB_PASSWORD is set
	// Skip if not running in integration test mode
	t.Skip("Integration test - requires running database")

	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     25432,
		User:     "keruberosu",
		Password: "keruberosu_test_password",
		Database: "keruberosu_test",
		SSLMode:  "disable",
	}

	pg, err := NewPostgres(cfg)
	if err != nil {
		t.Fatalf("NewPostgres() error = %v", err)
	}
	defer pg.Close()

	// Test HealthCheck
	if err := pg.HealthCheck(); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}

	// Test Close
	if err := pg.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should also work
	if err := pg.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}
