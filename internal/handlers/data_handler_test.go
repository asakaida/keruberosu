package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// === Data Management Tests ===

func TestDataHandler_Write_Relations_Success(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		batchWriteFunc: func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if len(tuples) != 2 {
				t.Errorf("expected 2 tuples, got %d", len(tuples))
			}
			return nil
		},
	}

	handler := NewDataHandler(mockRelationRepo, &mockAttributeRepository{})

	req := &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Subject{Type: "user", Id: "alice"},
			},
			{
				Entity:   &pb.Entity{Type: "document", Id: "2"},
				Relation: "editor",
				Subject:  &pb.Subject{Type: "user", Id: "bob"},
			},
		},
	}

	resp, err := handler.Write(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DataWriteResponse now only contains snap_token (Permify compatible)
	if resp.SnapToken == "" {
		t.Logf("snap_token is empty (expected for now)")
	}
}

func TestDataHandler_Write_Attributes_Success(t *testing.T) {
	writtenAttrs := 0
	mockAttrRepo := &mockAttributeRepository{
		writeFunc: func(ctx context.Context, tenantID string, attr *entities.Attribute) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			writtenAttrs++
			return nil
		},
	}

	handler := NewDataHandler(&mockRelationRepository{}, mockAttrRepo)

	req := &pb.DataWriteRequest{
		Attributes: []*pb.Attribute{
			{
				Entity:    &pb.Entity{Type: "document", Id: "1"},
				Attribute: "public",
				Value:     structpb.NewBoolValue(true),
			},
			{
				Entity:    &pb.Entity{Type: "document", Id: "1"},
				Attribute: "title",
				Value:     structpb.NewStringValue("Test Document"),
			},
		},
	}

	resp, err := handler.Write(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DataWriteResponse now only contains snap_token (Permify compatible)
	if resp.SnapToken == "" {
		t.Logf("snap_token is empty (expected for now)")
	}

	if writtenAttrs != 2 {
		t.Errorf("expected 2 attributes written, got %d", writtenAttrs)
	}
}

func TestDataHandler_Write_Combined_Success(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		batchWriteFunc: func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
			if len(tuples) != 1 {
				t.Errorf("expected 1 tuple, got %d", len(tuples))
			}
			return nil
		},
	}

	writtenAttrs := 0
	mockAttrRepo := &mockAttributeRepository{
		writeFunc: func(ctx context.Context, tenantID string, attr *entities.Attribute) error {
			writtenAttrs++
			return nil
		},
	}

	handler := NewDataHandler(mockRelationRepo, mockAttrRepo)

	req := &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Subject{Type: "user", Id: "alice"},
			},
		},
		Attributes: []*pb.Attribute{
			{
				Entity:    &pb.Entity{Type: "document", Id: "1"},
				Attribute: "public",
				Value:     structpb.NewBoolValue(true),
			},
		},
	}

	resp, err := handler.Write(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.SnapToken == "" {
		t.Logf("snap_token is empty (expected for now)")
	}

	if writtenAttrs != 1 {
		t.Errorf("expected 1 attribute written, got %d", writtenAttrs)
	}
}

func TestDataHandler_Write_EmptyRequest(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.DataWriteRequest{
		Tuples:     []*pb.Tuple{},
		Attributes: []*pb.Attribute{},
	}

	// Empty write is allowed
	resp, err := handler.Write(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp.SnapToken
}

func TestDataHandler_Write_InvalidTuple(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.DataWriteRequest{
		Tuples: []*pb.Tuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: ""}, // Missing ID
				Relation: "owner",
				Subject:  &pb.Subject{Type: "user", Id: "alice"},
			},
		},
	}

	_, err := handler.Write(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid tuple")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_Write_InvalidAttribute(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.DataWriteRequest{
		Attributes: []*pb.Attribute{
			{
				Entity:    &pb.Entity{Type: "", Id: "1"}, // Missing type
				Attribute: "public",
				Value:     structpb.NewBoolValue(true),
			},
		},
	}

	_, err := handler.Write(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid attribute")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_Write_RepositoryError(t *testing.T) {
	mockAttrRepo := &mockAttributeRepository{
		writeFunc: func(ctx context.Context, tenantID string, attr *entities.Attribute) error {
			return fmt.Errorf("database error")
		},
	}

	handler := NewDataHandler(&mockRelationRepository{}, mockAttrRepo)

	req := &pb.DataWriteRequest{
		Attributes: []*pb.Attribute{
			{
				Entity:    &pb.Entity{Type: "document", Id: "1"},
				Attribute: "public",
				Value:     structpb.NewBoolValue(true),
			},
		},
	}

	_, err := handler.Write(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for repository error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected Internal error, got %v", st.Code())
	}
}

