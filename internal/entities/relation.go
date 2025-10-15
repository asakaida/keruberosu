package entities

// Relation represents a relation definition in the schema
// Example: "relation owner @user" or "relation parent @document"
type Relation struct {
	Name       string // Relation name (e.g., "owner", "editor", "parent")
	TargetType string // Target entity type (e.g., "user", "document")
}
