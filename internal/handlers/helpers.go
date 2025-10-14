package handlers

import (
	"fmt"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// === Shared Helper Functions for all handlers ===

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

func expandNodeToProto(node *authorization.ExpandNode) *pb.ExpandTreeNode {
	if node == nil {
		return nil
	}

	protoNode := &pb.ExpandTreeNode{
		Operation: node.Type,
	}

	if len(node.Children) > 0 {
		protoNode.Children = make([]*pb.ExpandTreeNode, 0, len(node.Children))
		for _, child := range node.Children {
			protoNode.Children = append(protoNode.Children, expandNodeToProto(child))
		}
	}

	return protoNode
}

// protoToRelationTuple handles both pb.RelationTuple and pb.Tuple
func protoToRelationTuple(proto interface{}) (*entities.RelationTuple, error) {
	var entity *pb.Entity
	var relation string
	var subject *pb.Subject

	// Handle both pb.RelationTuple and pb.Tuple
	switch t := proto.(type) {
	case *pb.RelationTuple:
		entity = t.Entity
		relation = t.Relation
		subject = t.Subject
	case *pb.Tuple:
		entity = t.Entity
		relation = t.Relation
		subject = t.Subject
	default:
		return nil, fmt.Errorf("unsupported tuple type: %T", proto)
	}

	if entity == nil {
		return nil, fmt.Errorf("entity is required")
	}
	if entity.Type == "" {
		return nil, fmt.Errorf("entity type is required")
	}
	if entity.Id == "" {
		return nil, fmt.Errorf("entity id is required")
	}

	if relation == "" {
		return nil, fmt.Errorf("relation is required")
	}

	if subject == nil {
		return nil, fmt.Errorf("subject is required")
	}
	if subject.Type == "" {
		return nil, fmt.Errorf("subject type is required")
	}
	if subject.Id == "" {
		return nil, fmt.Errorf("subject id is required")
	}

	tuple := &entities.RelationTuple{
		EntityType:      entity.Type,
		EntityID:        entity.Id,
		Relation:        relation,
		SubjectType:     subject.Type,
		SubjectID:       subject.Id,
		SubjectRelation: subject.Relation,
	}

	if err := tuple.Validate(); err != nil {
		return nil, err
	}

	return tuple, nil
}

func protoToAttribute(proto *pb.Attribute) (*entities.Attribute, error) {
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

func interfaceToProtoValue(v interface{}) (*structpb.Value, error) {
	if v == nil {
		return structpb.NewNullValue(), nil
	}

	switch val := v.(type) {
	case bool:
		return structpb.NewBoolValue(val), nil
	case int:
		return structpb.NewNumberValue(float64(val)), nil
	case int32:
		return structpb.NewNumberValue(float64(val)), nil
	case int64:
		return structpb.NewNumberValue(float64(val)), nil
	case float32:
		return structpb.NewNumberValue(float64(val)), nil
	case float64:
		return structpb.NewNumberValue(val), nil
	case string:
		return structpb.NewStringValue(val), nil
	case []interface{}:
		listValues := make([]*structpb.Value, len(val))
		for i, item := range val {
			protoVal, err := interfaceToProtoValue(item)
			if err != nil {
				return nil, err
			}
			listValues[i] = protoVal
		}
		return structpb.NewListValue(&structpb.ListValue{Values: listValues}), nil
	case map[string]interface{}:
		structVal, err := structpb.NewStruct(val)
		if err != nil {
			return nil, err
		}
		return structpb.NewStructValue(structVal), nil
	default:
		return nil, fmt.Errorf("unsupported value type: %T", v)
	}
}

func handleReadSchemaError(err error) error {
	errMsg := err.Error()

	if containsAny(errMsg, "schema not found", "not found") {
		return status.Errorf(codes.NotFound, "schema not found")
	}

	if containsAny(errMsg, "is required") {
		return status.Errorf(codes.InvalidArgument, "%s", errMsg)
	}

	return status.Errorf(codes.Internal, "failed to read schema: %s", errMsg)
}

func containsAny(str string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
