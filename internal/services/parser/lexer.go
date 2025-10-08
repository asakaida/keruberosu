package parser

import (
	"fmt"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF

	// Identifiers and literals
	TOKEN_IDENTIFIER
	TOKEN_STRING // String literals (quoted)

	// Keywords
	TOKEN_ENTITY
	TOKEN_RELATION
	TOKEN_ATTRIBUTE
	TOKEN_PERMISSION
	TOKEN_RULE

	// Operators
	TOKEN_OR
	TOKEN_AND
	TOKEN_NOT
	TOKEN_EQUALS

	// Comparison operators (for CEL expressions in rule())
	TOKEN_EQ          // ==
	TOKEN_NEQ         // !=
	TOKEN_LT          // <
	TOKEN_LTE         // <=
	TOKEN_GT          // >
	TOKEN_GTE         // >=
	TOKEN_LOGICAL_AND // &&
	TOKEN_LOGICAL_OR  // ||
	TOKEN_AMPERSAND   // &
	TOKEN_PIPE        // |
	TOKEN_EXCLAMATION // !

	// Delimiters
	TOKEN_COLON
	TOKEN_LBRACE
	TOKEN_RBRACE
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_LBRACKET
	TOKEN_RBRACKET
	TOKEN_DOT
	TOKEN_COMMA
)

var tokenNames = map[TokenType]string{
	TOKEN_ILLEGAL:     "ILLEGAL",
	TOKEN_EOF:         "EOF",
	TOKEN_IDENTIFIER:  "IDENTIFIER",
	TOKEN_STRING:      "STRING",
	TOKEN_ENTITY:      "entity",
	TOKEN_RELATION:    "relation",
	TOKEN_ATTRIBUTE:   "attribute",
	TOKEN_PERMISSION:  "permission",
	TOKEN_RULE:        "rule",
	TOKEN_OR:          "or",
	TOKEN_AND:         "and",
	TOKEN_NOT:         "not",
	TOKEN_EQUALS:      "=",
	TOKEN_EQ:          "==",
	TOKEN_NEQ:         "!=",
	TOKEN_LT:          "<",
	TOKEN_LTE:         "<=",
	TOKEN_GT:          ">",
	TOKEN_GTE:         ">=",
	TOKEN_LOGICAL_AND: "&&",
	TOKEN_LOGICAL_OR:  "||",
	TOKEN_AMPERSAND:   "&",
	TOKEN_PIPE:        "|",
	TOKEN_EXCLAMATION: "!",
	TOKEN_COLON:       ":",
	TOKEN_LBRACE:      "{",
	TOKEN_RBRACE:      "}",
	TOKEN_LPAREN:      "(",
	TOKEN_RPAREN:      ")",
	TOKEN_LBRACKET:    "[",
	TOKEN_RBRACKET:    "]",
	TOKEN_DOT:         ".",
	TOKEN_COMMA:       ",",
}

var keywords = map[string]TokenType{
	"entity":     TOKEN_ENTITY,
	"relation":   TOKEN_RELATION,
	"attribute":  TOKEN_ATTRIBUTE,
	"permission": TOKEN_PERMISSION,
	"rule":       TOKEN_RULE,
	"or":         TOKEN_OR,
	"and":        TOKEN_AND,
	"not":        TOKEN_NOT,
}

// Token represents a lexical token
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// String returns a string representation of the token
func (t *Token) String() string {
	typeName := tokenNames[t.Type]
	if typeName == "" {
		typeName = fmt.Sprintf("UNKNOWN(%d)", t.Type)
	}
	return fmt.Sprintf("%s(%s) at %d:%d", typeName, t.Value, t.Line, t.Column)
}

// Lexer performs lexical analysis
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	column       int
}

// NewLexer creates a new Lexer
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++

	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// skipComment skips single-line comments starting with //
