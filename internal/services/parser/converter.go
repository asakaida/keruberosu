package parser

import (
	"fmt"

	"github.com/asakaida/keruberosu/internal/entities"
)

// ASTToSchema converts SchemaAST to entities.Schema
func ASTToSchema(tenantID string, ast *SchemaAST) (*entities.Schema, error) {
	schema := &entities.Schema{
		TenantID: tenantID,
		Entities: make([]*entities.Entity, 0, len(ast.Entities)),
	}

	for _, entityAST := range ast.Entities {
		entity, err := convertEntity(entityAST)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity %s: %w", entityAST.Name, err)
		}
		schema.Entities = append(schema.Entities, entity)
	}

	return schema, nil
}

// SchemaToAST converts entities.Schema to SchemaAST
func SchemaToAST(schema *entities.Schema) (*SchemaAST, error) {
	ast := &SchemaAST{
		Entities: make([]*EntityAST, 0, len(schema.Entities)),
	}

	for _, entity := range schema.Entities {
		entityAST, err := convertEntityToAST(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity %s: %w", entity.Name, err)
		}
		ast.Entities = append(ast.Entities, entityAST)
	}

	return ast, nil
}

// convertEntity converts EntityAST to entities.Entity
func convertEntity(ast *EntityAST) (*entities.Entity, error) {
	entity := &entities.Entity{
		Name:             ast.Name,
		Relations:        make([]*entities.Relation, 0, len(ast.Relations)),
		AttributeSchemas: make([]*entities.AttributeSchema, 0, len(ast.Attributes)),
		Permissions:      make([]*entities.Permission, 0, len(ast.Permissions)),
	}

	// Convert relations
	for _, relAST := range ast.Relations {
		entity.Relations = append(entity.Relations, &entities.Relation{
			Name:       relAST.Name,
			TargetType: relAST.TargetType,
		})
	}

	// Convert attributes
	for _, attrAST := range ast.Attributes {
		entity.AttributeSchemas = append(entity.AttributeSchemas, &entities.AttributeSchema{
			Name: attrAST.Name,
			Type: attrAST.Type,
		})
	}

	// Convert permissions
	for _, permAST := range ast.Permissions {
		rule, err := convertPermissionRule(permAST.Rule)
		if err != nil {
			return nil, fmt.Errorf("failed to convert permission %s: %w", permAST.Name, err)
		}
		entity.Permissions = append(entity.Permissions, &entities.Permission{
			Name: permAST.Name,
			Rule: rule,
		})
	}

	return entity, nil
}

// convertPermissionRule converts PermissionRuleAST to entities.PermissionRule
func convertPermissionRule(ast PermissionRuleAST) (entities.PermissionRule, error) {
	switch r := ast.(type) {
	case *RelationPermissionAST:
		return &entities.RelationRule{
			Relation: r.Relation,
		}, nil

	case *LogicalPermissionAST:
		left, err := convertPermissionRule(r.Left)
		if err != nil {
			return nil, fmt.Errorf("failed to convert left side: %w", err)
		}

		var right entities.PermissionRule
		if r.Right != nil {
			var err error
			right, err = convertPermissionRule(r.Right)
			if err != nil {
				return nil, fmt.Errorf("failed to convert right side: %w", err)
			}
		}

		return &entities.LogicalRule{
			Operator: r.Operator,
			Left:     left,
			Right:    right,
		}, nil

	case *HierarchicalPermissionAST:
		return &entities.HierarchicalRule{
			Relation:   r.Relation,
			Permission: r.Permission,
		}, nil

	case *RulePermissionAST:
		return &entities.ABACRule{
			Expression: r.Expression,
		}, nil

	default:
		return nil, fmt.Errorf("unknown rule type: %T", ast)
	}
}

// convertEntityToAST converts entities.Entity to EntityAST
func convertEntityToAST(entity *entities.Entity) (*EntityAST, error) {
	ast := &EntityAST{
		Name:        entity.Name,
		Relations:   make([]*RelationAST, 0, len(entity.Relations)),
		Attributes:  make([]*AttributeAST, 0, len(entity.AttributeSchemas)),
		Permissions: make([]*PermissionAST, 0, len(entity.Permissions)),
	}

	// Convert relations
	for _, rel := range entity.Relations {
		ast.Relations = append(ast.Relations, &RelationAST{
			Name:       rel.Name,
			TargetType: rel.TargetType,
		})
	}

	// Convert attributes
	for _, attr := range entity.AttributeSchemas {
		ast.Attributes = append(ast.Attributes, &AttributeAST{
			Name: attr.Name,
			Type: attr.Type,
		})
	}

	// Convert permissions
	for _, perm := range entity.Permissions {
		rule, err := convertPermissionRuleToAST(perm.Rule)
		if err != nil {
			return nil, fmt.Errorf("failed to convert permission %s: %w", perm.Name, err)
		}
		ast.Permissions = append(ast.Permissions, &PermissionAST{
			Name: perm.Name,
			Rule: rule,
		})
	}

	return ast, nil
}

// convertPermissionRuleToAST converts entities.PermissionRule to PermissionRuleAST
func convertPermissionRuleToAST(rule entities.PermissionRule) (PermissionRuleAST, error) {
	switch r := rule.(type) {
	case *entities.RelationRule:
		return &RelationPermissionAST{
			Relation: r.Relation,
		}, nil

	case *entities.LogicalRule:
		left, err := convertPermissionRuleToAST(r.Left)
		if err != nil {
			return nil, fmt.Errorf("failed to convert left side: %w", err)
		}

		var right PermissionRuleAST
		if r.Right != nil {
			var err error
			right, err = convertPermissionRuleToAST(r.Right)
			if err != nil {
				return nil, fmt.Errorf("failed to convert right side: %w", err)
			}
		}

		return &LogicalPermissionAST{
			Operator: r.Operator,
			Left:     left,
			Right:    right,
		}, nil

	case *entities.HierarchicalRule:
		return &HierarchicalPermissionAST{
			Relation:   r.Relation,
			Permission: r.Permission,
		}, nil

	case *entities.ABACRule:
		return &RulePermissionAST{
			Expression: r.Expression,
		}, nil

	default:
		return nil, fmt.Errorf("unknown rule type: %T", rule)
	}
}
