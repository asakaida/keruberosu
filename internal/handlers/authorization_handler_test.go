package handlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "github.com/asakaida/keruberosu/proto/keruberosu/v1"
	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// Mock SchemaService
type mockSchemaService struct {
	writeSchemaFunc     func(ctx context.Context, tenantID string, schemaDSL string) error
	readSchemaFunc      func(ctx context.Context, tenantID string) (string, error)
	getSchemaEntityFunc func(ctx context.Context, tenantID string) (*entities.Schema, error)
}

func (m *mockSchemaService) WriteSchema(ctx context.Context, tenantID string, schemaDSL string) error {
	if m.writeSchemaFunc != nil {
		return m.writeSchemaFunc(ctx, tenantID, schemaDSL)
	}
	return nil
}

func (m *mockSchemaService) ReadSchema(ctx context.Context, tenantID string) (string, error) {
	if m.readSchemaFunc != nil {
		return m.readSchemaFunc(ctx, tenantID)
	}
	return "", nil
}

func (m *mockSchemaService) ValidateSchema(ctx context.Context, schemaDSL string) error {
	return nil
}

func (m *mockSchemaService) DeleteSchema(ctx context.Context, tenantID string) error {
	return nil
}

func (m *mockSchemaService) GetSchemaEntity(ctx context.Context, tenantID string) (*entities.Schema, error) {
	if m.getSchemaEntityFunc != nil {
		return m.getSchemaEntityFunc(ctx, tenantID)
	}
	return nil, nil
}

// Mock RelationRepository
type mockRelationRepository struct {
	batchWriteFunc  func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
	batchDeleteFunc func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error
}

func (m *mockRelationRepository) Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	return nil
}

func (m *mockRelationRepository) Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	return nil
}

func (m *mockRelationRepository) Read(ctx context.Context, tenantID string, filter *repositories.RelationFilter) ([]*entities.RelationTuple, error) {
	return nil, nil
}

func (m *mockRelationRepository) CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
	return false, nil
}

func (m *mockRelationRepository) BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if m.batchWriteFunc != nil {
		return m.batchWriteFunc(ctx, tenantID, tuples)
	}
	return nil
}

func (m *mockRelationRepository) BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if m.batchDeleteFunc != nil {
		return m.batchDeleteFunc(ctx, tenantID, tuples)
	}
	return nil
}

// Mock AttributeRepository
type mockAttributeRepository struct {
	writeFunc func(ctx context.Context, tenantID string, attr *entities.Attribute) error
}

func (m *mockAttributeRepository) Write(ctx context.Context, tenantID string, attr *entities.Attribute) error {
	if m.writeFunc != nil {
		return m.writeFunc(ctx, tenantID, attr)
	}
	return nil
}

func (m *mockAttributeRepository) Read(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockAttributeRepository) Delete(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) error {
	return nil
}

func (m *mockAttributeRepository) GetValue(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) (interface{}, error) {
	return nil, nil
}

// Mock Checker
type mockChecker struct {
	checkFunc func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error)
}

func (m *mockChecker) Check(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, req)
	}
	return &authorization.CheckResponse{Allowed: false}, nil
}

// Mock Expander
type mockExpander struct {
	expandFunc func(ctx context.Context, req *authorization.ExpandRequest) (*authorization.ExpandResponse, error)
}

func (m *mockExpander) Expand(ctx context.Context, req *authorization.ExpandRequest) (*authorization.ExpandResponse, error) {
	if m.expandFunc != nil {
		return m.expandFunc(ctx, req)
	}
	return &authorization.ExpandResponse{Tree: &authorization.ExpandNode{Type: "leaf"}}, nil
}

// Mock Lookup
type mockLookup struct {
	lookupEntityFunc  func(ctx context.Context, req *authorization.LookupEntityRequest) (*authorization.LookupEntityResponse, error)
	lookupSubjectFunc func(ctx context.Context, req *authorization.LookupSubjectRequest) (*authorization.LookupSubjectResponse, error)
}

func (m *mockLookup) LookupEntity(ctx context.Context, req *authorization.LookupEntityRequest) (*authorization.LookupEntityResponse, error) {
	if m.lookupEntityFunc != nil {
		return m.lookupEntityFunc(ctx, req)
	}
	return &authorization.LookupEntityResponse{EntityIDs: []string{}}, nil
}

