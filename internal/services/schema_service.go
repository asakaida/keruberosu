package services

import (
	"context"
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services/parser"
)

// SchemaServiceInterface defines the interface for schema management operations
type SchemaServiceInterface interface {
	WriteSchema(ctx context.Context, tenantID string, schemaDSL string) (string, error)
	ReadSchema(ctx context.Context, tenantID string) (*entities.Schema, error)
	ValidateSchema(ctx context.Context, schemaDSL string) error
	DeleteSchema(ctx context.Context, tenantID string) error
	GetSchemaEntity(ctx context.Context, tenantID string, version string) (*entities.Schema, error)
}

// SchemaService handles schema management operations
type SchemaService struct {
	schemaRepo repositories.SchemaRepository
}

// NewSchemaService creates a new SchemaService
func NewSchemaService(schemaRepo repositories.SchemaRepository) *SchemaService {
	return &SchemaService{
		schemaRepo: schemaRepo,
	}
}

// WriteSchema parses DSL, validates it, and creates a new schema version
func (s *SchemaService) WriteSchema(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
	// Validate input
	if tenantID == "" {
		return "", fmt.Errorf("tenant ID is required")
	}
	if schemaDSL == "" {
		return "", fmt.Errorf("schema DSL is required")
	}

	// Parse DSL
	lexer := parser.NewLexer(schemaDSL)
	p := parser.NewParser(lexer)
	ast, err := p.Parse()
	if err != nil {
		return "", fmt.Errorf("failed to parse DSL: %w", err)
	}

	// Validate schema
	validator := parser.NewValidator(ast)
	if err := validator.Validate(); err != nil {
		return "", fmt.Errorf("schema validation failed: %w", err)
	}

	// Convert AST to entities.Schema for validation
	_, err = parser.ASTToSchema(tenantID, ast)
	if err != nil {
		return "", fmt.Errorf("failed to convert schema: %w", err)
	}

	// Always create a new version (Permify-compatible behavior)
	version, err := s.schemaRepo.Create(ctx, tenantID, schemaDSL)
	if err != nil {
		return "", fmt.Errorf("failed to create schema version: %w", err)
	}

	return version, nil
}

// ReadSchema retrieves the latest schema for a tenant
func (s *SchemaService) ReadSchema(ctx context.Context, tenantID string) (*entities.Schema, error) {
	// Validate input
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required")
	}

	// Get latest schema from database
	schema, err := s.schemaRepo.GetLatestVersion(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	if schema == nil {
		return nil, fmt.Errorf("schema not found for tenant: %s", tenantID)
	}

	return schema, nil
}

// ValidateSchema validates a DSL string without saving it
func (s *SchemaService) ValidateSchema(ctx context.Context, schemaDSL string) error {
	// Validate input
	if schemaDSL == "" {
		return fmt.Errorf("schema DSL is required")
	}

	// Parse DSL
	lexer := parser.NewLexer(schemaDSL)
	p := parser.NewParser(lexer)
	ast, err := p.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse DSL: %w", err)
	}

	// Validate schema
	validator := parser.NewValidator(ast)
	if err := validator.Validate(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	return nil
}

// DeleteSchema deletes the schema for a tenant
func (s *SchemaService) DeleteSchema(ctx context.Context, tenantID string) error {
	// Validate input
	if tenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}

	// Delete schema
	if err := s.schemaRepo.Delete(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	return nil
}

// GetSchemaEntity retrieves the parsed schema entity for internal use
// version="" means use the latest version
func (s *SchemaService) GetSchemaEntity(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
	// Validate input
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required")
	}

	// Get schema from database
	var dbSchema *entities.Schema
	var err error

	if version == "" {
		// Get latest version
		dbSchema, err = s.schemaRepo.GetLatestVersion(ctx, tenantID)
	} else {
		// Get specific version
		dbSchema, err = s.schemaRepo.GetByVersion(ctx, tenantID, version)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Parse DSL to populate Entities field
	lexer := parser.NewLexer(dbSchema.DSL)
	p := parser.NewParser(lexer)
	ast, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema DSL: %w", err)
	}

	// Convert AST to Schema with Entities populated
	parsedSchema, err := parser.ASTToSchema(tenantID, ast)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AST to schema: %w", err)
	}

	// Preserve metadata from database
	parsedSchema.Version = dbSchema.Version
	parsedSchema.CreatedAt = dbSchema.CreatedAt
	parsedSchema.UpdatedAt = dbSchema.UpdatedAt

	return parsedSchema, nil
}
