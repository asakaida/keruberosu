package parser

import (
	"strings"
	"testing"
)

func TestValidator_ValidSchema(t *testing.T) {
	input := `rule is_public(resource) {
  resource.public == true
}

entity user {}

entity document {
  relation owner @user
  relation editor @user
  relation viewer @user

  attribute public boolean
  attribute title string

  permission edit = owner or editor
  permission view = owner or editor or viewer or is_public(resource)
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
  relation owner @user
  relation owner @user
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
  attribute title string
  attribute title string
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
  relation owner @user
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
  relation owner @user
  attribute owner string
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
  attribute invalid unknown_type
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
  relation owner @user
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
  relation owner @user
  permission view = owner
}

entity document {
  relation parent @folder
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
  relation owner @user
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
  relation owner @user
}

entity document {
  relation parent @folder
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

	if !strings.Contains(err.Error(), "references undefined permission or relation edit in entity folder") {
		t.Errorf("expected undefined permission error, got: %v", err)
	}
}

func TestValidator_CircularPermissionReference(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner @user
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
  relation owner @user
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

func TestValidator_ValidAttributeTypes(t *testing.T) {
	input := `entity document {
  attribute title string
  attribute count integer
  attribute active boolean
  attribute score double
  attribute tags string[]
  attribute numbers integer[]
  attribute flags boolean[]
  attribute scores double[]
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
	input := `rule is_public(resource) {
  resource.public == true
}

entity user {}

entity organization {
  relation admin @user
  relation member @user
  permission edit = admin
  permission view = admin or member
}

entity folder {
  relation owner @user
  relation parent @folder
  relation org @organization

  attribute public boolean

  permission edit = owner or org.edit
  permission view = edit or parent.view or is_public(resource)
}

entity document {
  relation owner @user
  relation parent @folder

  attribute public boolean
  attribute title string

  permission edit = owner or parent.edit
  permission view = edit or parent.view or is_public(resource)
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
  relation owner @undefined_entity
  relation duplicate @user
  relation duplicate @user

  attribute invalid unknown_type

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

func TestValidator_ValidRuleCall(t *testing.T) {
	input := `rule is_public(resource) {
  resource.public == true
}

entity document {
  attribute public boolean
  permission view = is_public(resource)
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
		t.Errorf("expected valid rule call schema, got error: %v", err)
	}
}

func TestValidator_DuplicateRuleNames(t *testing.T) {
	input := `rule is_public(resource) {
  resource.public == true
}

rule is_public(resource) {
  resource.public == true
}

entity document {}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate()
	if err == nil {
		t.Fatal("expected validation error for duplicate rule names")
	}

	if !strings.Contains(err.Error(), "duplicate rule name: is_public") {
		t.Errorf("expected duplicate rule name error, got: %v", err)
	}
}

func TestValidator_UndefinedRuleCall(t *testing.T) {
	input := `entity document {
  attribute public boolean
  permission view = is_public(resource)
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
		t.Fatal("expected validation error for undefined rule")
	}

	if !strings.Contains(err.Error(), "calls undefined rule: is_public") {
		t.Errorf("expected undefined rule error, got: %v", err)
	}
}

func TestValidator_RuleCallArgumentCountMismatch(t *testing.T) {
	input := `rule is_public(resource) {
  resource.public == true
}

entity document {
  attribute public boolean
  permission view = is_public(resource, subject)
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
		t.Fatal("expected validation error for argument count mismatch")
	}

	if !strings.Contains(err.Error(), "with 2 arguments, expected 1") {
		t.Errorf("expected argument count mismatch error, got: %v", err)
	}
}

func TestValidator_RuleCallInvalidArgument(t *testing.T) {
	input := `rule is_public(resource) {
  resource.public == true
}

entity document {
  attribute public boolean
  permission view = is_public(invalid_arg)
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
		t.Fatal("expected validation error for invalid argument")
	}

	if !strings.Contains(err.Error(), "invalid argument: invalid_arg") {
		t.Errorf("expected invalid argument error, got: %v", err)
	}
}

func TestValidator_MultipleRulesAndCalls(t *testing.T) {
	input := `rule is_public(resource) {
  resource.public == true
}

rule can_edit(subject, resource) {
  subject.id == resource.owner_id
}

entity document {
  attribute public boolean
  attribute owner_id string

  permission view = is_public(resource)
  permission edit = can_edit(subject, resource)
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
		t.Errorf("expected valid schema with multiple rules, got error: %v", err)
	}
}
