package parser

import (
	"fmt"
	"strings"
)

// Validator validates the parsed schema AST
type Validator struct {
	schema  *SchemaAST
	errors  []string
	entities map[string]*EntityAST
}

// NewValidator creates a new Validator
func NewValidator(schema *SchemaAST) *Validator {
	entities := make(map[string]*EntityAST)
	for _, entity := range schema.Entities {
		entities[entity.Name] = entity
	}
	return &Validator{
		schema:   schema,
		errors:   []string{},
		entities: entities,
	}
}

// Validate validates the schema and returns error if invalid
func (v *Validator) Validate() error {
	v.validateUniqueEntityNames()
	v.validateEntityDefinitions()
	v.validateRelationTargets()
	v.validatePermissionReferences()
	v.validateCircularReferences()

	if len(v.errors) > 0 {
		return fmt.Errorf("validation errors:\n%s", strings.Join(v.errors, "\n"))
	}
	return nil
}

// validateUniqueEntityNames checks for duplicate entity names
func (v *Validator) validateUniqueEntityNames() {
	seen := make(map[string]bool)
	for _, entity := range v.schema.Entities {
		if seen[entity.Name] {
			v.errors = append(v.errors, fmt.Sprintf("duplicate entity name: %s", entity.Name))
		}
		seen[entity.Name] = true
	}
}

// validateEntityDefinitions validates each entity's internal structure
func (v *Validator) validateEntityDefinitions() {
	for _, entity := range v.schema.Entities {
		v.validateEntityUniqueness(entity)
		v.validateAttributeTypes(entity)
	}
}

// validateEntityUniqueness checks for duplicate names within an entity
func (v *Validator) validateEntityUniqueness(entity *EntityAST) {
	// Check duplicate relations
	relations := make(map[string]bool)
	for _, relation := range entity.Relations {
		if relations[relation.Name] {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: duplicate relation name: %s", entity.Name, relation.Name))
		}
		relations[relation.Name] = true
	}

	// Check duplicate attributes
	attributes := make(map[string]bool)
	for _, attribute := range entity.Attributes {
		if attributes[attribute.Name] {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: duplicate attribute name: %s", entity.Name, attribute.Name))
		}
		attributes[attribute.Name] = true
	}

	// Check duplicate permissions
	permissions := make(map[string]bool)
	for _, permission := range entity.Permissions {
		if permissions[permission.Name] {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: duplicate permission name: %s", entity.Name, permission.Name))
		}
		permissions[permission.Name] = true
	}

	// Check for name conflicts between relations, attributes, and permissions
	for attrName := range attributes {
		if relations[attrName] {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: name conflict between relation and attribute: %s", entity.Name, attrName))
		}
		if permissions[attrName] {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: name conflict between attribute and permission: %s", entity.Name, attrName))
		}
	}
	for relName := range relations {
		if permissions[relName] {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: name conflict between relation and permission: %s", entity.Name, relName))
		}
	}
}

// validateAttributeTypes validates attribute type declarations
func (v *Validator) validateAttributeTypes(entity *EntityAST) {
	validTypes := map[string]bool{
		"string":   true,
		"int":      true,
		"bool":     true,
		"float":    true,
		"string[]": true,
		"int[]":    true,
		"bool[]":   true,
		"float[]":  true,
	}

	for _, attribute := range entity.Attributes {
		if !validTypes[attribute.Type] {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: invalid attribute type: %s (attribute: %s)", entity.Name, attribute.Type, attribute.Name))
		}
	}
}

// validateRelationTargets checks that relation target types reference existing entities
func (v *Validator) validateRelationTargets() {
	for _, entity := range v.schema.Entities {
		for _, relation := range entity.Relations {
			if _, exists := v.entities[relation.TargetType]; !exists {
				v.errors = append(v.errors, fmt.Sprintf("entity %s: relation %s references undefined entity: %s", entity.Name, relation.Name, relation.TargetType))
			}
		}
	}
}

// validatePermissionReferences validates that permissions reference valid relations and entities
func (v *Validator) validatePermissionReferences() {
	for _, entity := range v.schema.Entities {
		for _, permission := range entity.Permissions {
			v.validatePermissionRule(entity, permission.Name, permission.Rule)
		}
	}
}

