package handlers

import (
	"context"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SchemaHandler handles Schema service gRPC requests
type SchemaHandler struct {
	pb.UnimplementedSchemaServer
	schemaService services.SchemaServiceInterface
	schemaRepo    repositories.SchemaRepository
}

// NewSchemaHandler creates a new SchemaHandler
func NewSchemaHandler(schemaService services.SchemaServiceInterface, schemaRepo repositories.SchemaRepository) *SchemaHandler {
	return &SchemaHandler{
		schemaService: schemaService,
		schemaRepo:    schemaRepo,
	}
}

// Write handles the Write RPC
func (h *SchemaHandler) Write(ctx context.Context, req *pb.SchemaWriteRequest) (*pb.SchemaWriteResponse, error) {
	if req.Schema == "" {
		return nil, status.Error(codes.InvalidArgument, "schema is required")
	}

	tenantID := "default"

	err := h.schemaService.WriteSchema(ctx, tenantID, req.Schema)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to write schema: %v", err)
	}

	return &pb.SchemaWriteResponse{
		SchemaVersion: "", // TODO: implement schema versioning later
	}, nil
}

// Read handles the Read RPC
func (h *SchemaHandler) Read(ctx context.Context, req *pb.SchemaReadRequest) (*pb.SchemaReadResponse, error) {
	tenantID := "default"

	schemaDSL, err := h.schemaService.ReadSchema(ctx, tenantID)
	if err != nil {
		return nil, handleReadSchemaError(err)
	}

	schema, err := h.schemaService.GetSchemaEntity(ctx, tenantID)
	if err != nil {
		return nil, handleReadSchemaError(err)
	}

	updatedAt := ""
	if !schema.UpdatedAt.IsZero() {
		updatedAt = schema.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return &pb.SchemaReadResponse{
		Schema:    schemaDSL,
		UpdatedAt: updatedAt,
	}, nil
}