func (l *Lexer) skipComment() {
	if l.ch == '/' && l.peekChar() == '/' {
		// Skip until end of line
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
	}
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number literal
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	// Handle decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// readString reads a string literal enclosed in quotes
func (l *Lexer) readString() string {
	position := l.position + 1 // Skip opening quote
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

// NextToken returns the next token
func (l *Lexer) NextToken() (*Token, error) {
	// Skip whitespace and comments in a loop
	for {
		l.skipWhitespace()
		if l.ch == '/' && l.peekChar() == '/' {
			l.skipComment()
		} else {
			break
		}
	}

	var tok *Token
	line := l.line
	column := l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = &Token{Type: TOKEN_EQ, Value: "==", Line: line, Column: column}
			l.readChar()
		} else {
			tok = &Token{Type: TOKEN_EQUALS, Value: "=", Line: line, Column: column}
			l.readChar()
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = &Token{Type: TOKEN_NEQ, Value: "!=", Line: line, Column: column}
			l.readChar()
		} else {
			tok = &Token{Type: TOKEN_EXCLAMATION, Value: "!", Line: line, Column: column}
			l.readChar()
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = &Token{Type: TOKEN_LTE, Value: "<=", Line: line, Column: column}
			l.readChar()
		} else {
			tok = &Token{Type: TOKEN_LT, Value: "<", Line: line, Column: column}
			l.readChar()
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = &Token{Type: TOKEN_GTE, Value: ">=", Line: line, Column: column}
			l.readChar()
		} else {
			tok = &Token{Type: TOKEN_GT, Value: ">", Line: line, Column: column}
			l.readChar()
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = &Token{Type: TOKEN_LOGICAL_AND, Value: "&&", Line: line, Column: column}
			l.readChar()
		} else {
			tok = &Token{Type: TOKEN_AMPERSAND, Value: "&", Line: line, Column: column}
			l.readChar()
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = &Token{Type: TOKEN_LOGICAL_OR, Value: "||", Line: line, Column: column}
			l.readChar()
		} else {
			tok = &Token{Type: TOKEN_PIPE, Value: "|", Line: line, Column: column}
			l.readChar()
		}
	case ':':
		tok = &Token{Type: TOKEN_COLON, Value: ":", Line: line, Column: column}
		l.readChar()
	case '{':
		tok = &Token{Type: TOKEN_LBRACE, Value: "{", Line: line, Column: column}
		l.readChar()
	case '}':
		tok = &Token{Type: TOKEN_RBRACE, Value: "}", Line: line, Column: column}
		l.readChar()
	case '(':
		tok = &Token{Type: TOKEN_LPAREN, Value: "(", Line: line, Column: column}
		l.readChar()
	case ')':
		tok = &Token{Type: TOKEN_RPAREN, Value: ")", Line: line, Column: column}
		l.readChar()
	case '[':
		tok = &Token{Type: TOKEN_LBRACKET, Value: "[", Line: line, Column: column}
		l.readChar()
	case ']':
		tok = &Token{Type: TOKEN_RBRACKET, Value: "]", Line: line, Column: column}
		l.readChar()
	case '.':
		tok = &Token{Type: TOKEN_DOT, Value: ".", Line: line, Column: column}
		l.readChar()
	case ',':
		tok = &Token{Type: TOKEN_COMMA, Value: ",", Line: line, Column: column}
		l.readChar()
	case '"':
		value := l.readString()
		tok = &Token{Type: TOKEN_STRING, Value: value, Line: line, Column: column}
		l.readChar() // Skip closing quote
	case 0:
		tok = &Token{Type: TOKEN_EOF, Value: "", Line: line, Column: column}
	default:
		if isLetter(l.ch) {
			value := l.readIdentifier()
			tokenType := TOKEN_IDENTIFIER
			if kw, ok := keywords[value]; ok {
				tokenType = kw
			}
			tok = &Token{Type: tokenType, Value: value, Line: line, Column: column}
			return tok, nil
		} else if isDigit(l.ch) {
			value := l.readNumber()
			tok = &Token{Type: TOKEN_IDENTIFIER, Value: value, Line: line, Column: column}
			return tok, nil
		} else {
			return nil, fmt.Errorf("illegal character '%c' at %d:%d", l.ch, line, column)
		}
	}

	return tok, nil
}

// isLetter checks if a character is a letter
func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}

// isDigit checks if a character is a digit
func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}