func (m *mockLookup) LookupSubject(ctx context.Context, req *authorization.LookupSubjectRequest) (*authorization.LookupSubjectResponse, error) {
	if m.lookupSubjectFunc != nil {
		return m.lookupSubjectFunc(ctx, req)
	}
	return &authorization.LookupSubjectResponse{SubjectIDs: []string{}}, nil
}

// Mock SchemaRepository
type mockSchemaRepository struct {
	getByTenantFunc func(ctx context.Context, tenantID string) (*entities.Schema, error)
}

func (m *mockSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) error {
	return nil
}

func (m *mockSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	if m.getByTenantFunc != nil {
		return m.getByTenantFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockSchemaRepository) Update(ctx context.Context, tenantID string, schemaDSL string) error {
	return nil
}

func (m *mockSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	return nil
}

// === Schema Management Tests ===

func TestAuthorizationHandler_WriteSchema_Success(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			return nil
		},
	}

	handler := NewAuthorizationHandler(
		mockService,
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteSchemaRequest{
		SchemaDsl: `entity user {}`,
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success=true, got false")
	}

	if resp.Message != "Schema written successfully" {
		t.Errorf("expected success message, got %s", resp.Message)
	}

	if len(resp.Errors) != 0 {
		t.Errorf("expected no errors, got %v", resp.Errors)
	}
}

func TestAuthorizationHandler_WriteSchema_EmptyDSL(t *testing.T) {
	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteSchemaRequest{
		SchemaDsl: "",
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success=false for empty DSL")
	}

	if len(resp.Errors) == 0 {
		t.Errorf("expected errors for empty DSL")
	}
}

func TestAuthorizationHandler_WriteSchema_ParseError(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) error {
			return fmt.Errorf("failed to parse DSL: syntax error at line 1")
		},
	}

	handler := NewAuthorizationHandler(
		mockService,
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteSchemaRequest{
		SchemaDsl: `invalid syntax`,
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success=false for parse error")
	}

	if resp.Message != "Failed to parse schema DSL" {
		t.Errorf("expected parse error message, got %s", resp.Message)
	}

	if len(resp.Errors) == 0 {
		t.Errorf("expected errors for parse error")
	}
}

func TestAuthorizationHandler_WriteSchema_ValidationError(t *testing.T) {
	mockService := &mockSchemaService{
		writeSchemaFunc: func(ctx context.Context, tenantID string, schemaDSL string) error {
			return fmt.Errorf("schema validation failed: undefined relation reference")
		},
	}

	handler := NewAuthorizationHandler(
		mockService,
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteSchemaRequest{
		SchemaDsl: `entity document { permission view = owner }`,
	}

	resp, err := handler.WriteSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success=false for validation error")
	}

	if resp.Message != "Schema validation failed" {
		t.Errorf("expected validation error message, got %s", resp.Message)
	}

	if len(resp.Errors) == 0 {
		t.Errorf("expected errors for validation error")
	}
}

func TestAuthorizationHandler_ReadSchema_Success(t *testing.T) {
	updatedAt := time.Now()
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (string, error) {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			return `entity user {}`, nil
		},
		getSchemaEntityFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID:  tenantID,
				DSL:       `entity user {}`,
				UpdatedAt: updatedAt,
			}, nil
		},
	}

	handler := NewAuthorizationHandler(
		mockService,
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.ReadSchemaRequest{}

	resp, err := handler.ReadSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.SchemaDsl != `entity user {}` {
		t.Errorf("expected schema DSL 'entity user {}', got %s", resp.SchemaDsl)
	}

	if resp.UpdatedAt == "" {
		t.Errorf("expected updated_at to be set")
	}
}

