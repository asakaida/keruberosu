package parser

import (
	"fmt"
	"strings"
)

// Generator generates DSL from AST
type Generator struct {
	indent string
}

// NewGenerator creates a new Generator
func NewGenerator() *Generator {
	return &Generator{
		indent: "  ",
	}
}

// Generate generates DSL string from SchemaAST
func (g *Generator) Generate(schema *SchemaAST) string {
	var sb strings.Builder

	for i, entity := range schema.Entities {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(g.generateEntity(entity))
	}

	return sb.String()
}

// generateEntity generates DSL for an entity
func (g *Generator) generateEntity(entity *EntityAST) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("entity %s {\n", entity.Name))

	// Generate relations
	for _, relation := range entity.Relations {
		sb.WriteString(g.indent)
		sb.WriteString(g.generateRelation(relation))
		sb.WriteString("\n")
	}

	// Generate attributes
	for _, attr := range entity.Attributes {
		sb.WriteString(g.indent)
		sb.WriteString(g.generateAttribute(attr))
		sb.WriteString("\n")
	}

	// Generate permissions
	for _, perm := range entity.Permissions {
		sb.WriteString(g.indent)
		sb.WriteString(g.generatePermission(perm))
		sb.WriteString("\n")
	}

	sb.WriteString("}")

	return sb.String()
}

// generateRelation generates DSL for a relation
func (g *Generator) generateRelation(relation *RelationAST) string {
	// TargetType is stored as "user team#member" (space-separated, no @)
	// We need to output "@user @team#member" (each type prefixed with @)
	types := strings.Split(relation.TargetType, " ")
	var prefixedTypes []string
	for _, t := range types {
		if t != "" {
			prefixedTypes = append(prefixedTypes, "@"+t)
		}
	}
	return fmt.Sprintf("relation %s %s", relation.Name, strings.Join(prefixedTypes, " "))
}

// generateAttribute generates DSL for an attribute
func (g *Generator) generateAttribute(attr *AttributeAST) string {
	return fmt.Sprintf("attribute %s: %s", attr.Name, attr.Type)
}

// generatePermission generates DSL for a permission
func (g *Generator) generatePermission(perm *PermissionAST) string {
	return fmt.Sprintf("permission %s = %s", perm.Name, g.generatePermissionRule(perm.Rule))
}

// generatePermissionRule generates DSL for a permission rule
func (g *Generator) generatePermissionRule(rule PermissionRuleAST) string {
	switch r := rule.(type) {
	case *RelationPermissionAST:
		return r.Relation

	case *LogicalPermissionAST:
		left := g.generatePermissionRule(r.Left)

		switch r.Operator {
		case "or":
			right := g.generatePermissionRule(r.Right)
			return fmt.Sprintf("%s or %s", left, right)

		case "and":
			right := g.generatePermissionRule(r.Right)
			// Add parentheses if needed for precedence
			leftStr := left
			rightStr := right
			if l, ok := r.Left.(*LogicalPermissionAST); ok && l.Operator == "or" {
				leftStr = fmt.Sprintf("(%s)", left)
			}
			if ri, ok := r.Right.(*LogicalPermissionAST); ok && ri.Operator == "or" {
				rightStr = fmt.Sprintf("(%s)", right)
			}
			return fmt.Sprintf("%s and %s", leftStr, rightStr)

		case "not":
			// For NOT, only use left side
			// Add parentheses if left is a logical operation
			if _, ok := r.Left.(*LogicalPermissionAST); ok {
				return fmt.Sprintf("not (%s)", left)
			}
			return fmt.Sprintf("not %s", left)

		default:
			return fmt.Sprintf("%s %s %s", left, r.Operator, g.generatePermissionRule(r.Right))
		}

	case *HierarchicalPermissionAST:
		return fmt.Sprintf("%s.%s", r.Relation, r.Permission)

	case *RulePermissionAST:
		return fmt.Sprintf("rule(%s)", r.Expression)

	default:
		return ""
	}
}
