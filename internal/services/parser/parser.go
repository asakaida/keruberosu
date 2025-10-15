package parser

import (
	"fmt"
	"strings"
)

// Parser parses the DSL into an AST
type Parser struct {
	lexer   *Lexer
	current *Token
	peek    *Token
	errors  []string
}

// NewParser creates a new Parser
func NewParser(lexer *Lexer) *Parser {
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}

	// Read two tokens to initialize current and peek
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.current = p.peek
	tok, err := p.lexer.NextToken()
	if err != nil {
		p.errors = append(p.errors, err.Error())
		p.peek = &Token{Type: TOKEN_EOF}
	} else {
		p.peek = tok
	}
}

// currentTokenIs checks if the current token is of the given type
func (p *Parser) currentTokenIs(t TokenType) bool {
	return p.current != nil && p.current.Type == t
}

// peekTokenIs checks if the peek token is of the given type
func (p *Parser) peekTokenIs(t TokenType) bool {
	return p.peek != nil && p.peek.Type == t
}

// expectPeek checks if the next token is of the expected type and advances
func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead at %d:%d",
		tokenNames[t], tokenNames[p.peek.Type], p.peek.Line, p.peek.Column)
	p.errors = append(p.errors, msg)
}

// Parse parses the entire schema
func (p *Parser) Parse() (*SchemaAST, error) {
	schema := &SchemaAST{
		Entities: []*EntityAST{},
	}

	for !p.currentTokenIs(TOKEN_EOF) {
		if p.currentTokenIs(TOKEN_ENTITY) {
			entity := p.parseEntity()
			if entity != nil {
				schema.Entities = append(schema.Entities, entity)
			} else {
				// If parseEntity failed, skip to next token to avoid infinite loop
				p.nextToken()
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("unexpected token %s at %d:%d, expected 'entity'",
				tokenNames[p.current.Type], p.current.Line, p.current.Column))
			p.nextToken()
		}
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors:\n%s", strings.Join(p.errors, "\n"))
	}

	return schema, nil
}

// parseEntity parses an entity definition
func (p *Parser) parseEntity() *EntityAST {
	entity := &EntityAST{
		Relations:   []*RelationAST{},
		Attributes:  []*AttributeAST{},
		Permissions: []*PermissionAST{},
	}

	// Expect identifier (entity name)
	if !p.expectPeek(TOKEN_IDENTIFIER) {
		return nil
	}
	entity.Name = p.current.Value

	// Expect {
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}

	// Parse entity body
	p.nextToken()
	for !p.currentTokenIs(TOKEN_RBRACE) && !p.currentTokenIs(TOKEN_EOF) {
		switch {
		case p.currentTokenIs(TOKEN_RELATION):
			relation := p.parseRelation()
			if relation != nil {
				entity.Relations = append(entity.Relations, relation)
			}
		case p.currentTokenIs(TOKEN_ATTRIBUTE):
			attribute := p.parseAttribute()
			if attribute != nil {
				entity.Attributes = append(entity.Attributes, attribute)
			}
		case p.currentTokenIs(TOKEN_PERMISSION):
			permission := p.parsePermission()
			if permission != nil {
				entity.Permissions = append(entity.Permissions, permission)
			}
		case p.currentTokenIs(TOKEN_ACTION): // Permify互換: actionはpermissionのエイリアス
			permission := p.parsePermission()
			if permission != nil {
				entity.Permissions = append(entity.Permissions, permission)
			}
		default:
			p.errors = append(p.errors, fmt.Sprintf("unexpected token %s in entity at %d:%d",
				tokenNames[p.current.Type], p.current.Line, p.current.Column))
			p.nextToken()
		}
	}

	// Expect }
	if !p.currentTokenIs(TOKEN_RBRACE) {
		p.errors = append(p.errors, fmt.Sprintf("expected '}' at end of entity, got %s at %d:%d",
			tokenNames[p.current.Type], p.current.Line, p.current.Column))
		return nil
	}

	p.nextToken()
	return entity
}

