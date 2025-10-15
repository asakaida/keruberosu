package entities

// Entity represents an entity definition in the schema
// Example: "entity document { relation owner @user; permission edit = owner }"
type Entity struct {
	Name             string             // Entity name (e.g., "document", "user", "organization")
	Relations        []*Relation        // Relation definitions
	AttributeSchemas []*AttributeSchema // Attribute type definitions
	Permissions      []*Permission      // Permission definitions
}

// GetRelation returns the relation definition by name
func (e *Entity) GetRelation(name string) *Relation {
	for _, r := range e.Relations {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// GetPermission returns the permission definition by name
func (e *Entity) GetPermission(name string) *Permission {
	for _, p := range e.Permissions {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// GetAttributeSchema returns the attribute schema by name
func (e *Entity) GetAttributeSchema(name string) *AttributeSchema {
	for _, a := range e.AttributeSchemas {
		if a.Name == name {
			return a
		}
	}
	return nil
}
