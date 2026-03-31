package repositories

import (
	"context"
	"database/sql"

	"github.com/asakaida/keruberosu/internal/entities"
)

// AttributeRepository defines the interface for attribute data access
type AttributeRepository interface {
	// Write creates or updates an attribute
	Write(ctx context.Context, tenantID string, attr *entities.Attribute) error

	// Read retrieves all attributes for a specific entity
	// Returns a map of attribute name to value
	Read(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error)

	// Delete removes a specific attribute from an entity
	Delete(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) error

	// GetValue retrieves a specific attribute value for an entity
	GetValue(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) (interface{}, error)

	// WriteInTx creates or updates an attribute within an existing transaction
	WriteInTx(ctx context.Context, tx *sql.Tx, tenantID string, attr *entities.Attribute) error
}
