package entities

import "testing"

func TestEntity_GetRelation(t *testing.T) {
	entity := &Entity{
		Name: "document",
		Relations: []*Relation{
			{Name: "owner", TargetType: "user"},
			{Name: "editor", TargetType: "user"},
			{Name: "viewer", TargetType: "user"},
		},
	}

	tests := []struct {
		name         string
		relationName string
		want         *Relation
	}{
		{
			name:         "existing relation - owner",
			relationName: "owner",
			want:         &Relation{Name: "owner", TargetType: "user"},
		},
		{
			name:         "existing relation - editor",
			relationName: "editor",
			want:         &Relation{Name: "editor", TargetType: "user"},
		},
		{
			name:         "non-existing relation",
			relationName: "nonexistent",
			want:         nil,
		},
		{
			name:         "empty name",
			relationName: "",
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entity.GetRelation(tt.relationName)
			if got == nil && tt.want == nil {
				return
			}
			if got == nil || tt.want == nil {
				t.Errorf("Entity.GetRelation() = %v, want %v", got, tt.want)
				return
			}
			if got.Name != tt.want.Name || got.TargetType != tt.want.TargetType {
				t.Errorf("Entity.GetRelation() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestEntity_GetPermission(t *testing.T) {
	entity := &Entity{
		Name: "document",
		Permissions: []*Permission{
			{Name: "edit", Rule: &RelationRule{Relation: "owner"}},
			{Name: "view", Rule: &RelationRule{Relation: "viewer"}},
		},
	}

	tests := []struct {
		name           string
		permissionName string
		want           *Permission
	}{
		{
			name:           "existing permission - edit",
			permissionName: "edit",
			want:           &Permission{Name: "edit", Rule: &RelationRule{Relation: "owner"}},
		},
		{
			name:           "existing permission - view",
			permissionName: "view",
			want:           &Permission{Name: "view", Rule: &RelationRule{Relation: "viewer"}},
		},
		{
			name:           "non-existing permission",
			permissionName: "delete",
			want:           nil,
		},
		{
			name:           "empty name",
			permissionName: "",
			want:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entity.GetPermission(tt.permissionName)
			if got == nil && tt.want == nil {
				return
			}
			if got == nil || tt.want == nil {
				t.Errorf("Entity.GetPermission() = %v, want %v", got, tt.want)
				return
			}
			if got.Name != tt.want.Name {
				t.Errorf("Entity.GetPermission() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestEntity_GetAttributeSchema(t *testing.T) {
	entity := &Entity{
		Name: "document",
		AttributeSchemas: []*AttributeSchema{
			{Name: "public", Type: "boolean"},
			{Name: "tags", Type: "string[]"},
			{Name: "version", Type: "int"},
		},
	}

	tests := []struct {
		name      string
		attrName  string
		want      *AttributeSchema
	}{
		{
			name:     "existing attribute - public",
			attrName: "public",
			want:     &AttributeSchema{Name: "public", Type: "boolean"},
		},
		{
			name:     "existing attribute - tags",
			attrName: "tags",
			want:     &AttributeSchema{Name: "tags", Type: "string[]"},
		},
		{
			name:     "existing attribute - version",
			attrName: "version",
			want:     &AttributeSchema{Name: "version", Type: "int"},
		},
		{
			name:     "non-existing attribute",
			attrName: "nonexistent",
			want:     nil,
		},
		{
			name:     "empty name",
			attrName: "",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entity.GetAttributeSchema(tt.attrName)
			if got == nil && tt.want == nil {
				return
			}
			if got == nil || tt.want == nil {
				t.Errorf("Entity.GetAttributeSchema() = %v, want %v", got, tt.want)
				return
			}
			if got.Name != tt.want.Name || got.Type != tt.want.Type {
				t.Errorf("Entity.GetAttributeSchema() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestEntity_EmptyLists(t *testing.T) {
	entity := &Entity{
		Name:             "empty",
		Relations:        []*Relation{},
		AttributeSchemas: []*AttributeSchema{},
		Permissions:      []*Permission{},
	}

	if got := entity.GetRelation("any"); got != nil {
		t.Errorf("Entity.GetRelation() on empty list = %v, want nil", got)
	}

	if got := entity.GetPermission("any"); got != nil {
		t.Errorf("Entity.GetPermission() on empty list = %v, want nil", got)
	}

	if got := entity.GetAttributeSchema("any"); got != nil {
		t.Errorf("Entity.GetAttributeSchema() on empty list = %v, want nil", got)
	}
}
