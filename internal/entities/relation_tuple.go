package entities

import (
	"fmt"
	"time"
)

// RelationTuple represents an actual relation data
// Example: document:1#owner@user:alice
// This means: user "alice" has "owner" relation with document "1"
type RelationTuple struct {
	EntityType      string // Entity type (e.g., "document")
	EntityID        string // Entity ID (e.g., "1")
	Relation        string // Relation name (e.g., "owner")
	SubjectType     string // Subject type (e.g., "user")
	SubjectID       string // Subject ID (e.g., "alice")
	SubjectRelation string // Subject relation (optional, e.g., "member" for group relations)
	CreatedAt       time.Time
}

// String returns a string representation of the relation tuple
// Format: entity_type:entity_id#relation@subject_type:subject_id[#subject_relation]
func (rt *RelationTuple) String() string {
	if rt.SubjectRelation != "" {
		return fmt.Sprintf("%s:%s#%s@%s:%s#%s",
			rt.EntityType, rt.EntityID, rt.Relation,
			rt.SubjectType, rt.SubjectID, rt.SubjectRelation)
	}
	return fmt.Sprintf("%s:%s#%s@%s:%s",
		rt.EntityType, rt.EntityID, rt.Relation,
		rt.SubjectType, rt.SubjectID)
}

// Validate checks if the relation tuple is valid
func (rt *RelationTuple) Validate() error {
	if rt.EntityType == "" {
		return fmt.Errorf("entity type is required")
	}
	if rt.EntityID == "" {
		return fmt.Errorf("entity ID is required")
	}
	if rt.Relation == "" {
		return fmt.Errorf("relation is required")
	}
	if rt.SubjectType == "" {
		return fmt.Errorf("subject type is required")
	}
	if rt.SubjectID == "" {
		return fmt.Errorf("subject ID is required")
	}
	return nil
}
