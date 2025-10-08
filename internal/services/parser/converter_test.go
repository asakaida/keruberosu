package parser

import (
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
)

func TestASTToSchema_Basic(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "user",
			},
			{
				Name: "document",
				Relations: []*RelationAST{
					{Name: "owner", TargetType: "user"},
				},
				Permissions: []*PermissionAST{
					{
						Name: "view",
						Rule: &RelationPermissionAST{Relation: "owner"},
					},
				},
			},
		},
	}

	schema, err := ASTToSchema("test-tenant", ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema.TenantID != "test-tenant" {
		t.Errorf("expected tenant ID 'test-tenant', got %s", schema.TenantID)
	}

	if len(schema.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(schema.Entities))
	}

	// Check user entity
	userEntity := schema.Entities[0]
	if userEntity.Name != "user" {
		t.Errorf("expected entity name 'user', got %s", userEntity.Name)
	}

	// Check document entity
	docEntity := schema.Entities[1]
	if docEntity.Name != "document" {
		t.Errorf("expected entity name 'document', got %s", docEntity.Name)
	}

	if len(docEntity.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(docEntity.Relations))
	}

	if docEntity.Relations[0].Name != "owner" {
		t.Errorf("expected relation name 'owner', got %s", docEntity.Relations[0].Name)
	}

	if len(docEntity.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(docEntity.Permissions))
	}

	if docEntity.Permissions[0].Name != "view" {
		t.Errorf("expected permission name 'view', got %s", docEntity.Permissions[0].Name)
	}
}

func TestASTToSchema_WithAttributes(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Attributes: []*AttributeAST{
					{Name: "public", Type: "boolean"},
					{Name: "title", Type: "string"},
				},
			},
		},
	}

	schema, err := ASTToSchema("test-tenant", ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entity := schema.Entities[0]
	if len(entity.AttributeSchemas) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(entity.AttributeSchemas))
	}

	if entity.AttributeSchemas[0].Name != "public" || entity.AttributeSchemas[0].Type != "boolean" {
		t.Errorf("expected attribute (public, boolean), got (%s, %s)",
			entity.AttributeSchemas[0].Name, entity.AttributeSchemas[0].Type)
	}
}

func TestASTToSchema_LogicalRule(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Relations: []*RelationAST{
					{Name: "owner", TargetType: "user"},
					{Name: "editor", TargetType: "user"},
				},
				Permissions: []*PermissionAST{
					{
						Name: "edit",
						Rule: &LogicalPermissionAST{
							Operator: "or",
							Left:     &RelationPermissionAST{Relation: "owner"},
							Right:    &RelationPermissionAST{Relation: "editor"},
						},
					},
				},
			},
		},
	}

	schema, err := ASTToSchema("test-tenant", ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	permission := schema.Entities[0].Permissions[0]
	logicalRule, ok := permission.Rule.(*entities.LogicalRule)
	if !ok {
		t.Fatalf("expected LogicalRule, got %T", permission.Rule)
	}

	if logicalRule.Operator != "or" {
		t.Errorf("expected operator 'or', got %s", logicalRule.Operator)
	}

	leftRule, ok := logicalRule.Left.(*entities.RelationRule)
	if !ok {
		t.Fatalf("expected left to be RelationRule, got %T", logicalRule.Left)
	}

	if leftRule.Relation != "owner" {
		t.Errorf("expected left relation 'owner', got %s", leftRule.Relation)
	}
}

func TestASTToSchema_HierarchicalRule(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Relations: []*RelationAST{
					{Name: "parent", TargetType: "folder"},
				},
				Permissions: []*PermissionAST{
					{
						Name: "view",
						Rule: &HierarchicalPermissionAST{
							Relation:   "parent",
							Permission: "view",
						},
					},
				},
			},
		},
	}

	schema, err := ASTToSchema("test-tenant", ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	permission := schema.Entities[0].Permissions[0]
	hierarchicalRule, ok := permission.Rule.(*entities.HierarchicalRule)
	if !ok {
		t.Fatalf("expected HierarchicalRule, got %T", permission.Rule)
	}

	if hierarchicalRule.Relation != "parent" {
		t.Errorf("expected relation 'parent', got %s", hierarchicalRule.Relation)
	}

	if hierarchicalRule.Permission != "view" {
		t.Errorf("expected permission 'view', got %s", hierarchicalRule.Permission)
	}
}

func TestASTToSchema_ABACRule(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Permissions: []*PermissionAST{
					{
						Name: "view",
						Rule: &RulePermissionAST{
							Expression: "resource.public == true",
						},
					},
				},
			},
		},
	}

	schema, err := ASTToSchema("test-tenant", ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	permission := schema.Entities[0].Permissions[0]
	abacRule, ok := permission.Rule.(*entities.ABACRule)
	if !ok {
		t.Fatalf("expected ABACRule, got %T", permission.Rule)
	}

	if abacRule.Expression != "resource.public == true" {
		t.Errorf("expected expression 'resource.public == true', got %s", abacRule.Expression)
	}
}

