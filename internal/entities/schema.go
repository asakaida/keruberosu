package entities

import "time"

// Schema represents the complete authorization schema for a tenant
type Schema struct {
	TenantID  string    // Tenant identifier
	DSL       string    // Original DSL text
	Entities  []*Entity // Entity definitions
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetEntity returns the entity definition by name
func (s *Schema) GetEntity(name string) *Entity {
	for _, e := range s.Entities {
		if e.Name == name {
			return e
		}
	}
	return nil
}

// GetPermission returns the permission definition for a given entity type and permission name
func (s *Schema) GetPermission(entityType, permissionName string) *Permission {
	entity := s.GetEntity(entityType)
	if entity == nil {
		return nil
	}
	return entity.GetPermission(permissionName)
}
