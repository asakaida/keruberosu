package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
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

// AuthorizationHandler handles all authorization service gRPC requests
// This includes schema management, data management, and authorization operations
type AuthorizationHandler struct {
	// Schema management
	schemaService services.SchemaServiceInterface

	// Data management (Relations & Attributes)
	relationRepo  repositories.RelationRepository
	attributeRepo repositories.AttributeRepository

	// Authorization operations
	checker    CheckerInterface
	expander   ExpanderInterface
	lookup     LookupInterface
	schemaRepo repositories.SchemaRepository

	pb.UnimplementedAuthorizationServiceServer
}

// NewAuthorizationHandler creates a new unified AuthorizationHandler
func NewAuthorizationHandler(
	schemaService services.SchemaServiceInterface,
	relationRepo repositories.RelationRepository,
	attributeRepo repositories.AttributeRepository,
	checker CheckerInterface,
	expander ExpanderInterface,
	lookup LookupInterface,
	schemaRepo repositories.SchemaRepository,
) *AuthorizationHandler {
	return &AuthorizationHandler{
		schemaService: schemaService,
		relationRepo:  relationRepo,
		attributeRepo: attributeRepo,
		checker:       checker,
		expander:      expander,
		lookup:        lookup,
		schemaRepo:    schemaRepo,
	}
}

// === Schema Management ===

// WriteSchema handles the WriteSchema RPC
func (h *AuthorizationHandler) WriteSchema(ctx context.Context, req *pb.WriteSchemaRequest) (*pb.WriteSchemaResponse, error) {
	// Validate request
	if req.SchemaDsl == "" {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "schema_dsl is required",
			Errors:  []string{"schema_dsl field cannot be empty"},
		}, nil
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Call schema service
	err := h.schemaService.WriteSchema(ctx, tenantID, req.SchemaDsl)
	if err != nil {
		return h.handleWriteSchemaError(err)
	}

	return &pb.WriteSchemaResponse{
		Success: true,
		Message: "Schema written successfully",
		Errors:  nil,
	}, nil
}

// ReadSchema handles the ReadSchema RPC
func (h *AuthorizationHandler) ReadSchema(ctx context.Context, req *pb.ReadSchemaRequest) (*pb.ReadSchemaResponse, error) {
	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Call schema service
	schemaDSL, err := h.schemaService.ReadSchema(ctx, tenantID)
	if err != nil {
		return nil, h.handleReadSchemaError(err)
	}

	// Get schema entity to retrieve metadata
	schema, err := h.schemaService.GetSchemaEntity(ctx, tenantID)
	if err != nil {
		return nil, h.handleReadSchemaError(err)
	}

	// Format updated_at as ISO8601
	updatedAt := ""
	if !schema.UpdatedAt.IsZero() {
		updatedAt = schema.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return &pb.ReadSchemaResponse{
		SchemaDsl: schemaDSL,
		UpdatedAt: updatedAt,
	}, nil
}

// === Data Management ===

// WriteRelations handles the WriteRelations RPC
func (h *AuthorizationHandler) WriteRelations(ctx context.Context, req *pb.WriteRelationsRequest) (*pb.WriteRelationsResponse, error) {
	// Validate request
	if len(req.Tuples) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one relation tuple is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Convert proto tuples to entities
	tuples := make([]*entities.RelationTuple, 0, len(req.Tuples))
	for i, protoTuple := range req.Tuples {
		tuple, err := protoToRelationTuple(protoTuple)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid tuple at index %d: %v", i, err)
		}
		tuples = append(tuples, tuple)
	}

	// Batch write to repository
	if err := h.relationRepo.BatchWrite(ctx, tenantID, tuples); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to write relations: %v", err)
	}

	return &pb.WriteRelationsResponse{
		WrittenCount: int32(len(tuples)),
	}, nil
}

// DeleteRelations handles the DeleteRelations RPC
func (h *AuthorizationHandler) DeleteRelations(ctx context.Context, req *pb.DeleteRelationsRequest) (*pb.DeleteRelationsResponse, error) {
	// Validate request
	if len(req.Tuples) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one relation tuple is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Convert proto tuples to entities
	tuples := make([]*entities.RelationTuple, 0, len(req.Tuples))
	for i, protoTuple := range req.Tuples {
		tuple, err := protoToRelationTuple(protoTuple)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid tuple at index %d: %v", i, err)
		}
		tuples = append(tuples, tuple)
	}

	// Batch delete from repository
	if err := h.relationRepo.BatchDelete(ctx, tenantID, tuples); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete relations: %v", err)
	}

	return &pb.DeleteRelationsResponse{
		DeletedCount: int32(len(tuples)),
	}, nil
}

// WriteAttributes handles the WriteAttributes RPC
func (h *AuthorizationHandler) WriteAttributes(ctx context.Context, req *pb.WriteAttributesRequest) (*pb.WriteAttributesResponse, error) {
	// Validate request
	if len(req.Attributes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one attribute is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	tenantID := "default"

	// Convert proto attributes to entities and write
	writtenCount := 0
	for i, protoAttr := range req.Attributes {
		attributes, err := protoToAttributes(protoAttr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid attribute at index %d: %v", i, err)
		}

		// Write each attribute
		for _, attr := range attributes {
			if err := h.attributeRepo.Write(ctx, tenantID, attr); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to write attribute %s: %v", attr.Name, err)
			}
			writtenCount++
		}
	}

	return &pb.WriteAttributesResponse{
		WrittenCount: int32(writtenCount),
	}, nil
}