func TestAuthorizationHandler_ReadSchema_NotFound(t *testing.T) {
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (string, error) {
			return "", fmt.Errorf("schema not found for tenant: %s", tenantID)
		},
	}

	handler := NewAuthorizationHandler(
		mockService,
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.ReadSchemaRequest{}

	_, err := handler.ReadSchema(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for schema not found")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_ReadSchema_NoUpdatedAt(t *testing.T) {
	mockService := &mockSchemaService{
		readSchemaFunc: func(ctx context.Context, tenantID string) (string, error) {
			return `entity user {}`, nil
		},
		getSchemaEntityFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID:  tenantID,
				DSL:       `entity user {}`,
				UpdatedAt: time.Time{}, // Zero value
			}, nil
		},
	}

	handler := NewAuthorizationHandler(
		mockService,
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.ReadSchemaRequest{}

	resp, err := handler.ReadSchema(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.UpdatedAt != "" {
		t.Errorf("expected empty updated_at, got %s", resp.UpdatedAt)
	}
}

// === Data Management Tests ===

func TestAuthorizationHandler_WriteRelations_Success(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		batchWriteFunc: func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if len(tuples) != 2 {
				t.Errorf("expected 2 tuples, got %d", len(tuples))
			}
			return nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		mockRelationRepo,
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
			{
				Entity:   &pb.Entity{Type: "document", Id: "2"},
				Relation: "editor",
				Subject:  &pb.Entity{Type: "user", Id: "bob"},
			},
		},
	}

	resp, err := handler.WriteRelations(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.WrittenCount != 2 {
		t.Errorf("expected written count 2, got %d", resp.WrittenCount)
	}
}

func TestAuthorizationHandler_WriteRelations_EmptyTuples(t *testing.T) {
	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{},
	}

	_, err := handler.WriteRelations(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty tuples")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_WriteRelations_InvalidTuple(t *testing.T) {
	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: ""}, // Missing ID
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
		},
	}

	_, err := handler.WriteRelations(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid tuple")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_DeleteRelations_Success(t *testing.T) {
	mockRelationRepo := &mockRelationRepository{
		batchDeleteFunc: func(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			if len(tuples) != 1 {
				t.Errorf("expected 1 tuple, got %d", len(tuples))
			}
			return nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		mockRelationRepo,
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.DeleteRelationsRequest{
		Tuples: []*pb.RelationTuple{
			{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
		},
	}

	resp, err := handler.DeleteRelations(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.DeletedCount != 1 {
		t.Errorf("expected deleted count 1, got %d", resp.DeletedCount)
	}
}

func TestAuthorizationHandler_DeleteRelations_EmptyTuples(t *testing.T) {
	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.DeleteRelationsRequest{
		Tuples: []*pb.RelationTuple{},
	}

	_, err := handler.DeleteRelations(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty tuples")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_WriteAttributes_Success(t *testing.T) {
	writtenAttrs := 0
	mockAttrRepo := &mockAttributeRepository{
		writeFunc: func(ctx context.Context, tenantID string, attr *entities.Attribute) error {
			if tenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", tenantID)
			}
			writtenAttrs++
			return nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		mockAttrRepo,
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
					"title":  structpb.NewStringValue("Test Document"),
				},
			},
		},
	}

	resp, err := handler.WriteAttributes(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.WrittenCount != 2 {
		t.Errorf("expected written count 2, got %d", resp.WrittenCount)
	}

	if writtenAttrs != 2 {
		t.Errorf("expected 2 attributes written, got %d", writtenAttrs)
	}
}

func TestAuthorizationHandler_WriteAttributes_EmptyAttributes(t *testing.T) {
	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{},
	}

	_, err := handler.WriteAttributes(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty attributes")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_WriteAttributes_InvalidAttribute(t *testing.T) {
	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity: &pb.Entity{Type: "", Id: "1"}, // Missing type
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
				},
			},
		},
	}

	_, err := handler.WriteAttributes(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid attribute")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_WriteAttributes_RepositoryError(t *testing.T) {
	mockAttrRepo := &mockAttributeRepository{
		writeFunc: func(ctx context.Context, tenantID string, attr *entities.Attribute) error {
			return fmt.Errorf("database error")
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		mockAttrRepo,
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.WriteAttributesRequest{
		Attributes: []*pb.AttributeData{
			{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
				},
			},
		},
	}

	_, err := handler.WriteAttributes(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for repository error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected Internal error, got %v", st.Code())
	}
}

// === Authorization Tests ===

func TestAuthorizationHandler_Check_Allowed(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			if req.EntityType != "document" {
				t.Errorf("expected entity type 'document', got %s", req.EntityType)
			}
			return &authorization.CheckResponse{Allowed: true}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.CheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected ALLOWED, got %v", resp.Can)
	}
}

