package authorization

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// extractBaseType extracts the base entity type from a relation target type.
// Handles formats like "folder", "folder#member", "folder user" → returns first type.
func extractBaseType(targetType string) string {
	// Split by space first (multiple types like "folder user")
	parts := strings.Fields(targetType)
	if len(parts) > 0 {
		targetType = parts[0]
	}
	// Strip #relation suffix (like "folder#member")
	if idx := strings.Index(targetType, "#"); idx >= 0 {
		return targetType[:idx]
	}
	return targetType
}

const (
	// MaxDepth is the maximum recursion depth for hierarchical permission evaluation
	MaxDepth = 100
	// MaxTuplesPerQuery is the maximum number of tuples returned per query
	MaxTuplesPerQuery = 10000
	// MaxUsersetDepth is the maximum recursion depth for computed userset expansion in SQL
	MaxUsersetDepth = 10
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
	SubjectRelation  string                    // Optional subject relation for subject set checks
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
	if req.Depth >= MaxDepth {
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
	case *entities.HierarchicalRuleCallRule:
		return e.evaluateHierarchicalRuleCall(ctx, req, r)
	default:
		return false, fmt.Errorf("unknown rule type: %T", rule)
	}
}

// evaluateRelation evaluates a relation-based rule.
// If the rule's relation name refers to another permission in the same entity,
// it recursively evaluates that permission (permission composition).
// Otherwise, it checks for relation tuples matching the rule.
func (e *Evaluator) evaluateRelation(
	ctx context.Context,
	req *EvaluationRequest,
	rule *entities.RelationRule,
) (bool, error) {
	// Check if the relation name actually refers to a permission in the same entity.
	// This supports permission composition like "permission manage = edit"
	// where "edit" is another permission.
	schema, err := e.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return false, fmt.Errorf("failed to get schema: %w", err)
	}
	entity := schema.GetEntity(req.EntityType)
	if entity != nil {
		// Only treat as permission reference if there's NO relation with this name
		// (relations take precedence over permissions with the same name, though
		// the validator prevents name conflicts)
		isRelation := entity.GetRelation(rule.Relation) != nil
		if !isRelation {
			if perm := entity.GetPermission(rule.Relation); perm != nil {
				return e.EvaluateRule(ctx, &EvaluationRequest{
					TenantID:         req.TenantID,
					SchemaVersion:    req.SchemaVersion,
					EntityType:       req.EntityType,
					EntityID:         req.EntityID,
					SubjectType:      req.SubjectType,
					SubjectID:        req.SubjectID,
					SubjectRelation:  req.SubjectRelation,
					ContextualTuples: req.ContextualTuples,
					Depth:            req.Depth + 1,
				}, perm.Rule)
			}
		}
	}

	// If the request includes a SubjectRelation (subject set check), we need to
	// match tuples where the subject_relation matches exactly. For example,
	// checking if "team:engineering#member" has permission means finding a tuple
	// like "document:doc1#viewer@team:engineering#member".
	if req.SubjectRelation != "" {
		// Check contextual tuples for subject set match
		for _, tuple := range req.ContextualTuples {
			if tuple.EntityType == req.EntityType &&
				tuple.EntityID == req.EntityID &&
				tuple.Relation == rule.Relation &&
				tuple.SubjectType == req.SubjectType &&
				tuple.SubjectID == req.SubjectID &&
				tuple.SubjectRelation == req.SubjectRelation {
				return true, nil
			}
		}

		// Check database for subject set match
		exists, err := e.relationRepo.ExistsWithSubjectRelation(ctx, req.TenantID,
			req.EntityType, req.EntityID, rule.Relation,
			req.SubjectType, req.SubjectID, req.SubjectRelation)
		if err != nil {
			return false, fmt.Errorf("failed to check relation existence with subject relation: %w", err)
		}
		return exists, nil
	}

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

	// Check in database for direct match using Exists (more efficient than Read)
	exists, err := e.relationRepo.Exists(ctx, req.TenantID, &entities.RelationTuple{
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		Relation:    rule.Relation,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check relation existence: %w", err)
	}
	if exists {
		return true, nil
	}

	// Check for subject relations (e.g., team:backend-team#member)
	// Get all tuples for this entity and relation using specialized query
	allTuples, err := e.relationRepo.FindByEntityWithRelation(ctx, req.TenantID, req.EntityType, req.EntityID, rule.Relation, MaxTuplesPerQuery)
	if err != nil {
		return false, fmt.Errorf("failed to find relations by entity with relation: %w", err)
	}

	// Add contextual tuples
	allTuples = append(allTuples, req.ContextualTuples...)

	// Check each tuple for subject relations
	for _, tuple := range allTuples {
		// Skip if not matching entity/relation
		if tuple.EntityType != req.EntityType || tuple.EntityID != req.EntityID || tuple.Relation != rule.Relation {
			continue
		}

		// If this tuple has a subject relation, expand it recursively.
		// Example: tuple is "repository:backend-api#contributor@team:backend-team#member"
		// We need to check if "user:frank" has relation "member" with "team:backend-team".
		// Using recursive evaluateRelation handles nested computed usersets:
		// e.g., team#member → group#member → user
		if tuple.SubjectRelation != "" {
			subjectReq := &EvaluationRequest{
				TenantID:         req.TenantID,
				SchemaVersion:    req.SchemaVersion,
				EntityType:       tuple.SubjectType,
				EntityID:         tuple.SubjectID,
				SubjectType:      req.SubjectType,
				SubjectID:        req.SubjectID,
				ContextualTuples: req.ContextualTuples,
				Depth:            req.Depth + 1,
			}
			result, err := e.evaluateRelation(ctx, subjectReq, &entities.RelationRule{Relation: tuple.SubjectRelation})
			if err != nil {
				return false, fmt.Errorf("failed to evaluate subject relation: %w", err)
			}
			if result {
				return true, nil
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

	// Optimization: For same-type hierarchies where the permission being checked
	// is the SAME relation as the traversal (e.g., permission view = parent.parent),
	// use CTE-based hierarchical search. This only works when rule.Permission == rule.Relation
	// on the same entity type, AND there are no contextual tuples to consider.
	targetType := extractBaseType(relation.TargetType)
	if targetType == req.EntityType && rule.Permission == rule.Relation && len(req.ContextualTuples) == 0 {
		found, err := e.relationRepo.FindHierarchicalWithSubject(
			ctx, req.TenantID, req.EntityType, req.EntityID,
			rule.Relation, req.SubjectType, req.SubjectID, MaxDepth)
		if err == nil {
			return found, nil
		}
		log.Printf("WARNING: hierarchical CTE query failed, falling back to recursive evaluation: %v", err)
	}

	// Get the parent entity(s) via the relation using specialized query
	tuples, err := e.relationRepo.FindByEntityWithRelation(ctx, req.TenantID, req.EntityType, req.EntityID, rule.Relation, MaxTuplesPerQuery)
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
				SubjectRelation:  req.SubjectRelation,
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
			// If not a permission, check if it's a relation (Permify compatibility).
			// Use EvaluateRule with a synthetic RelationRule to get full relation
			// evaluation including computed userset expansion and SubjectRelation propagation.
			parentEntityDef := schema.GetEntity(tuple.SubjectType)
			if parentEntityDef != nil && parentEntityDef.GetRelation(rule.Permission) != nil {
				parentReq := &EvaluationRequest{
					TenantID:         req.TenantID,
					SchemaVersion:    req.SchemaVersion,
					EntityType:       tuple.SubjectType,
					EntityID:         tuple.SubjectID,
					SubjectType:      req.SubjectType,
					SubjectID:        req.SubjectID,
					SubjectRelation:  req.SubjectRelation,
					ContextualTuples: req.ContextualTuples,
					Depth:            req.Depth + 1,
				}
				relResult, err := e.evaluateRelation(ctx, parentReq, &entities.RelationRule{Relation: rule.Permission})
				if err != nil {
					return false, fmt.Errorf("failed to evaluate parent relation: %w", err)
				}
				if relResult {
					return true, nil
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
	resourceAttrs = normalizeJSONNumbers(resourceAttrs)

	// Get subject attributes
	subjectAttrs, err := e.attributeRepo.Read(ctx, req.TenantID, req.SubjectType, req.SubjectID)
	if err != nil {
		return false, fmt.Errorf("failed to read subject attributes: %w", err)
	}
	subjectAttrs = normalizeJSONNumbers(subjectAttrs)

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

	// Get resource attributes
	resourceAttrs, err := e.attributeRepo.Read(ctx, req.TenantID, req.EntityType, req.EntityID)
	if err != nil {
		return false, fmt.Errorf("failed to read resource attributes: %w", err)
	}
	resourceAttrs = normalizeJSONNumbers(resourceAttrs)

	// Get subject attributes
	subjectAttrs, err := e.attributeRepo.Read(ctx, req.TenantID, req.SubjectType, req.SubjectID)
	if err != nil {
		return false, fmt.Errorf("failed to read subject attributes: %w", err)
	}
	subjectAttrs = normalizeJSONNumbers(subjectAttrs)

	// Map argument names to their context data
	argContexts := map[string]map[string]interface{}{
		"resource": resourceAttrs,
		"subject":  subjectAttrs,
		"request":  {},
	}

	// Build parameter-to-context mapping: each parameter name gets the context
	// specified by its corresponding argument.
	// Example: rule check(doc, user) called as check(resource, subject)
	//   → parameter "doc" gets resource context, parameter "user" gets subject context
	paramContexts := make(map[string]map[string]interface{}, len(ruleDef.Parameters))
	for i, paramName := range ruleDef.Parameters {
		argName := rule.Arguments[i]
		paramContexts[paramName] = argContexts[argName]
	}

	// Evaluate the CEL expression from the rule body
	result, err := e.celEngine.EvaluateRule(ruleDef.Body, paramContexts)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate rule %s: %w", rule.RuleName, err)
	}

	return result, nil
}

// evaluateHierarchicalRuleCall evaluates a hierarchical rule call
// Example: "parent.check_confidentiality(authority)"
// Traverses the relation to find parent entities, then calls the rule on each parent
func (e *Evaluator) evaluateHierarchicalRuleCall(
	ctx context.Context,
	req *EvaluationRequest,
	rule *entities.HierarchicalRuleCallRule,
) (bool, error) {
	schema, err := e.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return false, fmt.Errorf("failed to get schema: %w", err)
	}

	// Get parent entities via the relation
	filter := &repositories.RelationFilter{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Relation:   rule.Relation,
	}
	tuples, err := e.relationRepo.Read(ctx, req.TenantID, filter)
	if err != nil {
		return false, fmt.Errorf("failed to read parent relations: %w", err)
	}

	// Also check contextual tuples
	for _, ct := range req.ContextualTuples {
		if ct.EntityType == req.EntityType &&
			ct.EntityID == req.EntityID &&
			ct.Relation == rule.Relation {
			tuples = append(tuples, ct)
		}
	}

	// Get current entity's attributes (for argument values)
	currentAttrs, err := e.attributeRepo.Read(ctx, req.TenantID, req.EntityType, req.EntityID)
	if err != nil {
		return false, fmt.Errorf("failed to read current entity attributes: %w", err)
	}
	currentAttrs = normalizeJSONNumbers(currentAttrs)

	for _, tuple := range tuples {
		// Find rule definition (try namespaced first, then plain)
		namespacedName := tuple.SubjectType + "." + rule.RuleName
		ruleDef := schema.GetRule(namespacedName)
		if ruleDef == nil {
			ruleDef = schema.GetRule(rule.RuleName)
		}
		if ruleDef == nil {
			return false, fmt.Errorf("rule %s not found", rule.RuleName)
		}

		// Get parent entity's attributes → these become "this" context
		parentAttrs, err := e.attributeRepo.Read(ctx, req.TenantID, tuple.SubjectType, tuple.SubjectID)
		if err != nil {
			return false, fmt.Errorf("failed to read parent attributes: %w", err)
		}
		parentAttrs = normalizeJSONNumbers(parentAttrs)

		// Build parameter values: map rule parameter names → current entity attribute values
		paramMap := make(map[string]interface{})
		for i, paramName := range ruleDef.Parameters {
			if i < len(rule.Arguments) {
				argAttrName := rule.Arguments[i]
				if val, ok := currentAttrs[argAttrName]; ok {
					paramMap[paramName] = val
				}
			}
		}

		// Evaluate CEL with "this" = parent attributes, plus parameter values
		result, err := e.celEngine.EvaluateWithParams(ruleDef.Body, parentAttrs, paramMap)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate hierarchical rule %s: %w", rule.RuleName, err)
		}
		if result {
			return true, nil
		}
	}

	return false, nil
}

// normalizeJSONNumbers converts json.Number values in a map to int64 or float64.
// This ensures consistent numeric types for CEL evaluation.
func normalizeJSONNumbers(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = normalizeValue(v)
	}
	return result
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return string(val)
	case map[string]interface{}:
		return normalizeJSONNumbers(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = normalizeValue(item)
		}
		return result
	default:
		return v
	}
}
