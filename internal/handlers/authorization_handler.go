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
type AuthorizationHandler struct {
	schemaService services.SchemaServiceInterface
	relationRepo  repositories.RelationRepository
	attributeRepo repositories.AttributeRepository
	checker       CheckerInterface
	expander      ExpanderInterface
	lookup        LookupInterface
	schemaRepo    repositories.SchemaRepository

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

// WriteSchema handles the WriteSchema RPC
func (h *AuthorizationHandler) WriteSchema(ctx context.Context, req *pb.WriteSchemaRequest) (*pb.WriteSchemaResponse, error) {
	if req.SchemaDsl == "" {
		return nil, status.Error(codes.InvalidArgument, "schema_dsl is required")
	}

	tenantID := "default"

	err := h.schemaService.WriteSchema(ctx, tenantID, req.SchemaDsl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to write schema: %v", err)
	}

	return &pb.WriteSchemaResponse{
		SchemaVersion: "", // TODO: schema_version機能実装時に更新
	}, nil
}

// ReadSchema handles the ReadSchema RPC
func (h *AuthorizationHandler) ReadSchema(ctx context.Context, req *pb.ReadSchemaRequest) (*pb.ReadSchemaResponse, error) {
	tenantID := "default"

	schemaDSL, err := h.schemaService.ReadSchema(ctx, tenantID)
	if err != nil {
		return nil, h.handleReadSchemaError(err)
	}

	schema, err := h.schemaService.GetSchemaEntity(ctx, tenantID)
	if err != nil {
		return nil, h.handleReadSchemaError(err)
	}

	updatedAt := ""
	if !schema.UpdatedAt.IsZero() {
		updatedAt = schema.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return &pb.ReadSchemaResponse{
		SchemaDsl: schemaDSL,
		UpdatedAt: updatedAt,
	}, nil
}

// WriteRelations handles the WriteRelations RPC
func (h *AuthorizationHandler) WriteRelations(ctx context.Context, req *pb.WriteRelationsRequest) (*pb.WriteRelationsResponse, error) {
	tenantID := "default"

	// Write tuples if provided
	if len(req.Tuples) > 0 {
		tuples := make([]*entities.RelationTuple, 0, len(req.Tuples))
		for i, protoTuple := range req.Tuples {
			tuple, err := protoToRelationTuple(protoTuple)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid tuple at index %d: %v", i, err)
			}
			tuples = append(tuples, tuple)
		}

		if err := h.relationRepo.BatchWrite(ctx, tenantID, tuples); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to write relations: %v", err)
		}
	}

	// Write attributes if provided (Permify互換)
	if len(req.Attributes) > 0 {
		for i, protoAttr := range req.Attributes {
			attr, err := protoToAttribute(protoAttr)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid attribute at index %d: %v", i, err)
			}

			if err := h.attributeRepo.Write(ctx, tenantID, attr); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to write attribute: %v", err)
			}
		}
	}

	return &pb.WriteRelationsResponse{
		SnapToken: "", // TODO: cache機構実装時に更新
	}, nil
}

// DeleteRelations handles the DeleteRelations RPC (Permify互換: フィルター形式)
func (h *AuthorizationHandler) DeleteRelations(ctx context.Context, req *pb.DeleteRelationsRequest) (*pb.DeleteRelationsResponse, error) {
	if req.Filter == nil {
		return nil, status.Error(codes.InvalidArgument, "filter is required")
	}

	tenantID := "default"

	// Convert filter to repository format
	filter := &repositories.RelationFilter{
		EntityType:      req.Filter.Entity.GetType(),
		EntityIDs:       req.Filter.Entity.GetIds(),
		Relation:        req.Filter.GetRelation(),
		SubjectType:     req.Filter.Subject.GetType(),
		SubjectIDs:      req.Filter.Subject.GetIds(),
		SubjectRelation: req.Filter.Subject.GetRelation(),
	}

	if err := h.relationRepo.DeleteByFilter(ctx, tenantID, filter); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete relations: %v", err)
	}

	return &pb.DeleteRelationsResponse{
		SnapToken: "", // TODO: cache機構実装時に更新
	}, nil
}

// WriteAttributes handles the WriteAttributes RPC
func (h *AuthorizationHandler) WriteAttributes(ctx context.Context, req *pb.WriteAttributesRequest) (*pb.WriteAttributesResponse, error) {
	if len(req.Attributes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one attribute is required")
	}

	tenantID := "default"

	for i, protoAttr := range req.Attributes {
		attr, err := protoToAttribute(protoAttr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid attribute at index %d: %v", i, err)
		}

		if err := h.attributeRepo.Write(ctx, tenantID, attr); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to write attribute: %v", err)
		}
	}

	return &pb.WriteAttributesResponse{
		SnapToken: "", // TODO: cache機構実装時に更新
	}, nil
}

