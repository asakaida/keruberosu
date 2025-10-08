package services

import (
	"context"
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services/parser"
)

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

// WriteSchema parses DSL, validates it, and saves to the database
func (s *SchemaService) WriteSchema(ctx context.Context, tenantID string, schemaDSL string) error {
	// Validate input
	if tenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
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

	// Convert AST to entities.Schema for potential future use
	// (currently we just store the DSL string, but this validates the conversion works)
	_, err = parser.ASTToSchema(tenantID, ast)
	if err != nil {
		return fmt.Errorf("failed to convert schema: %w", err)
	}

	// Check if schema already exists
	existingSchema, err := s.schemaRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check existing schema: %w", err)
	}

	// Create or update schema
	if existingSchema == nil {
		// Create new schema
		if err := s.schemaRepo.Create(ctx, tenantID, schemaDSL); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	} else {
		// Update existing schema
		if err := s.schemaRepo.Update(ctx, tenantID, schemaDSL); err != nil {
			return fmt.Errorf("failed to update schema: %w", err)
		}
	}

	return nil
}

// ReadSchema retrieves the schema for a tenant
func (s *SchemaService) ReadSchema(ctx context.Context, tenantID string) (string, error) {
	// Validate input
	if tenantID == "" {
		return "", fmt.Errorf("tenant ID is required")
	}

	// Get schema from database
	schema, err := s.schemaRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get schema: %w", err)
	}

	if schema == nil {
		return "", fmt.Errorf("schema not found for tenant: %s", tenantID)
	}

	// For Phase 1, we return the stored DSL directly
	// In the future, we could parse and regenerate it for normalization:
	// lexer := parser.NewLexer(schema.DSL)
	// p := parser.NewParser(lexer)
	// ast, _ := p.Parse()
	// gen := parser.NewGenerator()
	// return gen.Generate(ast), nil

	return schema.DSL, nil
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
func (s *SchemaService) GetSchemaEntity(ctx context.Context, tenantID string) (*entities.Schema, error) {
	// Validate input
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required")
	}

	// Get schema from database
	schema, err := s.schemaRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	if schema == nil {
		return nil, fmt.Errorf("schema not found for tenant: %s", tenantID)
	}

	return schema, nil
}