func TestDataHandler_Delete_Success(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		batchDeleteFunc: func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			return nil
		},
	}

	handler := NewDataHandler(mockRelationRepo, &mockAttributeRepository{})

	req := &pb.DataDeleteRequest{
		Filter: &pb.TupleFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{"1"},
			},
			Relation: "owner",
			Subject: &pb.SubjectFilter{
				Type: "user",
				Ids:  []string{"alice"},
			},
		},
	}

	resp, err := handler.Delete(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DataDeleteResponse now only contains snap_token (Permify compatible)
	if resp.SnapToken == "" {
		t.Logf("snap_token is empty (expected for now)")
	}
}

func TestDataHandler_Delete_EmptyFilter(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.DataDeleteRequest{
		Filter: nil,
	}

	_, err := handler.Delete(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty filter")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

// === Read Tests ===

func TestDataHandler_Read_Success(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		readByFilterFunc: func(ctx context.Context, tenantID string, filter *repositories.RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if filter.EntityType != "document" {
				t.Errorf("expected entity type 'document', got %s", filter.EntityType)
			}
			if len(filter.EntityIDs) != 1 || filter.EntityIDs[0] != "1" {
				t.Errorf("expected entity IDs ['1'], got %v", filter.EntityIDs)
			}
			return []*entities.RelationTuple{
				{
					EntityType:  "document",
					EntityID:    "1",
					Relation:    "owner",
					SubjectType: "user",
					SubjectID:   "alice",
				},
				{
					EntityType:  "document",
					EntityID:    "1",
					Relation:    "editor",
					SubjectType: "user",
					SubjectID:   "bob",
				},
			}, "", nil
		},
	}

	handler := NewDataHandler(mockRelationRepo, &mockAttributeRepository{})

	req := &pb.DataReadRequest{
		Filter: &pb.TupleFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{"1"},
			},
		},
	}

	resp, err := handler.Read(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Tuples) != 2 {
		t.Fatalf("expected 2 tuples, got %d", len(resp.Tuples))
	}

	// Verify first tuple
	tuple0 := resp.Tuples[0]
	if tuple0.Entity.Type != "document" || tuple0.Entity.Id != "1" {
		t.Errorf("expected entity document:1, got %s:%s", tuple0.Entity.Type, tuple0.Entity.Id)
	}
	if tuple0.Relation != "owner" {
		t.Errorf("expected relation 'owner', got %s", tuple0.Relation)
	}
	if tuple0.Subject.Type != "user" || tuple0.Subject.Id != "alice" {
		t.Errorf("expected subject user:alice, got %s:%s", tuple0.Subject.Type, tuple0.Subject.Id)
	}

	// Verify second tuple
	tuple1 := resp.Tuples[1]
	if tuple1.Relation != "editor" {
		t.Errorf("expected relation 'editor', got %s", tuple1.Relation)
	}
	if tuple1.Subject.Id != "bob" {
		t.Errorf("expected subject ID 'bob', got %s", tuple1.Subject.Id)
	}
}

