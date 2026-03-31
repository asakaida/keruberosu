package authorization

import (
	"context"
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

const (
	defaultLookupLimit = 1000
	defaultBatchSize   = 300
	maxBatchSize       = 10000
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
	TenantID         string
	SchemaVersion    string
	EntityType       string
	Permission       string
	SubjectType      string
	SubjectID        string
	ContextualTuples []*entities.RelationTuple
	PageSize         int
	PageToken        string
}

// LookupEntityResponse contains the list of entities
type LookupEntityResponse struct {
	EntityIDs     []string
	NextPageToken string
}

// LookupSubjectRequest contains the parameters for looking up subjects
type LookupSubjectRequest struct {
	TenantID         string
	SchemaVersion    string
	EntityType       string
	EntityID         string
	Permission       string
	SubjectType      string
	SubjectRelation  string // optional: for computed userset lookups (e.g., "member")
	ContextualTuples []*entities.RelationTuple
	PageSize         int
	PageToken        string
}

// LookupSubjectResponse contains the list of subjects
type LookupSubjectResponse struct {
	SubjectIDs    []string
	NextPageToken string
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

// LookupEntity finds all entities of a given type that a subject has permission on.
// Uses optimized SQL lookup with closure table when possible, falling back to
// batched Check() loop for complex rules (ABAC/RuleCall).
func (l *Lookup) LookupEntity(ctx context.Context, req *LookupEntityRequest) (*LookupEntityResponse, error) {
	if err := l.validateLookupEntityRequest(req); err != nil {
		return nil, fmt.Errorf("invalid lookup entity request: %w", err)
	}

	schema, err := l.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	entity := schema.GetEntity(req.EntityType)
	if entity == nil {
		return nil, fmt.Errorf("entity type %s not found in schema", req.EntityType)
	}

	permission := entity.GetPermission(req.Permission)
	if permission == nil {
		return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
	}

	limit := req.PageSize
	if limit <= 0 {
		limit = defaultLookupLimit
	}

	// Extract relations with schema context for hierarchical expansion
	visited := make(map[string]bool)
	relations, parentRelations, hasUnresolvable := extractRelationsFromRuleWithContext(
		schema, req.EntityType, permission.Rule, visited)

	if !hasUnresolvable && (len(relations) > 0 || len(parentRelations) > 0) {
		// Optimized path: pure ReBAC with or without hierarchy
		entityIDs, err := l.relationRepo.LookupAccessibleEntitiesComplex(
			ctx, req.TenantID,
			req.EntityType, relations, parentRelations,
			req.SubjectType, req.SubjectID,
			MaxDepth, req.PageToken, limit+1)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup accessible entities: %w", err)
		}

		// If contextual tuples present, verify each result with Check
		if len(req.ContextualTuples) > 0 {
			return l.verifyEntitiesAndPaginate(ctx, req, entityIDs, limit)
		}

		nextPageToken := ""
		if len(entityIDs) > limit {
			entityIDs = entityIDs[:limit]
			nextPageToken = entityIDs[len(entityIDs)-1]
		}

		return &LookupEntityResponse{
			EntityIDs:     entityIDs,
			NextPageToken: nextPageToken,
		}, nil
	}

	// Fallback path: batched sorted entity IDs + Check loop
	return l.lookupEntityFallback(ctx, req, limit)
}

// LookupSubject finds all subjects of a given type that have permission on an entity.
// Uses optimized SQL lookup with closure table when possible.
func (l *Lookup) LookupSubject(ctx context.Context, req *LookupSubjectRequest) (*LookupSubjectResponse, error) {
	if err := l.validateLookupSubjectRequest(req); err != nil {
		return nil, fmt.Errorf("invalid lookup subject request: %w", err)
	}

	schema, err := l.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	entity := schema.GetEntity(req.EntityType)
	if entity == nil {
		return nil, fmt.Errorf("entity type %s not found in schema", req.EntityType)
	}

	permission := entity.GetPermission(req.Permission)
	if permission == nil {
		return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
	}

	limit := req.PageSize
	if limit <= 0 {
		limit = defaultLookupLimit
	}

	// Extract relations with schema context
	visited := make(map[string]bool)
	relations, parentRelations, hasUnresolvable := extractRelationsFromRuleWithContext(
		schema, req.EntityType, permission.Rule, visited)

	if !hasUnresolvable && (len(relations) > 0 || len(parentRelations) > 0) {
		// Optimized path
		subjectIDs, err := l.relationRepo.LookupAccessibleSubjectsComplex(
			ctx, req.TenantID,
			req.EntityType, req.EntityID, relations, parentRelations,
			req.SubjectType,
			MaxDepth, req.PageToken, limit+1)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup accessible subjects: %w", err)
		}

		if len(req.ContextualTuples) > 0 {
			return l.verifySubjectsAndPaginate(ctx, req, subjectIDs, limit)
		}

		nextPageToken := ""
		if len(subjectIDs) > limit {
			subjectIDs = subjectIDs[:limit]
			nextPageToken = subjectIDs[len(subjectIDs)-1]
		}

		return &LookupSubjectResponse{
			SubjectIDs:    subjectIDs,
			NextPageToken: nextPageToken,
		}, nil
	}

	// Fallback path
	return l.lookupSubjectFallback(ctx, req, limit)
}

