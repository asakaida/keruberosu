package handlers

import (
	"context"
	"strings"

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
		if strings.Contains(err.Error(), "parse") || strings.Contains(err.Error(), "validation") {
			return nil, status.Errorf(codes.InvalidArgument, "failed to write schema: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to write schema: %v", err)
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

	if schema == nil {
		return nil, status.Error(codes.NotFound, "schema not found")
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

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	cursor := req.ContinuousToken

	// Fetch one extra to determine if there's a next page
	versions, err := h.schemaRepo.ListVersions(ctx, tenantID, int(pageSize)+1, cursor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list schema versions: %v", err)
	}

	// Determine next page token
	var continuousToken string
	if len(versions) > int(pageSize) {
		continuousToken = versions[pageSize-1].Version
		versions = versions[:pageSize]
	}

	// Head is the latest version (only meaningful on first page)
	var head string
	if len(versions) > 0 && cursor == "" {
		head = versions[0].Version
	}

	schemaItems := make([]*pb.SchemaListItem, len(versions))
	for i, v := range versions {
		schemaItems[i] = &pb.SchemaListItem{
			Version:   v.Version,
			CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return &pb.SchemaListResponse{
		Head:            head,
		Schemas:         schemaItems,
		ContinuousToken: continuousToken,
	}, nil
}
