package authorization

import (
	"context"
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// LookupInterface defines the interface for entity and subject lookup
type LookupInterface interface {
	LookupEntity(ctx context.Context, req *LookupEntityRequest) (*LookupEntityResponse, error)
	LookupSubject(ctx context.Context, req *LookupSubjectRequest) (*LookupSubjectResponse, error)
}

// Lookup provides entity and subject lookup functionality
type Lookup struct {
	checker       CheckerInterface
	schemaService SchemaServiceInterface
	relationRepo  repositories.RelationRepository
}

// LookupEntityRequest contains the parameters for looking up entities
type LookupEntityRequest struct {
	TenantID         string                    // Tenant ID
	SchemaVersion    string                    // Schema version (empty = latest)
	EntityType       string                    // Entity type to search for (e.g., "document")
	Permission       string                    // Permission to check (e.g., "view")
	SubjectType      string                    // Subject type (e.g., "user")
	SubjectID        string                    // Subject ID (e.g., "alice")
	ContextualTuples []*entities.RelationTuple // Temporary tuples for this request
	PageSize         int                       // Maximum number of results to return (0 = unlimited)
	PageToken        string                    // Pagination token for next page
}

// LookupEntityResponse contains the list of entities
type LookupEntityResponse struct {
	EntityIDs     []string // Entity IDs that the subject has permission on
	NextPageToken string   // Token for fetching the next page (empty if no more results)
}

// LookupSubjectRequest contains the parameters for looking up subjects
type LookupSubjectRequest struct {
	TenantID         string                    // Tenant ID
	SchemaVersion    string                    // Schema version (empty = latest)
	EntityType       string                    // Entity type (e.g., "document")
	EntityID         string                    // Entity ID (e.g., "doc1")
	Permission       string                    // Permission to check (e.g., "view")
	SubjectType      string                    // Subject type to search for (e.g., "user")
	ContextualTuples []*entities.RelationTuple // Temporary tuples for this request
	PageSize         int                       // Maximum number of results to return (0 = unlimited)
	PageToken        string                    // Pagination token for next page
}

// LookupSubjectResponse contains the list of subjects
type LookupSubjectResponse struct {
	SubjectIDs    []string // Subject IDs that have permission on the entity
	NextPageToken string   // Token for fetching the next page (empty if no more results)
}

// NewLookup creates a new Lookup
func NewLookup(
	checker CheckerInterface,
	schemaService SchemaServiceInterface,
	relationRepo repositories.RelationRepository,
) *Lookup {
	return &Lookup{
		checker:       checker,
		schemaService: schemaService,
		relationRepo:  relationRepo,
	}
}

// LookupEntity finds all entities of a given type that a subject has permission on
// This is a brute-force implementation for Phase 1 (cacheless)
func (l *Lookup) LookupEntity(ctx context.Context, req *LookupEntityRequest) (*LookupEntityResponse, error) {
	// Validate request
	if err := l.validateLookupEntityRequest(req); err != nil {
		return nil, fmt.Errorf("invalid lookup entity request: %w", err)
	}

	// Get parsed schema
	schema, err := l.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Verify entity type exists
	entity := schema.GetEntity(req.EntityType)
	if entity == nil {
		return nil, fmt.Errorf("entity type %s not found in schema", req.EntityType)
	}

	// Verify permission exists
	permission := entity.GetPermission(req.Permission)
	if permission == nil {
		return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
	}

	// Get all entities that could potentially have this permission
	// by querying all relation tuples that reference this entity type
	candidateIDs, err := l.getCandidateEntityIDs(ctx, req.TenantID, req.EntityType, req.PageToken, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate entities: %w", err)
	}

	// Check each candidate entity
	allowedIDs := make([]string, 0)
	for _, entityID := range candidateIDs {
		checkReq := &CheckRequest{
			TenantID:         req.TenantID,
			SchemaVersion:    req.SchemaVersion,
			EntityType:       req.EntityType,
			EntityID:         entityID,
			Permission:       req.Permission,
			SubjectType:      req.SubjectType,
			SubjectID:        req.SubjectID,
			ContextualTuples: req.ContextualTuples,
		}

		resp, err := l.checker.Check(ctx, checkReq)
		if err != nil {
			// Skip entities that cause errors
			continue
		}

		if resp.Allowed {
			allowedIDs = append(allowedIDs, entityID)

			// Check page size limit
			if req.PageSize > 0 && len(allowedIDs) >= req.PageSize {
				break
			}
		}
	}

	// Generate next page token if needed
	nextPageToken := ""
	if req.PageSize > 0 && len(allowedIDs) == req.PageSize {
		// There might be more results
		// In a real implementation, we'd use the last entityID as the token
		if len(candidateIDs) > 0 {
			nextPageToken = candidateIDs[len(candidateIDs)-1]
		}
	}

	return &LookupEntityResponse{
		EntityIDs:     allowedIDs,
		NextPageToken: nextPageToken,
	}, nil
}

// LookupSubject finds all subjects of a given type that have permission on an entity
// This is a brute-force implementation for Phase 1 (cacheless)
func (l *Lookup) LookupSubject(ctx context.Context, req *LookupSubjectRequest) (*LookupSubjectResponse, error) {
	// Validate request
	if err := l.validateLookupSubjectRequest(req); err != nil {
		return nil, fmt.Errorf("invalid lookup subject request: %w", err)
	}

	// Get parsed schema
	schema, err := l.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Verify entity type exists
	entity := schema.GetEntity(req.EntityType)
	if entity == nil {
		return nil, fmt.Errorf("entity type %s not found in schema", req.EntityType)
	}

	// Verify permission exists
	permission := entity.GetPermission(req.Permission)
	if permission == nil {
		return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
	}

	// Get all subjects that could potentially have this permission
	// by querying all relation tuples that reference this subject type
	candidateIDs, err := l.getCandidateSubjectIDs(ctx, req.TenantID, req.SubjectType, req.PageToken, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate subjects: %w", err)
	}

	// Check each candidate subject
	allowedIDs := make([]string, 0)
	for _, subjectID := range candidateIDs {
		checkReq := &CheckRequest{
			TenantID:         req.TenantID,
			SchemaVersion:    req.SchemaVersion,
			EntityType:       req.EntityType,
			EntityID:         req.EntityID,
			Permission:       req.Permission,
			SubjectType:      req.SubjectType,
			SubjectID:        subjectID,
			ContextualTuples: req.ContextualTuples,
		}

		resp, err := l.checker.Check(ctx, checkReq)
		if err != nil {
			// Skip subjects that cause errors
			continue
		}

		if resp.Allowed {
			allowedIDs = append(allowedIDs, subjectID)

			// Check page size limit
			if req.PageSize > 0 && len(allowedIDs) >= req.PageSize {
				break
			}
		}
	}

	// Generate next page token if needed
	nextPageToken := ""
	if req.PageSize > 0 && len(allowedIDs) == req.PageSize {
		// There might be more results
		// In a real implementation, we'd use the last subjectID as the token
		if len(candidateIDs) > 0 {
			nextPageToken = candidateIDs[len(candidateIDs)-1]
		}
	}

	return &LookupSubjectResponse{
		SubjectIDs:    allowedIDs,
		NextPageToken: nextPageToken,
	}, nil
}

// getCandidateEntityIDs gets all unique entity IDs of a given type from relation tuples
func (l *Lookup) getCandidateEntityIDs(ctx context.Context, tenantID, entityType, pageToken string, pageSize int) ([]string, error) {
	// Query all relation tuples for this entity type
	filter := &repositories.RelationFilter{
		EntityType: entityType,
	}

	tuples, err := l.relationRepo.Read(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	// Extract unique entity IDs
	entityIDSet := make(map[string]bool)
	for _, tuple := range tuples {
		entityIDSet[tuple.EntityID] = true
	}

	// Convert set to slice
	entityIDs := make([]string, 0, len(entityIDSet))
	for id := range entityIDSet {
		entityIDs = append(entityIDs, id)
	}

	// Apply pagination
	startIdx := 0
	if pageToken != "" {
		// Find the index of the page token
		for i, id := range entityIDs {
			if id == pageToken {
				startIdx = i + 1
				break
			}
		}
	}

	if startIdx >= len(entityIDs) {
		return []string{}, nil
	}

	endIdx := len(entityIDs)
	if pageSize > 0 && startIdx+pageSize < endIdx {
		endIdx = startIdx + pageSize
	}

	return entityIDs[startIdx:endIdx], nil
}

// getCandidateSubjectIDs gets all unique subject IDs of a given type from relation tuples
func (l *Lookup) getCandidateSubjectIDs(ctx context.Context, tenantID, subjectType, pageToken string, pageSize int) ([]string, error) {
	// Query all relation tuples for this subject type
	filter := &repositories.RelationFilter{
		SubjectType: subjectType,
	}

	tuples, err := l.relationRepo.Read(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	// Extract unique subject IDs
	subjectIDSet := make(map[string]bool)
	for _, tuple := range tuples {
		subjectIDSet[tuple.SubjectID] = true
	}

	// Convert set to slice
	subjectIDs := make([]string, 0, len(subjectIDSet))
	for id := range subjectIDSet {
		subjectIDs = append(subjectIDs, id)
	}

	// Apply pagination
	startIdx := 0
	if pageToken != "" {
		// Find the index of the page token
		for i, id := range subjectIDs {
			if id == pageToken {
				startIdx = i + 1
				break
			}
		}
	}

	if startIdx >= len(subjectIDs) {
		return []string{}, nil
	}

	endIdx := len(subjectIDs)
	if pageSize > 0 && startIdx+pageSize < endIdx {
		endIdx = startIdx + pageSize
	}

	return subjectIDs[startIdx:endIdx], nil
}

// validateLookupEntityRequest validates the lookup entity request
func (l *Lookup) validateLookupEntityRequest(req *LookupEntityRequest) error {
	if req.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	if req.EntityType == "" {
		return fmt.Errorf("entity type is required")
	}
	if req.Permission == "" {
		return fmt.Errorf("permission is required")
	}
	if req.SubjectType == "" {
		return fmt.Errorf("subject type is required")
	}
	if req.SubjectID == "" {
		return fmt.Errorf("subject ID is required")
	}
	return nil
}

// validateLookupSubjectRequest validates the lookup subject request
func (l *Lookup) validateLookupSubjectRequest(req *LookupSubjectRequest) error {
	if req.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	if req.EntityType == "" {
		return fmt.Errorf("entity type is required")
	}
	if req.EntityID == "" {
		return fmt.Errorf("entity ID is required")
	}
	if req.Permission == "" {
		return fmt.Errorf("permission is required")
	}
	if req.SubjectType == "" {
		return fmt.Errorf("subject type is required")
	}
	return nil
}
