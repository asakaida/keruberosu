package repositories

import (
	"context"

	"github.com/asakaida/keruberosu/internal/entities"
)

// RelationFilter defines filter criteria for querying relations
type RelationFilter struct {
	EntityType      string   // Filter by entity type (optional)
	EntityID        string   // Filter by entity ID (optional)
	EntityIDs       []string // Filter by multiple entity IDs (Permify互換)
	Relation        string   // Filter by relation name (optional)
	SubjectType     string   // Filter by subject type (optional)
	SubjectID       string   // Filter by subject ID (optional)
	SubjectIDs      []string // Filter by multiple subject IDs (Permify互換)
	SubjectRelation string   // Filter by subject relation (optional)
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

	// DeleteByFilter removes relation tuples matching the filter (Permify互換)
	DeleteByFilter(ctx context.Context, tenantID string, filter *RelationFilter) error

	// ReadByFilter retrieves relation tuples matching filter with pagination (Permify互換)
	ReadByFilter(ctx context.Context, tenantID string, filter *RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error)

	// Exists checks if a specific relation tuple exists
	Exists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error)

	// ExistsWithSubjectRelation checks existence including subject relation
	ExistsWithSubjectRelation(ctx context.Context, tenantID string,
		entityType, entityID, relation, subjectType, subjectID, subjectRelation string) (bool, error)

	// FindByEntityWithRelation returns tuples for a specific entity and relation
	FindByEntityWithRelation(ctx context.Context, tenantID string,
		entityType, entityID, relation string) ([]*entities.RelationTuple, error)

	// LookupAncestorsViaRelation finds all ancestors via closure table
	LookupAncestorsViaRelation(ctx context.Context, tenantID string,
		entityType, entityID string, maxDepth int) ([]*entities.RelationTuple, error)

	// FindHierarchicalWithSubject checks if a subject exists in the hierarchy using recursive CTE
	FindHierarchicalWithSubject(ctx context.Context, tenantID string,
		entityType, entityID, relation, subjectType, subjectID string,
		maxDepth int) (bool, error)

	// RebuildClosure rebuilds the closure table for a tenant from scratch
	RebuildClosure(ctx context.Context, tenantID string) error
}