// validatePermissionRule recursively validates a permission rule
func (v *Validator) validatePermissionRule(entity *EntityAST, permissionName string, rule PermissionRuleAST) {
	switch r := rule.(type) {
	case *RelationPermissionAST:
		// Check if it's a relation
		foundRelation := false
		for _, relation := range entity.Relations {
			if relation.Name == r.Relation {
				foundRelation = true
				break
			}
		}

		// If not a relation, check if it's another permission in the same entity
		foundPermission := false
		if !foundRelation {
			for _, permission := range entity.Permissions {
				if permission.Name == r.Relation {
					foundPermission = true
					break
				}
			}
		}

		if !foundRelation && !foundPermission {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s references undefined relation or permission: %s", entity.Name, permissionName, r.Relation))
		}

	case *LogicalPermissionAST:
		if r.Left != nil {
			v.validatePermissionRule(entity, permissionName, r.Left)
		}
		if r.Right != nil {
			v.validatePermissionRule(entity, permissionName, r.Right)
		}

	case *HierarchicalPermissionAST:
		// Check if relation exists
		var targetEntity *EntityAST
		for _, relation := range entity.Relations {
			if relation.Name == r.Relation {
				targetEntity = v.entities[relation.TargetType]
				break
			}
		}
		if targetEntity == nil {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s references undefined relation: %s", entity.Name, permissionName, r.Relation))
			return
		}

		// Check if target entity has the referenced permission
		found := false
		for _, perm := range targetEntity.Permissions {
			if perm.Name == r.Permission {
				found = true
				break
			}
		}
		if !found {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s references undefined permission %s in entity %s", entity.Name, permissionName, r.Permission, targetEntity.Name))
		}

	case *RulePermissionAST:
		// Basic validation of rule expression (could be enhanced with CEL validation)
		if r.Expression == "" {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s has empty rule expression", entity.Name, permissionName))
		}
	}
}

// validateCircularReferences checks for circular hierarchical permission references
func (v *Validator) validateCircularReferences() {
	for _, entity := range v.schema.Entities {
		for _, permission := range entity.Permissions {
			visited := make(map[string]bool)
			currentPath := fmt.Sprintf("%s.%s", entity.Name, permission.Name)
			visited[currentPath] = true
			path := []string{currentPath}
			v.checkCircularPermission(entity, permission.Name, permission.Rule, visited, path)
		}
	}
}

// checkCircularPermission recursively checks for circular permission references
func (v *Validator) checkCircularPermission(entity *EntityAST, permissionName string, rule PermissionRuleAST, visited map[string]bool, path []string) {
	switch r := rule.(type) {
	case *RelationPermissionAST:
		// Check if this references another permission in the same entity
		for _, perm := range entity.Permissions {
			if perm.Name == r.Relation {
				// Found a permission reference - check for circular reference
				currentPath := fmt.Sprintf("%s.%s", entity.Name, perm.Name)

				if visited[currentPath] {
					cycle := append(path, currentPath)
					v.errors = append(v.errors, fmt.Sprintf("circular permission reference: %s", strings.Join(cycle, " -> ")))
					return
				}

				visited[currentPath] = true
				newPath := append(path, currentPath)
				v.checkCircularPermission(entity, perm.Name, perm.Rule, visited, newPath)
				delete(visited, currentPath)
				return
			}
		}
		// If it's a relation (not a permission), no circular reference possible

	case *LogicalPermissionAST:
		// Just recurse into the branches without adding to path
		if r.Left != nil {
			v.checkCircularPermission(entity, permissionName, r.Left, visited, path)
		}
		if r.Right != nil {
			v.checkCircularPermission(entity, permissionName, r.Right, visited, path)
		}

	case *HierarchicalPermissionAST:
		// Hierarchical permissions traverse to different instances via relations,
		// so they cannot create circular references at the schema level.
		// For example, folder.view = parent.view is valid because it references
		// a different folder instance via the parent relation.
		// Therefore, we don't check for circular references in hierarchical permissions.
		return
	}
}
