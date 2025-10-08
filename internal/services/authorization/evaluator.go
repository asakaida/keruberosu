package authorization

import (
	"context"
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

const (
	// MaxDepth is the maximum recursion depth for hierarchical permission evaluation
	MaxDepth = 100
)

// Evaluator evaluates permission rules
type Evaluator struct {
	schemaRepo    repositories.SchemaRepository
	relationRepo  repositories.RelationRepository
	attributeRepo repositories.AttributeRepository
	celEngine     *CELEngine
}

// EvaluationRequest contains all the context needed for rule evaluation
type EvaluationRequest struct {
	TenantID         string                    // Tenant ID
	EntityType       string                    // Resource entity type
	EntityID         string                    // Resource entity ID
	SubjectType      string                    // Subject type
	SubjectID        string                    // Subject ID
	ContextualTuples []*entities.RelationTuple // Temporary tuples for this request
	Depth            int                       // Current recursion depth
}

// NewEvaluator creates a new Evaluator
func NewEvaluator(
	schemaRepo repositories.SchemaRepository,
	relationRepo repositories.RelationRepository,
	attributeRepo repositories.AttributeRepository,
	celEngine *CELEngine,
) *Evaluator {
	return &Evaluator{
		schemaRepo:    schemaRepo,
		relationRepo:  relationRepo,
		attributeRepo: attributeRepo,
		celEngine:     celEngine,
	}
}

// EvaluateRule evaluates a permission rule and returns true if the subject has the permission
func (e *Evaluator) EvaluateRule(
	ctx context.Context,
	req *EvaluationRequest,
	rule entities.PermissionRule,
) (bool, error) {
	// Check depth limit
	if req.Depth > MaxDepth {
		return false, fmt.Errorf("maximum recursion depth exceeded (depth: %d)", req.Depth)
	}

	switch r := rule.(type) {
	case *entities.RelationRule:
		return e.evaluateRelation(ctx, req, r)
	case *entities.LogicalRule:
		return e.evaluateLogical(ctx, req, r)
	case *entities.HierarchicalRule:
		return e.evaluateHierarchical(ctx, req, r)
	case *entities.ABACRule:
		return e.evaluateABAC(ctx, req, r)
	default:
		return false, fmt.Errorf("unknown rule type: %T", rule)
	}
}

// evaluateRelation evaluates a relation-based rule
// Returns true if there exists a relation tuple matching the rule
func (e *Evaluator) evaluateRelation(
	ctx context.Context,
	req *EvaluationRequest,
	rule *entities.RelationRule,
) (bool, error) {
	// Check in contextual tuples first
	for _, tuple := range req.ContextualTuples {
		if tuple.EntityType == req.EntityType &&
			tuple.EntityID == req.EntityID &&
			tuple.Relation == rule.Relation &&
			tuple.SubjectType == req.SubjectType &&
			tuple.SubjectID == req.SubjectID {
			return true, nil
		}
	}

	// Check in database
	filter := &repositories.RelationFilter{
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		Relation:    rule.Relation,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
	}

	tuples, err := e.relationRepo.Read(ctx, req.TenantID, filter)
	if err != nil {
		return false, fmt.Errorf("failed to read relations: %w", err)
	}

	return len(tuples) > 0, nil
}

// evaluateLogical evaluates a logical operation (OR/AND/NOT)
func (e *Evaluator) evaluateLogical(
	ctx context.Context,
	req *EvaluationRequest,
	rule *entities.LogicalRule,
) (bool, error) {
	switch rule.Operator {
	case "or":
		// Evaluate left side
		leftResult, err := e.EvaluateRule(ctx, req, rule.Left)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate left side of OR: %w", err)
		}
		if leftResult {
			return true, nil // Short-circuit on true
		}

		// Evaluate right side
		rightResult, err := e.EvaluateRule(ctx, req, rule.Right)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate right side of OR: %w", err)
		}
		return rightResult, nil

	case "and":
		// Evaluate left side
		leftResult, err := e.EvaluateRule(ctx, req, rule.Left)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate left side of AND: %w", err)
		}
		if !leftResult {
			return false, nil // Short-circuit on false
		}

		// Evaluate right side
		rightResult, err := e.EvaluateRule(ctx, req, rule.Right)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate right side of AND: %w", err)
		}
		return rightResult, nil

	case "not":
		// Evaluate the expression
		result, err := e.EvaluateRule(ctx, req, rule.Left)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate NOT expression: %w", err)
		}
		return !result, nil

	default:
		return false, fmt.Errorf("unknown logical operator: %s", rule.Operator)
	}
}

