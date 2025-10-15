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

func TestSchemaHandler_Read_WithSpecificVersion(t *testing.T) {
	updatedAt := time.Now()
	mockService := &mockSchemaService{}
	mockRepo := &mockSchemaRepository{
		getByVersionFunc: func(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if version != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
				t.Errorf("expected version '01ARZ3NDEKTSV4RRFFQ69G5FAV', got %s", version)
			}
			return &entities.Schema{
				TenantID:  tenantID,
				Version:   version,
				DSL:       `entity document {}`,
				UpdatedAt: updatedAt,
			}, nil
		},
	}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaReadRequest{
		Metadata: &pb.PermissionCheckMetadata{
			SchemaVersion: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		},
	}

	resp, err := handler.Read(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Schema != `entity document {}` {
		t.Errorf("expected schema 'entity document {}', got %s", resp.Schema)
	}
}

func TestSchemaHandler_List_Success(t *testing.T) {
	createdAt1 := time.Now().Add(-2 * time.Hour)
	createdAt2 := time.Now().Add(-1 * time.Hour)
	createdAt3 := time.Now()

	mockService := &mockSchemaService{}
	mockRepo := &mockSchemaRepository{
		listVersionsFunc: func(ctx context.Context, tenantID string, limit int, offset int) ([]*entities.SchemaVersion, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if limit != 10 {
				t.Errorf("expected limit 10, got %d", limit)
			}
			return []*entities.SchemaVersion{
				{Version: "01ARZ3NDEKTSV4RRFFQ69G5FC", CreatedAt: createdAt3},
				{Version: "01ARZ3NDEKTSV4RRFFQ69G5FB", CreatedAt: createdAt2},
				{Version: "01ARZ3NDEKTSV4RRFFQ69G5FA", CreatedAt: createdAt1},
			}, nil
		},
	}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaListRequest{
		PageSize: 10,
	}

	resp, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Head != "01ARZ3NDEKTSV4RRFFQ69G5FC" {
		t.Errorf("expected head '01ARZ3NDEKTSV4RRFFQ69G5FC', got %s", resp.Head)
	}

	if len(resp.Schemas) != 3 {
		t.Fatalf("expected 3 schemas, got %d", len(resp.Schemas))
	}

	if resp.Schemas[0].Version != "01ARZ3NDEKTSV4RRFFQ69G5FC" {
		t.Errorf("expected first version '01ARZ3NDEKTSV4RRFFQ69G5FC', got %s", resp.Schemas[0].Version)
	}
	if resp.Schemas[0].CreatedAt == "" {
		t.Error("expected created_at to be set")
	}
}

func TestSchemaHandler_List_EmptyResult(t *testing.T) {
	mockService := &mockSchemaService{}
	mockRepo := &mockSchemaRepository{
		listVersionsFunc: func(ctx context.Context, tenantID string, limit int, offset int) ([]*entities.SchemaVersion, error) {
			return []*entities.SchemaVersion{}, nil
		},
	}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaListRequest{}

	resp, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Head != "" {
		t.Errorf("expected empty head, got %s", resp.Head)
	}

	if len(resp.Schemas) != 0 {
		t.Fatalf("expected 0 schemas, got %d", len(resp.Schemas))
	}
}

func TestSchemaHandler_List_DefaultPageSize(t *testing.T) {
	mockService := &mockSchemaService{}
	mockRepo := &mockSchemaRepository{
		listVersionsFunc: func(ctx context.Context, tenantID string, limit int, offset int) ([]*entities.SchemaVersion, error) {
			if limit != 10 {
				t.Errorf("expected default limit 10, got %d", limit)
			}
			return []*entities.SchemaVersion{}, nil
		},
	}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaListRequest{
		PageSize: 0, // Should default to 10
	}

	_, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchemaHandler_List_MaxPageSize(t *testing.T) {
	mockService := &mockSchemaService{}
	mockRepo := &mockSchemaRepository{
		listVersionsFunc: func(ctx context.Context, tenantID string, limit int, offset int) ([]*entities.SchemaVersion, error) {
			if limit != 100 {
				t.Errorf("expected max limit 100, got %d", limit)
			}
			return []*entities.SchemaVersion{}, nil
		},
	}

	handler := NewSchemaHandler(mockService, mockRepo)

	req := &pb.SchemaListRequest{
		PageSize: 150, // Should be capped at 100
	}

	_, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
