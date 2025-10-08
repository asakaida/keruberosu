package handlers

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// Mock RelationRepository
type mockRelationRepository struct {
	batchWriteFunc  func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
	batchDeleteFunc func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
}

func (m *mockRelationRepository) Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	return nil
}

func (m *mockRelationRepository) Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	return nil
}

func (m *mockRelationRepository) Read(ctx context.Context, tenantID string, filter *repositories.RelationFilter) ([]*entities.RelationTuple, error) {
	return nil, nil
}

func (m *mockRelationRepository) CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
	return false, nil
}

func (m *mockRelationRepository) BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if m.batchWriteFunc != nil {
		return m.batchWriteFunc(ctx, tenantID, tuples)
	}
	return nil
}

func (m *mockRelationRepository) BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if m.batchDeleteFunc != nil {
		return m.batchDeleteFunc(ctx, tenantID, tuples)
	}
	return nil
}

// Mock AttributeRepository
type mockAttributeRepository struct {
	writeFunc func(ctx context.Context, tenantID string, attr *entities.Attribute) error
}

func (m *mockAttributeRepository) Write(ctx context.Context, tenantID string, attr *entities.Attribute) error {
	if m.writeFunc != nil {
		return m.writeFunc(ctx, tenantID, attr)
	}
	return nil
}

func (m *mockAttributeRepository) Read(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockAttributeRepository) Delete(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) error {
	return nil
}

func (m *mockAttributeRepository) GetValue(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) (interface{}, error) {
	return nil, nil
}

func TestDataHandler_WriteRelations_Success(t *testing.T) {
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

	req := &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
			{
				Entity:   &pb.Entity{Type: "document", Id: "2"},
				Relation: "editor",
				Subject:  &pb.Entity{Type: "user", Id: "bob"},
			},
		},
	}

	resp, err := handler.WriteRelations(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.WrittenCount != 2 {
		t.Errorf("expected written count 2, got %d", resp.WrittenCount)
	}
}

func TestDataHandler_WriteRelations_EmptyTuples(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{},
	}

	_, err := handler.WriteRelations(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty tuples")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_WriteRelations_InvalidTuple(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: ""}, // Missing ID
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
		},
	}

	_, err := handler.WriteRelations(context.Background(), req)
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

func TestDataHandler_DeleteRelations_Success(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		batchDeleteFunc: func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if len(tuples) != 1 {
				t.Errorf("expected 1 tuple, got %d", len(tuples))
			}
			return nil
		},
	}

	handler := NewDataHandler(mockRelationRepo, &mockAttributeRepository{})

	req := &pb.DeleteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
		},
	}

	resp, err := handler.DeleteRelations(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.DeletedCount != 1 {
		t.Errorf("expected deleted count 1, got %d", resp.DeletedCount)
	}
}

func TestDataHandler_DeleteRelations_EmptyTuples(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.DeleteRelationsRequest{
		Tuples: []*pb.RelationTuple{},
	}

	_, err := handler.DeleteRelations(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty tuples")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_WriteAttributes_Success(t *testing.T) {
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

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
					"title":  structpb.NewStringValue("Test Document"),
				},
			},
		},
	}

	resp, err := handler.WriteAttributes(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.WrittenCount != 2 {
		t.Errorf("expected written count 2, got %d", resp.WrittenCount)
	}

	if writtenAttrs != 2 {
		t.Errorf("expected 2 attributes written, got %d", writtenAttrs)
	}
}

func TestDataHandler_WriteAttributes_EmptyAttributes(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{},
	}

	_, err := handler.WriteAttributes(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty attributes")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestDataHandler_WriteAttributes_InvalidAttribute(t *testing.T) {
	handler := NewDataHandler(&mockRelationRepository{}, &mockAttributeRepository{})

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity: &pb.Entity{Type: "", Id: "1"}, // Missing type
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
				},
			},
		},
	}

	_, err := handler.WriteAttributes(context.Background(), req)
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

func TestDataHandler_WriteAttributes_RepositoryError(t *testing.T) {
	mockAttrRepo := &mockAttributeRepository{
		writeFunc: func(ctx context.Context, tenantID string, attr *entities.Attribute) error {
			return fmt.Errorf("database error")
		},
	}

	handler := NewDataHandler(&mockRelationRepository{}, mockAttrRepo)

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
				},
			},
		},
	}

	_, err := handler.WriteAttributes(context.Background(), req)
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

func TestProtoToRelationTuple(t *testing.T) {
	tests := []struct {
		name      string
		proto     *pb.RelationTuple
		wantError bool
	}{
		{
			name: "valid tuple",
			proto: &pb.RelationTuple{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
			wantError: false,
		},
		{
			name: "missing entity",
			proto: &pb.RelationTuple{
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
			wantError: true,
		},
		{
			name: "missing relation",
			proto: &pb.RelationTuple{
				Entity:  &pb.Entity{Type: "document", Id: "1"},
				Subject: &pb.Entity{Type: "user", Id: "alice"},
			},
			wantError: true,
		},
		{
			name: "missing subject",
			proto: &pb.RelationTuple{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := protoToRelationTuple(tt.proto)
			if (err != nil) != tt.wantError {
				t.Errorf("protoToRelationTuple() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestProtoToAttributes(t *testing.T) {
	tests := []struct {
		name      string
		proto     *pb.AttributeData
		wantError bool
		wantCount int
	}{
		{
			name: "valid attributes",
			proto: &pb.AttributeData{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
					"title":  structpb.NewStringValue("Test"),
				},
			},
			wantError: false,
			wantCount: 2,
		},
		{
			name: "missing entity",
			proto: &pb.AttributeData{
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
				},
			},
			wantError: true,
		},
		{
			name: "empty data",
			proto: &pb.AttributeData{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data:   map[string]*structpb.Value{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs, err := protoToAttributes(tt.proto)
			if (err != nil) != tt.wantError {
				t.Errorf("protoToAttributes() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && len(attrs) != tt.wantCount {
				t.Errorf("protoToAttributes() got %d attributes, want %d", len(attrs), tt.wantCount)
			}
		})
	}
}
