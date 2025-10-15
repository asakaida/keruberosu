package parser

import (
	"testing"
)

func TestParser_SimpleEntity(t *testing.T) {
	input := `entity user {}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(schema.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(schema.Entities))
	}

	if schema.Entities[0].Name != "user" {
		t.Errorf("expected entity name 'user', got %s", schema.Entities[0].Name)
	}
}

func TestParser_EntityWithRelation(t *testing.T) {
	input := `entity document {
  relation owner @user
  relation editor @user
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(schema.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(schema.Entities))
	}

	entity := schema.Entities[0]
	if entity.Name != "document" {
		t.Errorf("expected entity name 'document', got %s", entity.Name)
	}

	if len(entity.Relations) != 2 {
		t.Fatalf("expected 2 relations, got %d", len(entity.Relations))
	}

	if entity.Relations[0].Name != "owner" {
		t.Errorf("expected relation name 'owner', got %s", entity.Relations[0].Name)
	}
	if entity.Relations[0].TargetType != "user" {
		t.Errorf("expected target type 'user', got %s", entity.Relations[0].TargetType)
	}

	if entity.Relations[1].Name != "editor" {
		t.Errorf("expected relation name 'editor', got %s", entity.Relations[1].Name)
	}
}

func TestParser_EntityWithAttribute(t *testing.T) {
	input := `entity document {
  attribute title string
  attribute public boolean
  attribute version integer
  attribute tags string[]
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	if len(entity.Attributes) != 4 {
		t.Fatalf("expected 4 attributes, got %d", len(entity.Attributes))
	}

	tests := []struct {
		name     string
		attrType string
	}{
		{"title", "string"},
		{"public", "boolean"},
		{"version", "integer"},
		{"tags", "string[]"},
	}

	for i, test := range tests {
		if entity.Attributes[i].Name != test.name {
			t.Errorf("expected attribute name %s, got %s", test.name, entity.Attributes[i].Name)
		}
		if entity.Attributes[i].Type != test.attrType {
			t.Errorf("expected attribute type %s, got %s", test.attrType, entity.Attributes[i].Type)
		}
	}
}

func TestParser_SimplePermission(t *testing.T) {
	input := `entity document {
  relation owner @user
  permission edit = owner
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	if len(entity.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(entity.Permissions))
	}

	perm := entity.Permissions[0]
	if perm.Name != "edit" {
		t.Errorf("expected permission name 'edit', got %s", perm.Name)
	}

	rule, ok := perm.Rule.(*RelationPermissionAST)
	if !ok {
		t.Fatalf("expected RelationPermissionAST, got %T", perm.Rule)
	}

	if rule.Relation != "owner" {
		t.Errorf("expected relation 'owner', got %s", rule.Relation)
	}
}

func TestParser_PermissionWithOr(t *testing.T) {
	input := `entity document {
  relation owner @user
  relation editor @user
  permission edit = owner or editor
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	perm := entity.Permissions[0]

	rule, ok := perm.Rule.(*LogicalPermissionAST)
	if !ok {
		t.Fatalf("expected LogicalPermissionAST, got %T", perm.Rule)
	}

	if rule.Operator != "or" {
		t.Errorf("expected operator 'or', got %s", rule.Operator)
	}

	leftRule, ok := rule.Left.(*RelationPermissionAST)
	if !ok {
		t.Fatalf("expected left to be RelationPermissionAST, got %T", rule.Left)
	}
	if leftRule.Relation != "owner" {
		t.Errorf("expected left relation 'owner', got %s", leftRule.Relation)
	}

	rightRule, ok := rule.Right.(*RelationPermissionAST)
	if !ok {
		t.Fatalf("expected right to be RelationPermissionAST, got %T", rule.Right)
	}
	if rightRule.Relation != "editor" {
		t.Errorf("expected right relation 'editor', got %s", rightRule.Relation)
	}
}

func TestParser_PermissionWithAnd(t *testing.T) {
	input := `entity document {
  relation member @user
  relation approved @user
  permission edit = member and approved
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	perm := entity.Permissions[0]

	rule, ok := perm.Rule.(*LogicalPermissionAST)
	if !ok {
		t.Fatalf("expected LogicalPermissionAST, got %T", perm.Rule)
	}

	if rule.Operator != "and" {
		t.Errorf("expected operator 'and', got %s", rule.Operator)
	}
}

func TestParser_PermissionWithNot(t *testing.T) {
	input := `entity document {
  relation banned @user
  permission view = not banned
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	perm := entity.Permissions[0]

	rule, ok := perm.Rule.(*LogicalPermissionAST)
	if !ok {
		t.Fatalf("expected LogicalPermissionAST, got %T", perm.Rule)
	}

	if rule.Operator != "not" {
		t.Errorf("expected operator 'not', got %s", rule.Operator)
	}

	if rule.Right != nil {
		t.Error("expected right to be nil for 'not' operator")
	}
}

func TestParser_PermissionWithComplexExpression(t *testing.T) {
	input := `entity document {
  relation owner @user
  relation editor @user
  relation banned @user
  permission edit = (owner or editor) and not banned
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	perm := entity.Permissions[0]

	// Top level should be 'and'
	rule, ok := perm.Rule.(*LogicalPermissionAST)
	if !ok {
		t.Fatalf("expected LogicalPermissionAST, got %T", perm.Rule)
	}

	if rule.Operator != "and" {
		t.Errorf("expected operator 'and', got %s", rule.Operator)
	}

	// Left should be 'or'
	leftRule, ok := rule.Left.(*LogicalPermissionAST)
	if !ok {
		t.Fatalf("expected left to be LogicalPermissionAST, got %T", rule.Left)
	}
	if leftRule.Operator != "or" {
		t.Errorf("expected left operator 'or', got %s", leftRule.Operator)
	}

	// Right should be 'not'
	rightRule, ok := rule.Right.(*LogicalPermissionAST)
	if !ok {
		t.Fatalf("expected right to be LogicalPermissionAST, got %T", rule.Right)
	}
	if rightRule.Operator != "not" {
		t.Errorf("expected right operator 'not', got %s", rightRule.Operator)
	}
}

func TestParser_HierarchicalPermission(t *testing.T) {
	input := `entity document {
  relation parent @folder
  permission view = parent.view
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	perm := entity.Permissions[0]

	rule, ok := perm.Rule.(*HierarchicalPermissionAST)
	if !ok {
		t.Fatalf("expected HierarchicalPermissionAST, got %T", perm.Rule)
	}

	if rule.Relation != "parent" {
		t.Errorf("expected relation 'parent', got %s", rule.Relation)
	}

	if rule.Permission != "view" {
		t.Errorf("expected permission 'view', got %s", rule.Permission)
	}
}

func TestParser_MultipleEntities(t *testing.T) {
	input := `entity user {}

entity organization {
  relation member @user
}

entity document {
  relation owner @user
  permission edit = owner
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(schema.Entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(schema.Entities))
	}

	entityNames := []string{"user", "organization", "document"}
	for i, name := range entityNames {
		if schema.Entities[i].Name != name {
			t.Errorf("expected entity[%d] name %s, got %s", i, name, schema.Entities[i].Name)
		}
	}
}

func TestParser_CompleteSchema(t *testing.T) {
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

	if len(schema.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(schema.Rules))
	}

	if len(schema.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(schema.Entities))
	}

	doc := schema.Entities[1]
	if doc.Name != "document" {
		t.Errorf("expected entity name 'document', got %s", doc.Name)
	}

	if len(doc.Relations) != 3 {
		t.Errorf("expected 3 relations, got %d", len(doc.Relations))
	}

	if len(doc.Attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(doc.Attributes))
	}

	if len(doc.Permissions) != 3 {
		t.Errorf("expected 3 permissions, got %d", len(doc.Permissions))
	}
}

