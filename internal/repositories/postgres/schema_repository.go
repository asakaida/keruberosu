package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/oklog/ulid/v2"
)

// PostgresSchemaRepository implements SchemaRepository using PostgreSQL
type PostgresSchemaRepository struct {
	db *sql.DB
}

// NewPostgresSchemaRepository creates a new PostgreSQL schema repository
func NewPostgresSchemaRepository(db *sql.DB) repositories.SchemaRepository {
	return &PostgresSchemaRepository{db: db}
}

// Create creates a new schema version for a tenant and returns the version ID
func (r *PostgresSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
	// Generate ULID for version
	version := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()

	query := `
		INSERT INTO schemas (tenant_id, version, schema_dsl, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, tenantID, version, schemaDSL, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to create schema: %w", err)
	}
	return version, nil
}

// GetLatestVersion retrieves the latest schema version for a tenant
func (r *PostgresSchemaRepository) GetLatestVersion(ctx context.Context, tenantID string) (*entities.Schema, error) {
	query := `
		SELECT version, schema_dsl, created_at, updated_at
		FROM schemas
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var version, schemaDSL string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&version, &schemaDSL, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schema not found for tenant %s: %w", tenantID, repositories.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	schema := &entities.Schema{
		TenantID:  tenantID,
		Version:   version,
		DSL:       schemaDSL,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		// Note: Entities will be populated by the parser in the service layer
	}

	return schema, nil
}

// GetByVersion retrieves a specific schema version for a tenant
func (r *PostgresSchemaRepository) GetByVersion(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
	query := `
		SELECT version, schema_dsl, created_at, updated_at
		FROM schemas
		WHERE tenant_id = $1 AND version = $2
	`
	var versionOut, schemaDSL string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, tenantID, version).Scan(&versionOut, &schemaDSL, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schema version %s not found for tenant %s: %w", version, tenantID, repositories.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	schema := &entities.Schema{
		TenantID:  tenantID,
		Version:   versionOut,
		DSL:       schemaDSL,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		// Note: Entities will be populated by the parser in the service layer
	}

	return schema, nil
}

// GetByTenant retrieves the latest schema for a tenant (backward compatibility)
func (r *PostgresSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	return r.GetLatestVersion(ctx, tenantID)
}

// Delete deletes all schemas for a tenant
func (r *PostgresSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	query := `DELETE FROM schemas WHERE tenant_id = $1`
	result, err := r.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("schema not found for tenant: %s", tenantID)
	}

	return nil
}
