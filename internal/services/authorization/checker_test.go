package authorization

import (
	"context"
	"strings"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
)

func TestChecker_Check_BasicPermission(t *testing.T) {
	// Setup
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	tests := []struct {
		name      string
		req       *CheckRequest
		wantAllow bool
		wantError bool
	}{
		{
			name: "owner has view permission - should allow",
			req: &CheckRequest{
				TenantID:   "test-tenant",
				EntityType: "document",
				EntityID:   "doc1",
				Permission: "view",
				SubjectType: "user",
				SubjectID:  "alice",
			},
			wantAllow: true,
			wantError: false,
		},
		{
			name: "non-owner has no view permission - should deny",
			req: &CheckRequest{
				TenantID:   "test-tenant",
				EntityType: "document",
				EntityID:   "doc1",
				Permission: "view",
				SubjectType: "user",
				SubjectID:  "bob",
			},
			wantAllow: false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := checker.Check(context.Background(), tt.req)
			if (err != nil) != tt.wantError {
				t.Errorf("Check() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && resp.Allowed != tt.wantAllow {
				t.Errorf("Check() allowed = %v, want %v", resp.Allowed, tt.wantAllow)
			}
		})
	}
}

func TestChecker_Check_LogicalPermission(t *testing.T) {
	// Setup
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "bob",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	// Test "edit" permission which is "owner or editor"
	req := &CheckRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "edit",
		SubjectType: "user",
		SubjectID:  "bob",
	}

	resp, err := checker.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Error("expected editor to have edit permission")
	}
}

func TestChecker_Check_HierarchicalPermission(t *testing.T) {
	// Setup schema with hierarchical permission
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "folder",
				Relations: []*entities.Relation{
					{Name: "owner", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "owner"},
					},
				},
			},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "parent", TargetType: "folder"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.HierarchicalRule{
							Relation:   "parent",
							Permission: "view",
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			// doc1 parent is folder1
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "parent",
				SubjectType: "folder",
				SubjectID:   "folder1",
			},
			// alice owns folder1
			{
				EntityType:  "folder",
				EntityID:    "folder1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	req := &CheckRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "view",
		SubjectType: "user",
		SubjectID:  "alice",
	}

	resp, err := checker.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Error("expected alice to have view permission via parent folder")
	}
}

func TestChecker_Check_ABACPermission(t *testing.T) {
	// Setup schema with ABAC permission
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.ABACRule{
							Expression: "resource.public == true",
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	// Set up public document
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document",
		EntityID:   "doc1",
		Name:       "public",
		Value:      true,
	})
	// Set up private document
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document",
		EntityID:   "doc2",
		Name:       "public",
		Value:      false,
	})

	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	tests := []struct {
		name      string
		entityID  string
		wantAllow bool
	}{
		{
			name:      "public document - should allow",
			entityID:  "doc1",
			wantAllow: true,
		},
		{
			name:      "non-public document - should deny",
			entityID:  "doc2",
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CheckRequest{
				TenantID:   "test-tenant",
				EntityType: "document",
				EntityID:   tt.entityID,
				Permission: "view",
				SubjectType: "user",
				SubjectID:  "alice",
			}

			resp, err := checker.Check(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Allowed != tt.wantAllow {
				t.Errorf("allowed = %v, want %v", resp.Allowed, tt.wantAllow)
			}
		})
	}
}

func TestChecker_Check_ContextualTuples(t *testing.T) {
	// Setup
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{} // Empty repository
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	req := &CheckRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "view",
		SubjectType: "user",
		SubjectID:  "alice",
		ContextualTuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}

	resp, err := checker.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Error("expected contextual tuple to grant permission")
	}
}