// lookupEntityFallback uses batched GetSortedEntityIDs + Check loop
func (l *Lookup) lookupEntityFallback(ctx context.Context, req *LookupEntityRequest, limit int) (*LookupEntityResponse, error) {
	batchSize := limit * 3
	if batchSize < defaultBatchSize {
		batchSize = defaultBatchSize
	}
	if batchSize > maxBatchSize {
		batchSize = maxBatchSize
	}

	cursor := req.PageToken
	var allowedIDs []string

	for {
		candidates, err := l.relationRepo.GetSortedEntityIDs(ctx, req.TenantID, req.EntityType, cursor, batchSize)
		if err != nil {
			return nil, fmt.Errorf("failed to get sorted entity IDs: %w", err)
		}

		if len(candidates) == 0 {
			break
		}

		var lastCandidate string
		for _, entityID := range candidates {
			lastCandidate = entityID
			resp, err := l.checker.Check(ctx, &CheckRequest{
				TenantID:         req.TenantID,
				SchemaVersion:    req.SchemaVersion,
				EntityType:       req.EntityType,
				EntityID:         entityID,
				Permission:       req.Permission,
				SubjectType:      req.SubjectType,
				SubjectID:        req.SubjectID,
				ContextualTuples: req.ContextualTuples,
			})
			if err != nil {
				continue
			}
			if resp.Allowed {
				allowedIDs = append(allowedIDs, entityID)
				if len(allowedIDs) >= limit {
					break
				}
			}
		}

		cursor = lastCandidate

		if len(allowedIDs) >= limit {
			break
		}

		if len(candidates) < batchSize {
			break
		}
	}

	nextPageToken := ""
	if len(allowedIDs) >= limit {
		nextPageToken = cursor
	}

	return &LookupEntityResponse{
		EntityIDs:     allowedIDs,
		NextPageToken: nextPageToken,
	}, nil
}

// lookupSubjectFallback uses batched GetSortedSubjectIDs + Check loop
func (l *Lookup) lookupSubjectFallback(ctx context.Context, req *LookupSubjectRequest, limit int) (*LookupSubjectResponse, error) {
	batchSize := limit * 3
	if batchSize < defaultBatchSize {
		batchSize = defaultBatchSize
	}
	if batchSize > maxBatchSize {
		batchSize = maxBatchSize
	}

	cursor := req.PageToken
	var allowedIDs []string

	for {
		candidates, err := l.relationRepo.GetSortedSubjectIDs(ctx, req.TenantID, req.SubjectType, cursor, batchSize)
		if err != nil {
			return nil, fmt.Errorf("failed to get sorted subject IDs: %w", err)
		}

		if len(candidates) == 0 {
			break
		}

		var lastCandidate string
		for _, subjectID := range candidates {
			lastCandidate = subjectID
			resp, err := l.checker.Check(ctx, &CheckRequest{
				TenantID:         req.TenantID,
				SchemaVersion:    req.SchemaVersion,
				EntityType:       req.EntityType,
				EntityID:         req.EntityID,
				Permission:       req.Permission,
				SubjectType:      req.SubjectType,
				SubjectID:        subjectID,
				ContextualTuples: req.ContextualTuples,
			})
			if err != nil {
				continue
			}
			if resp.Allowed {
				allowedIDs = append(allowedIDs, subjectID)
				if len(allowedIDs) >= limit {
					break
				}
			}
		}

		cursor = lastCandidate

		if len(allowedIDs) >= limit {
			break
		}

		if len(candidates) < batchSize {
			break
		}
	}

	nextPageToken := ""
	if len(allowedIDs) >= limit {
		nextPageToken = cursor
	}

	return &LookupSubjectResponse{
		SubjectIDs:    allowedIDs,
		NextPageToken: nextPageToken,
	}, nil
}

// verifyEntitiesAndPaginate verifies candidate entity IDs with Check and applies pagination
func (l *Lookup) verifyEntitiesAndPaginate(ctx context.Context, req *LookupEntityRequest, candidates []string, limit int) (*LookupEntityResponse, error) {
	var allowedIDs []string
	for _, entityID := range candidates {
		resp, err := l.checker.Check(ctx, &CheckRequest{
			TenantID:         req.TenantID,
			SchemaVersion:    req.SchemaVersion,
			EntityType:       req.EntityType,
			EntityID:         entityID,
			Permission:       req.Permission,
			SubjectType:      req.SubjectType,
			SubjectID:        req.SubjectID,
			ContextualTuples: req.ContextualTuples,
		})
		if err != nil {
			continue
		}
		if resp.Allowed {
			allowedIDs = append(allowedIDs, entityID)
			if len(allowedIDs) >= limit {
				break
			}
		}
	}

	nextPageToken := ""
	if len(allowedIDs) >= limit {
		nextPageToken = allowedIDs[len(allowedIDs)-1]
	}

	return &LookupEntityResponse{
		EntityIDs:     allowedIDs,
		NextPageToken: nextPageToken,
	}, nil
}

