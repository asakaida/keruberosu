package authorization

import (
	"context"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
)

func TestExpander_Expand_BasicRelation(t *testing.T) {
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
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "bob",
			},
		},
	}

	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "view",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// view = owner (RelationRule)
	// Should expand to a union of all users with owner relation
	if resp.Tree.Type != "union" {
		t.Errorf("expected union node, got %s", resp.Tree.Type)
	}
	if resp.Tree.Relation != "owner" {
		t.Errorf("expected relation 'owner', got %s", resp.Tree.Relation)
	}
	if len(resp.Tree.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(resp.Tree.Children))
	}

	// Check that both alice and bob are in the tree
	subjects := make(map[string]bool)
	for _, child := range resp.Tree.Children {
		if child.Type != "leaf" {
			t.Errorf("expected leaf node, got %s", child.Type)
		}
		subjects[child.Subject] = true
	}

	if !subjects["user:alice"] || !subjects["user:bob"] {
		t.Errorf("expected subjects user:alice and user:bob, got %v", subjects)
	}
}

func TestExpander_Expand_LogicalOR(t *testing.T) {
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
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "bob",
			},
		},
	}

	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "edit",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// edit = owner or editor (LogicalRule with OR)
	if resp.Tree.Type != "union" {
		t.Errorf("expected union node, got %s", resp.Tree.Type)
	}
	if resp.Tree.Relation != "or" {
		t.Errorf("expected relation 'or', got %s", resp.Tree.Relation)
	}
	if len(resp.Tree.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(resp.Tree.Children))
	}

	// First child should be owner relation
	if resp.Tree.Children[0].Relation != "owner" {
		t.Errorf("expected first child to be owner relation, got %s", resp.Tree.Children[0].Relation)
	}

	// Second child should be editor relation
	if resp.Tree.Children[1].Relation != "editor" {
		t.Errorf("expected second child to be editor relation, got %s", resp.Tree.Children[1].Relation)
	}
}

func TestExpander_Expand_LogicalAND(t *testing.T) {
	// Setup schema with AND permission
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "member", TargetType: "user"},
					{Name: "approved", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "access",
						Rule: &entities.LogicalRule{
							Operator: "and",
							Left:     &entities.RelationRule{Relation: "member"},
							Right:    &entities.RelationRule{Relation: "approved"},
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "member",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "approved",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}

	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "access",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Tree.Type != "intersection" {
		t.Errorf("expected intersection node, got %s", resp.Tree.Type)
	}
	if len(resp.Tree.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(resp.Tree.Children))
	}
}

func TestExpander_Expand_LogicalNOT(t *testing.T) {
	// Setup schema with NOT permission
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "blocked", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "restricted",
						Rule: &entities.LogicalRule{
							Operator: "not",
							Left:     &entities.RelationRule{Relation: "blocked"},
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "blocked",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}

	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "restricted",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Tree.Type != "exclusion" {
		t.Errorf("expected exclusion node, got %s", resp.Tree.Type)
	}
	if len(resp.Tree.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(resp.Tree.Children))
	}
}

func TestExpander_Expand_Hierarchical(t *testing.T) {
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
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "parent",
				SubjectType: "folder",
				SubjectID:   "folder1",
			},
			{
				EntityType:  "folder",
				EntityID:    "folder1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}

	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "view",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be a union node for the hierarchical rule
	if resp.Tree.Type != "union" {
		t.Errorf("expected union node, got %s", resp.Tree.Type)
	}
	if resp.Tree.Relation != "parent.view" {
		t.Errorf("expected relation 'parent.view', got %s", resp.Tree.Relation)
	}

	// Should have one child (the parent folder's view permission)
	if len(resp.Tree.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(resp.Tree.Children))
	}

	// The child should be the folder's owner relation
	childNode := resp.Tree.Children[0]
	if childNode.Type != "union" {
		t.Errorf("expected child to be union node, got %s", childNode.Type)
	}
	if childNode.Relation != "owner" {
		t.Errorf("expected child relation 'owner', got %s", childNode.Relation)
	}
	if childNode.Entity != "folder:folder1" {
		t.Errorf("expected child entity 'folder:folder1', got %s", childNode.Entity)
	}
}

func TestExpander_Expand_ABAC(t *testing.T) {
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
	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "view",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ABAC rules should return a leaf node with the expression
	if resp.Tree.Type != "leaf" {
		t.Errorf("expected leaf node, got %s", resp.Tree.Type)
	}
	if resp.Tree.Relation != "abac" {
		t.Errorf("expected relation 'abac', got %s", resp.Tree.Relation)
	}
	if resp.Tree.Subject != "expression:resource.public == true" {
		t.Errorf("expected subject with expression, got %s", resp.Tree.Subject)
	}
}

func TestExpander_Expand_EmptyRelation(t *testing.T) {
	// Setup with no relation tuples
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{},
	}

	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "view",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return a union node with no children
	if resp.Tree.Type != "union" {
		t.Errorf("expected union node, got %s", resp.Tree.Type)
	}
	if len(resp.Tree.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(resp.Tree.Children))
	}
}

