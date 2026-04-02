package authorization

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// Mock repositories for testing

type mockSchemaRepository struct {
	schema *entities.Schema
}

func (m *mockSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) (string, error) {
	return "v1", nil
}

func (m *mockSchemaRepository) GetLatestVersion(ctx context.Context, tenantID string) (*entities.Schema, error) {
	return m.schema, nil
}

func (m *mockSchemaRepository) GetByVersion(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
	return m.schema, nil
}

func (m *mockSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	return m.schema, nil
}

func (m *mockSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	return nil
}

// GetSchemaEntity implements SchemaServiceInterface
// In tests, the schema already has Entities populated, so we just return it
func (m *mockSchemaRepository) GetSchemaEntity(ctx context.Context, tenantID string, version string) (*entities.Schema, error) {
	return m.schema, nil
}

type mockRelationRepository struct {
	tuples                                 []*entities.RelationTuple
	lookupAccessibleEntitiesComplexFunc    func(ctx context.Context, tenantID string, entityType string, relations []string, parentRelations []string, subjectType string, subjectID string, maxDepth int, cursor string, limit int) ([]string, error)
	lookupAccessibleSubjectsComplexFunc    func(ctx context.Context, tenantID string, entityType string, entityID string, relations []string, parentRelations []string, subjectType string, maxDepth int, cursor string, limit int) ([]string, error)
}

func (m *mockRelationRepository) Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	m.tuples = append(m.tuples, tuple)
	return nil
}

func (m *mockRelationRepository) Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	return nil
}

func (m *mockRelationRepository) Read(ctx context.Context, tenantID string, filter *repositories.RelationFilter) ([]*entities.RelationTuple, error) {
	var result []*entities.RelationTuple
	for _, tuple := range m.tuples {
		if filter.EntityType != "" && tuple.EntityType != filter.EntityType {
			continue
		}
		if filter.EntityID != "" && tuple.EntityID != filter.EntityID {
			continue
		}
		if filter.Relation != "" && tuple.Relation != filter.Relation {
			continue
		}
		if filter.SubjectType != "" && tuple.SubjectType != filter.SubjectType {
			continue
		}
		if filter.SubjectID != "" && tuple.SubjectID != filter.SubjectID {
			continue
		}
		result = append(result, tuple)
	}
	return result, nil
}

func (m *mockRelationRepository) CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
	tuples, _ := m.Read(ctx, tenantID, &repositories.RelationFilter{
		EntityType:  tuple.EntityType,
		EntityID:    tuple.EntityID,
		Relation:    tuple.Relation,
		SubjectType: tuple.SubjectType,
		SubjectID:   tuple.SubjectID,
	})
	return len(tuples) > 0, nil
}

func (m *mockRelationRepository) BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	m.tuples = append(m.tuples, tuples...)
	return nil
}

func (m *mockRelationRepository) BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	return nil
}

func (m *mockRelationRepository) DeleteByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter) error {
	return nil
}

func (m *mockRelationRepository) ReadByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error) {
	return nil, "", nil
}

func (m *mockRelationRepository) Exists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
	return m.CheckExists(ctx, tenantID, tuple)
}

