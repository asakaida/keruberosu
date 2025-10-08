package services

import (
	"context"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
)

// Mock SchemaRepository
type mockSchemaRepository struct {
	schemas map[string]*entities.Schema
}

func newMockSchemaRepository() *mockSchemaRepository {
	return &mockSchemaRepository{
		schemas: make(map[string]*entities.Schema),
	}
}

func (m *mockSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) error {
	m.schemas[tenantID] = &entities.Schema{
		TenantID: tenantID,
		DSL:      schemaDSL,
	}
	return nil
}

func (m *mockSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	schema, exists := m.schemas[tenantID]
	if !exists {
		return nil, nil
	}
	return schema, nil
}

func (m *mockSchemaRepository) Update(ctx context.Context, tenantID string, schemaDSL string) error {
	if _, exists := m.schemas[tenantID]; !exists {
		return nil
	}
	m.schemas[tenantID].DSL = schemaDSL
	return nil
}

func (m *mockSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	delete(m.schemas, tenantID)
	return nil
}

func TestSchemaService_WriteSchema_Create(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	schemaDSL := `entity user {
}
entity document {
  relation owner: user
  permission view = owner
}`

	err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify schema was created
	schema, err := repo.GetByTenant(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected schema to be created")
	}

	if schema.DSL != schemaDSL {
		t.Errorf("schema DSL mismatch: got %s, want %s", schema.DSL, schemaDSL)
	}
}

func TestSchemaService_WriteSchema_Update(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	// Create initial schema
	initialDSL := `entity user {
}`
	err := service.WriteSchema(context.Background(), "test-tenant", initialDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update schema
	updatedDSL := `entity user {
}
entity document {
  relation owner: user
}`
	err = service.WriteSchema(context.Background(), "test-tenant", updatedDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify schema was updated
	schema, err := repo.GetByTenant(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema.DSL != updatedDSL {
		t.Errorf("schema DSL mismatch: got %s, want %s", schema.DSL, updatedDSL)
	}
}

func TestSchemaService_WriteSchema_InvalidDSL(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	invalidDSL := `entity user {
  invalid syntax here
}`

	err := service.WriteSchema(context.Background(), "test-tenant", invalidDSL)
	if err == nil {
		t.Fatal("expected error for invalid DSL")
	}
}

func TestSchemaService_WriteSchema_ValidationError(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	// DSL with validation error (undefined relation reference)
	invalidDSL := `entity document {
  permission view = owner
}`

	err := service.WriteSchema(context.Background(), "test-tenant", invalidDSL)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSchemaService_WriteSchema_MissingTenantID(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	err := service.WriteSchema(context.Background(), "", "entity user {}")
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}

func TestSchemaService_WriteSchema_MissingDSL(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	err := service.WriteSchema(context.Background(), "test-tenant", "")
	if err == nil {
		t.Fatal("expected error for missing DSL")
	}
}

func TestSchemaService_ReadSchema(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	schemaDSL := `entity user {
}`

	// Create schema
	err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read schema
	result, err := service.ReadSchema(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != schemaDSL {
		t.Errorf("schema DSL mismatch: got %s, want %s", result, schemaDSL)
	}
}

func TestSchemaService_ReadSchema_NotFound(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.ReadSchema(context.Background(), "nonexistent-tenant")
	if err == nil {
		t.Fatal("expected error for nonexistent schema")
	}
}

func TestSchemaService_ReadSchema_MissingTenantID(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.ReadSchema(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}

func TestSchemaService_ValidateSchema_Valid(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	validDSL := `entity user {
}
entity document {
  relation owner: user
  permission view = owner
}`

	err := service.ValidateSchema(context.Background(), validDSL)
	if err != nil {
		t.Fatalf("unexpected error for valid DSL: %v", err)
	}
}

func TestSchemaService_ValidateSchema_Invalid(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	invalidDSL := `entity user {
  invalid syntax
}`

	err := service.ValidateSchema(context.Background(), invalidDSL)
	if err == nil {
		t.Fatal("expected error for invalid DSL")
	}
}

func TestSchemaService_ValidateSchema_Missing(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	err := service.ValidateSchema(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for missing DSL")
	}
}

func TestSchemaService_DeleteSchema(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	schemaDSL := `entity user {
}`

	// Create schema
	err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete schema
	err = service.DeleteSchema(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify schema was deleted
	schema, err := repo.GetByTenant(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema != nil {
		t.Fatal("expected schema to be deleted")
	}
}

func TestSchemaService_DeleteSchema_MissingTenantID(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	err := service.DeleteSchema(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}

func TestSchemaService_GetSchemaEntity(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	schemaDSL := `entity user {
}
entity document {
  relation owner: user
  permission view = owner
}`

	// Create schema
	err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get schema entity
	schema, err := service.GetSchemaEntity(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected schema entity to be returned")
	}

	if schema.TenantID != "test-tenant" {
		t.Errorf("tenant ID mismatch: got %s, want test-tenant", schema.TenantID)
	}
}

func TestSchemaService_GetSchemaEntity_NotFound(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.GetSchemaEntity(context.Background(), "nonexistent-tenant")
	if err == nil {
		t.Fatal("expected error for nonexistent schema")
	}
}

func TestSchemaService_GetSchemaEntity_MissingTenantID(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.GetSchemaEntity(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}