func TestAuthorizationHandler_Check_Denied(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			return &authorization.CheckResponse{Allowed: false}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.CheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Can != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Errorf("expected DENIED, got %v", resp.Can)
	}
}

func TestAuthorizationHandler_Check_MissingEntity(t *testing.T) {
	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.CheckRequest{
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing entity")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_Expand_Success(t *testing.T) {
	mockExpander := &mockExpander{
		expandFunc: func(ctx context.Context, req *authorization.ExpandRequest) (*authorization.ExpandResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			return &authorization.ExpandResponse{
				Tree: &authorization.ExpandNode{
					Type: "union",
					Children: []*authorization.ExpandNode{
						{Type: "leaf", Subject: "user:alice"},
					},
				},
			}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		mockExpander,
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.ExpandRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
	}

	resp, err := handler.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Tree == nil {
		t.Fatal("expected tree to be set")
	}

	if resp.Tree.Operation != "union" {
		t.Errorf("expected operation 'union', got %s", resp.Tree.Operation)
	}
}

func TestAuthorizationHandler_LookupEntity_Success(t *testing.T) {
	mockLookup := &mockLookup{
		lookupEntityFunc: func(ctx context.Context, req *authorization.LookupEntityRequest) (*authorization.LookupEntityResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			return &authorization.LookupEntityResponse{
				EntityIDs:     []string{"doc1", "doc2"},
				NextPageToken: "",
			}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		mockLookup,
		&mockSchemaRepository{},
	)

	req := &pb.LookupEntityRequest{
		EntityType: "document",
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.LookupEntity(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.EntityIds) != 2 {
		t.Errorf("expected 2 entity IDs, got %d", len(resp.EntityIds))
	}
}

func TestAuthorizationHandler_LookupSubject_Success(t *testing.T) {
	mockLookup := &mockLookup{
		lookupSubjectFunc: func(ctx context.Context, req *authorization.LookupSubjectRequest) (*authorization.LookupSubjectResponse, error) {
			if req.TenantID != "default" {
				t.Errorf("expected tenant ID 'default', got %s", req.TenantID)
			}
			return &authorization.LookupSubjectResponse{
				SubjectIDs:    []string{"alice", "bob"},
				NextPageToken: "",
			}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		mockLookup,
		&mockSchemaRepository{},
	)

	req := &pb.LookupSubjectRequest{
		Entity:           &pb.Entity{Type: "document", Id: "1"},
		Permission:       "view",
		SubjectReference: &pb.SubjectReference{Type: "user"},
	}

	resp, err := handler.LookupSubject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.SubjectIds) != 2 {
		t.Errorf("expected 2 subject IDs, got %d", len(resp.SubjectIds))
	}
}

func TestAuthorizationHandler_SubjectPermission_Success(t *testing.T) {
	checkCount := 0
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			checkCount++
			// Allow "view", deny "edit"
			return &authorization.CheckResponse{Allowed: req.Permission == "view"}, nil
		},
	}

	mockSchemaRepo := &mockSchemaRepository{
		getByTenantFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID: tenantID,
				Entities: []*entities.Entity{
					{
						Name: "document",
						Permissions: []*entities.Permission{
							{Name: "view"},
							{Name: "edit"},
						},
					},
				},
			}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		mockSchemaRepo,
	)

	req := &pb.SubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	}

	resp, err := handler.SubjectPermission(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}

	if resp.Results["view"] != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected view to be ALLOWED")
	}

	if resp.Results["edit"] != pb.CheckResult_CHECK_RESULT_DENIED {
		t.Errorf("expected edit to be DENIED")
	}

	if checkCount != 2 {
		t.Errorf("expected 2 checks, got %d", checkCount)
	}
}