// ReadRelationships handles the ReadRelationships RPC (Permify互換)
func (h *AuthorizationHandler) ReadRelationships(ctx context.Context, req *pb.ReadRelationshipsRequest) (*pb.ReadRelationshipsResponse, error) {
	tenantID := "default"

	// Convert filter
	filter := &repositories.RelationFilter{}
	if req.Filter != nil {
		if req.Filter.Entity != nil {
			filter.EntityType = req.Filter.Entity.GetType()
			filter.EntityIDs = req.Filter.Entity.GetIds()
		}
		filter.Relation = req.Filter.GetRelation()
		if req.Filter.Subject != nil {
			filter.SubjectType = req.Filter.Subject.GetType()
			filter.SubjectIDs = req.Filter.Subject.GetIds()
			filter.SubjectRelation = req.Filter.Subject.GetRelation()
		}
	}

	// Set pagination
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 100 // default
	}

	// Read from repository
	tuples, nextToken, err := h.relationRepo.ReadByFilter(ctx, tenantID, filter, pageSize, req.ContinuousToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read relationships: %v", err)
	}

	// Convert to proto
	protoTuples := make([]*pb.RelationTuple, 0, len(tuples))
	for _, tuple := range tuples {
		protoTuples = append(protoTuples, &pb.RelationTuple{
			Entity:   &pb.Entity{Type: tuple.EntityType, Id: tuple.EntityID},
			Relation: tuple.Relation,
			Subject: &pb.Subject{
				Type:     tuple.SubjectType,
				Id:       tuple.SubjectID,
				Relation: tuple.SubjectRelation,
			},
		})
	}

	return &pb.ReadRelationshipsResponse{
		Tuples:          protoTuples,
		ContinuousToken: nextToken,
	}, nil
}

// Check handles the Check RPC
func (h *AuthorizationHandler) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	tenantID := "default"

	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

	checkReq := &authorization.CheckRequest{
		TenantID:         tenantID,
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

	return &pb.CheckResponse{
		Can: result,
		Metadata: &pb.CheckResponseMetadata{
			CheckCount: 1,
		},
	}, nil
}

// Expand handles the Expand RPC
func (h *AuthorizationHandler) Expand(ctx context.Context, req *pb.ExpandRequest) (*pb.ExpandResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}

	tenantID := "default"

	expandReq := &authorization.ExpandRequest{
		TenantID:   tenantID,
		EntityType: req.Entity.Type,
		EntityID:   req.Entity.Id,
		Permission: req.Permission,
	}

	expandResp, err := h.expander.Expand(ctx, expandReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expand failed: %v", err)
	}

	tree := expandNodeToProto(expandResp.Tree)

	return &pb.ExpandResponse{
		Tree: tree,
	}, nil
}

// LookupEntity handles the LookupEntity RPC
func (h *AuthorizationHandler) LookupEntity(ctx context.Context, req *pb.LookupEntityRequest) (*pb.LookupEntityResponse, error) {
	if req.EntityType == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_type is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	tenantID := "default"

	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

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
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}
	if req.SubjectReference == nil {
		return nil, status.Error(codes.InvalidArgument, "subject_reference is required")
	}

	tenantID := "default"

	contextualTuples, err := protoContextToTuples(req.Context)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid context: %v", err)
	}

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
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	if req.Subject == nil {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	tenantID := "default"

	schema, err := h.schemaService.GetSchemaEntity(ctx, tenantID)
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

	return &pb.SubjectPermissionResponse{
		Results: results,
	}, nil
}

// LookupEntityStream handles the LookupEntityStream RPC
func (h *AuthorizationHandler) LookupEntityStream(req *pb.LookupEntityRequest, stream pb.AuthorizationService_LookupEntityStreamServer) error {
	return status.Error(codes.Unimplemented, "LookupEntityStream not implemented")
}

// === Helper Functions ===

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

func expandNodeToProto(node *authorization.ExpandNode) *pb.ExpandNode {
	if node == nil {
		return nil
	}

	protoNode := &pb.ExpandNode{
		Operation: node.Type,
	}

	if len(node.Children) > 0 {
		protoNode.Children = make([]*pb.ExpandNode, 0, len(node.Children))
		for _, child := range node.Children {
			protoNode.Children = append(protoNode.Children, expandNodeToProto(child))
		}
	}

	return protoNode
}

func protoToRelationTuple(proto *pb.RelationTuple) (*entities.RelationTuple, error) {
	if proto.Entity == nil {
		return nil, fmt.Errorf("entity is required")
	}
	if proto.Entity.Type == "" {
		return nil, fmt.Errorf("entity type is required")
	}
	if proto.Entity.Id == "" {
		return nil, fmt.Errorf("entity id is required")
	}

	if proto.Relation == "" {
		return nil, fmt.Errorf("relation is required")
	}

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
		SubjectRelation: proto.Subject.Relation,
	}

	if err := tuple.Validate(); err != nil {
		return nil, err
	}

	return tuple, nil
}

func protoToAttribute(proto *pb.AttributeData) (*entities.Attribute, error) {
	if proto.Entity == nil {
		return nil, fmt.Errorf("entity is required")
	}
	if proto.Entity.Type == "" {
		return nil, fmt.Errorf("entity type is required")
	}
	if proto.Entity.Id == "" {
		return nil, fmt.Errorf("entity id is required")
	}

	if proto.Attribute == "" {
		return nil, fmt.Errorf("attribute name is required")
	}

	val, err := protoValueToInterface(proto.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %v", err)
	}

	attr := &entities.Attribute{
		EntityType: proto.Entity.Type,
		EntityID:   proto.Entity.Id,
		Name:       proto.Attribute,
		Value:      val,
	}

	if err := attr.Validate(); err != nil {
		return nil, err
	}

	return attr, nil
}

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

func (h *AuthorizationHandler) handleReadSchemaError(err error) error {
	errMsg := err.Error()

	if strings.Contains(errMsg, "schema not found") || strings.Contains(errMsg, "not found") {
		return status.Errorf(codes.NotFound, "schema not found")
	}

	if strings.Contains(errMsg, "is required") {
		return status.Errorf(codes.InvalidArgument, "%s", errMsg)
	}

	return status.Errorf(codes.Internal, "failed to read schema: %s", errMsg)
}