// === Authorization Operations ===

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

	// Get parsed schema to find all permissions for this entity type
	schema, err := h.schemaService.GetSchemaEntity(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "schema not found for tenant: %s", tenantID)
		}
		return nil, status.Errorf(codes.Internal, "failed to get schema: %v", err)
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

// === Helper Functions ===

// protoToRelationTuple converts proto RelationTuple to entities.RelationTuple
func protoToRelationTuple(proto *pb.RelationTuple) (*entities.RelationTuple, error) {
	// Validate entity
	if proto.Entity == nil {
		return nil, fmt.Errorf("entity is required")
	}
	if proto.Entity.Type == "" {
		return nil, fmt.Errorf("entity type is required")
	}
	if proto.Entity.Id == "" {
		return nil, fmt.Errorf("entity id is required")
	}

	// Validate relation
	if proto.Relation == "" {
		return nil, fmt.Errorf("relation is required")
	}

	// Validate subject
	if proto.Subject == nil {
		return nil, fmt.Errorf("subject is required")
	}
	if proto.Subject.Type == "" {
		return nil, fmt.Errorf("subject type is required")
	}
	if proto.Subject.Id == "" {
		return nil, fmt.Errorf("subject id is required")
	}

	tuple := &entities.RelationTuple{
		EntityType:      proto.Entity.Type,
		EntityID:        proto.Entity.Id,
		Relation:        proto.Relation,
		SubjectType:     proto.Subject.Type,
		SubjectID:       proto.Subject.Id,
		SubjectRelation: "", // proto.Subject is Entity type, not Subject type (no Relation field)
	}

	// Validate the tuple
	if err := tuple.Validate(); err != nil {
		return nil, err
	}

	return tuple, nil
}

// protoToAttributes converts proto AttributeData to entities.Attribute slice
func protoToAttributes(proto *pb.AttributeData) ([]*entities.Attribute, error) {
	// Validate entity
	if proto.Entity == nil {
		return nil, fmt.Errorf("entity is required")
	}
	if proto.Entity.Type == "" {
		return nil, fmt.Errorf("entity type is required")
	}
	if proto.Entity.Id == "" {
		return nil, fmt.Errorf("entity id is required")
	}

	// Validate data
	if len(proto.Data) == 0 {
		return nil, fmt.Errorf("at least one attribute is required")
	}

	// Convert map to attribute slice
	attributes := make([]*entities.Attribute, 0, len(proto.Data))
	for name, value := range proto.Data {
		// Convert protobuf Value to interface{}
		val, err := protoValueToInterface(value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for attribute %s: %v", name, err)
		}

		attr := &entities.Attribute{
			EntityType: proto.Entity.Type,
			EntityID:   proto.Entity.Id,
			Name:       name,
			Value:      val,
		}

		// Validate the attribute
		if err := attr.Validate(); err != nil {
			return nil, fmt.Errorf("validation failed for attribute %s: %v", name, err)
		}

		attributes = append(attributes, attr)
	}

	return attributes, nil
}

// protoValueToInterface converts protobuf Value to Go interface{}
func protoValueToInterface(v *structpb.Value) (interface{}, error) {
	if v == nil {
		return nil, fmt.Errorf("value cannot be nil")
	}

	switch v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil, nil
	case *structpb.Value_NumberValue:
		return v.GetNumberValue(), nil
	case *structpb.Value_StringValue:
		return v.GetStringValue(), nil
	case *structpb.Value_BoolValue:
		return v.GetBoolValue(), nil
	case *structpb.Value_StructValue:
		return v.GetStructValue().AsMap(), nil
	case *structpb.Value_ListValue:
		list := v.GetListValue().GetValues()
		result := make([]interface{}, len(list))
		for i, item := range list {
			val, err := protoValueToInterface(item)
			if err != nil {
				return nil, err
			}
			result[i] = val
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported value type: %T", v.Kind)
	}
}

// handleWriteSchemaError converts domain errors to WriteSchemaResponse with errors
func (h *AuthorizationHandler) handleWriteSchemaError(err error) (*pb.WriteSchemaResponse, error) {
	errMsg := err.Error()

	// Check error type and categorize
	if strings.Contains(errMsg, "failed to parse DSL") {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "Failed to parse schema DSL",
			Errors:  []string{errMsg},
		}, nil
	}

	if strings.Contains(errMsg, "schema validation failed") {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "Schema validation failed",
			Errors:  []string{errMsg},
		}, nil
	}

	if strings.Contains(errMsg, "is required") {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "Validation error",
			Errors:  []string{errMsg},
		}, nil
	}

	// Generic error
	return &pb.WriteSchemaResponse{
		Success: false,
		Message: "Failed to write schema",
		Errors:  []string{errMsg},
	}, nil
}

// handleReadSchemaError converts domain errors to gRPC status errors
func (h *AuthorizationHandler) handleReadSchemaError(err error) error {
	errMsg := err.Error()

	// Check error type and return appropriate gRPC status
	if strings.Contains(errMsg, "schema not found") || strings.Contains(errMsg, "not found") {
		return status.Errorf(codes.NotFound, "schema not found")
	}

	if strings.Contains(errMsg, "is required") {
		return status.Errorf(codes.InvalidArgument, "%s", errMsg)
	}

	// Generic error
	return status.Errorf(codes.Internal, "failed to read schema: %s", errMsg)
}