// verifySubjectsAndPaginate verifies candidate subject IDs with Check and applies pagination
func (l *Lookup) verifySubjectsAndPaginate(ctx context.Context, req *LookupSubjectRequest, candidates []string, limit int) (*LookupSubjectResponse, error) {
	var allowedIDs []string
	for _, subjectID := range candidates {
		resp, err := l.checker.Check(ctx, &CheckRequest{
			TenantID:         req.TenantID,
			SchemaVersion:    req.SchemaVersion,
			EntityType:       req.EntityType,
			EntityID:         req.EntityID,
			Permission:       req.Permission,
			SubjectType:      req.SubjectType,
			SubjectID:        subjectID,
			ContextualTuples: req.ContextualTuples,
		})
		if err != nil {
			continue
		}
		if resp.Allowed {
			allowedIDs = append(allowedIDs, subjectID)
			if len(allowedIDs) >= limit {
				break
			}
		}
	}

	nextPageToken := ""
	if len(allowedIDs) >= limit {
		nextPageToken = allowedIDs[len(allowedIDs)-1]
	}

	return &LookupSubjectResponse{
		SubjectIDs:    allowedIDs,
		NextPageToken: nextPageToken,
	}, nil
}

// extractRelationsFromRuleWithContext recursively extracts relation names from a
// permission rule, using schema context to expand hierarchical rules into concrete
// parent relation paths that the SQL layer can handle via closure table.
//
// Returns:
//   - relations: direct relation names (e.g., ["owner", "editor"])
//   - parentRelations: hierarchical paths (e.g., ["parent.owner", "parent.editor"])
//   - hasUnresolvable: true if ABAC/RuleCall/HierarchicalRuleCall prevents full expansion
func extractRelationsFromRuleWithContext(
	schema *entities.Schema,
	entityType string,
	rule entities.PermissionRule,
	visited map[string]bool,
) (relations []string, parentRelations []string, hasUnresolvable bool) {
	switch r := rule.(type) {
	case *entities.RelationRule:
		relations = append(relations, r.Relation)

	case *entities.LogicalRule:
		lr, lp, lu := extractRelationsFromRuleWithContext(schema, entityType, r.Left, visited)
		relations = append(relations, lr...)
		parentRelations = append(parentRelations, lp...)
		hasUnresolvable = hasUnresolvable || lu
		if r.Right != nil {
			rr, rp, ru := extractRelationsFromRuleWithContext(schema, entityType, r.Right, visited)
			relations = append(relations, rr...)
			parentRelations = append(parentRelations, rp...)
			hasUnresolvable = hasUnresolvable || ru
		}

	case *entities.HierarchicalRule:
		// Resolve the target entity type from the relation definition
		entity := schema.GetEntity(entityType)
		if entity == nil {
			hasUnresolvable = true
			return
		}
		relation := entity.GetRelation(r.Relation)
		if relation == nil {
			hasUnresolvable = true
			return
		}

		targetType := extractBaseType(relation.TargetType)

		// Cycle detection key
		key := targetType + "." + r.Permission
		if visited[key] {
			// Self-referential cycle (e.g., folder.view = parent.view where parent is folder)
			// The closure table handles transitive ancestry, so emit as parentRelation
			parentRelations = append(parentRelations, r.Relation+"."+r.Permission)
			return
		}
		visited[key] = true

		// Look up the permission on the target entity type
		targetPermission := schema.GetPermission(targetType, r.Permission)
		if targetPermission != nil {
			// Recursively expand the target permission
			childRels, childParentRels, childUnresolvable := extractRelationsFromRuleWithContext(
				schema, targetType, targetPermission.Rule, visited)

			// Direct relations on the target become parent relations for us
			for _, rel := range childRels {
				parentRelations = append(parentRelations, r.Relation+"."+rel)
			}

			// Multi-hop parent relations (target has its own hierarchical rules)
			// These require multi-level closure traversal which our SQL supports
			// since closure table stores transitive ancestors
			for _, pr := range childParentRels {
				// pr is like "parent.owner" on the target entity
				// We can't directly use multi-hop in a single closure query,
				// so fall back for nested hierarchies beyond one level
				_ = pr
				hasUnresolvable = true
			}

			hasUnresolvable = hasUnresolvable || childUnresolvable
		} else {
			// Not a permission, check if it's a relation on the target entity
			targetEntity := schema.GetEntity(targetType)
			if targetEntity != nil && targetEntity.GetRelation(r.Permission) != nil {
				parentRelations = append(parentRelations, r.Relation+"."+r.Permission)
			} else {
				hasUnresolvable = true
			}
		}

		// Allow other paths to explore the same node
		delete(visited, key)

	case *entities.ABACRule:
		hasUnresolvable = true
	case *entities.RuleCallRule:
		hasUnresolvable = true
	case *entities.HierarchicalRuleCallRule:
		hasUnresolvable = true
	}

	return
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

// extractBaseType is defined in evaluator.go - reused via package scope.
