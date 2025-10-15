package handlers

import (
	"context"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services"
	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
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

	version, err := h.schemaService.WriteSchema(ctx, tenantID, req.Schema)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to write schema: %v", err)
	}

	return &pb.SchemaWriteResponse{
		SchemaVersion: version,
	}, nil
}

// Read handles the Read RPC
func (h *SchemaHandler) Read(ctx context.Context, req *pb.SchemaReadRequest) (*pb.SchemaReadResponse, error) {
	tenantID := "default"

	var schema *entities.Schema
	var err error

	// Check if specific version is requested
	if req.Metadata != nil && req.Metadata.SchemaVersion != "" {
		schema, err = h.schemaRepo.GetByVersion(ctx, tenantID, req.Metadata.SchemaVersion)
	} else {
		schema, err = h.schemaService.ReadSchema(ctx, tenantID)
	}

	if err != nil {
		return nil, handleReadSchemaError(err)
	}

	updatedAt := ""
	if !schema.UpdatedAt.IsZero() {
		updatedAt = schema.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return &pb.SchemaReadResponse{
		Schema:    schema.DSL,
		UpdatedAt: updatedAt,
	}, nil
}

// List handles the List RPC
func (h *SchemaHandler) List(ctx context.Context, req *pb.SchemaListRequest) (*pb.SchemaListResponse, error) {
	tenantID := "default"

	// Set default page size if not specified
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// For now, we'll use a simple offset-based pagination
	// In production, you might want to implement cursor-based pagination
	offset := 0
	// TODO: Parse continuous_token to get offset if needed

	// Get versions from repository
	versions, err := h.schemaRepo.ListVersions(ctx, tenantID, int(pageSize), offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list schema versions: %v", err)
	}

	// Get the latest version (head)
	var head string
	if len(versions) > 0 {
		head = versions[0].Version
	}

	// Convert to proto format
	schemaItems := make([]*pb.SchemaListItem, len(versions))
	for i, v := range versions {
		schemaItems[i] = &pb.SchemaListItem{
			Version:   v.Version,
			CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return &pb.SchemaListResponse{
		Head:    head,
		Schemas: schemaItems,
		// TODO: Implement continuous_token for pagination
		ContinuousToken: "",
	}, nil
}
