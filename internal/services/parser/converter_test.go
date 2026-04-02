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

func TestASTToSchema_RuleCall(t *testing.T) {
	ast := &SchemaAST{
		Rules: []*RuleDefinitionAST{
			{
				Name:       "is_public",
				Parameters: []string{"resource"},
				Body:       "resource.public == true",
			},
		},
		Entities: []*EntityAST{
			{
				Name: "document",
				Attributes: []*AttributeAST{
					{Name: "public", Type: "boolean"},
				},
				Permissions: []*PermissionAST{
					{
						Name: "view",
						Rule: &RuleCallPermissionAST{
							RuleName:  "is_public",
							Arguments: []string{"resource"},
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

	// Check rule definition
	if len(schema.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(schema.Rules))
	}

	rule := schema.Rules[0]
	if rule.Name != "is_public" {
		t.Errorf("expected rule name 'is_public', got %s", rule.Name)
	}
	if len(rule.Parameters) != 1 || rule.Parameters[0] != "resource" {
		t.Errorf("expected parameters [resource], got %v", rule.Parameters)
	}
	if rule.Body != "resource.public == true" {
		t.Errorf("expected body 'resource.public == true', got %s", rule.Body)
	}

	// Check permission with rule call
	permission := schema.Entities[0].Permissions[0]
	ruleCall, ok := permission.Rule.(*entities.RuleCallRule)
	if !ok {
		t.Fatalf("expected RuleCallRule, got %T", permission.Rule)
	}

	if ruleCall.RuleName != "is_public" {
		t.Errorf("expected rule name 'is_public', got %s", ruleCall.RuleName)
	}
	if len(ruleCall.Arguments) != 1 || ruleCall.Arguments[0] != "resource" {
		t.Errorf("expected arguments [resource], got %v", ruleCall.Arguments)
	}
}

func TestSchemaToAST_RuleCall(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Rules: []*entities.RuleDefinition{
			{
				Name:       "is_public",
				Parameters: []string{"resource"},
				Body:       "resource.public == true",
			},
		},
		Entities: []*entities.Entity{
			{
				Name: "document",
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RuleCallRule{
							RuleName:  "is_public",
							Arguments: []string{"resource"},
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

	// Check rule definition
	if len(ast.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ast.Rules))
	}

	rule := ast.Rules[0]
	if rule.Name != "is_public" {
		t.Errorf("expected rule name 'is_public', got %s", rule.Name)
	}

	// Check permission with rule call
	permission := ast.Entities[0].Permissions[0]
	ruleCall, ok := permission.Rule.(*RuleCallPermissionAST)
	if !ok {
		t.Fatalf("expected RuleCallPermissionAST, got %T", permission.Rule)
	}

	if ruleCall.RuleName != "is_public" {
		t.Errorf("expected rule name 'is_public', got %s", ruleCall.RuleName)
	}
}

func TestConverter_HierarchicalRuleCallRule(t *testing.T) {
	// Test ASTToSchema for HierarchicalRuleCallPermissionAST
	ast := &SchemaAST{
		Rules: []*RuleDefinitionAST{
			{
				Name:       "check_confidentiality",
				Parameters: []string{"resource"},
				Body:       "resource.level >= 3",
			},
		},
		Entities: []*EntityAST{
			{
				Name: "document",
				Relations: []*RelationAST{
					{Name: "parent", TargetType: "folder"},
				},
				Attributes: []*AttributeAST{
					{Name: "authority", Type: "string"},
				},
				Permissions: []*PermissionAST{
					{
						Name: "view",
						Rule: &HierarchicalRuleCallPermissionAST{
							Relation:  "parent",
							RuleName:  "check_confidentiality",
							Arguments: []string{"authority"},
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

	// Verify the permission was converted correctly
	permission := schema.Entities[0].Permissions[0]
	hrcRule, ok := permission.Rule.(*entities.HierarchicalRuleCallRule)
	if !ok {
		t.Fatalf("expected HierarchicalRuleCallRule, got %T", permission.Rule)
	}

	if hrcRule.Relation != "parent" {
		t.Errorf("expected relation 'parent', got %s", hrcRule.Relation)
	}
	if hrcRule.RuleName != "check_confidentiality" {
		t.Errorf("expected rule name 'check_confidentiality', got %s", hrcRule.RuleName)
	}
	if len(hrcRule.Arguments) != 1 || hrcRule.Arguments[0] != "authority" {
		t.Errorf("expected arguments [authority], got %v", hrcRule.Arguments)
	}

	// Test SchemaToAST for HierarchicalRuleCallRule
	resultAST, err := SchemaToAST(schema)
	if err != nil {
		t.Fatalf("SchemaToAST failed: %v", err)
	}

	resultPerm := resultAST.Entities[0].Permissions[0]
	hrcAST, ok := resultPerm.Rule.(*HierarchicalRuleCallPermissionAST)
	if !ok {
		t.Fatalf("expected HierarchicalRuleCallPermissionAST, got %T", resultPerm.Rule)
	}

	if hrcAST.Relation != "parent" {
		t.Errorf("expected relation 'parent', got %s", hrcAST.Relation)
	}
	if hrcAST.RuleName != "check_confidentiality" {
		t.Errorf("expected rule name 'check_confidentiality', got %s", hrcAST.RuleName)
	}
	if len(hrcAST.Arguments) != 1 || hrcAST.Arguments[0] != "authority" {
		t.Errorf("expected arguments [authority], got %v", hrcAST.Arguments)
	}
}

func TestConverter_ABACRule(t *testing.T) {
	// ABACRule exists in entities but has no corresponding AST type in the parser.
	// SchemaToAST should return an error when encountering an ABACRule.
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

	_, err := SchemaToAST(schema)
	if err == nil {
		t.Fatal("expected error when converting ABACRule to AST, but got nil")
	}
}

func TestConverter_RoundTrip_ComplexSchema(t *testing.T) {
	// Full round-trip (AST -> Schema -> AST) with all supported rule types
	originalAST := &SchemaAST{
		Rules: []*RuleDefinitionAST{
			{
				Name:       "is_public",
				Parameters: []string{"resource"},
				Body:       "resource.public == true",
			},
			{
				Name:       "check_level",
				Parameters: []string{"resource", "subject"},
				Body:       "subject.level >= resource.required_level",
			},
		},
		Entities: []*EntityAST{
			{Name: "user"},
			{
				Name: "team",
				Relations: []*RelationAST{
					{Name: "member", TargetType: "user"},
				},
			},
			{
				Name: "folder",
				Relations: []*RelationAST{
					{Name: "owner", TargetType: "user"},
					{Name: "viewer", TargetType: "user team#member"},
					{Name: "parent", TargetType: "folder"},
				},
				Attributes: []*AttributeAST{
					{Name: "public", Type: "boolean"},
					{Name: "required_level", Type: "integer"},
				},
				Permissions: []*PermissionAST{
					// RelationPermissionAST
					{
						Name: "delete",
						Rule: &RelationPermissionAST{Relation: "owner"},
					},
					// LogicalPermissionAST
					{
						Name: "edit",
						Rule: &LogicalPermissionAST{
							Operator: "or",
							Left:     &RelationPermissionAST{Relation: "owner"},
							Right:    &RelationPermissionAST{Relation: "viewer"},
						},
					},
					// HierarchicalPermissionAST
					{
						Name: "view_parent",
						Rule: &HierarchicalPermissionAST{
							Relation:   "parent",
							Permission: "delete",
						},
					},
					// RuleCallPermissionAST
					{
						Name: "view_public",
						Rule: &RuleCallPermissionAST{
							RuleName:  "is_public",
							Arguments: []string{"resource"},
						},
					},
					// HierarchicalRuleCallPermissionAST
					{
						Name: "view_classified",
						Rule: &HierarchicalRuleCallPermissionAST{
							Relation:  "parent",
							RuleName:  "check_level",
							Arguments: []string{"resource", "subject"},
						},
					},
				},
			},
		},
	}

	// AST -> Schema
	schema, err := ASTToSchema("test-tenant", originalAST)
	if err != nil {
		t.Fatalf("ASTToSchema failed: %v", err)
	}

	// Schema -> AST
	resultAST, err := SchemaToAST(schema)
	if err != nil {
		t.Fatalf("SchemaToAST failed: %v", err)
	}

	// Verify rules
	if len(resultAST.Rules) != len(originalAST.Rules) {
		t.Fatalf("rule count mismatch: got %d, want %d", len(resultAST.Rules), len(originalAST.Rules))
	}
	for i, rule := range resultAST.Rules {
		orig := originalAST.Rules[i]
		if rule.Name != orig.Name {
			t.Errorf("rule[%d] name mismatch: got %s, want %s", i, rule.Name, orig.Name)
		}
		if rule.Body != orig.Body {
			t.Errorf("rule[%d] body mismatch: got %s, want %s", i, rule.Body, orig.Body)
		}
		if len(rule.Parameters) != len(orig.Parameters) {
			t.Errorf("rule[%d] parameter count mismatch: got %d, want %d", i, len(rule.Parameters), len(orig.Parameters))
		}
	}

	// Verify entities
	if len(resultAST.Entities) != len(originalAST.Entities) {
		t.Fatalf("entity count mismatch: got %d, want %d", len(resultAST.Entities), len(originalAST.Entities))
	}

	// Verify the folder entity in detail (index 2)
	resultFolder := resultAST.Entities[2]
	originalFolder := originalAST.Entities[2]

	if resultFolder.Name != originalFolder.Name {
		t.Errorf("entity name mismatch: got %s, want %s", resultFolder.Name, originalFolder.Name)
	}
	if len(resultFolder.Relations) != len(originalFolder.Relations) {
		t.Errorf("relation count mismatch: got %d, want %d", len(resultFolder.Relations), len(originalFolder.Relations))
	}
	for i, rel := range resultFolder.Relations {
		orig := originalFolder.Relations[i]
		if rel.Name != orig.Name {
			t.Errorf("relation[%d] name mismatch: got %s, want %s", i, rel.Name, orig.Name)
		}
		if rel.TargetType != orig.TargetType {
			t.Errorf("relation[%d] target type mismatch: got %s, want %s", i, rel.TargetType, orig.TargetType)
		}
	}
	if len(resultFolder.Attributes) != len(originalFolder.Attributes) {
		t.Errorf("attribute count mismatch: got %d, want %d", len(resultFolder.Attributes), len(originalFolder.Attributes))
	}
	if len(resultFolder.Permissions) != len(originalFolder.Permissions) {
		t.Fatalf("permission count mismatch: got %d, want %d", len(resultFolder.Permissions), len(originalFolder.Permissions))
	}

	// Verify each permission rule type survived the round-trip
	// Permission 0: RelationPermissionAST
	if _, ok := resultFolder.Permissions[0].Rule.(*RelationPermissionAST); !ok {
		t.Errorf("permission[0] expected RelationPermissionAST, got %T", resultFolder.Permissions[0].Rule)
	}
	// Permission 1: LogicalPermissionAST
	if _, ok := resultFolder.Permissions[1].Rule.(*LogicalPermissionAST); !ok {
		t.Errorf("permission[1] expected LogicalPermissionAST, got %T", resultFolder.Permissions[1].Rule)
	}
	// Permission 2: HierarchicalPermissionAST
	if _, ok := resultFolder.Permissions[2].Rule.(*HierarchicalPermissionAST); !ok {
		t.Errorf("permission[2] expected HierarchicalPermissionAST, got %T", resultFolder.Permissions[2].Rule)
	}
	// Permission 3: RuleCallPermissionAST
	if _, ok := resultFolder.Permissions[3].Rule.(*RuleCallPermissionAST); !ok {
		t.Errorf("permission[3] expected RuleCallPermissionAST, got %T", resultFolder.Permissions[3].Rule)
	}
	// Permission 4: HierarchicalRuleCallPermissionAST
	hrcResult, ok := resultFolder.Permissions[4].Rule.(*HierarchicalRuleCallPermissionAST)
	if !ok {
		t.Errorf("permission[4] expected HierarchicalRuleCallPermissionAST, got %T", resultFolder.Permissions[4].Rule)
	} else {
		if hrcResult.Relation != "parent" {
			t.Errorf("permission[4] relation mismatch: got %s, want parent", hrcResult.Relation)
		}
		if hrcResult.RuleName != "check_level" {
			t.Errorf("permission[4] rule name mismatch: got %s, want check_level", hrcResult.RuleName)
		}
		if len(hrcResult.Arguments) != 2 || hrcResult.Arguments[0] != "resource" || hrcResult.Arguments[1] != "subject" {
			t.Errorf("permission[4] arguments mismatch: got %v, want [resource subject]", hrcResult.Arguments)
		}
	}
}

func TestRoundTrip_ASTToSchemaToAST(t *testing.T) {
	// Original AST
	originalAST := &SchemaAST{
		Rules: []*RuleDefinitionAST{
			{
				Name:       "is_public",
				Parameters: []string{"resource"},
				Body:       "resource.public == true",
			},
		},
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
							Right:    &RuleCallPermissionAST{RuleName: "is_public", Arguments: []string{"resource"}},
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
