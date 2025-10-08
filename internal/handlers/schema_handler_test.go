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

// Mock SchemaService
type mockSchemaService struct {
	writeSchemaFunc     func(ctx context.Context, tenantID string, schemaDSL string) error
	readSchemaFunc      func(ctx context.Context, tenantID string) (string, error)
	getSchemaEntityFunc func(ctx context.Context, tenantID string) (*entities.Schema, error)
}

func (m *mockSchemaService) WriteSchema(ctx context.Context, tenantID string, schemaDSL string) error {
	if m.writeSchemaFunc != nil {
		return m.writeSchemaFunc(ctx, tenantID, schemaDSL)
	}
	return nil
}

func (m *mockSchemaService) ReadSchema(ctx context.Context, tenantID string) (string, error) {
	if m.readSchemaFunc != nil {
		return m.readSchemaFunc(ctx, tenantID)
	}
	return "", nil
}

func (m *mockSchemaService) ValidateSchema(ctx context.Context, schemaDSL string) error {
	return nil
}

func (m *mockSchemaService) DeleteSchema(ctx context.Context, tenantID string) error {
	return nil
}

func (m *mockSchemaService) GetSchemaEntity(ctx context.Context, tenantID string) (*entities.Schema, error) {
	if m.getSchemaEntityFunc != nil {
		return m.getSchemaEntityFunc(ctx, tenantID)
	}
	return nil, nil
}

func TestSchemaHandler_WriteSchema_Success(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			return nil
		},
	}

	handler := NewSchemaHandler(mockService)

	req := &pb.WriteSchemaRequest{
		SchemaDsl: `entity user {}`,
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success=true, got false")
	}

	if resp.Message != "Schema written successfully" {
		t.Errorf("expected success message, got %s", resp.Message)
	}

	if len(resp.Errors) != 0 {
		t.Errorf("expected no errors, got %v", resp.Errors)
	}
}

func TestSchemaHandler_WriteSchema_EmptyDSL(t *testing.T) {
	handler := NewSchemaHandler(&mockSchemaService{})

	req := &pb.WriteSchemaRequest{
		SchemaDsl: "",
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success=false for empty DSL")
	}

	if len(resp.Errors) == 0 {
		t.Errorf("expected errors for empty DSL")
	}
}

func TestSchemaHandler_WriteSchema_ParseError(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) error {
			return fmt.Errorf("failed to parse DSL: syntax error at line 1")
		},
	}

	handler := NewSchemaHandler(mockService)

	req := &pb.WriteSchemaRequest{
		SchemaDsl: `invalid syntax`,
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success=false for parse error")
	}

	if resp.Message != "Failed to parse schema DSL" {
		t.Errorf("expected parse error message, got %s", resp.Message)
	}

	if len(resp.Errors) == 0 {
		t.Errorf("expected errors for parse error")
	}
}

func TestSchemaHandler_WriteSchema_ValidationError(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) error {
			return fmt.Errorf("schema validation failed: undefined relation reference")
		},
	}

	handler := NewSchemaHandler(mockService)

	req := &pb.WriteSchemaRequest{
		SchemaDsl: `entity document { permission view = owner }`,
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success=false for validation error")
	}

	if resp.Message != "Schema validation failed" {
		t.Errorf("expected validation error message, got %s", resp.Message)
	}

	if len(resp.Errors) == 0 {
		t.Errorf("expected errors for validation error")
	}
}

func TestSchemaHandler_ReadSchema_Success(t *testing.T) {
	updatedAt := time.Now()
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (string, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			return `entity user {}`, nil
		},
		getSchemaEntityFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID:  tenantID,
				DSL:       `entity user {}`,
				UpdatedAt: updatedAt,
			}, nil
		},
	}

	handler := NewSchemaHandler(mockService)

	req := &pb.ReadSchemaRequest{}

	resp, err := handler.ReadSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.SchemaDsl != `entity user {}` {
		t.Errorf("expected schema DSL 'entity user {}', got %s", resp.SchemaDsl)
	}

	if resp.UpdatedAt == "" {
		t.Errorf("expected updated_at to be set")
	}
}

func TestSchemaHandler_ReadSchema_NotFound(t *testing.T) {
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (string, error) {
			return "", fmt.Errorf("schema not found for tenant: %s", tenantID)
		},
	}

	handler := NewSchemaHandler(mockService)

	req := &pb.ReadSchemaRequest{}

	_, err := handler.ReadSchema(context.Background(), req)
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

func TestSchemaHandler_ReadSchema_NoUpdatedAt(t *testing.T) {
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (string, error) {
			return `entity user {}`, nil
		},
		getSchemaEntityFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID:  tenantID,
				DSL:       `entity user {}`,
				UpdatedAt: time.Time{}, // Zero value (no updated_at)
			}, nil
		},
	}

	handler := NewSchemaHandler(mockService)

	req := &pb.ReadSchemaRequest{}

	resp, err := handler.ReadSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.UpdatedAt != "" {
		t.Errorf("expected empty updated_at, got %s", resp.UpdatedAt)
	}
}