func TestDataHandler_Read_WithPagination(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		readByFilterFunc: func(ctx context.Context, tenantID string, filter *repositories.RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error) {
			if pageSize != 10 {
				t.Errorf("expected page size 10, got %d", pageSize)
			}
			if pageToken != "token_abc" {
				t.Errorf("expected page token 'token_abc', got %s", pageToken)
			}
			return []*entities.RelationTuple{
				{
					EntityType:  "document",
					EntityID:    "1",
					Relation:    "viewer",
					SubjectType: "user",
					SubjectID:   "charlie",
				},
			}, "next_token_xyz", nil
		},
	}

	handler := NewDataHandler(mockRelationRepo, &mockAttributeRepository{})

	req := &pb.DataReadRequest{
		Filter: &pb.TupleFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{"1"},
			},
		},
		PageSize:        10,
		ContinuousToken: "token_abc",
	}

	resp, err := handler.Read(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Tuples) != 1 {
		t.Fatalf("expected 1 tuple, got %d", len(resp.Tuples))
	}

	if resp.ContinuousToken != "next_token_xyz" {
		t.Errorf("expected continuous token 'next_token_xyz', got %s", resp.ContinuousToken)
	}
}

func TestDataHandler_Read_NilFilter(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.DataReadRequest{
		Filter: nil,
	}

	_, err := handler.Read(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nil filter")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_Read_EmptyFilter(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.DataReadRequest{
		Filter: &pb.TupleFilter{},
	}

	_, err := handler.Read(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty filter")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_Read_SubjectRelation(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		readByFilterFunc: func(ctx context.Context, tenantID string, filter *repositories.RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error) {
			return []*entities.RelationTuple{
				{
					EntityType:      "document",
					EntityID:        "1",
					Relation:        "viewer",
					SubjectType:     "team",
					SubjectID:       "eng",
					SubjectRelation: "member",
				},
			}, "", nil
		},
	}

	handler := NewDataHandler(mockRelationRepo, &mockAttributeRepository{})

	req := &pb.DataReadRequest{
		Filter: &pb.TupleFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{"1"},
			},
		},
	}

	resp, err := handler.Read(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Tuples) != 1 {
		t.Fatalf("expected 1 tuple, got %d", len(resp.Tuples))
	}

	tuple := resp.Tuples[0]
	if tuple.Subject.Type != "team" {
		t.Errorf("expected subject type 'team', got %s", tuple.Subject.Type)
	}
	if tuple.Subject.Id != "eng" {
		t.Errorf("expected subject ID 'eng', got %s", tuple.Subject.Id)
	}
	if tuple.Subject.Relation != "member" {
		t.Errorf("expected subject relation 'member', got %s", tuple.Subject.Relation)
	}
}

// === ReadAttributes Tests ===

func TestDataHandler_ReadAttributes_Success(t *testing.T) {
	mockAttrRepo := &mockAttributeRepository{
		readFunc: func(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if entityType != "document" {
				t.Errorf("expected entity type 'document', got %s", entityType)
			}
			if entityID != "1" {
				t.Errorf("expected entity ID '1', got %s", entityID)
			}
			return map[string]interface{}{
				"public": true,
				"title":  "Test Document",
			}, nil
		},
	}

	handler := NewDataHandler(&mockRelationRepository{}, mockAttrRepo)

	req := &pb.AttributeReadRequest{
		Filter: &pb.AttributeFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{"1"},
			},
		},
	}

	resp, err := handler.ReadAttributes(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(resp.Attributes))
	}

	// Build a map for easier assertion (order may vary)
	attrMap := make(map[string]*pb.Attribute)
	for _, attr := range resp.Attributes {
		attrMap[attr.Attribute] = attr
	}

	publicAttr, ok := attrMap["public"]
	if !ok {
		t.Fatal("expected 'public' attribute in response")
	}
	if publicAttr.Entity.Type != "document" || publicAttr.Entity.Id != "1" {
		t.Errorf("expected entity document:1, got %s:%s", publicAttr.Entity.Type, publicAttr.Entity.Id)
	}
	if publicAttr.Value.GetBoolValue() != true {
		t.Errorf("expected public=true, got %v", publicAttr.Value.GetBoolValue())
	}

	titleAttr, ok := attrMap["title"]
	if !ok {
		t.Fatal("expected 'title' attribute in response")
	}
	if titleAttr.Value.GetStringValue() != "Test Document" {
		t.Errorf("expected title='Test Document', got %s", titleAttr.Value.GetStringValue())
	}
}

