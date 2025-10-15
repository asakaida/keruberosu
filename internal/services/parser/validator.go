package parser

import (
	"fmt"
	"strings"
)

// Validator validates the parsed schema AST
type Validator struct {
	schema   *SchemaAST
	errors   []string
	entities map[string]*EntityAST
	rules    map[string]*RuleDefinitionAST
}

// NewValidator creates a new Validator
func NewValidator(schema *SchemaAST) *Validator {
	entities := make(map[string]*EntityAST)
	for _, entity := range schema.Entities {
		entities[entity.Name] = entity
	}
	rules := make(map[string]*RuleDefinitionAST)
	for _, rule := range schema.Rules {
		rules[rule.Name] = rule
	}
	return &Validator{
		schema:   schema,
		errors:   []string{},
		entities: entities,
		rules:    rules,
	}
}

// Validate validates the schema and returns error if invalid
func (v *Validator) Validate() error {
	v.validateUniqueRuleNames()
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

// validateUniqueRuleNames checks for duplicate rule names
func (v *Validator) validateUniqueRuleNames() {
	seen := make(map[string]bool)
	for _, rule := range v.schema.Rules {
		if seen[rule.Name] {
			v.errors = append(v.errors, fmt.Sprintf("duplicate rule name: %s", rule.Name))
		}
		seen[rule.Name] = true
	}
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
// Only Permify standard types are supported
func (v *Validator) validateAttributeTypes(entity *EntityAST) {
	validTypes := map[string]bool{
		"string":    true,
		"integer":   true,
		"boolean":   true,
		"double":    true,
		"string[]":  true,
		"integer[]": true,
		"boolean[]": true,
		"double[]":  true,
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
			// Split by space to handle multiple types (e.g., "user team#member")
			types := strings.Fields(relation.TargetType)
			for _, typeStr := range types {

				// Check if it's a subject relation (e.g., "team#member")
				if strings.Contains(typeStr, "#") {
					parts := strings.Split(typeStr, "#")
					if len(parts) != 2 {
						v.errors = append(v.errors, fmt.Sprintf("entity %s: relation %s has invalid subject relation format: %s", entity.Name, relation.Name, typeStr))
						continue
					}

					entityName := parts[0]
					relationName := parts[1]

					// Check if the referenced entity exists
					targetEntity, exists := v.entities[entityName]
					if !exists {
						v.errors = append(v.errors, fmt.Sprintf("entity %s: relation %s references undefined entity: %s (in %s)", entity.Name, relation.Name, entityName, typeStr))
						continue
					}

					// Check if the referenced relation exists in the target entity
					foundRelation := false
					for _, rel := range targetEntity.Relations {
						if rel.Name == relationName {
							foundRelation = true
							break
						}
					}
					if !foundRelation {
						v.errors = append(v.errors, fmt.Sprintf("entity %s: relation %s references undefined relation %s in entity %s", entity.Name, relation.Name, relationName, entityName))
					}
				} else {
					// Simple entity type reference
					if _, exists := v.entities[typeStr]; !exists {
						v.errors = append(v.errors, fmt.Sprintf("entity %s: relation %s references undefined entity: %s", entity.Name, relation.Name, typeStr))
					}
				}
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

		// Check if target entity has the referenced permission or relation
		// (Permify compatibility: allow both relation and permission references)
		foundPermission := false
		for _, perm := range targetEntity.Permissions {
			if perm.Name == r.Permission {
				foundPermission = true
				break
			}
		}

		foundRelation := false
		if !foundPermission {
			for _, rel := range targetEntity.Relations {
				if rel.Name == r.Permission {
					foundRelation = true
					break
				}
			}
		}

		if !foundPermission && !foundRelation {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s references undefined permission or relation %s in entity %s", entity.Name, permissionName, r.Permission, targetEntity.Name))
		}

	case *RuleCallPermissionAST:
		// Check if the rule exists
		ruleDef, exists := v.rules[r.RuleName]
		if !exists {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s calls undefined rule: %s", entity.Name, permissionName, r.RuleName))
			return
		}

		// Check if argument count matches parameter count
		if len(r.Arguments) != len(ruleDef.Parameters) {
			v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s calls rule %s with %d arguments, expected %d",
				entity.Name, permissionName, r.RuleName, len(r.Arguments), len(ruleDef.Parameters)))
		}

		// Validate that arguments are valid identifiers (resource, subject, etc.)
		validArguments := map[string]bool{
			"resource": true,
			"subject":  true,
			"request":  true,
		}
		for _, arg := range r.Arguments {
			if !validArguments[arg] {
				v.errors = append(v.errors, fmt.Sprintf("entity %s: permission %s calls rule %s with invalid argument: %s (must be 'resource', 'subject', or 'request')",
					entity.Name, permissionName, r.RuleName, arg))
			}
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