// parseRelation parses a relation definition
// Permify syntax only: "relation owner @user"
func (p *Parser) parseRelation() *RelationAST {
	relation := &RelationAST{}

	// Expect identifier (relation name)
	if !p.expectPeek(TOKEN_IDENTIFIER) {
		return nil
	}
	relation.Name = p.current.Value

	// Expect @ (Permify syntax)
	if !p.expectPeek(TOKEN_AT) {
		return nil
	}

	// Expect identifier (target type)
	if !p.expectPeek(TOKEN_IDENTIFIER) {
		return nil
	}
	relation.TargetType = p.current.Value

	// Check for additional types (e.g., "user | team#member" or "@user @team#member")
	// This handles multi-type relations
	for p.peekTokenIs(TOKEN_PIPE) || p.peekTokenIs(TOKEN_AT) {
		p.nextToken() // consume | or @
		if !p.expectPeek(TOKEN_IDENTIFIER) {
			return nil
		}
		// Append additional type with pipe separator (normalize to | syntax internally)
		relation.TargetType += " | " + p.current.Value

		// Handle subject relation (e.g., team#member)
		if p.peekTokenIs(TOKEN_DOT) || p.peekTokenIs(TOKEN_COMMA) {
			// Skip for now - complex multi-type with relations
			// This would be handled by the authorization engine
		}
	}

	p.nextToken()
	return relation
}

// parseAttribute parses an attribute definition
func (p *Parser) parseAttribute() *AttributeAST {
	attribute := &AttributeAST{}

	// Expect identifier (attribute name)
	if !p.expectPeek(TOKEN_IDENTIFIER) {
		return nil
	}
	attribute.Name = p.current.Value

	// Expect :
	if !p.expectPeek(TOKEN_COLON) {
		return nil
	}

	// Expect identifier (type)
	if !p.expectPeek(TOKEN_IDENTIFIER) {
		return nil
	}
	attributeType := p.current.Value

	// Check for array type (e.g., string[])
	if p.peekTokenIs(TOKEN_LBRACKET) {
		p.nextToken() // consume [
		if !p.expectPeek(TOKEN_RBRACKET) {
			return nil
		}
		attributeType += "[]"
	}

	attribute.Type = attributeType

	p.nextToken()
	return attribute
}

// parsePermission parses a permission definition
func (p *Parser) parsePermission() *PermissionAST {
	permission := &PermissionAST{}

	// Expect identifier (permission name)
	if !p.expectPeek(TOKEN_IDENTIFIER) {
		return nil
	}
	permission.Name = p.current.Value

	// Expect =
	if !p.expectPeek(TOKEN_EQUALS) {
		return nil
	}

	// Parse permission rule
	p.nextToken()
	rule := p.parsePermissionRule()
	if rule == nil {
		return nil
	}
	permission.Rule = rule

	return permission
}

// parsePermissionRule parses a permission rule (recursive)
func (p *Parser) parsePermissionRule() PermissionRuleAST {
	return p.parseOrExpression()
}

// parseOrExpression parses OR expressions
func (p *Parser) parseOrExpression() PermissionRuleAST {
	left := p.parseAndExpression()
	if left == nil {
		return nil
	}

	for p.currentTokenIs(TOKEN_OR) {
		p.nextToken()
		right := p.parseAndExpression()
		if right == nil {
			return nil
		}
		left = &LogicalPermissionAST{
			Operator: "or",
			Left:     left,
			Right:    right,
		}
	}

	return left
}

// parseAndExpression parses AND expressions
func (p *Parser) parseAndExpression() PermissionRuleAST {
	left := p.parseUnaryExpression()
	if left == nil {
		return nil
	}

	for p.currentTokenIs(TOKEN_AND) {
		p.nextToken()
		right := p.parseUnaryExpression()
		if right == nil {
			return nil
		}
		left = &LogicalPermissionAST{
			Operator: "and",
			Left:     left,
			Right:    right,
		}
	}

	return left
}