func TestDataHandler_ReadAttributes_WithAttributeFilter(t *testing.T) {
	mockAttrRepo := &mockAttributeRepository{
		readFunc: func(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"public": true,
				"title":  "Test Document",
				"owner":  "alice",
			}, nil
		},
	}

	handler := NewDataHandler(&mockRelationRepository{}, mockAttrRepo)

	req := &pb.AttributeReadRequest{
		Filter: &pb.AttributeFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{"1"},
			},
			Attributes: []string{"title"},
		},
	}

	resp, err := handler.ReadAttributes(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(resp.Attributes))
	}

	if resp.Attributes[0].Attribute != "title" {
		t.Errorf("expected attribute name 'title', got %s", resp.Attributes[0].Attribute)
	}
	if resp.Attributes[0].Value.GetStringValue() != "Test Document" {
		t.Errorf("expected value 'Test Document', got %s", resp.Attributes[0].Value.GetStringValue())
	}
}

func TestDataHandler_ReadAttributes_NilFilter(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.AttributeReadRequest{
		Filter: nil,
	}

	_, err := handler.ReadAttributes(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nil filter")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_ReadAttributes_MissingEntityType(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.AttributeReadRequest{
		Filter: &pb.AttributeFilter{
			Entity: &pb.EntityFilter{
				Type: "",
				Ids:  []string{"1"},
			},
		},
	}

	_, err := handler.ReadAttributes(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing entity type")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_ReadAttributes_MissingEntityID(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.AttributeReadRequest{
		Filter: &pb.AttributeFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{},
			},
		},
	}

	_, err := handler.ReadAttributes(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing entity ID")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_ReadAttributes_BoolStringNumberValues(t *testing.T) {
	mockAttrRepo := &mockAttributeRepository{
		readFunc: func(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"is_public":   true,
				"title":       "My Doc",
				"view_count":  float64(42),
				"rating":      3.14,
				"json_number": json.Number("99"),
			}, nil
		},
	}

	handler := NewDataHandler(&mockRelationRepository{}, mockAttrRepo)

	req := &pb.AttributeReadRequest{
		Filter: &pb.AttributeFilter{
			Entity: &pb.EntityFilter{
				Type: "document",
				Ids:  []string{"1"},
			},
		},
	}

	resp, err := handler.ReadAttributes(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Attributes) != 5 {
		t.Fatalf("expected 5 attributes, got %d", len(resp.Attributes))
	}

	attrMap := make(map[string]*pb.Attribute)
	for _, attr := range resp.Attributes {
		attrMap[attr.Attribute] = attr
	}

	// bool
	if attrMap["is_public"].Value.GetBoolValue() != true {
		t.Errorf("expected is_public=true, got %v", attrMap["is_public"].Value.GetBoolValue())
	}

	// string
	if attrMap["title"].Value.GetStringValue() != "My Doc" {
		t.Errorf("expected title='My Doc', got %s", attrMap["title"].Value.GetStringValue())
	}

	// float64 number
	if attrMap["view_count"].Value.GetNumberValue() != 42 {
		t.Errorf("expected view_count=42, got %v", attrMap["view_count"].Value.GetNumberValue())
	}

	// float64 decimal
	if attrMap["rating"].Value.GetNumberValue() != 3.14 {
		t.Errorf("expected rating=3.14, got %v", attrMap["rating"].Value.GetNumberValue())
	}

	// json.Number
	if attrMap["json_number"].Value.GetNumberValue() != 99 {
		t.Errorf("expected json_number=99, got %v", attrMap["json_number"].Value.GetNumberValue())
	}
}

// === interfaceToProtoValue Tests ===

