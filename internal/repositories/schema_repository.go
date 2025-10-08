package repositories

import (
	"context"

	"github.com/asakaida/keruberosu/internal/entities"
)

// SchemaRepository defines the interface for schema data access
type SchemaRepository interface {
	// Create creates a new schema for a tenant
	Create(ctx context.Context, tenantID string, schemaDSL string) error

	// GetByTenant retrieves the schema for a tenant
	GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error)

	// Update updates the schema for a tenant
	Update(ctx context.Context, tenantID string, schemaDSL string) error

	// Delete deletes the schema for a tenant
	Delete(ctx context.Context, tenantID string) error
}
