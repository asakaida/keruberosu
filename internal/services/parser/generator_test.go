package parser

import (
	"strings"
	"testing"
)

func TestGenerator_Generate_BasicEntity(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "user",
			},
		},
	}

	gen := NewGenerator()
	result := gen.Generate(ast)

	expected := `entity user {
}`

	if result != expected {
		t.Errorf("generated DSL mismatch:\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestGenerator_Generate_WithRelation(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Relations: []*RelationAST{
					{Name: "owner", TargetType: "user"},
				},
			},
		},
	}

	gen := NewGenerator()
	result := gen.Generate(ast)

	expected := `entity document {
  relation owner: user
}`

	if result != expected {
		t.Errorf("generated DSL mismatch:\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestGenerator_Generate_WithAttribute(t *testing.T) {
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

	gen := NewGenerator()
	result := gen.Generate(ast)

	expected := `entity document {
  attribute public: boolean
  attribute title: string
}`

	if result != expected {
		t.Errorf("generated DSL mismatch:\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestGenerator_Generate_WithPermission(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
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

	gen := NewGenerator()
	result := gen.Generate(ast)

	expected := `entity document {
  relation owner: user
  permission view = owner
}`

	if result != expected {
		t.Errorf("generated DSL mismatch:\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestGenerator_Generate_LogicalOR(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
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

	gen := NewGenerator()
	result := gen.Generate(ast)

	if !strings.Contains(result, "permission edit = owner or editor") {
		t.Errorf("generated DSL should contain 'permission edit = owner or editor', got:\n%s", result)
	}
}

func TestGenerator_Generate_LogicalAND(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Permissions: []*PermissionAST{
					{
						Name: "delete",
						Rule: &LogicalPermissionAST{
							Operator: "and",
							Left:     &RelationPermissionAST{Relation: "owner"},
							Right:    &RelationPermissionAST{Relation: "admin"},
						},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	result := gen.Generate(ast)

	if !strings.Contains(result, "permission delete = owner and admin") {
		t.Errorf("generated DSL should contain 'permission delete = owner and admin', got:\n%s", result)
	}
}

func TestGenerator_Generate_LogicalNOT(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Permissions: []*PermissionAST{
					{
						Name: "restricted",
						Rule: &LogicalPermissionAST{
							Operator: "not",
							Left:     &RelationPermissionAST{Relation: "blocked"},
						},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	result := gen.Generate(ast)

	if !strings.Contains(result, "permission restricted = not blocked") {
		t.Errorf("generated DSL should contain 'permission restricted = not blocked', got:\n%s", result)
	}
}

func TestGenerator_Generate_OperatorPrecedence(t *testing.T) {
	// (owner or editor) and admin
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Permissions: []*PermissionAST{
					{
						Name: "manage",
						Rule: &LogicalPermissionAST{
							Operator: "and",
							Left: &LogicalPermissionAST{
								Operator: "or",
								Left:     &RelationPermissionAST{Relation: "owner"},
								Right:    &RelationPermissionAST{Relation: "editor"},
							},
							Right: &RelationPermissionAST{Relation: "admin"},
						},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	result := gen.Generate(ast)

	// Should have parentheses around OR when it's inside AND
	if !strings.Contains(result, "(owner or editor) and admin") {
		t.Errorf("generated DSL should contain '(owner or editor) and admin', got:\n%s", result)
	}
}

func TestGenerator_Generate_HierarchicalPermission(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
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

	gen := NewGenerator()
	result := gen.Generate(ast)

	if !strings.Contains(result, "permission view = parent.view") {
		t.Errorf("generated DSL should contain 'permission view = parent.view', got:\n%s", result)
	}
}

func TestGenerator_Generate_ABACRule(t *testing.T) {
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

	gen := NewGenerator()
	result := gen.Generate(ast)

	if !strings.Contains(result, "permission view = rule(resource.public == true)") {
		t.Errorf("generated DSL should contain 'permission view = rule(resource.public == true)', got:\n%s", result)
	}
}

func TestGenerator_Generate_ComplexSchema(t *testing.T) {
	ast := &SchemaAST{
		Entities: []*EntityAST{
			{Name: "user"},
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
						Rule: &RelationPermissionAST{Relation: "owner"},
					},
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

	gen := NewGenerator()
	result := gen.Generate(ast)

	// Check that multiple entities are separated by newlines
	if !strings.Contains(result, "entity user {\n}\nentity document {") {
		t.Errorf("entities should be separated by newline, got:\n%s", result)
	}

	// Check that all elements are present
	if !strings.Contains(result, "relation owner: user") {
		t.Errorf("missing relation owner, got:\n%s", result)
	}
	if !strings.Contains(result, "attribute public: boolean") {
		t.Errorf("missing attribute public, got:\n%s", result)
	}
	if !strings.Contains(result, "permission view = owner") {
		t.Errorf("missing permission view, got:\n%s", result)
	}
}

func TestGenerator_RoundTrip_ParseAndGenerate(t *testing.T) {
	originalDSL := `entity user {
}
entity document {
  relation owner: user
  relation editor: user
  attribute public: boolean
  permission view = owner
  permission edit = owner or editor
}`

	// Parse original DSL
	lexer := NewLexer(originalDSL)
	parser := NewParser(lexer)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("failed to parse original DSL: %v", err)
	}

	// Generate DSL from AST
	gen := NewGenerator()
	generatedDSL := gen.Generate(ast)

	// Parse generated DSL
	lexer2 := NewLexer(generatedDSL)
	parser2 := NewParser(lexer2)
	ast2, err := parser2.Parse()
	if err != nil {
		t.Fatalf("failed to parse generated DSL: %v", err)
	}

	// Check that both ASTs have the same structure
	if len(ast.Entities) != len(ast2.Entities) {
		t.Errorf("entity count mismatch: original %d, generated %d",
			len(ast.Entities), len(ast2.Entities))
	}

	// Check document entity
	if len(ast.Entities) > 1 {
		originalDoc := ast.Entities[1]
		generatedDoc := ast2.Entities[1]

		if len(originalDoc.Relations) != len(generatedDoc.Relations) {
			t.Errorf("relation count mismatch: original %d, generated %d",
				len(originalDoc.Relations), len(generatedDoc.Relations))
		}

		if len(originalDoc.Permissions) != len(generatedDoc.Permissions) {
			t.Errorf("permission count mismatch: original %d, generated %d",
				len(originalDoc.Permissions), len(generatedDoc.Permissions))
		}
	}
}
