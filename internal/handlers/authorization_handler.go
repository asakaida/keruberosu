package handlers

import (
	"context"
	"fmt"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CheckerInterface defines the interface for permission checking
type CheckerInterface interface {
	Check(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error)
}

// ExpanderInterface defines the interface for permission tree expansion
type ExpanderInterface interface {
	Expand(ctx context.Context, req *authorization.ExpandRequest) (*authorization.ExpandResponse, error)
}

// LookupInterface defines the interface for entity/subject lookup
type LookupInterface interface {
	LookupEntity(ctx context.Context, req *authorization.LookupEntityRequest) (*authorization.LookupEntityResponse, error)
	LookupSubject(ctx context.Context, req *authorization.LookupSubjectRequest) (*authorization.LookupSubjectResponse, error)
}

// AuthorizationHandler handles authorization gRPC requests
type AuthorizationHandler struct {
	checker    CheckerInterface
	expander   ExpanderInterface
	lookup     LookupInterface
	schemaRepo repositories.SchemaRepository
}

// NewAuthorizationHandler creates a new AuthorizationHandler
func NewAuthorizationHandler(
	checker CheckerInterface,
	expander ExpanderInterface,
	lookup LookupInterface,
	schemaRepo repositories.SchemaRepository,
) *AuthorizationHandler {
	return &AuthorizationHandler{
		checker:    checker,
		expander:   expander,
		lookup:     lookup,
		schemaRepo: schemaRepo,
	}
}

// Check handles the Check RPC
func (h *AuthorizationHandler) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	// Validate request
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Convert contextual tuples
	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	// Create check request
	checkReq := &authorization.CheckRequest{
		TenantID:         tenantID,
		EntityType:       req.Entity.Type,
		EntityID:         req.Entity.Id,
		Permission:       req.Permission,
		SubjectType:      req.Subject.Type,
		SubjectID:        req.Subject.Id,
		ContextualTuples: contextualTuples,
	}

	// Execute check
	checkResp, err := h.checker.Check(ctx, checkReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check failed: %v", err)
	}

	// Convert result
	result := pb.CheckResult_CHECK_RESULT_DENIED
	if checkResp.Allowed {
		result = pb.CheckResult_CHECK_RESULT_ALLOWED
	}

	return &pb.CheckResponse{
		Can: result,
		Metadata: &pb.CheckResponseMetadata{
			CheckCount: 1, // Simple implementation: one check performed
		},
	}, nil
}

// Expand handles the Expand RPC
func (h *AuthorizationHandler) Expand(ctx context.Context, req *pb.ExpandRequest) (*pb.ExpandResponse, error) {
	// Validate request
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Create expand request
	expandReq := &authorization.ExpandRequest{
		TenantID:   tenantID,
		EntityType: req.Entity.Type,
		EntityID:   req.Entity.Id,
		Permission: req.Permission,
	}

	// Execute expand
	expandResp, err := h.expander.Expand(ctx, expandReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expand failed: %v", err)
	}

	// Convert tree to proto
	tree := expandNodeToProto(expandResp.Tree)

	return &pb.ExpandResponse{
		Tree: tree,
	}, nil
}

// LookupEntity handles the LookupEntity RPC
func (h *AuthorizationHandler) LookupEntity(ctx context.Context, req *pb.LookupEntityRequest) (*pb.LookupEntityResponse, error) {
	// Validate request
	if req.EntityType == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Convert contextual tuples
	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	// Create lookup request
	lookupReq := &authorization.LookupEntityRequest{
		TenantID:         tenantID,
		EntityType:       req.EntityType,
		Permission:       req.Permission,
		SubjectType:      req.Subject.Type,
		SubjectID:        req.Subject.Id,
		ContextualTuples: contextualTuples,
		PageSize:         int(req.PageSize),
		PageToken:        req.ContinuousToken,
	}

	// Execute lookup
	lookupResp, err := h.lookup.LookupEntity(ctx, lookupReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lookup entity failed: %v", err)
	}

	return &pb.LookupEntityResponse{
		EntityIds:       lookupResp.EntityIDs,
		ContinuousToken: lookupResp.NextPageToken,
	}, nil
}

