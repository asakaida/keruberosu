package entities

import "testing"

func TestSchema_GetEntity(t *testing.T) {
	schema := &Schema{
		TenantID: "tenant1",
		Entities: []*Entity{
			{Name: "document"},
			{Name: "user"},
			{Name: "organization"},
		},
	}

	tests := []struct {
		name       string
		entityName string
		want       *Entity
	}{
		{
			name:       "existing entity - document",
			entityName: "document",
			want:       &Entity{Name: "document"},
		},
		{
			name:       "existing entity - user",
			entityName: "user",
			want:       &Entity{Name: "user"},
		},
		{
			name:       "existing entity - organization",
			entityName: "organization",
			want:       &Entity{Name: "organization"},
		},
		{
			name:       "non-existing entity",
			entityName: "nonexistent",
			want:       nil,
		},
		{
			name:       "empty name",
			entityName: "",
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schema.GetEntity(tt.entityName)
			if got == nil && tt.want == nil {
				return
			}
			if got == nil || tt.want == nil {
				t.Errorf("Schema.GetEntity() = %v, want %v", got, tt.want)
				return
			}
			if got.Name != tt.want.Name {
				t.Errorf("Schema.GetEntity() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSchema_GetPermission(t *testing.T) {
	schema := &Schema{
		TenantID: "tenant1",
		Entities: []*Entity{
			{
				Name: "document",
				Permissions: []*Permission{
					{Name: "edit", Rule: &RelationRule{Relation: "owner"}},
					{Name: "view", Rule: &RelationRule{Relation: "viewer"}},
				},
			},
			{
				Name: "user",
				Permissions: []*Permission{
					{Name: "manage", Rule: &RelationRule{Relation: "admin"}},
				},
			},
		},
	}

	tests := []struct {
		name           string
		entityType     string
		permissionName string
		want           *Permission
	}{
		{
			name:           "existing permission - document.edit",
			entityType:     "document",
			permissionName: "edit",
			want:           &Permission{Name: "edit", Rule: &RelationRule{Relation: "owner"}},
		},
		{
			name:           "existing permission - document.view",
			entityType:     "document",
			permissionName: "view",
			want:           &Permission{Name: "view", Rule: &RelationRule{Relation: "viewer"}},
		},
		{
			name:           "existing permission - user.manage",
			entityType:     "user",
			permissionName: "manage",
			want:           &Permission{Name: "manage", Rule: &RelationRule{Relation: "admin"}},
		},
		{
			name:           "non-existing entity",
			entityType:     "nonexistent",
			permissionName: "edit",
			want:           nil,
		},
		{
			name:           "non-existing permission",
			entityType:     "document",
			permissionName: "delete",
			want:           nil,
		},
		{
			name:           "empty entity type",
			entityType:     "",
			permissionName: "edit",
			want:           nil,
		},
		{
			name:           "empty permission name",
			entityType:     "document",
			permissionName: "",
			want:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schema.GetPermission(tt.entityType, tt.permissionName)
			if got == nil && tt.want == nil {
				return
			}
			if got == nil || tt.want == nil {
				t.Errorf("Schema.GetPermission() = %v, want %v", got, tt.want)
				return
			}
			if got.Name != tt.want.Name {
				t.Errorf("Schema.GetPermission() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSchema_EmptyEntities(t *testing.T) {
	schema := &Schema{
		TenantID: "tenant1",
		Entities: []*Entity{},
	}

	if got := schema.GetEntity("any"); got != nil {
		t.Errorf("Schema.GetEntity() on empty list = %v, want nil", got)
	}

	if got := schema.GetPermission("any", "any"); got != nil {
		t.Errorf("Schema.GetPermission() on empty list = %v, want nil", got)
	}
}
