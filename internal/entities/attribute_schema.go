package entities

// AttributeSchema represents an attribute type definition in the schema
// Example: "attribute public: boolean" or "attribute tags: string[]"
type AttributeSchema struct {
	Name string // Attribute name (e.g., "public", "tags", "created_at")
	Type string // Attribute type (e.g., "string", "int", "bool", "string[]")
}
