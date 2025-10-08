package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// PostgresAttributeRepository implements AttributeRepository using PostgreSQL
type PostgresAttributeRepository struct {
	db *sql.DB
}

// NewPostgresAttributeRepository creates a new PostgreSQL attribute repository
func NewPostgresAttributeRepository(db *sql.DB) repositories.AttributeRepository {
	return &PostgresAttributeRepository{db: db}
}

// Write creates or updates an attribute
func (r *PostgresAttributeRepository) Write(ctx context.Context, tenantID string, attr *entities.Attribute) error {
	if err := attr.Validate(); err != nil {
		return fmt.Errorf("invalid attribute: %w", err)
	}

	// Serialize value to JSON
	valueJSON, err := json.Marshal(attr.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal attribute value: %w", err)
	}

	query := `
		INSERT INTO attributes (tenant_id, entity_type, entity_id, attribute, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, entity_type, entity_id, attribute)
		DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	_, err = r.db.ExecContext(ctx, query,
		tenantID, attr.EntityType, attr.EntityID, attr.Name, string(valueJSON), now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to write attribute: %w", err)
	}

	return nil
}

// Read retrieves all attributes for a specific entity
func (r *PostgresAttributeRepository) Read(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error) {
	query := `
		SELECT attribute, value
		FROM attributes
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to read attributes: %w", err)
	}
	defer rows.Close()

	attributes := make(map[string]interface{})
	for rows.Next() {
		var name, valueJSON string
		if err := rows.Scan(&name, &valueJSON); err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}

		var value interface{}
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attribute value: %w", err)
		}

		attributes[name] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attributes: %w", err)
	}

	return attributes, nil
}

// Delete removes a specific attribute from an entity
func (r *PostgresAttributeRepository) Delete(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) error {
	query := `
		DELETE FROM attributes
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 AND attribute = $4
	`
	_, err := r.db.ExecContext(ctx, query, tenantID, entityType, entityID, attrName)
	if err != nil {
		return fmt.Errorf("failed to delete attribute: %w", err)
	}

	return nil
}

// GetValue retrieves a specific attribute value for an entity
func (r *PostgresAttributeRepository) GetValue(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) (interface{}, error) {
	query := `
		SELECT value
		FROM attributes
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 AND attribute = $4
	`
	var valueJSON string
	err := r.db.QueryRowContext(ctx, query, tenantID, entityType, entityID, attrName).Scan(&valueJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attribute not found: %s", attrName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute value: %w", err)
	}

	var value interface{}
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attribute value: %w", err)
	}

	return value, nil
}
