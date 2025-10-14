package handlers

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
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
