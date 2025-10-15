package parser

// SchemaAST represents the parsed schema AST
type SchemaAST struct {
	Rules    []*RuleDefinitionAST // Top-level rule definitions (Permify compatible)
	Entities []*EntityAST
}

// RuleDefinitionAST represents a top-level rule definition
// Permify syntax: rule rule_name(param1, param2) { expression }
type RuleDefinitionAST struct {
	Name       string   // Rule name
	Parameters []string // Parameter names (e.g., ["resource", "subject"])
	Body       string   // CEL expression
}

// EntityAST represents an entity definition in the AST
type EntityAST struct {
	Name        string
	Relations   []*RelationAST
	Attributes  []*AttributeAST
	Permissions []*PermissionAST
}

// RelationAST represents a relation definition in the AST
type RelationAST struct {
	Name       string
	TargetType string
}

// AttributeAST represents an attribute definition in the AST
type AttributeAST struct {
	Name string
	Type string // "string", "int", "bool", "string[]", etc.
}

// PermissionAST represents a permission definition in the AST
type PermissionAST struct {
	Name string
	Rule PermissionRuleAST
}

// PermissionRuleAST is the interface for all permission rule types
type PermissionRuleAST interface {
	isPermissionRule()
}

// RelationPermissionAST represents a relation-based permission rule
// Example: "permission edit = owner"
type RelationPermissionAST struct {
	Relation string
}

func (r *RelationPermissionAST) isPermissionRule() {}

// LogicalPermissionAST represents a logical operation on permission rules
// Example: "permission edit = owner or editor"
type LogicalPermissionAST struct {
	Operator string             // "or", "and", "not"
	Left     PermissionRuleAST  // Left operand
	Right    PermissionRuleAST  // Right operand (nil for "not")
}

func (r *LogicalPermissionAST) isPermissionRule() {}

// HierarchicalPermissionAST represents a hierarchical permission check
// Example: "permission edit = parent.edit"
type HierarchicalPermissionAST struct {
	Relation   string // The relation to traverse
	Permission string // The permission to check on the related entity
}

func (r *HierarchicalPermissionAST) isPermissionRule() {}

// RuleCallPermissionAST represents a call to a top-level rule (Permify syntax)
// Example: "permission view = is_public(resource)"
type RuleCallPermissionAST struct {
	RuleName  string   // Name of the rule to call
	Arguments []string // Argument list (e.g., ["resource", "subject"])
}

func (r *RuleCallPermissionAST) isPermissionRule() {}
