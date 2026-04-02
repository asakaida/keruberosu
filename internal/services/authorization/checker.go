package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories/postgres"
	"github.com/asakaida/keruberosu/pkg/cache"
)

// CheckerInterface defines the interface for permission checking
type CheckerInterface interface {
	Check(ctx context.Context, req *CheckRequest) (*CheckResponse, error)
	CheckMultiple(ctx context.Context, req *CheckRequest, permissions []string) (map[string]bool, error)
}

// Checker provides permission checking functionality
type Checker struct {
	schemaService   SchemaServiceInterface
	evaluator       *Evaluator
	cache           cache.Cache               // Optional cache for check results
	snapshotManager postgres.SnapshotProvider // Optional snapshot provider for cache consistency
	cacheTTL        time.Duration             // TTL for cached results
}

// CheckRequest contains the parameters for a permission check
type CheckRequest struct {
	TenantID         string                    // Tenant ID
	SchemaVersion    string                    // Schema version (empty = latest)
	EntityType       string                    // Resource entity type (e.g., "document")
	EntityID         string                    // Resource entity ID (e.g., "doc1")
	Permission       string                    // Permission to check (e.g., "view", "edit")
	SubjectType      string                    // Subject type (e.g., "user")
	SubjectID        string                    // Subject ID (e.g., "alice")
	SubjectRelation  string                    // Optional subject relation (e.g., "member" for subject set checks)
	ContextualTuples []*entities.RelationTuple // Temporary relation tuples for this check
	SnapshotToken    string                    // Optional snapshot token for cache consistency
}

// CheckResponse contains the result of a permission check
type CheckResponse struct {
	Allowed bool // Whether the subject has the permission
}

// NewChecker creates a new Checker without caching
func NewChecker(schemaService SchemaServiceInterface, evaluator *Evaluator) *Checker {
	return &Checker{
		schemaService: schemaService,
		evaluator:     evaluator,
	}
}

// NewCheckerWithCache creates a new Checker with caching enabled
func NewCheckerWithCache(
	schemaService SchemaServiceInterface,
	evaluator *Evaluator,
	c cache.Cache,
	snapshotManager postgres.SnapshotProvider,
	cacheTTL time.Duration,
) *Checker {
	return &Checker{
		schemaService:   schemaService,
		evaluator:       evaluator,
		cache:           c,
		snapshotManager: snapshotManager,
		cacheTTL:        cacheTTL,
	}
}

// generateCacheKey generates a cache key for the check request.
// schemaVersion is the resolved schema version (not the requested one, which may be empty).
func (c *Checker) generateCacheKey(req *CheckRequest, snapshotToken string, schemaVersion string) string {
	// Create a key from the request parameters, snapshot token, and resolved schema version.
	// Using the resolved schemaVersion ensures that schema changes invalidate cached results
	// even when no data writes have occurred.
	keyData := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%s",
		req.TenantID,
		schemaVersion,
		req.EntityType,
		req.EntityID,
		req.Permission,
		req.SubjectType,
		req.SubjectID,
		req.SubjectRelation,
		snapshotToken,
	)
	// Hash the key to keep it short
	hash := sha256.Sum256([]byte(keyData))
	return hex.EncodeToString(hash[:])
}

// Check performs a permission check
// Returns true if the subject has the specified permission on the resource
func (c *Checker) Check(ctx context.Context, req *CheckRequest) (*CheckResponse, error) {
	// Validate request
	if err := c.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid check request: %w", err)
	}

	// Skip cache if contextual tuples are present (they make the result unique)
	useCache := c.cache != nil && c.snapshotManager != nil && len(req.ContextualTuples) == 0

	// Get parsed schema (needed for both cache key and evaluation)
	schema, err := c.schemaService.GetSchemaEntity(ctx, req.TenantID, req.SchemaVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	var snapshotToken string
	var cacheKey string

	if useCache {
		// Get current snapshot token for cache key
		if req.SnapshotToken != "" {
			snapshotToken = req.SnapshotToken
		} else {
			snapshot, err := c.snapshotManager.GetCurrentSnapshotForRead(ctx)
			if err != nil {
				// Log error but continue without cache
				useCache = false
			} else {
				snapshotToken = snapshot.String()
			}
		}

		if useCache {
			// Include the resolved schema version in the cache key so that
			// schema changes invalidate cached results even when no data
			// writes (which update the snapshot token) have occurred.
			cacheKey = c.generateCacheKey(req, snapshotToken, schema.Version)

			// Try to get from cache
			if cached, found := c.cache.Get(ctx, cacheKey); found {
				if result, ok := cached.(bool); ok {
					return &CheckResponse{Allowed: result}, nil
				}
			}
		}
	}

	// Get entity definition
	entity := schema.GetEntity(req.EntityType)
	if entity == nil {
		return nil, fmt.Errorf("entity type %s not found in schema", req.EntityType)
	}

	// Get permission definition.
	// If no permission is found, check if it's a relation name and evaluate
	// as a direct relation check. This supports SubjectPermission reporting
	// on both permissions and relations.
	permission := entity.GetPermission(req.Permission)
	if permission == nil {
		if entity.GetRelation(req.Permission) != nil {
			permission = &entities.Permission{
				Name: req.Permission,
				Rule: &entities.RelationRule{Relation: req.Permission},
			}
		} else {
			return nil, fmt.Errorf("permission %s not found in entity %s", req.Permission, req.EntityType)
		}
	}

	// Create evaluation request.
	// Use the resolved schema version (not the requested one, which may be empty)
	// to ensure consistent schema usage throughout the evaluation.
	evalReq := &EvaluationRequest{
		TenantID:         req.TenantID,
		SchemaVersion:    schema.Version,
		EntityType:       req.EntityType,
		EntityID:         req.EntityID,
		SubjectType:      req.SubjectType,
		SubjectID:        req.SubjectID,
		SubjectRelation:  req.SubjectRelation,
		ContextualTuples: req.ContextualTuples,
		Depth:            0, // Start at depth 0
	}

	// Evaluate the permission rule
	allowed, err := c.evaluator.EvaluateRule(ctx, evalReq, permission.Rule)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate permission: %w", err)
	}

	// Store result in cache
	if useCache && cacheKey != "" {
		_ = c.cache.Set(ctx, cacheKey, allowed, c.cacheTTL)
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
			SchemaVersion:    req.SchemaVersion,
			EntityType:       req.EntityType,
			EntityID:         req.EntityID,
			Permission:       permission,
			SubjectType:      req.SubjectType,
			SubjectID:        req.SubjectID,
			SubjectRelation:  req.SubjectRelation,
			ContextualTuples: req.ContextualTuples,
			SnapshotToken:    req.SnapshotToken,
		}

		resp, err := c.Check(ctx, checkReq)
		if err != nil {
			return nil, fmt.Errorf("failed to check permission %s: %w", permission, err)
		}

		results[permission] = resp.Allowed
	}

	return results, nil
}