func (m *mockRelationRepository) ExistsWithSubjectRelation(ctx context.Context, tenantID string, entityType, entityID, relation, subjectType, subjectID, subjectRelation string) (bool, error) {
	for _, tuple := range m.tuples {
		if tuple.EntityType == entityType &&
			tuple.EntityID == entityID &&
			tuple.Relation == relation &&
			tuple.SubjectType == subjectType &&
			tuple.SubjectID == subjectID &&
			tuple.SubjectRelation == subjectRelation {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRelationRepository) FindByEntityWithRelation(ctx context.Context, tenantID string, entityType, entityID, relation string, limit int) ([]*entities.RelationTuple, error) {
	return m.Read(ctx, tenantID, &repositories.RelationFilter{
		EntityType: entityType,
		EntityID:   entityID,
		Relation:   relation,
	})
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
	seen := make(map[string]bool)
	var ids []string
	for _, t := range m.tuples {
		if t.EntityType == entityType && !seen[t.EntityID] {
			seen[t.EntityID] = true
			ids = append(ids, t.EntityID)
		}
	}
	sort.Strings(ids)

	// Apply cursor
	if cursor != "" {
		filtered := ids[:0]
		for _, id := range ids {
			if id > cursor {
				filtered = append(filtered, id)
			}
		}
		ids = filtered
	}

	// Apply limit
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func (m *mockRelationRepository) GetSortedSubjectIDs(ctx context.Context, tenantID string, subjectType string, cursor string, limit int) ([]string, error) {
	seen := make(map[string]bool)
	var ids []string
	for _, t := range m.tuples {
		if t.SubjectType == subjectType && !seen[t.SubjectID] {
			seen[t.SubjectID] = true
			ids = append(ids, t.SubjectID)
		}
	}
	sort.Strings(ids)

	// Apply cursor
	if cursor != "" {
		filtered := ids[:0]
		for _, id := range ids {
			if id > cursor {
				filtered = append(filtered, id)
			}
		}
		ids = filtered
	}

	// Apply limit
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func (m *mockRelationRepository) LookupAccessibleEntitiesComplex(ctx context.Context, tenantID string, entityType string, relations []string, parentRelations []string, subjectType string, subjectID string, maxDepth int, cursor string, limit int) ([]string, error) {
	if m.lookupAccessibleEntitiesComplexFunc != nil {
		return m.lookupAccessibleEntitiesComplexFunc(ctx, tenantID, entityType, relations, parentRelations, subjectType, subjectID, maxDepth, cursor, limit)
	}
	return nil, nil
}

func (m *mockRelationRepository) LookupAccessibleSubjectsComplex(ctx context.Context, tenantID string, entityType string, entityID string, relations []string, parentRelations []string, subjectType string, maxDepth int, cursor string, limit int) ([]string, error) {
	if m.lookupAccessibleSubjectsComplexFunc != nil {
		return m.lookupAccessibleSubjectsComplexFunc(ctx, tenantID, entityType, entityID, relations, parentRelations, subjectType, maxDepth, cursor, limit)
	}
	return nil, nil
}

func (m *mockRelationRepository) BatchWriteInTx(ctx context.Context, tx *sql.Tx, tenantID string, tuples []*entities.RelationTuple) error {
	m.tuples = append(m.tuples, tuples...)
	return nil
}

type mockAttributeRepository struct {
	attributes map[string]map[string]interface{} // key: entityType:entityID, value: attributes
}

func newMockAttributeRepository() *mockAttributeRepository {
	return &mockAttributeRepository{
		attributes: make(map[string]map[string]interface{}),
	}
}

func (m *mockAttributeRepository) Write(ctx context.Context, tenantID string, attr *entities.Attribute) error {
	key := attr.EntityType + ":" + attr.EntityID
	if m.attributes[key] == nil {
		m.attributes[key] = make(map[string]interface{})
	}
	m.attributes[key][attr.Name] = attr.Value
	return nil
}

func (m *mockAttributeRepository) Read(ctx context.Context, tenantID string, entityType string, entityID string) (map[string]interface{}, error) {
	key := entityType + ":" + entityID
	if m.attributes[key] == nil {
		return make(map[string]interface{}), nil
	}
	return m.attributes[key], nil
}

func (m *mockAttributeRepository) Delete(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) error {
	key := entityType + ":" + entityID
	if m.attributes[key] != nil {
		delete(m.attributes[key], attrName)
	}
	return nil
}

func (m *mockAttributeRepository) GetValue(ctx context.Context, tenantID string, entityType string, entityID string, attrName string) (interface{}, error) {
	key := entityType + ":" + entityID
	if m.attributes[key] == nil {
		return nil, nil
	}
	return m.attributes[key][attrName], nil
}

func (m *mockAttributeRepository) WriteInTx(ctx context.Context, tx *sql.Tx, tenantID string, attr *entities.Attribute) error {
	return m.Write(ctx, tenantID, attr)
}

func (m *mockAttributeRepository) GetSortedEntityIDs(ctx context.Context, tenantID string, entityType string, cursor string, limit int) ([]string, error) {
	seen := make(map[string]bool)
	var ids []string
	for key := range m.attributes {
		parts := splitKey(key)
		if len(parts) == 2 && parts[0] == entityType {
			id := parts[1]
			if !seen[id] && (cursor == "" || id > cursor) {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}

// Test helper functions

func createTestSchema() *entities.Schema {
	return &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{
				Name: "user",
			},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "owner", TargetType: "user"},
					{Name: "editor", TargetType: "user"},
					{Name: "parent", TargetType: "folder"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "owner"},
					},
					{
						Name: "edit",
						Rule: &entities.LogicalRule{
							Operator: "or",
							Left:     &entities.RelationRule{Relation: "owner"},
							Right:    &entities.RelationRule{Relation: "editor"},
						},
					},
					{
						Name: "delete",
						Rule: &entities.RelationRule{Relation: "owner"},
					},
				},
			},
			{
				Name: "folder",
				Relations: []*entities.Relation{
					{Name: "owner", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "owner"},
					},
				},
			},
		},
	}
}

