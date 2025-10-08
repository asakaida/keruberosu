package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// PostgresSchemaRepository implements SchemaRepository using PostgreSQL
type PostgresSchemaRepository struct {
	db *sql.DB
}

// NewPostgresSchemaRepository creates a new PostgreSQL schema repository
func NewPostgresSchemaRepository(db *sql.DB) repositories.SchemaRepository {
	return &PostgresSchemaRepository{db: db}
}

// Create creates a new schema for a tenant
func (r *PostgresSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) error {
	query := `
		INSERT INTO schemas (tenant_id, schema_dsl, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, tenantID, schemaDSL, now, now)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// GetByTenant retrieves the schema for a tenant
func (r *PostgresSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	query := `
		SELECT schema_dsl, created_at, updated_at
		FROM schemas
		WHERE tenant_id = $1
	`
	var schemaDSL string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&schemaDSL, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schema not found for tenant: %s", tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	schema := &entities.Schema{
		TenantID:  tenantID,
		DSL:       schemaDSL,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		// Note: Entities will be populated by the parser in the service layer
	}

	return schema, nil
}

// Update updates the schema for a tenant
func (r *PostgresSchemaRepository) Update(ctx context.Context, tenantID string, schemaDSL string) error {
	query := `
		UPDATE schemas
		SET schema_dsl = $1, updated_at = $2
		WHERE tenant_id = $3
	`
	result, err := r.db.ExecContext(ctx, query, schemaDSL, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update schema: %w", err)
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

// Delete deletes the schema for a tenant
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
