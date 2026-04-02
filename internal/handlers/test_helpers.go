package handlers

import (
	"context"
	"database/sql"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/asakaida/keruberosu/internal/services/authorization"
)

// Mock SchemaService
type mockSchemaService struct {
	writeSchemaFunc     func(ctx context.Context, tenantID string, schemaDSL string) (string, error)
	readSchemaFunc      func(ctx context.Context, tenantID string) (*entities.Schema, error)
	getSchemaEntityFunc func(ctx context.Context, tenantID string, version string) (*entities.Schema, error)
}

func (m *mockSchemaService) WriteSchema(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
	if m.writeSchemaFunc != nil {
		return m.writeSchemaFunc(ctx, tenantID, schemaDSL)
	}
	return "v1", nil
}

func (m *mockSchemaService) ReadSchema(ctx context.Context, tenantID string) (*entities.Schema, error) {
	if m.readSchemaFunc != nil {
		return m.readSchemaFunc(ctx, tenantID)
	}
	return &entities.Schema{}, nil
}

func (m *mockSchemaService) ValidateSchema(ctx context.Context, schemaDSL string) error {
	return nil
}

func (m *mockSchemaService) DeleteSchema(ctx context.Context, tenantID string) error {
	return nil
}

func (m *mockSchemaService) GetSchemaEntity(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
	if m.getSchemaEntityFunc != nil {
		return m.getSchemaEntityFunc(ctx, tenantID, version)
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

func (m *mockRelationRepository) BatchWriteInTx(ctx context.Context, tx *sql.Tx, tenantID string, tuples []*entities.RelationTuple) error {
	return nil
}

func (m *mockRelationRepository) BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if m.batchDeleteFunc != nil {
		return m.batchDeleteFunc(ctx, tenantID, tuples)
	}
	return nil
}

func (m *mockRelationRepository) DeleteByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter) error {
	return nil
}

func (m *mockRelationRepository) ReadByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error) {
	return nil, "", nil
}

func (m *mockRelationRepository) Exists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
	return false, nil
}

func (m *mockRelationRepository) ExistsWithSubjectRelation(ctx context.Context, tenantID string, entityType, entityID, relation, subjectType, subjectID, subjectRelation string) (bool, error) {
	return false, nil
}

func (m *mockRelationRepository) FindByEntityWithRelation(ctx context.Context, tenantID string, entityType, entityID, relation string, limit int) ([]*entities.RelationTuple, error) {
	return nil, nil
}

func (m *mockRelationRepository) LookupAncestorsViaRelation(ctx context.Context, tenantID string, entityType, entityID string, maxDepth int) ([]*entities.RelationTuple, error) {
	return nil, nil
}

func (m *mockRelationRepository) FindHierarchicalWithSubject(ctx context.Context, tenantID string, entityType, entityID, relation, subjectType, subjectID string, maxDepth int) (bool, error) {
	return false, nil
}

func (m *mockRelationRepository) RebuildClosure(ctx context.Context, tenantID string) error {
	return nil
}

func (m *mockRelationRepository) GetSortedEntityIDs(ctx context.Context, tenantID string, entityType string, cursor string, limit int) ([]string, error) {
	return nil, nil
}

func (m *mockRelationRepository) GetSortedSubjectIDs(ctx context.Context, tenantID string, subjectType string, cursor string, limit int) ([]string, error) {
	return nil, nil
}

func (m *mockRelationRepository) LookupAccessibleEntitiesComplex(ctx context.Context, tenantID string, entityType string, relations []string, parentRelations []string, subjectType string, subjectID string, maxDepth int, cursor string, limit int) ([]string, error) {
	return nil, nil
}

func (m *mockRelationRepository) LookupAccessibleSubjectsComplex(ctx context.Context, tenantID string, entityType string, entityID string, relations []string, parentRelations []string, subjectType string, maxDepth int, cursor string, limit int) ([]string, error) {
	return nil, nil
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

func (m *mockAttributeRepository) WriteInTx(ctx context.Context, tx *sql.Tx, tenantID string, attr *entities.Attribute) error {
	return nil
}

func (m *mockAttributeRepository) GetSortedEntityIDs(ctx context.Context, tenantID string, entityType string, cursor string, limit int) ([]string, error) {
	return nil, nil
}

// Mock Checker - implements authorization.CheckerInterface
type mockChecker struct {
	checkFunc func(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error)
}

func (m *mockChecker) Check(ctx context.Context, req *authorization.CheckRequest) (*authorization.CheckResponse, error) {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, req)
	}
	return &authorization.CheckResponse{Allowed: false}, nil
}

func (m *mockChecker) CheckMultiple(ctx context.Context, req *authorization.CheckRequest, permissions []string) (map[string]bool, error) {
	return nil, nil
}

// Mock Expander - implements authorization.ExpanderInterface
type mockExpander struct {
	expandFunc func(ctx context.Context, req *authorization.ExpandRequest) (*authorization.ExpandResponse, error)
}

func (m *mockExpander) Expand(ctx context.Context, req *authorization.ExpandRequest) (*authorization.ExpandResponse, error) {
	if m.expandFunc != nil {
		return m.expandFunc(ctx, req)
	}
	return &authorization.ExpandResponse{Tree: &authorization.ExpandNode{Type: "leaf"}}, nil
}

// Mock Lookup - implements authorization.LookupInterface
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
	getLatestVersionFunc func(ctx context.Context, tenantID string) (*entities.Schema, error)
	getByVersionFunc     func(ctx context.Context, tenantID string, version string) (*entities.Schema, error)
	listVersionsFunc     func(ctx context.Context, tenantID string, limit int, cursor string) ([]*entities.SchemaVersion, error)
}

func (m *mockSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
	return "v1", nil
}

func (m *mockSchemaRepository) GetLatestVersion(ctx context.Context, tenantID string) (*entities.Schema, error) {
	if m.getLatestVersionFunc != nil {
		return m.getLatestVersionFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockSchemaRepository) GetByVersion(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
	if m.getByVersionFunc != nil {
		return m.getByVersionFunc(ctx, tenantID, version)
	}
	return nil, nil
}

func (m *mockSchemaRepository) ListVersions(ctx context.Context, tenantID string, limit int, cursor string) ([]*entities.SchemaVersion, error) {
	if m.listVersionsFunc != nil {
		return m.listVersionsFunc(ctx, tenantID, limit, cursor)
	}
	return nil, nil
}

func (m *mockSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	return m.GetLatestVersion(ctx, tenantID)
}

func (m *mockSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	return nil
}
