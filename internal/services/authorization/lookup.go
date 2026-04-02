package authorization

import (
	"context"
	"fmt"
	"log"
	"sort"

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
	attributeRepo repositories.AttributeRepository
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

// NewLookup creates a new Lookup.
// attributeRepo is optional; if nil, entities accessible only through attributes
// will not appear in fallback lookup results.
func NewLookup(
	checker CheckerInterface,
	schemaService SchemaServiceInterface,
	relationRepo repositories.RelationRepository,
	attributeRepo ...repositories.AttributeRepository,
) *Lookup {
	l := &Lookup{
		checker:       checker,
		schemaService: schemaService,
		relationRepo:  relationRepo,
	}
	if len(attributeRepo) > 0 {
		l.attributeRepo = attributeRepo[0]
	}
	return l
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
		// Fall back to relation name (consistency with Check API)
		if entity.GetRelation(req.Permission) != nil {
			permission = &entities.Permission{
				Name: req.Permission,
				Rule: &entities.RelationRule{Relation: req.Permission},
			}
		} else {
			return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
		}
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

		// If contextual tuples present, merge candidates from contextual tuples
		// and verify each result with Check.
		// Filter contextual tuple IDs by page cursor to prevent duplicates across pages.
		if len(req.ContextualTuples) > 0 {
			ctxIDs := extractEntityIDsFromContextualTuples(req.ContextualTuples, req.EntityType)
			if req.PageToken != "" {
				ctxIDs = filterIDsAfterCursor(ctxIDs, req.PageToken)
			}
			merged := mergeSortedUnique(entityIDs, ctxIDs, len(entityIDs)+len(ctxIDs))
			return l.verifyEntitiesAndPaginate(ctx, req, merged, limit)
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
		// Fall back to relation name (consistency with Check API)
		if entity.GetRelation(req.Permission) != nil {
			permission = &entities.Permission{
				Name: req.Permission,
				Rule: &entities.RelationRule{Relation: req.Permission},
			}
		} else {
			return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
		}
	}

	limit := req.PageSize
	if limit <= 0 {
		limit = defaultLookupLimit
	}

	// Extract relations with schema context
	visited := make(map[string]bool)
	relations, parentRelations, hasUnresolvable := extractRelationsFromRuleWithContext(
		schema, req.EntityType, permission.Rule, visited)

	// When SubjectRelation is set, the optimized SQL path doesn't support
	// filtering by subject_relation, so always use the fallback Check loop.
	if req.SubjectRelation == "" && !hasUnresolvable && (len(relations) > 0 || len(parentRelations) > 0) {
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
			ctxIDs := extractSubjectIDsFromContextualTuples(req.ContextualTuples, req.EntityType, req.EntityID, req.SubjectType)
			if req.PageToken != "" {
				ctxIDs = filterIDsAfterCursor(ctxIDs, req.PageToken)
			}
			merged := mergeSortedUnique(subjectIDs, ctxIDs, len(subjectIDs)+len(ctxIDs))
			return l.verifySubjectsAndPaginate(ctx, req, merged, limit)
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

// lookupEntityFallback uses batched GetSortedEntityIDs + Check loop.
// Merges candidates from both relations and attributes tables to ensure
// entities accessible only through attributes (ABAC) are also found.
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
		candidates := l.getMergedEntityCandidates(ctx, req.TenantID, req.EntityType, cursor, batchSize)

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
				log.Printf("WARNING: Check failed for entity %s:%s: %v", req.EntityType, entityID, err)
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

// getMergedEntityCandidates returns sorted unique entity IDs from both
// relations and attributes tables.
func (l *Lookup) getMergedEntityCandidates(ctx context.Context, tenantID, entityType, cursor string, batchSize int) []string {
	relCandidates, err := l.relationRepo.GetSortedEntityIDs(ctx, tenantID, entityType, cursor, batchSize)
	if err != nil {
		log.Printf("WARNING: failed to get entity IDs from relations: %v", err)
		relCandidates = nil
	}

	if l.attributeRepo == nil {
		return relCandidates
	}

	attrCandidates, err := l.attributeRepo.GetSortedEntityIDs(ctx, tenantID, entityType, cursor, batchSize)
	if err != nil {
		log.Printf("WARNING: failed to get entity IDs from attributes: %v", err)
		attrCandidates = nil
	}

	return mergeSortedUnique(relCandidates, attrCandidates, batchSize)
}

// lookupSubjectFallback uses batched sorted subject IDs + Check loop.
// Merges candidates from both relations and attributes tables to ensure
// subjects accessible only through attributes (ABAC) are also found.
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
		candidates := l.getMergedSubjectCandidates(ctx, req.TenantID, req.SubjectType, cursor, batchSize)

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
				SubjectRelation:  req.SubjectRelation,
				ContextualTuples: req.ContextualTuples,
			})
			if err != nil {
				log.Printf("WARNING: Check failed for subject %s:%s: %v", req.SubjectType, subjectID, err)
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

// getMergedSubjectCandidates returns sorted unique subject IDs from both
// relations and attributes tables.
func (l *Lookup) getMergedSubjectCandidates(ctx context.Context, tenantID, subjectType, cursor string, batchSize int) []string {
	relCandidates, err := l.relationRepo.GetSortedSubjectIDs(ctx, tenantID, subjectType, cursor, batchSize)
	if err != nil {
		log.Printf("WARNING: failed to get subject IDs from relations: %v", err)
		relCandidates = nil
	}

	if l.attributeRepo == nil {
		return relCandidates
	}

	attrCandidates, err := l.attributeRepo.GetSortedEntityIDs(ctx, tenantID, subjectType, cursor, batchSize)
	if err != nil {
		log.Printf("WARNING: failed to get subject IDs from attributes: %v", err)
		attrCandidates = nil
	}

	return mergeSortedUnique(relCandidates, attrCandidates, batchSize)
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
			SubjectRelation:  req.SubjectRelation,
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
		// Check if the relation name refers to a permission (permission composition).
		// If so, recursively expand the referenced permission's rule.
		entity := schema.GetEntity(entityType)
		if entity != nil {
			isRelation := entity.GetRelation(r.Relation) != nil
			if !isRelation {
				if perm := entity.GetPermission(r.Relation); perm != nil {
					pr, pp, pu := extractRelationsFromRuleWithContext(schema, entityType, perm.Rule, visited)
					relations = append(relations, pr...)
					parentRelations = append(parentRelations, pp...)
					hasUnresolvable = hasUnresolvable || pu
					return
				}
			}
		}
		relations = append(relations, r.Relation)

	case *entities.LogicalRule:
		// Only "or" can be resolved via SQL UNION. "and" and "not" require
		// semantic evaluation (intersection/exclusion), so they must fall back
		// to the Check loop.
		if r.Operator == "and" || r.Operator == "not" {
			hasUnresolvable = true
			return
		}
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
		// Resolve the target entity types from the relation definition.
		// A relation may have multiple target types (e.g., "@folder @organization"),
		// so we expand each type's permission to collect all resolvable relations.
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

		// Extract all target types (e.g., "folder organization" → ["folder", "organization"])
		targetTypes := extractAllBaseTypes(relation.TargetType)
		if len(targetTypes) == 0 {
			hasUnresolvable = true
			return
		}

		resolvedAny := false
		for _, targetType := range targetTypes {
			// Cycle detection key
			key := targetType + "." + r.Permission
			if visited[key] {
				// Self-referential cycle (e.g., folder.view = parent.view where parent is folder)
				// The closure table handles transitive ancestry, so emit as parentRelation
				parentRelations = append(parentRelations, r.Relation+"."+r.Permission)
				resolvedAny = true
				continue
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
				for _, pr := range childParentRels {
					_ = pr
					hasUnresolvable = true
				}

				hasUnresolvable = hasUnresolvable || childUnresolvable
				resolvedAny = true
			} else {
				// Not a permission, check if it's a relation on the target entity
				targetEntity := schema.GetEntity(targetType)
				if targetEntity != nil && targetEntity.GetRelation(r.Permission) != nil {
					parentRelations = append(parentRelations, r.Relation+"."+r.Permission)
					resolvedAny = true
				}
				// If this target type doesn't have the permission/relation,
				// skip it (other target types may have it)
			}

			// Allow other paths to explore the same node
			delete(visited, key)
		}

		if !resolvedAny {
			hasUnresolvable = true
		}

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

// mergeSortedUnique merges two sorted string slices into a single sorted slice
// with unique values, limited to maxLen elements.
func mergeSortedUnique(a, b []string, maxLen int) []string {
	result := make([]string, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) && len(result) < maxLen {
		if a[i] < b[j] {
			result = append(result, a[i])
			i++
		} else if a[i] > b[j] {
			result = append(result, b[j])
			j++
		} else {
			result = append(result, a[i])
			i++
			j++
		}
	}
	for i < len(a) && len(result) < maxLen {
		result = append(result, a[i])
		i++
	}
	for j < len(b) && len(result) < maxLen {
		result = append(result, b[j])
		j++
	}
	return result
}

// extractEntityIDsFromContextualTuples extracts sorted unique entity IDs
// from contextual tuples matching the given entity type.
func extractEntityIDsFromContextualTuples(tuples []*entities.RelationTuple, entityType string) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, t := range tuples {
		if t.EntityType == entityType && !seen[t.EntityID] {
			seen[t.EntityID] = true
			ids = append(ids, t.EntityID)
		}
	}
	sort.Strings(ids)
	return ids
}

// extractSubjectIDsFromContextualTuples extracts sorted unique subject IDs
// from contextual tuples matching the given entity and subject type.
func extractSubjectIDsFromContextualTuples(tuples []*entities.RelationTuple, entityType, entityID, subjectType string) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, t := range tuples {
		if t.EntityType == entityType && t.EntityID == entityID && t.SubjectType == subjectType && !seen[t.SubjectID] {
			seen[t.SubjectID] = true
			ids = append(ids, t.SubjectID)
		}
	}
	sort.Strings(ids)
	return ids
}

// filterIDsAfterCursor returns only the IDs that are strictly greater than cursor.
// The input must be sorted in ascending order.
func filterIDsAfterCursor(ids []string, cursor string) []string {
	// Use binary search to find the insertion point
	idx := sort.SearchStrings(ids, cursor)
	// Skip the cursor value itself if it's present
	for idx < len(ids) && ids[idx] <= cursor {
		idx++
	}
	if idx >= len(ids) {
		return nil
	}
	return ids[idx:]
}

// extractBaseType is defined in evaluator.go - reused via package scope.
