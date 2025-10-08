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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

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
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)
	checker := NewChecker(evaluator)

	req := &CheckRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		SubjectType: "user",
		SubjectID:  "alice",
	}

	permissions := []string{"view", "nonexistent", "delete"}
	results, err := checker.CheckMultiple(context.Background(), req, permissions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nonexistent permission should return false
	if results["nonexistent"] != false {
		t.Error("expected nonexistent permission to return false")
	}
}
