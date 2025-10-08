package repositories

import (
	"context"

	"github.com/asakaida/keruberosu/internal/entities"
)

// RelationFilter defines filter criteria for querying relations
type RelationFilter struct {
	EntityType      string // Filter by entity type (optional)
	EntityID        string // Filter by entity ID (optional)
	Relation        string // Filter by relation name (optional)
	SubjectType     string // Filter by subject type (optional)
	SubjectID       string // Filter by subject ID (optional)
	SubjectRelation string // Filter by subject relation (optional)
}

// RelationRepository defines the interface for relation data access
type RelationRepository interface {
	// Write creates a new relation tuple
	Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error

	// Delete removes a relation tuple
	Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error

	// Read retrieves relation tuples matching the filter
	Read(ctx context.Context, tenantID string, filter *RelationFilter) ([]*entities.RelationTuple, error)

	// CheckExists checks if a specific relation tuple exists
	CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error)

	// BatchWrite creates multiple relation tuples in a single transaction
	BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error

	// BatchDelete removes multiple relation tuples in a single transaction
	BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
}