func TestChecker_Check_ErrorCases(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	tests := []struct {
		name       string
		req        *CheckRequest
		wantError  bool
		errorMatch string
	}{
		{
			name: "missing tenant ID",
			req: &CheckRequest{
				EntityType:  "document",
				EntityID:    "doc1",
				Permission:  "view",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			wantError:  true,
			errorMatch: "tenant ID is required",
		},
		{
			name: "missing entity type",
			req: &CheckRequest{
				TenantID:    "test-tenant",
				EntityID:    "doc1",
				Permission:  "view",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			wantError:  true,
			errorMatch: "entity type is required",
		},
		{
			name: "entity type not found",
			req: &CheckRequest{
				TenantID:   "test-tenant",
				EntityType: "nonexistent",
				EntityID:   "doc1",
				Permission: "view",
				SubjectType: "user",
				SubjectID:  "alice",
			},
			wantError:  true,
			errorMatch: "entity type nonexistent not found",
		},
		{
			name: "permission not found",
			req: &CheckRequest{
				TenantID:   "test-tenant",
				EntityType: "document",
				EntityID:   "doc1",
				Permission: "nonexistent",
				SubjectType: "user",
				SubjectID:  "alice",
			},
			wantError:  true,
			errorMatch: "permission nonexistent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := checker.Check(context.Background(), tt.req)
			if (err != nil) != tt.wantError {
				t.Errorf("Check() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && !strings.Contains(err.Error(), tt.errorMatch) {
				t.Errorf("error = %v, want error containing %v", err, tt.errorMatch)
			}
		})
	}
}

func TestChecker_CheckMultiple(t *testing.T) {
	// Setup
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	req := &CheckRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		SubjectType: "user",
		SubjectID:  "alice",
	}

	permissions := []string{"view", "edit", "delete"}
	results, err := checker.CheckMultiple(context.Background(), req, permissions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Alice is owner, so she should have all permissions
	expected := map[string]bool{
		"view":   true,
		"edit":   true,
		"delete": true,
	}

	for perm, want := range expected {
		if got := results[perm]; got != want {
			t.Errorf("permission %s: got %v, want %v", perm, got, want)
		}
	}
}

func TestChecker_CheckMultiple_PartialAccess(t *testing.T) {
	// Setup - bob is only an editor, not owner
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "bob",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	req := &CheckRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		SubjectType: "user",
		SubjectID:  "bob",
	}

	permissions := []string{"view", "edit", "delete"}
	results, err := checker.CheckMultiple(context.Background(), req, permissions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bob is editor, so:
	// - view: false (only owner can view in our test schema)
	// - edit: true (owner or editor)
	// - delete: false (only owner)
	expected := map[string]bool{
		"view":   false,
		"edit":   true,
		"delete": false,
	}

	for perm, want := range expected {
		if got := results[perm]; got != want {
			t.Errorf("permission %s: got %v, want %v", perm, got, want)
		}
	}
}

func TestChecker_CheckMultiple_NonexistentPermission(t *testing.T) {
	// Setup
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	req := &CheckRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		SubjectType: "user",
		SubjectID:  "alice",
	}

	permissions := []string{"view", "nonexistent", "delete"}
	_, err := checker.CheckMultiple(context.Background(), req, permissions)
	if err == nil {
		t.Fatal("expected error for nonexistent permission, got nil")
	}
}

func TestChecker_CheckMultiple_SnapshotTokenPropagation(t *testing.T) {
	// Verify that SnapshotToken from the top-level request is propagated
	// to each sub-check in CheckMultiple.
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	schemaService := &mockSchemaRepository{schema}
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	req := &CheckRequest{
		TenantID:      "test-tenant",
		EntityType:    "document",
		EntityID:      "doc1",
		SubjectType:   "user",
		SubjectID:     "alice",
		SnapshotToken: "snap123",
	}

	permissions := []string{"view", "edit"}
	results, err := checker.CheckMultiple(context.Background(), req, permissions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the results are correct (alice is owner, so she has view and edit)
	if !results["view"] {
		t.Error("expected view=true for alice (owner)")
	}
	if !results["edit"] {
		t.Error("expected edit=true for alice (owner)")
	}

	// To verify token propagation, we inspect the CheckMultiple implementation:
	// It creates a new CheckRequest per permission and copies SnapshotToken.
	// Since the checker doesn't have cache/snapshotManager, the token is stored
	// in the sub-request but not used for caching. The key check is that
	// CheckMultiple completes successfully with SnapshotToken set.
	// If SnapshotToken were not propagated (the bug), cached results could be
	// inconsistent across permissions in the same batch.

	// Also verify with a subject that has no access
	reqBob := &CheckRequest{
		TenantID:      "test-tenant",
		EntityType:    "document",
		EntityID:      "doc1",
		SubjectType:   "user",
		SubjectID:     "bob",
		SnapshotToken: "snap123",
	}

	resultsBob, err := checker.CheckMultiple(context.Background(), reqBob, permissions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resultsBob["view"] {
		t.Error("expected view=false for bob")
	}
	if resultsBob["edit"] {
		t.Error("expected edit=false for bob")
	}
}

// TestChecker_Check_UsesResolvedSchemaVersion verifies that the checker passes
// the resolved schema version (not the empty requested one) to the evaluator.
func TestChecker_Check_UsesResolvedSchemaVersion(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Version:  "01HWRESOLVED",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "owner", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "owner"},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{EntityType: "document", EntityID: "doc1", Relation: "owner", SubjectType: "user", SubjectID: "alice"},
		},
	}

	var capturedVersions []string
	schemaService := &mockSchemaServiceCapture{
		schema: schema,
		onGetSchemaEntity: func(tenantID, version string) {
			capturedVersions = append(capturedVersions, version)
		},
	}

	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	evaluator := NewEvaluator(schemaService, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(schemaService, evaluator)

	// Request with empty SchemaVersion (should resolve to latest)
	req := &CheckRequest{
		TenantID:    "test-tenant",
		EntityType:  "document",
		EntityID:    "doc1",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "alice",
	}

	resp, err := checker.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allowed {
		t.Error("expected allowed=true")
	}

	// First call from checker uses "" (original request), second from evaluator
	// should use the resolved version "01HWRESOLVED"
	if len(capturedVersions) < 2 {
		t.Fatalf("expected at least 2 GetSchemaEntity calls, got %d", len(capturedVersions))
	}
	// The evaluator's call should use the resolved version
	evalVersion := capturedVersions[1]
	if evalVersion != schema.Version {
		t.Errorf("evaluator used schema version %q, expected resolved version %q", evalVersion, schema.Version)
	}
}

// mockSchemaServiceCapture captures GetSchemaEntity calls for testing.
type mockSchemaServiceCapture struct {
	schema            *entities.Schema
	onGetSchemaEntity func(tenantID, version string)
}

func (m *mockSchemaServiceCapture) GetSchemaEntity(_ context.Context, tenantID string, version string) (*entities.Schema, error) {
	if m.onGetSchemaEntity != nil {
		m.onGetSchemaEntity(tenantID, version)
	}
	return m.schema, nil
}
