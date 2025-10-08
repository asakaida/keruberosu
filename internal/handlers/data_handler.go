package handlers

import (
	"context"
	"fmt"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// DataHandler handles data management gRPC requests
type DataHandler struct {
	relationRepo  repositories.RelationRepository
	attributeRepo repositories.AttributeRepository
}

// NewDataHandler creates a new DataHandler
func NewDataHandler(
	relationRepo repositories.RelationRepository,
	attributeRepo repositories.AttributeRepository,
) *DataHandler {
	return &DataHandler{
		relationRepo:  relationRepo,
		attributeRepo: attributeRepo,
	}
}

// WriteRelations handles the WriteRelations RPC
func (h *DataHandler) WriteRelations(ctx context.Context, req *pb.WriteRelationsRequest) (*pb.WriteRelationsResponse, error) {
	// Validate request
	if len(req.Tuples) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one relation tuple is required")
	}

	// Phase 1: Use fixed tenant ID "default"
	// In future phases, extract tenant ID from gRPC metadata
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
func (h *DataHandler) DeleteRelations(ctx context.Context, req *pb.DeleteRelationsRequest) (*pb.DeleteRelationsResponse, error) {
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
func (h *DataHandler) WriteAttributes(ctx context.Context, req *pb.WriteAttributesRequest) (*pb.WriteAttributesResponse, error) {
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
