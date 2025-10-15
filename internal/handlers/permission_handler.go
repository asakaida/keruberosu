package handlers

import (
	"context"
	"errors"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PermissionHandler handles Permission service gRPC requests
type PermissionHandler struct {
	pb.UnimplementedPermissionServer
	checker       authorization.CheckerInterface
	expander      authorization.ExpanderInterface
	lookup        authorization.LookupInterface
	schemaService services.SchemaServiceInterface
}

// NewPermissionHandler creates a new PermissionHandler
func NewPermissionHandler(
	checker authorization.CheckerInterface,
	expander authorization.ExpanderInterface,
	lookup authorization.LookupInterface,
	schemaService services.SchemaServiceInterface,
) *PermissionHandler {
	return &PermissionHandler{
		checker:       checker,
		expander:      expander,
		lookup:        lookup,
		schemaService: schemaService,
	}
}

// Check handles the Check RPC
func (h *PermissionHandler) Check(ctx context.Context, req *pb.PermissionCheckRequest) (*pb.PermissionCheckResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	tenantID := req.TenantId
	if tenantID == "" {
		tenantID = "default"
	}

	schemaVersion := ""
	if req.Metadata != nil {
		schemaVersion = req.Metadata.SchemaVersion
	}

	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	checkReq := &authorization.CheckRequest{
		TenantID:         tenantID,
		SchemaVersion:    schemaVersion,
		EntityType:       req.Entity.Type,
		EntityID:         req.Entity.Id,
		Permission:       req.Permission,
		SubjectType:      req.Subject.Type,
		SubjectID:        req.Subject.Id,
		ContextualTuples: contextualTuples,
	}

	checkResp, err := h.checker.Check(ctx, checkReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check failed: %v", err)
	}

	result := pb.CheckResult_CHECK_RESULT_DENIED
	if checkResp.Allowed {
		result = pb.CheckResult_CHECK_RESULT_ALLOWED
	}

	return &pb.PermissionCheckResponse{
		Can: result,
		Metadata: &pb.PermissionCheckResponseMetadata{
			CheckCount: 1,
		},
	}, nil
}

// Expand handles the Expand RPC
func (h *PermissionHandler) Expand(ctx context.Context, req *pb.PermissionExpandRequest) (*pb.PermissionExpandResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}

	tenantID := req.TenantId
	if tenantID == "" {
		tenantID = "default"
	}

	schemaVersion := ""
	if req.Metadata != nil {
		schemaVersion = req.Metadata.SchemaVersion
	}

	expandReq := &authorization.ExpandRequest{
		TenantID:      tenantID,
		SchemaVersion: schemaVersion,
		EntityType:    req.Entity.Type,
		EntityID:      req.Entity.Id,
		Permission:    req.Permission,
	}

	expandResp, err := h.expander.Expand(ctx, expandReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expand failed: %v", err)
	}

	tree := expandNodeToProto(expandResp.Tree)

	return &pb.PermissionExpandResponse{
		Tree: tree,
	}, nil
}

// LookupEntity handles the LookupEntity RPC
func (h *PermissionHandler) LookupEntity(ctx context.Context, req *pb.PermissionLookupEntityRequest) (*pb.PermissionLookupEntityResponse, error) {
	if req.EntityType == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	tenantID := req.TenantId
	if tenantID == "" {
		tenantID = "default"
	}

	schemaVersion := ""
	if req.Metadata != nil {
		schemaVersion = req.Metadata.SchemaVersion
	}

	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	lookupReq := &authorization.LookupEntityRequest{
		TenantID:         tenantID,
		SchemaVersion:    schemaVersion,
		EntityType:       req.EntityType,
		Permission:       req.Permission,
		SubjectType:      req.Subject.Type,
		SubjectID:        req.Subject.Id,
		ContextualTuples: contextualTuples,
		PageSize:         int(req.PageSize),
		PageToken:        req.ContinuousToken,
	}

	lookupResp, err := h.lookup.LookupEntity(ctx, lookupReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lookup entity failed: %v", err)
	}

	return &pb.PermissionLookupEntityResponse{
		EntityIds:       lookupResp.EntityIDs,
		ContinuousToken: lookupResp.NextPageToken,
	}, nil
}

// LookupSubject handles the LookupSubject RPC
func (h *PermissionHandler) LookupSubject(ctx context.Context, req *pb.PermissionLookupSubjectRequest) (*pb.PermissionLookupSubjectResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.SubjectReference == nil {
		return nil, status.Error(codes.InvalidArgument, "subject_reference is required")
	}

	tenantID := req.TenantId
	if tenantID == "" {
		tenantID = "default"
	}

	schemaVersion := ""
	if req.Metadata != nil {
		schemaVersion = req.Metadata.SchemaVersion
	}

	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	lookupReq := &authorization.LookupSubjectRequest{
		TenantID:         tenantID,
		SchemaVersion:    schemaVersion,
		EntityType:       req.Entity.Type,
		EntityID:         req.Entity.Id,
		Permission:       req.Permission,
		SubjectType:      req.SubjectReference.Type,
		ContextualTuples: contextualTuples,
		PageSize:         int(req.PageSize),
		PageToken:        req.ContinuousToken,
	}

	lookupResp, err := h.lookup.LookupSubject(ctx, lookupReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lookup subject failed: %v", err)
	}

	return &pb.PermissionLookupSubjectResponse{
		SubjectIds:      lookupResp.SubjectIDs,
		ContinuousToken: lookupResp.NextPageToken,
	}, nil
}

// LookupEntityStream handles the LookupEntityStream RPC
func (h *PermissionHandler) LookupEntityStream(req *pb.PermissionLookupEntityRequest, stream pb.Permission_LookupEntityStreamServer) error {
	return status.Error(codes.Unimplemented, "LookupEntityStream not implemented")
}

// SubjectPermission handles the SubjectPermission RPC
func (h *PermissionHandler) SubjectPermission(ctx context.Context, req *pb.PermissionSubjectPermissionRequest) (*pb.PermissionSubjectPermissionResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	tenantID := req.TenantId
	if tenantID == "" {
		tenantID = "default"
	}

	schemaVersion := ""
	if req.Metadata != nil {
		schemaVersion = req.Metadata.SchemaVersion
	}

	schema, err := h.schemaService.GetSchemaEntity(ctx, tenantID, schemaVersion)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "schema not found for tenant: %s", tenantID)
		}
		return nil, status.Errorf(codes.Internal, "failed to get schema: %v", err)
	}

	entity := schema.GetEntity(req.Entity.Type)
	if entity == nil {
		return nil, status.Errorf(codes.NotFound, "entity type %s not found in schema", req.Entity.Type)
	}

	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	results := make(map[string]pb.CheckResult)
	for _, permission := range entity.Permissions {
		checkReq := &authorization.CheckRequest{
			TenantID:         tenantID,
			SchemaVersion:    schemaVersion,
			EntityType:       req.Entity.Type,
			EntityID:         req.Entity.Id,
			Permission:       permission.Name,
			SubjectType:      req.Subject.Type,
			SubjectID:        req.Subject.Id,
			ContextualTuples: contextualTuples,
		}

		checkResp, err := h.checker.Check(ctx, checkReq)
		if err != nil {
			results[permission.Name] = pb.CheckResult_CHECK_RESULT_DENIED
			continue
		}

		if checkResp.Allowed {
			results[permission.Name] = pb.CheckResult_CHECK_RESULT_ALLOWED
		} else {
			results[permission.Name] = pb.CheckResult_CHECK_RESULT_DENIED
		}
	}

	return &pb.PermissionSubjectPermissionResponse{
		Results: results,
	}, nil
}