func TestParser_WithComments(t *testing.T) {
	input := `// User entity
entity user {}

// Document entity
entity document {
  // Owner relation
  relation owner @user

  // Edit permission
  permission edit = owner // only owner can edit
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(schema.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(schema.Entities))
	}
}

func TestParser_ErrorMissingEntityName(t *testing.T) {
	input := `entity {}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	_, err := parser.Parse()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestParser_ErrorMissingBrace(t *testing.T) {
	input := `entity user`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	_, err := parser.Parse()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestParser_ErrorInvalidRelation(t *testing.T) {
	input := `entity document {
  relation owner
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	_, err := parser.Parse()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestParser_ErrorInvalidPermission(t *testing.T) {
	input := `entity document {
  permission edit
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	_, err := parser.Parse()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestParser_TopLevelRuleDefinition(t *testing.T) {
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

	// Check rule call in permission
	if len(schema.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(schema.Entities))
	}

	entity := schema.Entities[0]
	if len(entity.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(entity.Permissions))
	}

	perm := entity.Permissions[0]
	ruleCall, ok := perm.Rule.(*RuleCallPermissionAST)
	if !ok {
		t.Fatalf("expected RuleCallPermissionAST, got %T", perm.Rule)
	}

	if ruleCall.RuleName != "is_public" {
		t.Errorf("expected rule name 'is_public', got %s", ruleCall.RuleName)
	}
	if len(ruleCall.Arguments) != 1 || ruleCall.Arguments[0] != "resource" {
		t.Errorf("expected arguments [resource], got %v", ruleCall.Arguments)
	}
}

func TestParser_RuleWithMultipleParameters(t *testing.T) {
	input := `rule can_edit(subject, resource) {
  subject.id == resource.owner_id
}

entity document {
  attribute owner_id string
  permission edit = can_edit(subject, resource)
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	rule := schema.Rules[0]
	if len(rule.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(rule.Parameters))
	}
	if rule.Parameters[0] != "subject" || rule.Parameters[1] != "resource" {
		t.Errorf("expected parameters [subject, resource], got %v", rule.Parameters)
	}

	entity := schema.Entities[0]
	perm := entity.Permissions[0]
	ruleCall := perm.Rule.(*RuleCallPermissionAST)
	if len(ruleCall.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(ruleCall.Arguments))
	}
	if ruleCall.Arguments[0] != "subject" || ruleCall.Arguments[1] != "resource" {
		t.Errorf("expected arguments [subject, resource], got %v", ruleCall.Arguments)
	}
}

func TestParser_AllAttributeTypes(t *testing.T) {
	// Test all Permify attribute types
	input := `entity document {
  attribute title string
  attribute active boolean
  attribute count integer
  attribute price double
  attribute tags string[]
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	entity := schema.Entities[0]
	if len(entity.Attributes) != 5 {
		t.Fatalf("expected 5 attributes, got %d", len(entity.Attributes))
	}

	tests := []struct {
		name     string
		attrType string
	}{
		{"title", "string"},
		{"active", "boolean"},
		{"count", "integer"},
		{"price", "double"},
		{"tags", "string[]"},
	}

	for i, test := range tests {
		if entity.Attributes[i].Name != test.name {
			t.Errorf("expected attribute name %s, got %s", test.name, entity.Attributes[i].Name)
		}
		if entity.Attributes[i].Type != test.attrType {
			t.Errorf("expected attribute type %s, got %s", test.attrType, entity.Attributes[i].Type)
		}
	}
}
