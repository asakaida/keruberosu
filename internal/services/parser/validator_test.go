package parser

import (
	"strings"
	"testing"
)

func TestValidator_ValidSchema(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user

  attribute public: bool
  attribute title: string

  permission edit = owner or editor
  permission view = owner or editor or viewer or rule(resource.public == true)
  permission delete = owner
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err != nil {
		t.Errorf("expected valid schema, got error: %v", err)
	}
}

func TestValidator_DuplicateEntityNames(t *testing.T) {
	input := `entity user {}
entity user {}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for duplicate entity names")
	}

	if !strings.Contains(err.Error(), "duplicate entity name: user") {
		t.Errorf("expected duplicate entity name error, got: %v", err)
	}
}

func TestValidator_DuplicateRelationNames(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner: user
  relation owner: user
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for duplicate relation names")
	}

	if !strings.Contains(err.Error(), "duplicate relation name: owner") {
		t.Errorf("expected duplicate relation name error, got: %v", err)
	}
}

func TestValidator_DuplicateAttributeNames(t *testing.T) {
	input := `entity document {
  attribute title: string
  attribute title: string
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for duplicate attribute names")
	}

	if !strings.Contains(err.Error(), "duplicate attribute name: title") {
		t.Errorf("expected duplicate attribute name error, got: %v", err)
	}
}

func TestValidator_DuplicatePermissionNames(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner: user
  permission edit = owner
  permission edit = owner
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for duplicate permission names")
	}

	if !strings.Contains(err.Error(), "duplicate permission name: edit") {
		t.Errorf("expected duplicate permission name error, got: %v", err)
	}
}

func TestValidator_NameConflicts(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner: user
  attribute owner: string
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for name conflict")
	}

	if !strings.Contains(err.Error(), "name conflict") {
		t.Errorf("expected name conflict error, got: %v", err)
	}
}

func TestValidator_InvalidAttributeType(t *testing.T) {
	input := `entity document {
  attribute invalid: unknown_type
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid attribute type")
	}

	if !strings.Contains(err.Error(), "invalid attribute type: unknown_type") {
		t.Errorf("expected invalid attribute type error, got: %v", err)
	}
}

func TestValidator_UndefinedEntityInRelation(t *testing.T) {
	input := `entity document {
  relation owner: user
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for undefined entity")
	}

	if !strings.Contains(err.Error(), "references undefined entity: user") {
		t.Errorf("expected undefined entity error, got: %v", err)
	}
}

func TestValidator_UndefinedRelationInPermission(t *testing.T) {
	input := `entity document {
  permission edit = owner
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for undefined relation")
	}

	if !strings.Contains(err.Error(), "references undefined relation or permission: owner") {
		t.Errorf("expected undefined relation error, got: %v", err)
	}
}

func TestValidator_HierarchicalPermission(t *testing.T) {
	input := `entity user {}

entity folder {
  relation owner: user
  permission view = owner
}

entity document {
  relation parent: folder
  permission view = parent.view
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err != nil {
		t.Errorf("expected valid hierarchical permission schema, got error: %v", err)
	}
}

func TestValidator_UndefinedHierarchicalRelation(t *testing.T) {
	input := `entity user {}

entity folder {
  relation owner: user
  permission view = owner
}

entity document {
  permission view = parent.view
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for undefined relation in hierarchical permission")
	}

	if !strings.Contains(err.Error(), "references undefined relation: parent") {
		t.Errorf("expected undefined relation error, got: %v", err)
	}
}

func TestValidator_UndefinedHierarchicalPermission(t *testing.T) {
	input := `entity user {}

entity folder {
  relation owner: user
}

entity document {
  relation parent: folder
  permission view = parent.edit
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for undefined permission in hierarchical permission")
	}

	if !strings.Contains(err.Error(), "references undefined permission edit in entity folder") {
		t.Errorf("expected undefined permission error, got: %v", err)
	}
}

func TestValidator_CircularPermissionReference(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner: user
  permission view = edit
  permission edit = view
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for circular permission reference")
	}

	if !strings.Contains(err.Error(), "circular permission reference") {
		t.Errorf("expected circular permission reference error, got: %v", err)
	}
}

func TestValidator_CircularPermissionWithinEntity(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner: user
  permission delete = edit
  permission edit = view
  permission view = delete
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for circular permission reference")
	}

	if !strings.Contains(err.Error(), "circular permission reference") {
		t.Errorf("expected circular permission reference error, got: %v", err)
	}
}

func TestValidator_EmptyRuleExpression(t *testing.T) {
	schema := &SchemaAST{
		Entities: []*EntityAST{
			{
				Name: "document",
				Permissions: []*PermissionAST{
					{
						Name: "view",
						Rule: &RulePermissionAST{Expression: ""},
					},
				},
			},
		},
	}

	validator := NewValidator(schema)
	err := validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty rule expression")
	}

	if !strings.Contains(err.Error(), "empty rule expression") {
		t.Errorf("expected empty rule expression error, got: %v", err)
	}
}

func TestValidator_ValidAttributeTypes(t *testing.T) {
	input := `entity document {
  attribute title: string
  attribute count: int
  attribute active: bool
  attribute score: float
  attribute tags: string[]
  attribute numbers: int[]
  attribute flags: bool[]
  attribute scores: float[]
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err != nil {
		t.Errorf("expected valid attribute types, got error: %v", err)
	}
}

func TestValidator_ComplexValidSchema(t *testing.T) {
	input := `entity user {}

entity organization {
  relation admin: user
  relation member: user
  permission edit = admin
  permission view = admin or member
}

entity folder {
  relation owner: user
  relation parent: folder
  relation org: organization

  attribute public: bool

  permission edit = owner or org.edit
  permission view = edit or parent.view or rule(resource.public == true)
}

entity document {
  relation owner: user
  relation parent: folder

  attribute public: bool
  attribute title: string

  permission edit = owner or parent.edit
  permission view = edit or parent.view or rule(resource.public == true)
  permission delete = owner
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err != nil {
		t.Errorf("expected valid complex schema, got error: %v", err)
	}
}

func TestValidator_MultipleErrors(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner: undefined_entity
  relation duplicate: user
  relation duplicate: user

  attribute invalid: unknown_type

  permission edit = undefined_relation
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}

	// Check that multiple errors are reported
	errorMsg := err.Error()
	errorCount := strings.Count(errorMsg, "\n")
	if errorCount < 3 {
		t.Errorf("expected at least 4 errors, got: %s", errorMsg)
	}
}