func TestExpander_Expand_ComplexNested(t *testing.T) {
	// Setup schema with complex nested permissions
	// delete = owner and (editor or admin)
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "owner", TargetType: "user"},
					{Name: "editor", TargetType: "user"},
					{Name: "admin", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "delete",
						Rule: &entities.LogicalRule{
							Operator: "and",
							Left:     &entities.RelationRule{Relation: "owner"},
							Right: &entities.LogicalRule{
								Operator: "or",
								Left:     &entities.RelationRule{Relation: "editor"},
								Right:    &entities.RelationRule{Relation: "admin"},
							},
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "bob",
			},
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "admin",
				SubjectType: "user",
				SubjectID:   "charlie",
			},
		},
	}

	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "document",
		EntityID:   "doc1",
		Permission: "delete",
	}

	resp, err := expander.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Root should be intersection (AND)
	if resp.Tree.Type != "intersection" {
		t.Errorf("expected intersection node, got %s", resp.Tree.Type)
	}
	if len(resp.Tree.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(resp.Tree.Children))
	}

	// First child should be owner relation
	if resp.Tree.Children[0].Relation != "owner" {
		t.Errorf("expected first child to be owner, got %s", resp.Tree.Children[0].Relation)
	}

	// Second child should be union (OR)
	secondChild := resp.Tree.Children[1]
	if secondChild.Type != "union" {
		t.Errorf("expected second child to be union, got %s", secondChild.Type)
	}
	if len(secondChild.Children) != 2 {
		t.Errorf("expected second child to have 2 children, got %d", len(secondChild.Children))
	}
}

func TestExpander_Expand_ErrorCases(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{}
	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	tests := []struct {
		name       string
		req        *ExpandRequest
		wantError  bool
		errorMatch string
	}{
		{
			name: "missing tenant ID",
			req: &ExpandRequest{
				EntityType: "document",
				EntityID:   "doc1",
				Permission: "view",
			},
			wantError:  true,
			errorMatch: "tenant ID is required",
		},
		{
			name: "missing entity type",
			req: &ExpandRequest{
				TenantID:   "test-tenant",
				EntityID:   "doc1",
				Permission: "view",
			},
			wantError:  true,
			errorMatch: "entity type is required",
		},
		{
			name: "missing entity ID",
			req: &ExpandRequest{
				TenantID:   "test-tenant",
				EntityType: "document",
				Permission: "view",
			},
			wantError:  true,
			errorMatch: "entity ID is required",
		},
		{
			name: "missing permission",
			req: &ExpandRequest{
				TenantID:   "test-tenant",
				EntityType: "document",
				EntityID:   "doc1",
			},
			wantError:  true,
			errorMatch: "permission is required",
		},
		{
			name: "entity type not found",
			req: &ExpandRequest{
				TenantID:   "test-tenant",
				EntityType: "nonexistent",
				EntityID:   "doc1",
				Permission: "view",
			},
			wantError:  true,
			errorMatch: "entity type nonexistent not found",
		},
		{
			name: "permission not found",
			req: &ExpandRequest{
				TenantID:   "test-tenant",
				EntityType: "document",
				EntityID:   "doc1",
				Permission: "nonexistent",
			},
			wantError:  true,
			errorMatch: "permission nonexistent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := expander.Expand(context.Background(), tt.req)
			if (err != nil) != tt.wantError {
				t.Errorf("Expand() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && err != nil {
				if !contains(err.Error(), tt.errorMatch) {
					t.Errorf("error = %v, want error containing %v", err, tt.errorMatch)
				}
			}
		})
	}
}

func TestParseEntityRef(t *testing.T) {
	tests := []struct {
		name          string
		ref           string
		wantType      string
		wantID        string
		wantError     bool
		errorContains string
	}{
		{
			name:     "valid reference",
			ref:      "document:doc1",
			wantType: "document",
			wantID:   "doc1",
		},
		{
			name:     "valid reference with complex ID",
			ref:      "user:alice@example.com",
			wantType: "user",
			wantID:   "alice@example.com",
		},
		{
			name:          "missing colon",
			ref:           "documentdoc1",
			wantError:     true,
			errorContains: "invalid entity reference format",
		},
		{
			name:          "empty type",
			ref:           ":doc1",
			wantError:     true,
			errorContains: "invalid entity reference",
		},
		{
			name:          "empty ID",
			ref:           "document:",
			wantError:     true,
			errorContains: "invalid entity reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotID, err := parseEntityRef(tt.ref)
			if (err != nil) != tt.wantError {
				t.Errorf("parseEntityRef() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError {
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errorContains)
				}
				return
			}
			if gotType != tt.wantType {
				t.Errorf("parseEntityRef() type = %v, want %v", gotType, tt.wantType)
			}
			if gotID != tt.wantID {
				t.Errorf("parseEntityRef() ID = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestExpander_Expand_MaxDepth(t *testing.T) {
	// Create a schema with deep hierarchical structure
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "folder",
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

	// Create a very deep hierarchy that exceeds MaxDepth
	tuples := make([]*entities.RelationTuple, MaxDepth+5)
	for i := 0; i < MaxDepth+5; i++ {
		tuples[i] = &entities.RelationTuple{
			EntityType:  "folder",
			EntityID:    formatInt(i),
			Relation:    "parent",
			SubjectType: "folder",
			SubjectID:   formatInt(i + 1),
		}
	}

	relationRepo := &mockRelationRepository{tuples: tuples}
	schemaService := &mockSchemaRepository{schema}
	expander := NewExpander(schemaService, relationRepo)

	req := &ExpandRequest{
		TenantID:   "test-tenant",
		EntityType: "folder",
		EntityID:   "0",
		Permission: "view",
	}

	_, err := expander.Expand(context.Background(), req)
	if err == nil {
		t.Error("expected error for exceeding max depth")
	}
	if !contains(err.Error(), "maximum recursion depth exceeded") {
		t.Errorf("expected max depth error, got: %v", err)
	}
}

// Helper function to format int as string
func formatInt(i int) string {
	return string(rune('0' + i%10))
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
