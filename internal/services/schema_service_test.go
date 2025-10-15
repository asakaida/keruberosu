package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// Mock SchemaRepository
type mockSchemaRepository struct {
	schemas       map[string]map[string]*entities.Schema // tenantID -> version -> schema
	versionCounts map[string]int                         // tenantID -> version count
}

func newMockSchemaRepository() *mockSchemaRepository {
	return &mockSchemaRepository{
		schemas:       make(map[string]map[string]*entities.Schema),
		versionCounts: make(map[string]int),
	}
}

func (m *mockSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
	if m.schemas[tenantID] == nil {
		m.schemas[tenantID] = make(map[string]*entities.Schema)
	}

	// Generate simple version ID (v1, v2, etc.)
	m.versionCounts[tenantID]++
	version := fmt.Sprintf("v%d", m.versionCounts[tenantID])

	m.schemas[tenantID][version] = &entities.Schema{
		TenantID: tenantID,
		Version:  version,
		DSL:      schemaDSL,
	}
	return version, nil
}

func (m *mockSchemaRepository) GetLatestVersion(ctx context.Context, tenantID string) (*entities.Schema, error) {
	versions, exists := m.schemas[tenantID]
	if !exists || len(versions) == 0 {
		return nil, fmt.Errorf("schema not found for tenant %s: %w", tenantID, repositories.ErrNotFound)
	}

	// Return the latest version (highest version number)
	latestVersion := fmt.Sprintf("v%d", m.versionCounts[tenantID])
	schema, exists := versions[latestVersion]
	if !exists {
		return nil, fmt.Errorf("schema not found for tenant %s: %w", tenantID, repositories.ErrNotFound)
	}
	return schema, nil
}

func (m *mockSchemaRepository) GetByVersion(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
	versions, exists := m.schemas[tenantID]
	if !exists {
		return nil, fmt.Errorf("schema version %s not found for tenant %s: %w", version, tenantID, repositories.ErrNotFound)
	}

	schema, exists := versions[version]
	if !exists {
		return nil, fmt.Errorf("schema version %s not found for tenant %s: %w", version, tenantID, repositories.ErrNotFound)
	}
	return schema, nil
}

func (m *mockSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	return m.GetLatestVersion(ctx, tenantID)
}

func (m *mockSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	if _, exists := m.schemas[tenantID]; !exists {
		return fmt.Errorf("schema not found for tenant %s", tenantID)
	}
	delete(m.schemas, tenantID)
	delete(m.versionCounts, tenantID)
	return nil
}

func TestSchemaService_WriteSchema_Create(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	schemaDSL := `entity user {
}
entity document {
  relation owner @user
  permission view = owner
}`

	version, err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version == "" {
		t.Fatal("expected version to be returned")
	}

	// Verify schema was created
	schema, err := repo.GetLatestVersion(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected schema to be created")
	}

	if schema.DSL != schemaDSL {
		t.Errorf("schema DSL mismatch: got %s, want %s", schema.DSL, schemaDSL)
	}

	if schema.Version != version {
		t.Errorf("version mismatch: got %s, want %s", schema.Version, version)
	}
}

func TestSchemaService_WriteSchema_MultipleVersions(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	// Create initial schema
	initialDSL := `entity user {
}`
	version1, err := service.WriteSchema(context.Background(), "test-tenant", initialDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create new version with updated schema
	updatedDSL := `entity user {
}
entity document {
  relation owner @user
}`
	version2, err := service.WriteSchema(context.Background(), "test-tenant", updatedDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify different versions were created
	if version1 == version2 {
		t.Error("expected different versions for different writes")
	}

	// Verify latest version has updated DSL
	latestSchema, err := repo.GetLatestVersion(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if latestSchema.DSL != updatedDSL {
		t.Errorf("latest schema DSL mismatch: got %s, want %s", latestSchema.DSL, updatedDSL)
	}

	// Verify old version still exists
	oldSchema, err := repo.GetByVersion(context.Background(), "test-tenant", version1)
	if err != nil {
		t.Fatalf("unexpected error getting old version: %v", err)
	}

	if oldSchema.DSL != initialDSL {
		t.Errorf("old schema DSL mismatch: got %s, want %s", oldSchema.DSL, initialDSL)
	}
}

func TestSchemaService_WriteSchema_InvalidDSL(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	invalidDSL := `entity user {
  invalid syntax here
}`

	_, err := service.WriteSchema(context.Background(), "test-tenant", invalidDSL)
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

	_, err := service.WriteSchema(context.Background(), "test-tenant", invalidDSL)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSchemaService_WriteSchema_MissingTenantID(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.WriteSchema(context.Background(), "", "entity user {}")
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}

func TestSchemaService_WriteSchema_MissingDSL(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.WriteSchema(context.Background(), "test-tenant", "")
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
	_, err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read schema (returns *entities.Schema now)
	result, err := service.ReadSchema(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected schema to be returned")
	}

	if result.DSL != schemaDSL {
		t.Errorf("schema DSL mismatch: got %s, want %s", result.DSL, schemaDSL)
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
  relation owner @user
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
	_, err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete schema
	err = service.DeleteSchema(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify schema was deleted - should return ErrNotFound
	schema, err := repo.GetByTenant(context.Background(), "test-tenant")
	if err == nil {
		t.Fatal("expected ErrNotFound after deletion")
	}

	if schema != nil {
		t.Fatal("expected nil schema after deletion")
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
  relation owner @user
  permission view = owner
}`

	// Create schema
	version, err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get schema entity (latest version)
	schema, err := service.GetSchemaEntity(context.Background(), "test-tenant", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected schema entity to be returned")
	}

	if schema.TenantID != "test-tenant" {
		t.Errorf("tenant ID mismatch: got %s, want test-tenant", schema.TenantID)
	}

	if schema.Version != version {
		t.Errorf("version mismatch: got %s, want %s", schema.Version, version)
	}

	// Get schema entity (specific version)
	schemaByVersion, err := service.GetSchemaEntity(context.Background(), "test-tenant", version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schemaByVersion.Version != version {
		t.Errorf("version mismatch: got %s, want %s", schemaByVersion.Version, version)
	}
}

func TestSchemaService_GetSchemaEntity_MultipleVersions(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	// Create first version
	schemaDSL1 := `entity user {
}`
	version1, err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create second version
	schemaDSL2 := `entity user {
}
entity document {
  relation owner @user
}`
	version2, err := service.WriteSchema(context.Background(), "test-tenant", schemaDSL2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get latest (should be version2)
	latestSchema, err := service.GetSchemaEntity(context.Background(), "test-tenant", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if latestSchema.Version != version2 {
		t.Errorf("expected latest version %s, got %s", version2, latestSchema.Version)
	}

	// Get old version explicitly
	oldSchema, err := service.GetSchemaEntity(context.Background(), "test-tenant", version1)
	if err != nil {
		t.Fatalf("unexpected error getting old version: %v", err)
	}

	if oldSchema.Version != version1 {
		t.Errorf("expected version %s, got %s", version1, oldSchema.Version)
	}
}

func TestSchemaService_GetSchemaEntity_NotFound(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.GetSchemaEntity(context.Background(), "nonexistent-tenant", "")
	if err == nil {
		t.Fatal("expected error for nonexistent schema")
	}
}

func TestSchemaService_GetSchemaEntity_MissingTenantID(t *testing.T) {
	repo := newMockSchemaRepository()
	service := NewSchemaService(repo)

	_, err := service.GetSchemaEntity(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}