// Tests

func TestEvaluator_RelationRule(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	tests := []struct {
		name     string
		req      *EvaluationRequest
		rule     entities.PermissionRule
		expected bool
	}{
		{
			name: "relation exists - should return true",
			req: &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			rule:     &entities.RelationRule{Relation: "owner"},
			expected: true,
		},
		{
			name: "relation does not exist - should return false",
			req: &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   "bob",
			},
			rule:     &entities.RelationRule{Relation: "owner"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateRule(context.Background(), tt.req, tt.rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_LogicalRule_OR(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "bob",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.LogicalRule{
		Operator: "or",
		Left:     &entities.RelationRule{Relation: "owner"},
		Right:    &entities.RelationRule{Relation: "editor"},
	}

	tests := []struct {
		name      string
		subjectID string
		expected  bool
	}{
		{
			name:      "right side true - should return true",
			subjectID: "bob",
			expected:  true,
		},
		{
			name:      "both false - should return false",
			subjectID: "charlie",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_LogicalRule_AND(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.LogicalRule{
		Operator: "and",
		Left:     &entities.RelationRule{Relation: "owner"},
		Right:    &entities.RelationRule{Relation: "editor"},
	}

	tests := []struct {
		name      string
		subjectID string
		expected  bool
	}{
		{
			name:      "both true - should return true",
			subjectID: "alice",
			expected:  true,
		},
		{
			name:      "both false - should return false",
			subjectID: "bob",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_LogicalRule_NOT(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.LogicalRule{
		Operator: "not",
		Left:     &entities.RelationRule{Relation: "owner"},
	}

	tests := []struct {
		name      string
		subjectID string
		expected  bool
	}{
		{
			name:      "relation exists - should return false",
			subjectID: "alice",
			expected:  false,
		},
		{
			name:      "relation does not exist - should return true",
			subjectID: "bob",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_HierarchicalRule(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			// document doc1 has parent folder1
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "parent",
				SubjectType: "folder",
				SubjectID:   "folder1",
			},
			// alice is owner of folder1
			{
				EntityType:  "folder",
				EntityID:    "folder1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.HierarchicalRule{
		Relation:   "parent",
		Permission: "view",
	}

	tests := []struct {
		name      string
		subjectID string
		expected  bool
	}{
		{
			name:      "parent grants permission - should return true",
			subjectID: "alice",
			expected:  true,
		},
		{
			name:      "parent does not grant permission - should return false",
			subjectID: "bob",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_ABACRule(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	// Set up attributes
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document",
		EntityID:   "doc1",
		Name:       "public",
		Value:      true,
	})
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document",
		EntityID:   "doc2",
		Name:       "public",
		Value:      false,
	})

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.ABACRule{
		Expression: "resource.public == true",
	}

	tests := []struct {
		name     string
		entityID string
		expected bool
	}{
		{
			name:     "public document - should return true",
			entityID: "doc1",
			expected: true,
		},
		{
			name:     "private document - should return false",
			entityID: "doc2",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    tt.entityID,
				SubjectType: "user",
				SubjectID:   "alice",
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_ContextualTuples(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.RelationRule{Relation: "owner"}

	req := &EvaluationRequest{
		TenantID:    "test-tenant",
		EntityType:  "document",
		EntityID:    "doc1",
		SubjectType: "user",
		SubjectID:   "alice",
		ContextualTuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "owner",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}

	result, err := evaluator.EvaluateRule(context.Background(), req, rule)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true from contextual tuple, got false")
	}
}

func TestEvaluator_MaxDepthExceeded(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.RelationRule{Relation: "owner"}

	req := &EvaluationRequest{
		TenantID:    "test-tenant",
		EntityType:  "document",
		EntityID:    "doc1",
		SubjectType: "user",
		SubjectID:   "alice",
		Depth:       MaxDepth + 1,
	}

	_, err := evaluator.EvaluateRule(context.Background(), req, rule)
	if err == nil {
		t.Error("expected error for max depth exceeded, got nil")
	}
}

func TestEvaluator_ComplexRule(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "editor",
				SubjectType: "user",
				SubjectID:   "bob",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document",
		EntityID:   "doc1",
		Name:       "public",
		Value:      true,
	})
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	// (owner or editor) and rule(resource.public == true)
	rule := &entities.LogicalRule{
		Operator: "and",
		Left: &entities.LogicalRule{
			Operator: "or",
			Left:     &entities.RelationRule{Relation: "owner"},
			Right:    &entities.RelationRule{Relation: "editor"},
		},
		Right: &entities.ABACRule{
			Expression: "resource.public == true",
		},
	}

	tests := []struct {
		name      string
		subjectID string
		expected  bool
	}{
		{
			name:      "editor on public document - should return true",
			subjectID: "bob",
			expected:  true,
		},
		{
			name:      "neither owner nor editor - should return false",
			subjectID: "charlie",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, rule)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_RuleCallRule_StandardParams(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Rules: []*entities.RuleDefinition{
			{
				Name:       "is_public",
				Parameters: []string{"resource"},
				Body:       "resource.public == true",
			},
		},
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RuleCallRule{
							RuleName:  "is_public",
							Arguments: []string{"resource"},
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document", EntityID: "doc1", Name: "public", Value: true,
	})
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document", EntityID: "doc2", Name: "public", Value: false,
	})

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	tests := []struct {
		name     string
		entityID string
		expected bool
	}{
		{"public doc - allowed", "doc1", true},
		{"private doc - denied", "doc2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID: "test-tenant", EntityType: "document", EntityID: tt.entityID,
				SubjectType: "user", SubjectID: "alice",
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, schema.Entities[1].Permissions[0].Rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_RuleCallRule_NonStandardParamNames(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Rules: []*entities.RuleDefinition{
			{
				Name:       "check_public",
				Parameters: []string{"doc"},
				Body:       "doc.public == true",
			},
		},
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RuleCallRule{
							RuleName:  "check_public",
							Arguments: []string{"resource"},
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document", EntityID: "doc1", Name: "public", Value: true,
	})

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	req := &EvaluationRequest{
		TenantID: "test-tenant", EntityType: "document", EntityID: "doc1",
		SubjectType: "user", SubjectID: "alice",
	}
	result, err := evaluator.EvaluateRule(context.Background(), req, schema.Entities[1].Permissions[0].Rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true (doc1 is public), got false")
	}
}

func TestEvaluator_RuleCallRule_SwappedArguments(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Rules: []*entities.RuleDefinition{
			{
				Name:       "level_check",
				Parameters: []string{"resource", "subject"},
				Body:       "resource.level > subject.level",
			},
		},
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Permissions: []*entities.Permission{
					{
						Name: "special",
						Rule: &entities.RuleCallRule{
							RuleName:  "level_check",
							Arguments: []string{"subject", "resource"},
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document", EntityID: "doc1", Name: "level", Value: int64(2),
	})
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "user", EntityID: "alice", Name: "level", Value: int64(5),
	})

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	req := &EvaluationRequest{
		TenantID: "test-tenant", EntityType: "document", EntityID: "doc1",
		SubjectType: "user", SubjectID: "alice",
	}
	// Arguments: ["subject", "resource"]
	// Parameter "resource" <- argument "subject" (user alice, level=5)
	// Parameter "subject"  <- argument "resource" (document doc1, level=2)
	// Body: resource.level > subject.level => 5 > 2 => true
	result, err := evaluator.EvaluateRule(context.Background(), req, schema.Entities[1].Permissions[0].Rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true (5 > 2 with swapped arguments), got false")
	}
}

func TestEvaluator_RuleCallRule_TwoParamsNonStandard(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Rules: []*entities.RuleDefinition{
			{
				Name:       "dept_match",
				Parameters: []string{"doc", "usr"},
				Body:       "doc.department == usr.department",
			},
		},
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RuleCallRule{
							RuleName:  "dept_match",
							Arguments: []string{"resource", "subject"},
						},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document", EntityID: "doc1", Name: "department", Value: "engineering",
	})
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "user", EntityID: "alice", Name: "department", Value: "engineering",
	})
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "user", EntityID: "bob", Name: "department", Value: "marketing",
	})

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	tests := []struct {
		name      string
		subjectID string
		expected  bool
	}{
		{"same department - allowed", "alice", true},
		{"different department - denied", "bob", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID: "test-tenant", EntityType: "document", EntityID: "doc1",
				SubjectType: "user", SubjectID: tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, schema.Entities[1].Permissions[0].Rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_PermissionReferencesPermission(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "owner", TargetType: "user"},
					{Name: "editor", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "edit",
						Rule: &entities.LogicalRule{
							Operator: "or",
							Left:     &entities.RelationRule{Relation: "owner"},
							Right:    &entities.RelationRule{Relation: "editor"},
						},
					},
					{
						Name: "manage",
						Rule: &entities.RelationRule{Relation: "edit"},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{EntityType: "document", EntityID: "doc1", Relation: "editor", SubjectType: "user", SubjectID: "alice"},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	tests := []struct {
		name       string
		permission string
		subjectID  string
		expected   bool
	}{
		{"editor has edit", "edit", "alice", true},
		{"non-editor denied edit", "edit", "bob", false},
		{"editor has manage (via edit)", "manage", "alice", true},
		{"non-editor denied manage", "manage", "bob", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := schema.Entities[1].GetPermission(tt.permission)
			req := &EvaluationRequest{
				TenantID: "test-tenant", EntityType: "document", EntityID: "doc1",
				SubjectType: "user", SubjectID: tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, perm.Rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluator_PermissionChainThreeLevels(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "owner", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{Name: "view", Rule: &entities.RelationRule{Relation: "owner"}},
					{Name: "edit", Rule: &entities.RelationRule{Relation: "view"}},
					{Name: "admin", Rule: &entities.RelationRule{Relation: "edit"}},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{EntityType: "document", EntityID: "doc1", Relation: "owner", SubjectType: "user", SubjectID: "alice"},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	for _, perm := range []string{"view", "edit", "admin"} {
		t.Run(perm, func(t *testing.T) {
			p := schema.Entities[1].GetPermission(perm)
			req := &EvaluationRequest{
				TenantID: "test-tenant", EntityType: "document", EntityID: "doc1",
				SubjectType: "user", SubjectID: "alice",
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, p.Rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result {
				t.Errorf("expected ALLOWED for %s (via owner chain), got DENIED", perm)
			}
		})
	}
}

func TestEvaluator_EvaluateRelation_SubjectRelation(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "team"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "viewer", TargetType: "team"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "viewer"},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:      "document",
				EntityID:        "doc1",
				Relation:        "viewer",
				SubjectType:     "team",
				SubjectID:       "engineering",
				SubjectRelation: "member",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	req := &EvaluationRequest{
		TenantID:        "test-tenant",
		EntityType:      "document",
		EntityID:        "doc1",
		SubjectType:     "team",
		SubjectID:       "engineering",
		SubjectRelation: "member",
	}

	rule := &entities.RelationRule{Relation: "viewer"}

	result, err := evaluator.EvaluateRule(context.Background(), req, rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true (subject set with matching relation), got false")
	}
}

func TestEvaluator_EvaluateRelation_SubjectRelation_NoMatch(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "team"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "viewer", TargetType: "team"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "viewer"},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:      "document",
				EntityID:        "doc1",
				Relation:        "viewer",
				SubjectType:     "team",
				SubjectID:       "engineering",
				SubjectRelation: "member",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	req := &EvaluationRequest{
		TenantID:        "test-tenant",
		EntityType:      "document",
		EntityID:        "doc1",
		SubjectType:     "team",
		SubjectID:       "engineering",
		SubjectRelation: "admin",
	}

	rule := &entities.RelationRule{Relation: "viewer"}

	result, err := evaluator.EvaluateRule(context.Background(), req, rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false (subject relation 'admin' does not match 'member'), got true")
	}
}

// --- Bug fix regression tests ---

// TestBugfix_SubjectRelationPreservedInPermissionComposition verifies that
// SubjectRelation is propagated through permission composition chains.
// Bug: "permission manage = edit" dropped SubjectRelation when recursing into "edit".
func TestBugfix_SubjectRelationPreservedInPermissionComposition(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "team"},
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "viewer", TargetType: "user team#member"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "edit",
						Rule: &entities.RelationRule{Relation: "viewer"},
					},
					{
						Name: "manage",
						Rule: &entities.RelationRule{Relation: "edit"}, // composition: manage → edit → viewer
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:      "document",
				EntityID:        "doc1",
				Relation:        "viewer",
				SubjectType:     "team",
				SubjectID:       "eng",
				SubjectRelation: "member",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	// Check that team:eng#member has manage permission through the composition chain
	req := &EvaluationRequest{
		TenantID:        "test-tenant",
		EntityType:      "document",
		EntityID:        "doc1",
		SubjectType:     "team",
		SubjectID:       "eng",
		SubjectRelation: "member",
	}

	result, err := evaluator.EvaluateRule(context.Background(), req,
		schema.GetPermission("document", "manage").Rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true: SubjectRelation 'member' should be preserved through permission composition manage → edit → viewer")
	}
}

// TestBugfix_SubjectRelationPreservedInHierarchicalEvaluation verifies that
// SubjectRelation is propagated when evaluating hierarchical permissions on parent entities.
func TestBugfix_SubjectRelationPreservedInHierarchicalEvaluation(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "team"},
			{Name: "user"},
			{
				Name: "folder",
				Relations: []*entities.Relation{
					{Name: "viewer", TargetType: "user team#member"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "viewer"},
					},
				},
			},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "parent", TargetType: "folder"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.HierarchicalRule{Relation: "parent", Permission: "view"},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "parent",
				SubjectType: "folder",
				SubjectID:   "f1",
			},
			{
				EntityType:      "folder",
				EntityID:        "f1",
				Relation:        "viewer",
				SubjectType:     "team",
				SubjectID:       "eng",
				SubjectRelation: "member",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	req := &EvaluationRequest{
		TenantID:        "test-tenant",
		EntityType:      "document",
		EntityID:        "doc1",
		SubjectType:     "team",
		SubjectID:       "eng",
		SubjectRelation: "member",
	}

	result, err := evaluator.EvaluateRule(context.Background(), req,
		schema.GetPermission("document", "view").Rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true: SubjectRelation 'member' should be preserved through hierarchical permission parent.view")
	}
}

// TestBugfix_HierarchicalRelationExpandsComputedUsersets verifies that
// when a hierarchical rule targets a parent's RELATION (not permission),
// computed usersets (subject_relation) are properly expanded.
func TestBugfix_HierarchicalRelationExpandsComputedUsersets(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "group",
				Relations: []*entities.Relation{
					{Name: "member", TargetType: "user"},
				},
			},
			{
				Name: "folder",
				Relations: []*entities.Relation{
					{Name: "viewer", TargetType: "user group#member"},
				},
			},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "parent", TargetType: "folder"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						// Directly references parent's RELATION (not permission)
						Rule: &entities.HierarchicalRule{Relation: "parent", Permission: "viewer"},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{
				EntityType:  "document",
				EntityID:    "doc1",
				Relation:    "parent",
				SubjectType: "folder",
				SubjectID:   "f1",
			},
			{
				EntityType:      "folder",
				EntityID:        "f1",
				Relation:        "viewer",
				SubjectType:     "group",
				SubjectID:       "eng",
				SubjectRelation: "member",
			},
			{
				EntityType:  "group",
				EntityID:    "eng",
				Relation:    "member",
				SubjectType: "user",
				SubjectID:   "alice",
			},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	req := &EvaluationRequest{
		TenantID:    "test-tenant",
		EntityType:  "document",
		EntityID:    "doc1",
		SubjectType: "user",
		SubjectID:   "alice",
	}

	result, err := evaluator.EvaluateRule(context.Background(), req,
		schema.GetPermission("document", "view").Rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true: user:alice should have view through parent.viewer → group:eng#member expansion")
	}

	// user:bob should NOT have access
	reqBob := &EvaluationRequest{
		TenantID:    "test-tenant",
		EntityType:  "document",
		EntityID:    "doc1",
		SubjectType: "user",
		SubjectID:   "bob",
	}
	resultBob, err := evaluator.EvaluateRule(context.Background(), reqBob,
		schema.GetPermission("document", "view").Rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultBob {
		t.Error("expected false: user:bob is not a member of group:eng")
	}
}

// TestEvaluateRelation_CyclicComputedUserset tests that cyclic computed usersets
// return an error instead of causing infinite recursion / stack overflow.
func TestEvaluateRelation_CyclicComputedUserset(t *testing.T) {
	schema := &entities.Schema{
		Entities: []*entities.Entity{
			{
				Name: "team",
				Relations: []*entities.Relation{
					{Name: "member", TargetType: "team#member user"},
				},
			},
			{
				Name: "user",
			},
		},
	}

	mockSchemaService := &mockSchemaRepository{schema: schema}

	// Cyclic tuples: team:A#member@team:B#member and team:B#member@team:A#member
	mockRelRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{EntityType: "team", EntityID: "A", Relation: "member", SubjectType: "team", SubjectID: "B", SubjectRelation: "member"},
			{EntityType: "team", EntityID: "B", Relation: "member", SubjectType: "team", SubjectID: "A", SubjectRelation: "member"},
		},
	}
	mockAttrRepo := &mockAttributeRepository{}
	celEngine, _ := NewCELEngine()

	evaluator := NewEvaluator(mockSchemaService, mockRelRepo, mockAttrRepo, celEngine)

	req := &EvaluationRequest{
		TenantID:    "test-tenant",
		EntityType:  "team",
		EntityID:    "A",
		SubjectType: "user",
		SubjectID:   "alice",
	}

	_, err := evaluator.EvaluateRule(context.Background(), req,
		&entities.RelationRule{Relation: "member"})

	if err == nil {
		t.Fatal("expected error for cyclic computed userset, got nil")
	}
	t.Logf("correctly returned error: %v", err)
}

// TestEvaluateHierarchical_MultipleParentTypes verifies that hierarchical permission
// evaluation works when a relation targets multiple parent types (e.g., "relation parent @folder @organization").
// Permissions should resolve correctly through each parent type independently.
func TestEvaluateHierarchical_MultipleParentTypes(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "folder",
				Relations: []*entities.Relation{
					{Name: "viewer", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "viewer"},
					},
				},
			},
			{
				Name: "organization",
				Relations: []*entities.Relation{
					{Name: "member", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.RelationRule{Relation: "member"},
					},
				},
			},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "parent", TargetType: "folder organization"},
				},
				Permissions: []*entities.Permission{
					{
						Name: "view",
						Rule: &entities.HierarchicalRule{Relation: "parent", Permission: "view"},
					},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			// doc1 has folder parent
			{EntityType: "document", EntityID: "doc1", Relation: "parent", SubjectType: "folder", SubjectID: "f1"},
			{EntityType: "folder", EntityID: "f1", Relation: "viewer", SubjectType: "user", SubjectID: "alice"},
			// doc2 has organization parent
			{EntityType: "document", EntityID: "doc2", Relation: "parent", SubjectType: "organization", SubjectID: "org1"},
			{EntityType: "organization", EntityID: "org1", Relation: "member", SubjectType: "user", SubjectID: "bob"},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	tests := []struct {
		name      string
		entityID  string
		subjectID string
		expected  bool
	}{
		{"alice has view on doc1 via folder parent", "doc1", "alice", true},
		{"bob has no view on doc1 (not in folder)", "doc1", "bob", false},
		{"bob has view on doc2 via org parent", "doc2", "bob", true},
		{"alice has no view on doc2 (not in org)", "doc2", "alice", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    tt.entityID,
				SubjectType: "user",
				SubjectID:   tt.subjectID,
			}
			result, err := evaluator.EvaluateRule(context.Background(), req,
				schema.GetPermission("document", "view").Rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestEvaluateABAC_WithContextualAttributes verifies that contextual attributes
// override database attributes during ABAC evaluation.
func TestEvaluateABAC_WithContextualAttributes(t *testing.T) {
	schema := createTestSchema()
	relationRepo := &mockRelationRepository{}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()

	// Set up DB attributes: doc1 is NOT public
	attributeRepo.Write(context.Background(), "test-tenant", &entities.Attribute{
		EntityType: "document",
		EntityID:   "doc1",
		Name:       "public",
		Value:      false,
	})

	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	rule := &entities.ABACRule{
		Expression: "resource.public == true",
	}

	// Without contextual attributes: should be denied (public=false in DB)
	reqDenied := &EvaluationRequest{
		TenantID:    "test-tenant",
		EntityType:  "document",
		EntityID:    "doc1",
		SubjectType: "user",
		SubjectID:   "alice",
	}
	result, err := evaluator.EvaluateRule(context.Background(), reqDenied, rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false without contextual attributes (DB has public=false), got true")
	}

	// With contextual attributes overriding public=true: should be allowed
	reqAllowed := &EvaluationRequest{
		TenantID:    "test-tenant",
		EntityType:  "document",
		EntityID:    "doc1",
		SubjectType: "user",
		SubjectID:   "alice",
		ContextualAttributes: []*entities.Attribute{
			{
				EntityType: "document",
				EntityID:   "doc1",
				Name:       "public",
				Value:      true,
			},
		},
	}
	result, err = evaluator.EvaluateRule(context.Background(), reqAllowed, rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true with contextual attribute override (public=true), got false")
	}
}

// TestEvaluateRelation_PermissionCompositionChain verifies that a 4-level permission
// composition chain correctly propagates access: super_admin = admin, admin = manage,
// manage = edit, edit = viewer.
func TestEvaluateRelation_PermissionCompositionChain(t *testing.T) {
	schema := &entities.Schema{
		TenantID: "test-tenant",
		Entities: []*entities.Entity{
			{Name: "user"},
			{
				Name: "document",
				Relations: []*entities.Relation{
					{Name: "viewer", TargetType: "user"},
				},
				Permissions: []*entities.Permission{
					{Name: "edit", Rule: &entities.RelationRule{Relation: "viewer"}},
					{Name: "manage", Rule: &entities.RelationRule{Relation: "edit"}},
					{Name: "admin", Rule: &entities.RelationRule{Relation: "manage"}},
					{Name: "super_admin", Rule: &entities.RelationRule{Relation: "admin"}},
				},
			},
		},
	}

	relationRepo := &mockRelationRepository{
		tuples: []*entities.RelationTuple{
			{EntityType: "document", EntityID: "doc1", Relation: "viewer", SubjectType: "user", SubjectID: "alice"},
		},
	}
	attributeRepo := newMockAttributeRepository()
	celEngine, _ := NewCELEngine()
	evaluator := NewEvaluator(&mockSchemaRepository{schema}, relationRepo, attributeRepo, celEngine)

	for _, perm := range []string{"edit", "manage", "admin", "super_admin"} {
		t.Run(perm, func(t *testing.T) {
			p := schema.Entities[1].GetPermission(perm)
			if p == nil {
				t.Fatalf("permission %s not found", perm)
			}
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   "alice",
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, p.Rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result {
				t.Errorf("expected ALLOWED for %s (via 4-level chain from viewer), got DENIED", perm)
			}
		})
	}

	// Verify non-viewer is denied at all levels
	for _, perm := range []string{"edit", "manage", "admin", "super_admin"} {
		t.Run("denied_"+perm, func(t *testing.T) {
			p := schema.Entities[1].GetPermission(perm)
			req := &EvaluationRequest{
				TenantID:    "test-tenant",
				EntityType:  "document",
				EntityID:    "doc1",
				SubjectType: "user",
				SubjectID:   "bob",
			}
			result, err := evaluator.EvaluateRule(context.Background(), req, p.Rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result {
				t.Errorf("expected DENIED for %s (bob is not a viewer), got ALLOWED", perm)
			}
		})
	}
}
