package authorization

import (
	"context"
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// ExpandNode represents a node in the permission tree
type ExpandNode struct {
	Type     string        // "union", "intersection", "exclusion", "leaf"
	Entity   string        // Entity reference (e.g., "document:doc1")
	Relation string        // Relation/permission name
	Subject  string        // Subject reference (e.g., "user:alice"), only for leaf nodes
	Children []*ExpandNode // Child nodes for logical operations
}

// Expander builds permission trees showing why permissions are granted
type Expander struct {
	schemaService SchemaServiceInterface
	relationRepo  repositories.RelationRepository
}

// ExpandRequest contains the parameters for expanding a permission tree
type ExpandRequest struct {
	TenantID   string // Tenant ID
	EntityType string // Resource entity type (e.g., "document")
	EntityID   string // Resource entity ID (e.g., "doc1")
	Permission string // Permission to expand (e.g., "view")
}

// ExpandResponse contains the resulting permission tree
type ExpandResponse struct {
	Tree *ExpandNode // Root node of the permission tree
}

// NewExpander creates a new Expander
func NewExpander(
	schemaService SchemaServiceInterface,
	relationRepo repositories.RelationRepository,
) *Expander {
	return &Expander{
		schemaService: schemaService,
		relationRepo:  relationRepo,
	}
}

// Expand builds a permission tree for the given resource and permission
// The tree shows all possible paths that could grant the permission
func (e *Expander) Expand(ctx context.Context, req *ExpandRequest) (*ExpandResponse, error) {
	// Validate request
	if err := e.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid expand request: %w", err)
	}

	// Get parsed schema
	schema, err := e.schemaService.GetSchemaEntity(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Get entity definition
	entity := schema.GetEntity(req.EntityType)
	if entity == nil {
		return nil, fmt.Errorf("entity type %s not found in schema", req.EntityType)
	}

	// Get permission definition
	permission := entity.GetPermission(req.Permission)
	if permission == nil {
		return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
	}

	// Build the tree
	entityRef := fmt.Sprintf("%s:%s", req.EntityType, req.EntityID)
	tree, err := e.expandRule(ctx, req.TenantID, schema, entityRef, permission.Rule, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to expand permission: %w", err)
	}

	return &ExpandResponse{
		Tree: tree,
	}, nil
}

// expandRule builds a tree node for the given rule
func (e *Expander) expandRule(
	ctx context.Context,
	tenantID string,
	schema *entities.Schema,
	entityRef string,
	rule entities.PermissionRule,
	depth int,
) (*ExpandNode, error) {
	// Check depth limit
	if depth > MaxDepth {
		return nil, fmt.Errorf("maximum recursion depth exceeded (depth: %d)", depth)
	}

	switch r := rule.(type) {
	case *entities.RelationRule:
		return e.expandRelation(ctx, tenantID, entityRef, r)

	case *entities.LogicalRule:
		return e.expandLogical(ctx, tenantID, schema, entityRef, r, depth)

	case *entities.HierarchicalRule:
		return e.expandHierarchical(ctx, tenantID, schema, entityRef, r, depth)

	case *entities.ABACRule:
		// ABAC rules can't be expanded into a tree since they depend on runtime attributes
		// Return a leaf node indicating ABAC evaluation
		return &ExpandNode{
			Type:     "leaf",
			Entity:   entityRef,
			Relation: "abac",
			Subject:  fmt.Sprintf("expression:%s", r.Expression),
		}, nil

	default:
		return nil, fmt.Errorf("unknown rule type: %T", rule)
	}
}

// expandRelation expands a relation-based rule by finding all subjects with that relation
func (e *Expander) expandRelation(
	ctx context.Context,
	tenantID string,
	entityRef string,
	rule *entities.RelationRule,
) (*ExpandNode, error) {
	// Parse entity reference
	entityType, entityID, err := parseEntityRef(entityRef)
	if err != nil {
		return nil, err
	}

	// Query all tuples with this relation
	filter := &repositories.RelationFilter{
		EntityType: entityType,
		EntityID:   entityID,
		Relation:   rule.Relation,
	}

	tuples, err := e.relationRepo.Read(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to read relations: %w", err)
	}

	// Create a union node with all subjects
	node := &ExpandNode{
		Type:     "union",
		Entity:   entityRef,
		Relation: rule.Relation,
		Children: make([]*ExpandNode, 0, len(tuples)),
	}

	for _, tuple := range tuples {
		subjectRef := fmt.Sprintf("%s:%s", tuple.SubjectType, tuple.SubjectID)
		node.Children = append(node.Children, &ExpandNode{
			Type:     "leaf",
			Entity:   entityRef,
			Relation: rule.Relation,
			Subject:  subjectRef,
		})
	}

	return node, nil
}

// expandLogical expands a logical operation (OR/AND/NOT)
func (e *Expander) expandLogical(
	ctx context.Context,
	tenantID string,
	schema *entities.Schema,
	entityRef string,
	rule *entities.LogicalRule,
	depth int,
) (*ExpandNode, error) {
	var nodeType string
	switch rule.Operator {
	case "or":
		nodeType = "union"
	case "and":
		nodeType = "intersection"
	case "not":
		nodeType = "exclusion"
	default:
		return nil, fmt.Errorf("unknown logical operator: %s", rule.Operator)
	}

	node := &ExpandNode{
		Type:     nodeType,
		Entity:   entityRef,
		Relation: rule.Operator,
		Children: make([]*ExpandNode, 0, 2),
	}

	// Expand left side
	leftNode, err := e.expandRule(ctx, tenantID, schema, entityRef, rule.Left, depth+1)
	if err != nil {
		return nil, fmt.Errorf("failed to expand left side of %s: %w", rule.Operator, err)
	}
	node.Children = append(node.Children, leftNode)

	// Expand right side (if exists)
	if rule.Right != nil {
		rightNode, err := e.expandRule(ctx, tenantID, schema, entityRef, rule.Right, depth+1)
		if err != nil {
			return nil, fmt.Errorf("failed to expand right side of %s: %w", rule.Operator, err)
		}
		node.Children = append(node.Children, rightNode)
	}

	return node, nil
}

// expandHierarchical expands a hierarchical permission (e.g., parent.view)
func (e *Expander) expandHierarchical(
	ctx context.Context,
	tenantID string,
	schema *entities.Schema,
	entityRef string,
	rule *entities.HierarchicalRule,
	depth int,
) (*ExpandNode, error) {
	// Parse entity reference
	entityType, entityID, err := parseEntityRef(entityRef)
	if err != nil {
		return nil, err
	}

	// Get parent entities via the relation
	filter := &repositories.RelationFilter{
		EntityType: entityType,
		EntityID:   entityID,
		Relation:   rule.Relation,
	}

	tuples, err := e.relationRepo.Read(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to read relations: %w", err)
	}

	// Create a union node with all parent permissions
	node := &ExpandNode{
		Type:     "union",
		Entity:   entityRef,
		Relation: fmt.Sprintf("%s.%s", rule.Relation, rule.Permission),
		Children: make([]*ExpandNode, 0, len(tuples)),
	}

	for _, tuple := range tuples {
		// Get the parent permission definition
		parentPermission := schema.GetPermission(tuple.SubjectType, rule.Permission)
		if parentPermission == nil {
			continue // Permission not found in parent entity, skip
		}

		// Recursively expand the parent permission
		parentRef := fmt.Sprintf("%s:%s", tuple.SubjectType, tuple.SubjectID)
		parentNode, err := e.expandRule(ctx, tenantID, schema, parentRef, parentPermission.Rule, depth+1)
		if err != nil {
			return nil, fmt.Errorf("failed to expand hierarchical permission: %w", err)
		}

		node.Children = append(node.Children, parentNode)
	}

	return node, nil
}

// validateRequest validates the expand request
func (e *Expander) validateRequest(req *ExpandRequest) error {
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
	return nil
}

// parseEntityRef parses an entity reference like "document:doc1" into type and ID
func parseEntityRef(ref string) (string, string, error) {
	for i := 0; i < len(ref); i++ {
		if ref[i] == ':' {
			if i == 0 || i == len(ref)-1 {
				return "", "", fmt.Errorf("invalid entity reference: %s", ref)
			}
			return ref[:i], ref[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid entity reference format: %s", ref)
}