func TestAuthorizationHandler_SubjectPermission_SchemaNotFound(t *testing.T) {
	mockSchemaRepo := &mockSchemaRepository{
		getByTenantFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return nil, nil // Schema not found
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		mockSchemaRepo,
	)

	req := &pb.SubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.SubjectPermission(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for schema not found")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_SubjectPermission_EntityNotFound(t *testing.T) {
	mockSchemaRepo := &mockSchemaRepository{
		getByTenantFunc: func(ctx context.Context, tenantID string) (*entities.Schema, error) {
			return &entities.Schema{
				TenantID: tenantID,
				Entities: []*entities.Entity{
					{Name: "other_entity"},
				},
			}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		&mockChecker{},
		&mockExpander{},
		&mockLookup{},
		mockSchemaRepo,
	)

	req := &pb.SubjectPermissionRequest{
		Entity:  &pb.Entity{Type: "document", Id: "1"},
		Subject: &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.SubjectPermission(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for entity not found")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

func TestAuthorizationHandler_Check_WithContextualTuples(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			if len(req.ContextualTuples) != 1 {
				t.Errorf("expected 1 contextual tuple, got %d", len(req.ContextualTuples))
			}
			return &authorization.CheckResponse{Allowed: true}, nil
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.CheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
		Context: &pb.Context{
			Tuples: []*pb.RelationTuple{
				{
					Entity:   &pb.Entity{Type: "document", Id: "1"},
					Relation: "owner",
					Subject:  &pb.Entity{Type: "user", Id: "alice"},
				},
			},
		},
	}

	resp, err := handler.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Can != pb.CheckResult_CHECK_RESULT_ALLOWED {
		t.Errorf("expected ALLOWED, got %v", resp.Can)
	}
}

func TestAuthorizationHandler_Check_CheckerError(t *testing.T) {
	mockChecker := &mockChecker{
		checkFunc: func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
			return nil, fmt.Errorf("internal error")
		},
	}

	handler := NewAuthorizationHandler(
		&mockSchemaService{},
		&mockRelationRepository{},
		&mockAttributeRepository{},
		mockChecker,
		&mockExpander{},
		&mockLookup{},
		&mockSchemaRepository{},
	)

	req := &pb.CheckRequest{
		Entity:     &pb.Entity{Type: "document", Id: "1"},
		Permission: "view",
		Subject:    &pb.Subject{Type: "user", Id: "alice"},
	}

	_, err := handler.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for checker error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected Internal error, got %v", st.Code())
	}
}

// === Helper Function Tests ===

func TestProtoToRelationTuple(t *testing.T) {
	tests := []struct {
		name      string
		proto     *pb.RelationTuple
		wantError bool
	}{
		{
			name: "valid tuple",
			proto: &pb.RelationTuple{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
			wantError: false,
		},
		{
			name: "missing entity",
			proto: &pb.RelationTuple{
				Relation: "owner",
				Subject:  &pb.Entity{Type: "user", Id: "alice"},
			},
			wantError: true,
		},
		{
			name: "missing relation",
			proto: &pb.RelationTuple{
				Entity:  &pb.Entity{Type: "document", Id: "1"},
				Subject: &pb.Entity{Type: "user", Id: "alice"},
			},
			wantError: true,
		},
		{
			name: "missing subject",
			proto: &pb.RelationTuple{
				Entity:   &pb.Entity{Type: "document", Id: "1"},
				Relation: "owner",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := protoToRelationTuple(tt.proto)
			if (err != nil) != tt.wantError {
				t.Errorf("protoToRelationTuple() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestProtoToAttributes(t *testing.T) {
	tests := []struct {
		name      string
		proto     *pb.AttributeData
		wantError bool
		wantCount int
	}{
		{
			name: "valid attributes",
			proto: &pb.AttributeData{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
					"title":  structpb.NewStringValue("Test"),
				},
			},
			wantError: false,
			wantCount: 2,
		},
		{
			name: "missing entity",
			proto: &pb.AttributeData{
				Data: map[string]*structpb.Value{
					"public": structpb.NewBoolValue(true),
				},
			},
			wantError: true,
		},
		{
			name: "empty data",
			proto: &pb.AttributeData{
				Entity: &pb.Entity{Type: "document", Id: "1"},
				Data:   map[string]*structpb.Value{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs, err := protoToAttributes(tt.proto)
			if (err != nil) != tt.wantError {
				t.Errorf("protoToAttributes() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && len(attrs) != tt.wantCount {
				t.Errorf("protoToAttributes() got %d attributes, want %d", len(attrs), tt.wantCount)
			}
		})
	}
}
