package handlers

import (
	"context"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DataHandler handles Data service gRPC requests
type DataHandler struct {
	pb.UnimplementedDataServer
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

// Write handles the Write RPC - writes both tuples and attributes
func (h *DataHandler) Write(ctx context.Context, req *pb.DataWriteRequest) (*pb.DataWriteResponse, error) {
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

	// Write attributes if provided
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

	return &pb.DataWriteResponse{
		SnapToken: "", // TODO: implement snapshot token later
	}, nil
}

// Delete handles the Delete RPC
func (h *DataHandler) Delete(ctx context.Context, req *pb.DataDeleteRequest) (*pb.DataDeleteResponse, error) {
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

	return &pb.DataDeleteResponse{
		SnapToken: "", // TODO: implement snapshot token later
	}, nil
}

// Read handles the Read RPC
func (h *DataHandler) Read(ctx context.Context, req *pb.DataReadRequest) (*pb.DataReadResponse, error) {
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
	protoTuples := make([]*pb.Tuple, 0, len(tuples))
	for _, tuple := range tuples {
		protoTuples = append(protoTuples, &pb.Tuple{
			Entity:   &pb.Entity{Type: tuple.EntityType, Id: tuple.EntityID},
			Relation: tuple.Relation,
			Subject: &pb.Subject{
				Type:     tuple.SubjectType,
				Id:       tuple.SubjectID,
				Relation: tuple.SubjectRelation,
			},
		})
	}

	return &pb.DataReadResponse{
		Tuples:          protoTuples,
		ContinuousToken: nextToken,
	}, nil
}

// ReadAttributes handles the ReadAttributes RPC
func (h *DataHandler) ReadAttributes(ctx context.Context, req *pb.AttributeReadRequest) (*pb.AttributeReadResponse, error) {
	tenantID := "default"

	if req.Filter == nil || req.Filter.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "filter with entity is required")
	}

	entityType := req.Filter.Entity.GetType()
	if entityType == "" {
		return nil, status.Error(codes.InvalidArgument, "entity type is required")
	}

	entityIDs := req.Filter.Entity.GetIds()
	if len(entityIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one entity ID is required")
	}

	// For now, we only support reading attributes for a single entity
	// TODO: Implement batch reading if needed
	if len(entityIDs) > 1 {
		return nil, status.Error(codes.Unimplemented, "reading attributes for multiple entities not yet supported")
	}

	entityID := entityIDs[0]

	// Read all attributes for the entity
	attrMap, err := h.attributeRepo.Read(ctx, tenantID, entityType, entityID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read attributes: %v", err)
	}

	// Filter by attribute names if specified
	requestedAttrs := req.Filter.GetAttributes()
	var filteredMap map[string]interface{}

	if len(requestedAttrs) > 0 {
		filteredMap = make(map[string]interface{})
		for _, attrName := range requestedAttrs {
			if val, ok := attrMap[attrName]; ok {
				filteredMap[attrName] = val
			}
		}
	} else {
		filteredMap = attrMap
	}

	// Convert to proto attributes
	protoAttrs := make([]*pb.Attribute, 0, len(filteredMap))
	for attrName, attrValue := range filteredMap {
		protoValue, err := interfaceToProtoValue(attrValue)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert attribute %s: %v", attrName, err)
		}

		protoAttrs = append(protoAttrs, &pb.Attribute{
			Entity: &pb.Entity{
				Type: entityType,
				Id:   entityID,
			},
			Attribute: attrName,
			Value:     protoValue,
		})
	}

	return &pb.AttributeReadResponse{
		Attributes:      protoAttrs,
		ContinuousToken: "", // TODO: implement pagination for attributes
	}, nil
}
