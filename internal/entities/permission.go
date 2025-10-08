package entities

// Permission represents a permission definition in the schema
// Example: "permission edit = owner or editor"
type Permission struct {
	Name string         // Permission name (e.g., "edit", "view")
	Rule PermissionRule // The rule that defines this permission
}
