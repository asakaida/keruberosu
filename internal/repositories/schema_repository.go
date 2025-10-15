package repositories

import (
	"context"

	"github.com/asakaida/keruberosu/internal/entities"
)

// SchemaRepository defines the interface for schema data access
type SchemaRepository interface {
	// Create creates a new schema version for a tenant and returns the version ID
	Create(ctx context.Context, tenantID string, schemaDSL string) (string, error)

	// GetLatestVersion retrieves the latest schema version for a tenant
	GetLatestVersion(ctx context.Context, tenantID string) (*entities.Schema, error)

	// GetByVersion retrieves a specific schema version for a tenant
	GetByVersion(ctx context.Context, tenantID string, version string) (*entities.Schema, error)

	// Delete deletes all schemas for a tenant
	Delete(ctx context.Context, tenantID string) error

	// Deprecated: Use GetLatestVersion instead
	// GetByTenant retrieves the latest schema for a tenant (for backward compatibility)
	GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error)
}
