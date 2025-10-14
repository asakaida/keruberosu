package authorization

import (
	"context"
	"testing"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// Mock repositories for testing

type mockSchemaRepository struct {
	schema *entities.Schema
}

func (m *mockSchemaRepository) Create(ctx context.Context, tenantID string, schemaDSL string) error {
	return nil
}

func (m *mockSchemaRepository) GetByTenant(ctx context.Context, tenantID string) (*entities.Schema, error) {
	return m.schema, nil
}

func (m *mockSchemaRepository) Update(ctx context.Context, tenantID string, schemaDSL string) error {
	return nil
}

func (m *mockSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	return nil
}

// GetSchemaEntity implements SchemaServiceInterface
// In tests, the schema already has Entities populated, so we just return it
func (m *mockSchemaRepository) GetSchemaEntity(ctx context.Context, tenantID string) (*entities.Schema, error) {
	return m.schema, nil
}

type mockRelationRepository struct {
	tuples []*entities.RelationTuple
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
