package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// PostgresAttributeRepository implements AttributeRepository using PostgreSQL
type PostgresAttributeRepository struct {
	cluster *database.DBCluster
}

// NewPostgresAttributeRepository creates a new PostgreSQL attribute repository
func NewPostgresAttributeRepository(cluster *database.DBCluster) repositories.AttributeRepository {
	return &PostgresAttributeRepository{cluster: cluster}
}

// Write creates or updates an attribute
func (r *PostgresAttributeRepository) Write(ctx context.Context, tenantID string, attr *entities.Attribute) error {
	if err := attr.Validate(); err != nil {
		return fmt.Errorf("invalid attribute: %w", err)
	}

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
	_, err = r.cluster.Writer().ExecContext(ctx, query,
		tenantID, attr.EntityType, attr.EntityID, attr.Name, string(valueJSON), now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to write attribute: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return nil
}

// WriteInTx creates or updates an attribute within an existing transaction
func (r *PostgresAttributeRepository) WriteInTx(ctx context.Context, tx *sql.Tx, tenantID string, attr *entities.Attribute) error {
	if err := attr.Validate(); err != nil {
		return fmt.Errorf("invalid attribute: %w", err)
	}

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
	_, err = tx.ExecContext(ctx, query,
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
	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, tenantID, entityType, entityID)
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
		dec := json.NewDecoder(strings.NewReader(valueJSON))
		dec.UseNumber()
		if err := dec.Decode(&value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attribute value: %w", err)
		}

		attributes[name] = normalizeJSONValue(value)
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
	_, err := r.cluster.Writer().ExecContext(ctx, query, tenantID, entityType, entityID, attrName)
	if err != nil {
		return fmt.Errorf("failed to delete attribute: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return nil
}

// GetValue retrieves a specific attribute value for an entity
func (r *PostgresAttributeRepository) GetValue(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) (interface{}, error) {
	query := `
		SELECT value
		FROM attributes
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 AND attribute = $4
	`
	db := r.cluster.ReaderFor(tenantID)
	var valueJSON string
	err := db.QueryRowContext(ctx, query, tenantID, entityType, entityID, attrName).Scan(&valueJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attribute not found: %s", attrName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute value: %w", err)
	}

	var value interface{}
	dec := json.NewDecoder(strings.NewReader(valueJSON))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attribute value: %w", err)
	}

	return normalizeJSONValue(value), nil
}

// normalizeJSONValue converts json.Number to int64 or float64 recursively.
// This ensures callers receive standard Go types instead of json.Number,
// which is an implementation detail of UseNumber()-based JSON decoding.
func normalizeJSONValue(v interface{}) interface{} {
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return string(val)
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, item := range val {
			result[k] = normalizeJSONValue(item)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = normalizeJSONValue(item)
		}
		return result
	default:
		return v
	}
}

// ensure interface compliance
var _ repositories.AttributeRepository = (*PostgresAttributeRepository)(nil)
