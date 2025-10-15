package authorization

import (
	"context"
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
)

// CheckerInterface defines the interface for permission checking
type CheckerInterface interface {
	Check(ctx context.Context, req *CheckRequest) (*CheckResponse, error)
	CheckMultiple(ctx context.Context, req *CheckRequest, permissions []string) (map[string]bool, error)
}

// Checker provides permission checking functionality
type Checker struct {
	schemaService SchemaServiceInterface
	evaluator     *Evaluator
}

// CheckRequest contains the parameters for a permission check
type CheckRequest struct {
	TenantID         string                    // Tenant ID
	EntityType       string                    // Resource entity type (e.g., "document")
	EntityID         string                    // Resource entity ID (e.g., "doc1")
	Permission       string                    // Permission to check (e.g., "view", "edit")
	SubjectType      string                    // Subject type (e.g., "user")
	SubjectID        string                    // Subject ID (e.g., "alice")
	ContextualTuples []*entities.RelationTuple // Temporary relation tuples for this check
}

// CheckResponse contains the result of a permission check
type CheckResponse struct {
	Allowed bool // Whether the subject has the permission
}

// NewChecker creates a new Checker
func NewChecker(schemaService SchemaServiceInterface, evaluator *Evaluator) *Checker {
	return &Checker{
		schemaService: schemaService,
		evaluator:     evaluator,
	}
}

// Check performs a permission check
// Returns true if the subject has the specified permission on the resource
func (c *Checker) Check(ctx context.Context, req *CheckRequest) (*CheckResponse, error) {
	// Validate request
	if err := c.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid check request: %w", err)
	}

	// Get parsed schema
	schema, err := c.schemaService.GetSchemaEntity(ctx, req.TenantID, "")
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

	// Create evaluation request
	evalReq := &EvaluationRequest{
		TenantID:         req.TenantID,
		EntityType:       req.EntityType,
		EntityID:         req.EntityID,
		SubjectType:      req.SubjectType,
		SubjectID:        req.SubjectID,
		ContextualTuples: req.ContextualTuples,
		Depth:            0, // Start at depth 0
	}

	// Evaluate the permission rule
	allowed, err := c.evaluator.EvaluateRule(ctx, evalReq, permission.Rule)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate permission: %w", err)
	}

	return &CheckResponse{
		Allowed: allowed,
	}, nil
}

// validateRequest validates the check request
func (c *Checker) validateRequest(req *CheckRequest) error {
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
	if req.SubjectID == "" {
		return fmt.Errorf("subject ID is required")
	}
	return nil
}

// CheckMultiple performs multiple permission checks in a single call
// Returns a map of permission name to whether it's allowed
func (c *Checker) CheckMultiple(ctx context.Context, req *CheckRequest, permissions []string) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, permission := range permissions {
		checkReq := &CheckRequest{
			TenantID:         req.TenantID,
			EntityType:       req.EntityType,
			EntityID:         req.EntityID,
			Permission:       permission,
			SubjectType:      req.SubjectType,
			SubjectID:        req.SubjectID,
			ContextualTuples: req.ContextualTuples,
		}

		resp, err := c.Check(ctx, checkReq)
		if err != nil {
			// If permission not found or other error, mark as false
			results[permission] = false
			continue
		}

		results[permission] = resp.Allowed
	}

	return results, nil
}