// LookupSubject handles the LookupSubject RPC
func (h *AuthorizationHandler) LookupSubject(ctx context.Context, req *pb.LookupSubjectRequest) (*pb.LookupSubjectResponse, error) {
	// Validate request
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.SubjectReference == nil {
		return nil, status.Error(codes.InvalidArgument, "subject_reference is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Convert contextual tuples
	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	// Create lookup request
	lookupReq := &authorization.LookupSubjectRequest{
		TenantID:         tenantID,
		EntityType:       req.Entity.Type,
		EntityID:         req.Entity.Id,
		Permission:       req.Permission,
		SubjectType:      req.SubjectReference.Type,
		ContextualTuples: contextualTuples,
		PageSize:         int(req.PageSize),
		PageToken:        req.ContinuousToken,
	}

	// Execute lookup
	lookupResp, err := h.lookup.LookupSubject(ctx, lookupReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lookup subject failed: %v", err)
	}

	return &pb.LookupSubjectResponse{
		SubjectIds:      lookupResp.SubjectIDs,
		ContinuousToken: lookupResp.NextPageToken,
	}, nil
}

// SubjectPermission handles the SubjectPermission RPC
func (h *AuthorizationHandler) SubjectPermission(ctx context.Context, req *pb.SubjectPermissionRequest) (*pb.SubjectPermissionResponse, error) {
	// Validate request
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Get schema to find all permissions for this entity type
	schema, err := h.schemaRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get schema: %v", err)
	}
	if schema == nil {
		return nil, status.Error(codes.NotFound, "schema not found")
	}

	// Get entity definition
	entity := schema.GetEntity(req.Entity.Type)
	if entity == nil {
		return nil, status.Errorf(codes.NotFound, "entity type %s not found in schema", req.Entity.Type)
	}

	// Convert contextual tuples
	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	// Check each permission
	results := make(map[string]pb.CheckResult)
	for _, permission := range entity.Permissions {
		checkReq := &authorization.CheckRequest{
			TenantID:         tenantID,
			EntityType:       req.Entity.Type,
			EntityID:         req.Entity.Id,
			Permission:       permission.Name,
			SubjectType:      req.Subject.Type,
			SubjectID:        req.Subject.Id,
			ContextualTuples: contextualTuples,
		}

		checkResp, err := h.checker.Check(ctx, checkReq)
		if err != nil {
			// If check fails, mark as denied
			results[permission.Name] = pb.CheckResult_CHECK_RESULT_DENIED
			continue
		}

		if checkResp.Allowed {
			results[permission.Name] = pb.CheckResult_CHECK_RESULT_ALLOWED
		} else {
			results[permission.Name] = pb.CheckResult_CHECK_RESULT_DENIED
		}
	}

	return &pb.SubjectPermissionResponse{
		Results: results,
	}, nil
}

// LookupEntityStream handles the LookupEntityStream RPC (streaming version)
// Phase 1: Not implemented yet
func (h *AuthorizationHandler) LookupEntityStream(req *pb.LookupEntityRequest, stream pb.AuthorizationService_LookupEntityStreamServer) error {
	return status.Error(codes.Unimplemented, "LookupEntityStream not implemented in Phase 1")
}

// protoContextToTuples converts proto Context to entities.RelationTuple slice
func protoContextToTuples(ctx *pb.Context) ([]*entities.RelationTuple, error) {
	if ctx == nil || len(ctx.Tuples) == 0 {
		return nil, nil
	}

	tuples := make([]*entities.RelationTuple, 0, len(ctx.Tuples))
	for i, protoTuple := range ctx.Tuples {
		tuple, err := protoToRelationTuple(protoTuple)
		if err != nil {
			return nil, fmt.Errorf("invalid tuple at index %d: %v", i, err)
		}
		tuples = append(tuples, tuple)
	}

	return tuples, nil
}

// expandNodeToProto converts authorization.ExpandNode to proto ExpandNode
func expandNodeToProto(node *authorization.ExpandNode) *pb.ExpandNode {
	if node == nil {
		return nil
	}

	protoNode := &pb.ExpandNode{
		Operation: node.Type,
	}

	// Convert children recursively
	if len(node.Children) > 0 {
		protoNode.Children = make([]*pb.ExpandNode, 0, len(node.Children))
		for _, child := range node.Children {
			protoNode.Children = append(protoNode.Children, expandNodeToProto(child))
		}
	}

	// For leaf nodes, set entity and subject
	if node.Type == "leaf" && node.Subject != "" {
		// Parse entity reference (e.g., "document:doc1")
		// Parse subject reference (e.g., "user:alice")
		// For simplicity in Phase 1, store as string in operation
		// Full implementation would parse these properly
	}

	return protoNode
}