// evaluateHierarchical evaluates a hierarchical permission check (e.g., parent.edit)
func (e *Evaluator) evaluateHierarchical(
	ctx context.Context,
	req *EvaluationRequest,
	rule *entities.HierarchicalRule,
) (bool, error) {
	// Get schema to find the relation's target type
	schema, err := e.schemaRepo.GetByTenant(ctx, req.TenantID)
	if err != nil {
		return false, fmt.Errorf("failed to get schema: %w", err)
	}

	entity := schema.GetEntity(req.EntityType)
	if entity == nil {
		return false, fmt.Errorf("entity type %s not found in schema", req.EntityType)
	}

	relation := entity.GetRelation(rule.Relation)
	if relation == nil {
		return false, fmt.Errorf("relation %s not found in entity %s", rule.Relation, req.EntityType)
	}

	// Get the parent entity(s) via the relation
	filter := &repositories.RelationFilter{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Relation:   rule.Relation,
	}

	tuples, err := e.relationRepo.Read(ctx, req.TenantID, filter)
	if err != nil {
		return false, fmt.Errorf("failed to read relations: %w", err)
	}

	// Check contextual tuples as well
	for _, ctxTuple := range req.ContextualTuples {
		if ctxTuple.EntityType == req.EntityType &&
			ctxTuple.EntityID == req.EntityID &&
			ctxTuple.Relation == rule.Relation {
			tuples = append(tuples, ctxTuple)
		}
	}

	// For each parent entity, check if the subject has the specified permission
	for _, tuple := range tuples {
		// Get the permission definition for the parent entity type
		parentPermission := schema.GetPermission(tuple.SubjectType, rule.Permission)
		if parentPermission == nil {
			continue // Permission not found in parent entity, try next
		}

		// Create a new request for the parent entity
		parentReq := &EvaluationRequest{
			TenantID:         req.TenantID,
			EntityType:       tuple.SubjectType,
			EntityID:         tuple.SubjectID,
			SubjectType:      req.SubjectType,
			SubjectID:        req.SubjectID,
			ContextualTuples: req.ContextualTuples,
			Depth:            req.Depth + 1, // Increment depth
		}

		// Recursively evaluate the parent permission
		result, err := e.EvaluateRule(ctx, parentReq, parentPermission.Rule)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate hierarchical permission: %w", err)
		}
		if result {
			return true, nil // Found at least one parent that grants permission
		}
	}

	return false, nil // No parent grants the permission
}

// evaluateABAC evaluates an ABAC rule using CEL
func (e *Evaluator) evaluateABAC(
	ctx context.Context,
	req *EvaluationRequest,
	rule *entities.ABACRule,
) (bool, error) {
	// Get resource attributes
	resourceAttrs, err := e.attributeRepo.Read(ctx, req.TenantID, req.EntityType, req.EntityID)
	if err != nil {
		return false, fmt.Errorf("failed to read resource attributes: %w", err)
	}

	// Get subject attributes
	subjectAttrs, err := e.attributeRepo.Read(ctx, req.TenantID, req.SubjectType, req.SubjectID)
	if err != nil {
		return false, fmt.Errorf("failed to read subject attributes: %w", err)
	}

	// Prepare CEL evaluation context
	evalContext := &EvaluationContext{
		Resource: resourceAttrs,
		Subject:  subjectAttrs,
		Request:  map[string]interface{}{}, // Can be extended with request metadata
	}

	// Evaluate the CEL expression
	result, err := e.celEngine.Evaluate(rule.Expression, evalContext)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate ABAC rule: %w", err)
	}

	return result, nil
}