// parseUnaryExpression parses unary expressions (NOT)
func (p *Parser) parseUnaryExpression() PermissionRuleAST {
	if p.currentTokenIs(TOKEN_NOT) {
		p.nextToken()
		expr := p.parsePrimaryExpression()
		if expr == nil {
			return nil
		}
		return &LogicalPermissionAST{
			Operator: "not",
			Left:     expr,
			Right:    nil,
		}
	}

	return p.parsePrimaryExpression()
}

// parsePrimaryExpression parses primary expressions
func (p *Parser) parsePrimaryExpression() PermissionRuleAST {
	switch {
	case p.currentTokenIs(TOKEN_LPAREN):
		// Grouped expression
		p.nextToken()
		expr := p.parsePermissionRule()
		if expr == nil {
			return nil
		}
		if !p.currentTokenIs(TOKEN_RPAREN) {
			p.errors = append(p.errors, fmt.Sprintf("expected ')' at %d:%d", p.current.Line, p.current.Column))
			return nil
		}
		p.nextToken()
		return expr

	case p.currentTokenIs(TOKEN_RULE):
		// ABAC rule
		return p.parseRuleExpression()

	case p.currentTokenIs(TOKEN_IDENTIFIER):
		// Could be relation or hierarchical permission
		name := p.current.Value

		// Check for hierarchical permission (relation.permission)
		if p.peekTokenIs(TOKEN_DOT) {
			p.nextToken() // consume .
			if !p.expectPeek(TOKEN_IDENTIFIER) {
				return nil
			}
			permissionName := p.current.Value
			p.nextToken()
			return &HierarchicalPermissionAST{
				Relation:   name,
				Permission: permissionName,
			}
		}

		// Simple relation permission
		p.nextToken()
		return &RelationPermissionAST{
			Relation: name,
		}

	default:
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s in permission rule at %d:%d",
			tokenNames[p.current.Type], p.current.Line, p.current.Column))
		return nil
	}
}

// parseRuleExpression parses a rule() expression
func (p *Parser) parseRuleExpression() PermissionRuleAST {
	// Expect (
	if !p.expectPeek(TOKEN_LPAREN) {
		return nil
	}

	// Read the CEL expression until closing )
	p.nextToken()
	var expressionParts []string
	parenCount := 1
	prevToken := &Token{Type: TOKEN_LPAREN}

	for parenCount > 0 && !p.currentTokenIs(TOKEN_EOF) {
		if p.currentTokenIs(TOKEN_LPAREN) {
			parenCount++
		} else if p.currentTokenIs(TOKEN_RPAREN) {
			parenCount--
			if parenCount == 0 {
				break
			}
		}

		// Add token value with proper spacing and quoting
		tokenValue := p.current.Value

		// Add quotes back for string literals
		if p.current.Type == TOKEN_STRING {
			tokenValue = `"` + tokenValue + `"`
		}

		// Add space before token if needed
		if len(expressionParts) > 0 && needsSpaceBefore(prevToken, p.current) {
			expressionParts = append(expressionParts, " ")
		}

		expressionParts = append(expressionParts, tokenValue)
		prevToken = p.current
		p.nextToken()
	}

	if !p.currentTokenIs(TOKEN_RPAREN) {
		p.errors = append(p.errors, "expected ')' at end of rule expression")
		return nil
	}

	expression := strings.Join(expressionParts, "")

	p.nextToken()
	return &RulePermissionAST{
		Expression: expression,
	}
}

// needsSpaceBefore determines if a space is needed between two tokens
func needsSpaceBefore(prev, current *Token) bool {
	// No space after opening paren or before closing paren
	if prev.Type == TOKEN_LPAREN || current.Type == TOKEN_RPAREN {
		return false
	}
	// No space before/after dot
	if prev.Type == TOKEN_DOT || current.Type == TOKEN_DOT {
		return false
	}
	// No space before comma
	if current.Type == TOKEN_COMMA {
		return false
	}
	// No space after comma gets space (handled by next token)
	// No space before opening bracket or after closing bracket
	if current.Type == TOKEN_LBRACKET || prev.Type == TOKEN_RBRACKET {
		return false
	}
	// Default: add space between tokens
	return true
}
