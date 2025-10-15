package handlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// === Schema Management Tests ===

func TestSchemaHandler_Write_Success(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			return "01ARZ3NDEKTSV4RRFFQ69G5FAV", nil
		},
	}

	mockRepo := &mockSchemaRepository{}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaWriteRequest{
		Schema: `entity user {}`,
	}

	resp, err := handler.Write(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.SchemaVersion == "" {
		t.Error("expected non-empty schema version")
	}

	if resp.SchemaVersion != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Errorf("expected version '01ARZ3NDEKTSV4RRFFQ69G5FAV', got %s", resp.SchemaVersion)
	}
}

func TestSchemaHandler_Write_EmptySchema(t *testing.T) {
	mockService := &mockSchemaService{}
	mockRepo := &mockSchemaRepository{}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaWriteRequest{
		Schema: "",
	}

	_, err := handler.Write(context.Background(), req)
	// SchemaWriteResponse now only contains SchemaVersion (Permify compatible)
	// Errors are returned via gRPC error, not in response fields
	if err == nil {
		t.Errorf("expected error for empty schema")
	}
}

func TestSchemaHandler_Write_ParseError(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
			return "", fmt.Errorf("failed to parse DSL: syntax error at line 1")
		},
	}

	mockRepo := &mockSchemaRepository{}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaWriteRequest{
		Schema: `invalid syntax`,
	}

	_, err := handler.Write(context.Background(), req)
	// SchemaWriteResponse now only contains SchemaVersion (Permify compatible)
	// Errors are returned via gRPC error, not in response fields
	if err == nil {
		t.Errorf("expected error for parse error")
	}
}

func TestSchemaHandler_Write_ValidationError(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
			return "", fmt.Errorf("schema validation failed: undefined relation reference")
		},
	}

	mockRepo := &mockSchemaRepository{}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaWriteRequest{
		Schema: `entity document { permission view = owner }`,
	}

	_, err := handler.Write(context.Background(), req)
	// SchemaWriteResponse now only contains SchemaVersion (Permify compatible)
	// Errors are returned via gRPC error, not in response fields
	if err == nil {
		t.Errorf("expected error for validation error")
	}
}

func TestSchemaHandler_Read_Success(t *testing.T) {
	updatedAt := time.Now()
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			return &entities.Schema{
				TenantID:  tenantID,
				DSL:       `entity user {}`,
				UpdatedAt: updatedAt,
			}, nil
		},
	}

	mockRepo := &mockSchemaRepository{}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaReadRequest{}

	resp, err := handler.Read(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Schema != `entity user {}` {
		t.Errorf("expected schema 'entity user {}', got %s", resp.Schema)
	}

	if resp.UpdatedAt == "" {
		t.Errorf("expected updated_at to be set")
	}
}

func TestSchemaHandler_Read_NotFound(t *testing.T) {
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return nil, fmt.Errorf("schema not found for tenant: %s", tenantID)
		},
	}

	mockRepo := &mockSchemaRepository{}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaReadRequest{}

	_, err := handler.Read(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for schema not found")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

func TestSchemaHandler_Read_NoUpdatedAt(t *testing.T) {
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID:  tenantID,
				DSL:       `entity user {}`,
				UpdatedAt: time.Time{}, // Zero value
			}, nil
		},
	}

	mockRepo := &mockSchemaRepository{}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaReadRequest{}

	resp, err := handler.Read(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.UpdatedAt != "" {
		t.Errorf("expected empty updated_at, got %s", resp.UpdatedAt)
	}
}
