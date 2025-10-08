package entities

import "testing"

func TestRelationRule_isPermissionRule(t *testing.T) {
	// This test verifies that RelationRule implements PermissionRule interface
	var _ PermissionRule = &RelationRule{}

	rule := &RelationRule{Relation: "owner"}

	// Just call the method to ensure it compiles and runs
	rule.isPermissionRule()
}

func TestLogicalRule_isPermissionRule(t *testing.T) {
	// This test verifies that LogicalRule implements PermissionRule interface
	var _ PermissionRule = &LogicalRule{}

	rule := &LogicalRule{
		Operator: "or",
		Left:     &RelationRule{Relation: "owner"},
		Right:    &RelationRule{Relation: "editor"},
	}

	// Just call the method to ensure it compiles and runs
	rule.isPermissionRule()
}

func TestHierarchicalRule_isPermissionRule(t *testing.T) {
	// This test verifies that HierarchicalRule implements PermissionRule interface
	var _ PermissionRule = &HierarchicalRule{}

	rule := &HierarchicalRule{
		Relation:   "parent",
		Permission: "edit",
	}

	// Just call the method to ensure it compiles and runs
	rule.isPermissionRule()
}

func TestABACRule_isPermissionRule(t *testing.T) {
	// This test verifies that ABACRule implements PermissionRule interface
	var _ PermissionRule = &ABACRule{}

	rule := &ABACRule{
		Expression: "resource.public == true",
	}

	// Just call the method to ensure it compiles and runs
	rule.isPermissionRule()
}

func TestPermissionRuleInterface(t *testing.T) {
	// Test that all rule types can be used as PermissionRule
	tests := []struct {
		name string
		rule PermissionRule
	}{
		{
			name: "RelationRule",
			rule: &RelationRule{Relation: "owner"},
		},
		{
			name: "LogicalRule - OR",
			rule: &LogicalRule{
				Operator: "or",
				Left:     &RelationRule{Relation: "owner"},
				Right:    &RelationRule{Relation: "editor"},
			},
		},
		{
			name: "LogicalRule - AND",
			rule: &LogicalRule{
				Operator: "and",
				Left:     &RelationRule{Relation: "owner"},
				Right:    &RelationRule{Relation: "member"},
			},
		},
		{
			name: "LogicalRule - NOT",
			rule: &LogicalRule{
				Operator: "not",
				Left:     &RelationRule{Relation: "banned"},
				Right:    nil,
			},
		},
		{
			name: "HierarchicalRule",
			rule: &HierarchicalRule{
				Relation:   "parent",
				Permission: "edit",
			},
		},
		{
			name: "ABACRule",
			rule: &ABACRule{
				Expression: "resource.public == true || resource.owner == subject.id",
			},
		},
		{
			name: "Nested LogicalRule",
			rule: &LogicalRule{
				Operator: "or",
				Left: &LogicalRule{
					Operator: "and",
					Left:     &RelationRule{Relation: "owner"},
					Right:    &RelationRule{Relation: "active"},
				},
				Right: &RelationRule{Relation: "admin"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify that the rule implements the interface
			tt.rule.isPermissionRule()
		})
	}
}

func TestRelationRule_Fields(t *testing.T) {
	rule := &RelationRule{Relation: "owner"}
	if rule.Relation != "owner" {
		t.Errorf("RelationRule.Relation = %v, want owner", rule.Relation)
	}
}

func TestLogicalRule_Fields(t *testing.T) {
	left := &RelationRule{Relation: "owner"}
	right := &RelationRule{Relation: "editor"}
	rule := &LogicalRule{
		Operator: "or",
		Left:     left,
		Right:    right,
	}

	if rule.Operator != "or" {
		t.Errorf("LogicalRule.Operator = %v, want or", rule.Operator)
	}
	if rule.Left != left {
		t.Errorf("LogicalRule.Left = %v, want %v", rule.Left, left)
	}
	if rule.Right != right {
		t.Errorf("LogicalRule.Right = %v, want %v", rule.Right, right)
	}
}

func TestHierarchicalRule_Fields(t *testing.T) {
	rule := &HierarchicalRule{
		Relation:   "parent",
		Permission: "edit",
	}

	if rule.Relation != "parent" {
		t.Errorf("HierarchicalRule.Relation = %v, want parent", rule.Relation)
	}
	if rule.Permission != "edit" {
		t.Errorf("HierarchicalRule.Permission = %v, want edit", rule.Permission)
	}
}

func TestABACRule_Fields(t *testing.T) {
	expr := "resource.public == true"
	rule := &ABACRule{Expression: expr}

	if rule.Expression != expr {
		t.Errorf("ABACRule.Expression = %v, want %v", rule.Expression, expr)
	}
}
