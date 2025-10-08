package entities

// PermissionRule represents a rule for permission evaluation
type PermissionRule interface {
	isPermissionRule()
}

// RelationRule represents a relation-based permission rule
// Example: "owner" in "permission edit = owner"
type RelationRule struct {
	Relation string
}

func (r *RelationRule) isPermissionRule() {}

// LogicalRule represents a logical operation (OR/AND/NOT) on permission rules
// Example: "owner or editor" or "owner and not banned"
type LogicalRule struct {
	Operator string         // "or", "and", "not"
	Left     PermissionRule // Left operand
	Right    PermissionRule // Right operand (nil for "not" operator)
}

func (r *LogicalRule) isPermissionRule() {}

// HierarchicalRule represents a hierarchical permission check
// Example: "parent.edit" means check "edit" permission on the parent relation
type HierarchicalRule struct {
	Relation   string // The relation to traverse (e.g., "parent")
	Permission string // The permission to check on the related entity (e.g., "edit")
}

func (r *HierarchicalRule) isPermissionRule() {}

// ABACRule represents an attribute-based access control rule using CEL
// Example: "rule(resource.public == true || resource.owner == subject.id)"
type ABACRule struct {
	Expression string // CEL expression
}

func (r *ABACRule) isPermissionRule() {}
