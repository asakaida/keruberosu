package handlers

import (
	"context"
	"strings"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SchemaHandler handles schema management gRPC requests
type SchemaHandler struct {
	schemaService services.SchemaServiceInterface
}

// NewSchemaHandler creates a new SchemaHandler
func NewSchemaHandler(schemaService services.SchemaServiceInterface) *SchemaHandler {
	return &SchemaHandler{
		schemaService: schemaService,
	}
}

// WriteSchema handles the WriteSchema RPC
func (h *SchemaHandler) WriteSchema(ctx context.Context, req *pb.WriteSchemaRequest) (*pb.WriteSchemaResponse, error) {
	// Validate request
	if req.SchemaDsl == "" {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "schema_dsl is required",
			Errors:  []string{"schema_dsl field cannot be empty"},
		}, nil
	}

	// Phase 1: Use fixed tenant ID "default"
	// In future phases, extract tenant ID from gRPC metadata
	tenantID := "default"

	// Call schema service
	err := h.schemaService.WriteSchema(ctx, tenantID, req.SchemaDsl)
	if err != nil {
		return h.handleWriteSchemaError(err)
	}

	return &pb.WriteSchemaResponse{
		Success: true,
		Message: "Schema written successfully",
		Errors:  nil,
	}, nil
}

// ReadSchema handles the ReadSchema RPC
func (h *SchemaHandler) ReadSchema(ctx context.Context, req *pb.ReadSchemaRequest) (*pb.ReadSchemaResponse, error) {
	// Phase 1: Use fixed tenant ID "default"
	// In future phases, extract tenant ID from gRPC metadata
	tenantID := "default"

	// Call schema service
	schemaDSL, err := h.schemaService.ReadSchema(ctx, tenantID)
	if err != nil {
		return nil, h.handleReadSchemaError(err)
	}

	// Get schema entity to retrieve metadata
	schema, err := h.schemaService.GetSchemaEntity(ctx, tenantID)
	if err != nil {
		return nil, h.handleReadSchemaError(err)
	}

	// Format updated_at as ISO8601
	updatedAt := ""
	if !schema.UpdatedAt.IsZero() {
		updatedAt = schema.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return &pb.ReadSchemaResponse{
		SchemaDsl: schemaDSL,
		UpdatedAt: updatedAt,
	}, nil
}

// handleWriteSchemaError converts domain errors to WriteSchemaResponse with errors
func (h *SchemaHandler) handleWriteSchemaError(err error) (*pb.WriteSchemaResponse, error) {
	errMsg := err.Error()

	// Check error type and categorize
	if strings.Contains(errMsg, "failed to parse DSL") {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "Failed to parse schema DSL",
			Errors:  []string{errMsg},
		}, nil
	}

	if strings.Contains(errMsg, "schema validation failed") {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "Schema validation failed",
			Errors:  []string{errMsg},
		}, nil
	}

	if strings.Contains(errMsg, "is required") {
		return &pb.WriteSchemaResponse{
			Success: false,
			Message: "Validation error",
			Errors:  []string{errMsg},
		}, nil
	}

	// Generic error
	return &pb.WriteSchemaResponse{
		Success: false,
		Message: "Failed to write schema",
		Errors:  []string{errMsg},
	}, nil
}

// handleReadSchemaError converts domain errors to gRPC status errors
func (h *SchemaHandler) handleReadSchemaError(err error) error {
	errMsg := err.Error()

	// Check error type and return appropriate gRPC status
	if strings.Contains(errMsg, "schema not found") || strings.Contains(errMsg, "not found") {
		return status.Errorf(codes.NotFound, "schema not found")
	}

	if strings.Contains(errMsg, "is required") {
		return status.Errorf(codes.InvalidArgument, "%s", errMsg)
	}

	// Generic error
	return status.Errorf(codes.Internal, "failed to read schema: %s", errMsg)
}
