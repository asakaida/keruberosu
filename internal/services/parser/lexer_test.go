package parser

import (
	"testing"
)

func TestLexer_Keywords(t *testing.T) {
	input := `entity relation attribute permission rule or and not`

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TOKEN_ENTITY, "entity"},
		{TOKEN_RELATION, "relation"},
		{TOKEN_ATTRIBUTE, "attribute"},
		{TOKEN_PERMISSION, "permission"},
		{TOKEN_RULE, "rule"},
		{TOKEN_OR, "or"},
		{TOKEN_AND, "and"},
		{TOKEN_NOT, "not"},
		{TOKEN_EOF, ""},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Value != exp.value {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexer_Operators(t *testing.T) {
	input := `= : { } ( ) [ ] . ,`

	expected := []TokenType{
		TOKEN_EQUALS,
		TOKEN_COLON,
		TOKEN_LBRACE,
		TOKEN_RBRACE,
		TOKEN_LPAREN,
		TOKEN_RPAREN,
		TOKEN_LBRACKET,
		TOKEN_RBRACKET,
		TOKEN_DOT,
		TOKEN_COMMA,
		TOKEN_EOF,
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp, tok.Type)
		}
	}
}

func TestLexer_Identifiers(t *testing.T) {
	input := `user document owner editor my_field field123`

	expected := []string{
		"user",
		"document",
		"owner",
		"editor",
		"my_field",
		"field123",
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != TOKEN_IDENTIFIER {
			t.Errorf("test[%d]: expected token type IDENTIFIER, got %v", i, tok.Type)
		}

		if tok.Value != exp {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp, tok.Value)
		}
	}
}

func TestLexer_SimpleSchema(t *testing.T) {
	input := `entity user {}

entity document {
  relation owner @user
  permission edit = owner
}`

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TOKEN_ENTITY, "entity"},
		{TOKEN_IDENTIFIER, "user"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_RBRACE, "}"},

		{TOKEN_ENTITY, "entity"},
		{TOKEN_IDENTIFIER, "document"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_RELATION, "relation"},
		{TOKEN_IDENTIFIER, "owner"},
		{TOKEN_AT, "@"},
		{TOKEN_IDENTIFIER, "user"},
		{TOKEN_PERMISSION, "permission"},
		{TOKEN_IDENTIFIER, "edit"},
		{TOKEN_EQUALS, "="},
		{TOKEN_IDENTIFIER, "owner"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_EOF, ""},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Value != exp.value {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexer_LogicalOperators(t *testing.T) {
	input := `permission edit = owner or editor and not banned`

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TOKEN_PERMISSION, "permission"},
		{TOKEN_IDENTIFIER, "edit"},
		{TOKEN_EQUALS, "="},
		{TOKEN_IDENTIFIER, "owner"},
		{TOKEN_OR, "or"},
		{TOKEN_IDENTIFIER, "editor"},
		{TOKEN_AND, "and"},
		{TOKEN_NOT, "not"},
		{TOKEN_IDENTIFIER, "banned"},
		{TOKEN_EOF, ""},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Value != exp.value {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexer_HierarchicalPermission(t *testing.T) {
	input := `permission edit = parent.edit`

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TOKEN_PERMISSION, "permission"},
		{TOKEN_IDENTIFIER, "edit"},
		{TOKEN_EQUALS, "="},
		{TOKEN_IDENTIFIER, "parent"},
		{TOKEN_DOT, "."},
		{TOKEN_IDENTIFIER, "edit"},
		{TOKEN_EOF, ""},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Value != exp.value {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexer_RuleExpression(t *testing.T) {
	input := `permission view = rule(resource.public == true)`

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TOKEN_PERMISSION, "permission"},
		{TOKEN_IDENTIFIER, "view"},
		{TOKEN_EQUALS, "="},
		{TOKEN_RULE, "rule"},
		{TOKEN_LPAREN, "("},
		{TOKEN_IDENTIFIER, "resource"},
		{TOKEN_DOT, "."},
		{TOKEN_IDENTIFIER, "public"},
		{TOKEN_EQ, "=="},
		{TOKEN_IDENTIFIER, "true"},
		{TOKEN_RPAREN, ")"},
		{TOKEN_EOF, ""},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Value != exp.value {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexer_Comments(t *testing.T) {
	input := `// This is a comment
entity user {} // another comment
// final comment`

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TOKEN_ENTITY, "entity"},
		{TOKEN_IDENTIFIER, "user"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_EOF, ""},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Value != exp.value {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexer_AttributeTypes(t *testing.T) {
	input := `attribute name: string
attribute age: int
attribute active: bool
attribute tags: string[]`

	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TOKEN_ATTRIBUTE, "attribute"},
		{TOKEN_IDENTIFIER, "name"},
		{TOKEN_COLON, ":"},
		{TOKEN_IDENTIFIER, "string"},

		{TOKEN_ATTRIBUTE, "attribute"},
		{TOKEN_IDENTIFIER, "age"},
		{TOKEN_COLON, ":"},
		{TOKEN_IDENTIFIER, "int"},

		{TOKEN_ATTRIBUTE, "attribute"},
		{TOKEN_IDENTIFIER, "active"},
		{TOKEN_COLON, ":"},
		{TOKEN_IDENTIFIER, "bool"},

		{TOKEN_ATTRIBUTE, "attribute"},
		{TOKEN_IDENTIFIER, "tags"},
		{TOKEN_COLON, ":"},
		{TOKEN_IDENTIFIER, "string"},
		{TOKEN_LBRACKET, "["},
		{TOKEN_RBRACKET, "]"},

		{TOKEN_EOF, ""},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Value != exp.value {
			t.Errorf("test[%d]: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexer_IllegalCharacter(t *testing.T) {
	// Note: @ is now a legal character (TOKEN_AT for Permify compatibility)
	// Using $ as an illegal character instead
	input := `entity user { $ }`

	lexer := NewLexer(input)

	// entity
	_, err := lexer.NextToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// user
	_, err = lexer.NextToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// {
	_, err = lexer.NextToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// $ should cause error
	_, err = lexer.NextToken()
	if err == nil {
		t.Fatal("expected error for illegal character, got nil")
	}
}

func TestLexer_LineAndColumn(t *testing.T) {
	input := `entity user {
  relation owner @user
}`

	expected := []struct {
		tokenType TokenType
		line      int
		column    int
	}{
		{TOKEN_ENTITY, 1, 1},
		{TOKEN_IDENTIFIER, 1, 8},
		{TOKEN_LBRACE, 1, 13},
		{TOKEN_RELATION, 2, 3},
		{TOKEN_IDENTIFIER, 2, 12},
		{TOKEN_AT, 2, 18},
		{TOKEN_IDENTIFIER, 2, 19},
		{TOKEN_RBRACE, 3, 1},
	}

	lexer := NewLexer(input)

	for i, exp := range expected {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("test[%d]: unexpected error: %v", i, err)
		}

		if tok.Type != exp.tokenType {
			t.Errorf("test[%d]: expected token type %v, got %v", i, exp.tokenType, tok.Type)
		}

		if tok.Line != exp.line {
			t.Errorf("test[%d]: expected line %d, got %d", i, exp.line, tok.Line)
		}

		if tok.Column != exp.column {
			t.Errorf("test[%d]: expected column %d, got %d", i, exp.column, tok.Column)
		}
	}
}
