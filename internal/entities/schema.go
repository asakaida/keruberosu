package entities

import "time"

// Schema represents the complete authorization schema for a tenant
type Schema struct {
	TenantID  string            // Tenant identifier
	Version   string            // Schema version (ULID)
	DSL       string            // Original DSL text
	Rules     []*RuleDefinition // Top-level rule definitions (Permify compatible)
	Entities  []*Entity         // Entity definitions
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SchemaVersion represents a lightweight schema version for listing
type SchemaVersion struct {
	Version   string    // Schema version (ULID)
	CreatedAt time.Time // When the version was created
}

// GetRule returns the rule definition by name
func (s *Schema) GetRule(name string) *RuleDefinition {
	for _, r := range s.Rules {
		if r.Name == name {
			return r
		}
	}
	return nil
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
