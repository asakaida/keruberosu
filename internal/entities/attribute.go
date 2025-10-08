package entities

import (
	"encoding/json"
	"fmt"
	"time"
)

// Attribute represents an actual attribute data
// Example: document:1.public = true
// This means: document "1" has attribute "public" with value true
type Attribute struct {
	EntityType string      // Entity type (e.g., "document")
	EntityID   string      // Entity ID (e.g., "1")
	Name       string      // Attribute name (e.g., "public", "tags")
	Value      interface{} // Attribute value (can be string, int, bool, []string, etc.)
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// String returns a string representation of the attribute
// Format: entity_type:entity_id.name = value
func (a *Attribute) String() string {
	return fmt.Sprintf("%s:%s.%s = %v", a.EntityType, a.EntityID, a.Name, a.Value)
}

// Validate checks if the attribute is valid
func (a *Attribute) Validate() error {
	if a.EntityType == "" {
		return fmt.Errorf("entity type is required")
	}
	if a.EntityID == "" {
		return fmt.Errorf("entity ID is required")
	}
	if a.Name == "" {
		return fmt.Errorf("attribute name is required")
	}
	if a.Value == nil {
		return fmt.Errorf("attribute value is required")
	}
	return nil
}

// MarshalValue serializes the attribute value to JSON string for storage
func (a *Attribute) MarshalValue() (string, error) {
	data, err := json.Marshal(a.Value)
	if err != nil {
		return "", fmt.Errorf("failed to marshal attribute value: %w", err)
	}
	return string(data), nil
}

// UnmarshalValue deserializes the JSON string to attribute value
func (a *Attribute) UnmarshalValue(data string) error {
	if err := json.Unmarshal([]byte(data), &a.Value); err != nil {
		return fmt.Errorf("failed to unmarshal attribute value: %w", err)
	}
	return nil
}