func TestSchemaToAST_Basic(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
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

	ast, err := SchemaToAST(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ast.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(ast.Entities))
	}

	// Check user entity
	if ast.Entities[0].Name != "user" {
		t.Errorf("expected entity name 'user', got %s", ast.Entities[0].Name)
	}

	// Check document entity
	docEntity := ast.Entities[1]
	if docEntity.Name != "document" {
		t.Errorf("expected entity name 'document', got %s", docEntity.Name)
	}

	if len(docEntity.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(docEntity.Relations))
	}

	if docEntity.Relations[0].Name != "owner" {
		t.Errorf("expected relation name 'owner', got %s", docEntity.Relations[0].Name)
	}

	if len(docEntity.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(docEntity.Permissions))
	}

	permRule, ok := docEntity.Permissions[0].Rule.(*RelationPermissionAST)
	if !ok {
		t.Fatalf("expected RelationPermissionAST, got %T", docEntity.Permissions[0].Rule)
	}

	if permRule.Relation != "owner" {
		t.Errorf("expected relation 'owner', got %s", permRule.Relation)
	}
}

func TestSchemaToAST_LogicalRule(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{
				Name: "document",
				Permissions: []*entities.Permission{
					{
						Name: "edit",
						Rule: &entities.LogicalRule{
							Operator: "or",
							Left:     &entities.RelationRule{Relation: "owner"},
							Right:    &entities.RelationRule{Relation: "editor"},
						},
					},
				},
			},
		},
	}

	ast, err := SchemaToAST(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	permission := ast.Entities[0].Permissions[0]
	logicalRule, ok := permission.Rule.(*LogicalPermissionAST)
	if !ok {
		t.Fatalf("expected LogicalPermissionAST, got %T", permission.Rule)
	}

	if logicalRule.Operator != "or" {
		t.Errorf("expected operator 'or', got %s", logicalRule.Operator)
	}
}

func TestSchemaToAST_HierarchicalRule(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{
				Name: "document",
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

	ast, err := SchemaToAST(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	permission := ast.Entities[0].Permissions[0]
	hierarchicalRule, ok := permission.Rule.(*HierarchicalPermissionAST)
	if !ok {
		t.Fatalf("expected HierarchicalPermissionAST, got %T", permission.Rule)
	}

	if hierarchicalRule.Relation != "parent" {
		t.Errorf("expected relation 'parent', got %s", hierarchicalRule.Relation)
	}

	if hierarchicalRule.Permission != "view" {
		t.Errorf("expected permission 'view', got %s", hierarchicalRule.Permission)
	}
}

func TestSchemaToAST_ABACRule(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
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

	ast, err := SchemaToAST(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	permission := ast.Entities[0].Permissions[0]
	abacRule, ok := permission.Rule.(*RulePermissionAST)
	if !ok {
		t.Fatalf("expected RulePermissionAST, got %T", permission.Rule)
	}

	if abacRule.Expression != "resource.public == true" {
		t.Errorf("expected expression 'resource.public == true', got %s", abacRule.Expression)
	}
}

func TestRoundTrip_ASTToSchemaToAST(t *testing.T) {
	// Original AST
	originalAST := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Relations: []*RelationAST{
					{Name: "owner", TargetType: "user"},
					{Name: "editor", TargetType: "user"},
				},
				Attributes: []*AttributeAST{
					{Name: "public", Type: "boolean"},
				},
				Permissions: []*PermissionAST{
					{
						Name: "view",
						Rule: &LogicalPermissionAST{
							Operator: "or",
							Left:     &RelationPermissionAST{Relation: "owner"},
							Right:    &RulePermissionAST{Expression: "resource.public == true"},
						},
					},
				},
			},
		},
	}

	// Convert AST → Schema
	schema, err := ASTToSchema("test-tenant", originalAST)
	if err != nil {
		t.Fatalf("ASTToSchema failed: %v", err)
	}

	// Convert Schema → AST
	resultAST, err := SchemaToAST(schema)
	if err != nil {
		t.Fatalf("SchemaToAST failed: %v", err)
	}

	// Verify the result matches the original
	if len(resultAST.Entities) != len(originalAST.Entities) {
		t.Errorf("entity count mismatch: got %d, want %d",
			len(resultAST.Entities), len(originalAST.Entities))
	}

	resultEntity := resultAST.Entities[0]
	originalEntity := originalAST.Entities[0]

	if resultEntity.Name != originalEntity.Name {
		t.Errorf("entity name mismatch: got %s, want %s",
			resultEntity.Name, originalEntity.Name)
	}

	if len(resultEntity.Relations) != len(originalEntity.Relations) {
		t.Errorf("relation count mismatch: got %d, want %d",
			len(resultEntity.Relations), len(originalEntity.Relations))
	}

	if len(resultEntity.Attributes) != len(originalEntity.Attributes) {
		t.Errorf("attribute count mismatch: got %d, want %d",
			len(resultEntity.Attributes), len(originalEntity.Attributes))
	}

	if len(resultEntity.Permissions) != len(originalEntity.Permissions) {
		t.Errorf("permission count mismatch: got %d, want %d",
			len(resultEntity.Permissions), len(originalEntity.Permissions))
	}
}
