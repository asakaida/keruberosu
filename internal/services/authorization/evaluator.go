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

// SchemaServiceInterface defines the interface for schema operations
// This interface is defined here to avoid circular dependency
type SchemaServiceInterface interface {
	GetSchemaEntity(ctx context.Context, tenantID string, version string) (*entities.Schema, error)
}

// Evaluator evaluates permission rules
type Evaluator struct {
	schemaService SchemaServiceInterface
	relationRepo  repositories.RelationRepository
	attributeRepo repositories.AttributeRepository
	celEngine     *CELEngine
}

// EvaluationRequest contains all the context needed for rule evaluation
type EvaluationRequest struct {
	TenantID         string                    // Tenant ID
	SchemaVersion    string                    // Schema version (empty = latest)
	EntityType       string                    // Resource entity type
	EntityID         string                    // Resource entity ID
	SubjectType      string                    // Subject type
	SubjectID        string                    // Subject ID
	ContextualTuples []*entities.RelationTuple // Temporary tuples for this request
	Depth            int                       // Current recursion depth
}

// NewEvaluator creates a new Evaluator
func NewEvaluator(
	schemaService SchemaServiceInterface,
	relationRepo repositories.RelationRepository,
	attributeRepo repositories.AttributeRepository,
	celEngine *CELEngine,
) *Evaluator {
	return &Evaluator{
		schemaService: schemaService,
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
	case *entities.RuleCallRule:
		return e.evaluateRuleCall(ctx, req, r)
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
	// Check in contextual tuples first for direct match
	for _, tuple := range req.ContextualTuples {
		if tuple.EntityType == req.EntityType &&
			tuple.EntityID == req.EntityID &&
			tuple.Relation == rule.Relation &&
			tuple.SubjectType == req.SubjectType &&
			tuple.SubjectID == req.SubjectID &&
			tuple.SubjectRelation == "" {
			return true, nil
		}
	}

	// Check in database for direct match
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

	if len(tuples) > 0 {
		return true, nil
	}

	// Check for subject relations (e.g., team:backend-team#member)
	// Get all tuples for this entity and relation, regardless of subject
	allTuplesFilter := &repositories.RelationFilter{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Relation:   rule.Relation,
	}

	allTuples, err := e.relationRepo.Read(ctx, req.TenantID, allTuplesFilter)
	if err != nil {
		return false, fmt.Errorf("failed to read all relations: %w", err)
	}

	// Add contextual tuples
	allTuples = append(allTuples, req.ContextualTuples...)

	// Check each tuple for subject relations
	for _, tuple := range allTuples {
		// Skip if not matching entity/relation
		if tuple.EntityType != req.EntityType || tuple.EntityID != req.EntityID || tuple.Relation != rule.Relation {
			continue
		}

		// If this tuple has a subject relation, expand it
		if tuple.SubjectRelation != "" {
			// Check if the current subject has the subject relation
			// Example: tuple is "repository:backend-api#contributor@team:backend-team#member"
			// We need to check if "user:frank" has relation "member" with "team:backend-team"
			subjectRelationFilter := &repositories.RelationFilter{
				EntityType:  tuple.SubjectType,
				EntityID:    tuple.SubjectID,
				Relation:    tuple.SubjectRelation,
				SubjectType: req.SubjectType,
				SubjectID:   req.SubjectID,
			}

			subjectTuples, err := e.relationRepo.Read(ctx, req.TenantID, subjectRelationFilter)
			if err != nil {
				return false, fmt.Errorf("failed to read subject relations: %w", err)
			}

			// Check contextual tuples as well
			for _, ctxTuple := range req.ContextualTuples {
				if ctxTuple.EntityType == tuple.SubjectType &&
					ctxTuple.EntityID == tuple.SubjectID &&
					ctxTuple.Relation == tuple.SubjectRelation &&
					ctxTuple.SubjectType == req.SubjectType &&
					ctxTuple.SubjectID == req.SubjectID &&
					ctxTuple.SubjectRelation == "" {
					subjectTuples = append(subjectTuples, ctxTuple)
				}
			}

			if len(subjectTuples) > 0 {
				return true, nil // Subject has the relation via subject relation
			}
		}
	}

	return false, nil
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
	// Get parsed schema to find the relation's target type
	schema, err := e.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
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

	// For each parent entity, check if the subject has the specified permission or relation
	for _, tuple := range tuples {
		// First, check if it's a permission
		parentPermission := schema.GetPermission(tuple.SubjectType, rule.Permission)
		if parentPermission != nil {
			// Create a new request for the parent entity
			parentReq := &EvaluationRequest{
				TenantID:         req.TenantID,
				SchemaVersion:    req.SchemaVersion,
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
		} else {
			// If not a permission, check if it's a relation (Permify compatibility)
			parentEntity := schema.GetEntity(tuple.SubjectType)
			if parentEntity != nil && parentEntity.GetRelation(rule.Permission) != nil {
				// Check if the subject has this relation to the parent entity
				relationFilter := &repositories.RelationFilter{
					EntityType:  tuple.SubjectType,
					EntityID:    tuple.SubjectID,
					Relation:    rule.Permission,
					SubjectType: req.SubjectType,
					SubjectID:   req.SubjectID,
				}

				relationTuples, err := e.relationRepo.Read(ctx, req.TenantID, relationFilter)
				if err != nil {
					return false, fmt.Errorf("failed to read parent relations: %w", err)
				}

				// Check contextual tuples as well
				for _, ctxTuple := range req.ContextualTuples {
					if ctxTuple.EntityType == tuple.SubjectType &&
						ctxTuple.EntityID == tuple.SubjectID &&
						ctxTuple.Relation == rule.Permission &&
						ctxTuple.SubjectType == req.SubjectType &&
						ctxTuple.SubjectID == req.SubjectID {
						relationTuples = append(relationTuples, ctxTuple)
					}
				}

				if len(relationTuples) > 0 {
					return true, nil // Subject has the relation to parent
				}
			}
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

// evaluateRuleCall evaluates a rule call by looking up the rule definition and evaluating it
func (e *Evaluator) evaluateRuleCall(
	ctx context.Context,
	req *EvaluationRequest,
	rule *entities.RuleCallRule,
) (bool, error) {
	// Get schema to access rule definitions
	schema, err := e.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return false, fmt.Errorf("failed to get schema: %w", err)
	}

	// Find the rule definition
	ruleDef := schema.GetRule(rule.RuleName)
	if ruleDef == nil {
		return false, fmt.Errorf("rule %s not found", rule.RuleName)
	}

	// Validate argument count
	if len(rule.Arguments) != len(ruleDef.Parameters) {
		return false, fmt.Errorf("rule %s expects %d arguments, got %d",
			rule.RuleName, len(ruleDef.Parameters), len(rule.Arguments))
	}

	// The rule body is a CEL expression that uses parameter names
	// The arguments specify which context variables to use (e.g., "resource", "subject")
	// We evaluate the rule body using the same context as ABAC rules

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
	// The rule body uses parameter names (e.g., "resource.public")
	// The arguments tell us which variables to map those parameters to
	evalContext := &EvaluationContext{
		Resource: resourceAttrs,
		Subject:  subjectAttrs,
		Request:  map[string]interface{}{}, // Can be extended with request metadata
	}

	// Note: For now, we assume the standard mapping where:
	// - "resource" parameter maps to resource attributes
	// - "subject" parameter maps to subject attributes
	// In the future, we could support more flexible parameter-to-context mapping

	// Evaluate the CEL expression from the rule body
	result, err := e.celEngine.Evaluate(ruleDef.Body, evalContext)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate rule %s: %w", rule.RuleName, err)
	}

	return result, nil
}