func TestInterfaceToProtoValue_Nil(t *testing.T) {
	val, err := interfaceToProtoValue(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNullValue() != structpb.NullValue_NULL_VALUE {
		t.Errorf("expected null value, got %v", val)
	}
}

func TestInterfaceToProtoValue_Bool(t *testing.T) {
	val, err := interfaceToProtoValue(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetBoolValue() != true {
		t.Errorf("expected true, got %v", val.GetBoolValue())
	}
}

func TestInterfaceToProtoValue_String(t *testing.T) {
	val, err := interfaceToProtoValue("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetStringValue() != "hello" {
		t.Errorf("expected 'hello', got %s", val.GetStringValue())
	}
}

func TestInterfaceToProtoValue_Int(t *testing.T) {
	val, err := interfaceToProtoValue(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNumberValue() != 42 {
		t.Errorf("expected 42, got %v", val.GetNumberValue())
	}
}

func TestInterfaceToProtoValue_Int32(t *testing.T) {
	val, err := interfaceToProtoValue(int32(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNumberValue() != 100 {
		t.Errorf("expected 100, got %v", val.GetNumberValue())
	}
}

func TestInterfaceToProtoValue_Int64(t *testing.T) {
	val, err := interfaceToProtoValue(int64(999))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNumberValue() != 999 {
		t.Errorf("expected 999, got %v", val.GetNumberValue())
	}
}

func TestInterfaceToProtoValue_Float32(t *testing.T) {
	val, err := interfaceToProtoValue(float32(1.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNumberValue() != float64(float32(1.5)) {
		t.Errorf("expected 1.5, got %v", val.GetNumberValue())
	}
}

func TestInterfaceToProtoValue_Float64(t *testing.T) {
	val, err := interfaceToProtoValue(3.14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNumberValue() != 3.14 {
		t.Errorf("expected 3.14, got %v", val.GetNumberValue())
	}
}

func TestInterfaceToProtoValue_JsonNumber_Int(t *testing.T) {
	val, err := interfaceToProtoValue(json.Number("123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNumberValue() != 123 {
		t.Errorf("expected 123, got %v", val.GetNumberValue())
	}
}

func TestInterfaceToProtoValue_JsonNumber_Float(t *testing.T) {
	val, err := interfaceToProtoValue(json.Number("3.14"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.GetNumberValue() != 3.14 {
		t.Errorf("expected 3.14, got %v", val.GetNumberValue())
	}
}

func TestInterfaceToProtoValue_JsonNumber_Invalid(t *testing.T) {
	val, err := interfaceToProtoValue(json.Number("not_a_number"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls back to string representation
	if val.GetStringValue() != "not_a_number" {
		t.Errorf("expected string fallback 'not_a_number', got %v", val)
	}
}

func TestInterfaceToProtoValue_List(t *testing.T) {
	input := []interface{}{"a", float64(1), true}
	val, err := interfaceToProtoValue(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listVal := val.GetListValue()
	if listVal == nil {
		t.Fatal("expected list value")
	}
	if len(listVal.Values) != 3 {
		t.Fatalf("expected 3 list elements, got %d", len(listVal.Values))
	}
	if listVal.Values[0].GetStringValue() != "a" {
		t.Errorf("expected list[0]='a', got %v", listVal.Values[0])
	}
	if listVal.Values[1].GetNumberValue() != 1 {
		t.Errorf("expected list[1]=1, got %v", listVal.Values[1])
	}
	if listVal.Values[2].GetBoolValue() != true {
		t.Errorf("expected list[2]=true, got %v", listVal.Values[2])
	}
}

func TestInterfaceToProtoValue_Map(t *testing.T) {
	input := map[string]interface{}{
		"key": "value",
	}
	val, err := interfaceToProtoValue(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	structVal := val.GetStructValue()
	if structVal == nil {
		t.Fatal("expected struct value")
	}
	fields := structVal.GetFields()
	if fields["key"].GetStringValue() != "value" {
		t.Errorf("expected key='value', got %v", fields["key"])
	}
}

func TestInterfaceToProtoValue_UnsupportedType(t *testing.T) {
	type custom struct{}
	_, err := interfaceToProtoValue(custom{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}
