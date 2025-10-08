package authorization

import (
	"strings"
	"testing"
)

func TestCELEngine_ComparisonOperators(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		context    *EvaluationContext
		expected   bool
		wantError  bool
	}{
		// Equality operator ==
		{
			name:       "equality - string equal",
			expression: `resource.status == "active"`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"status": "active"},
			},
			expected: true,
		},
		{
			name:       "equality - string not equal",
			expression: `resource.status == "inactive"`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"status": "active"},
			},
			expected: false,
		},
		{
			name:       "equality - int equal",
			expression: `resource.age == 25`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},
		{
			name:       "equality - bool equal",
			expression: `resource.public == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"public": true},
			},
			expected: true,
		},

		// Inequality operator !=
		{
			name:       "inequality - string not equal",
			expression: `resource.status != "inactive"`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"status": "active"},
			},
			expected: true,
		},
		{
			name:       "inequality - int not equal",
			expression: `resource.age != 30`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},

		// Greater than operator >
		{
			name:       "greater than - true",
			expression: `resource.age > 18`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},
		{
			name:       "greater than - false",
			expression: `resource.age > 30`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: false,
		},

		// Greater than or equal operator >=
		{
			name:       "greater than or equal - greater",
			expression: `resource.age >= 18`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},
		{
			name:       "greater than or equal - equal",
			expression: `resource.age >= 25`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},
		{
			name:       "greater than or equal - false",
			expression: `resource.age >= 30`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: false,
		},

		// Less than operator <
		{
			name:       "less than - true",
			expression: `resource.age < 30`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},
		{
			name:       "less than - false",
			expression: `resource.age < 18`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: false,
		},

		// Less than or equal operator <=
		{
			name:       "less than or equal - less",
			expression: `resource.age <= 30`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},
		{
			name:       "less than or equal - equal",
			expression: `resource.age <= 25`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: true,
		},
		{
			name:       "less than or equal - false",
			expression: `resource.age <= 18`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, tt.context)
			if (err != nil) != tt.wantError {
				t.Errorf("Evaluate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCELEngine_LogicalOperators(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		context    *EvaluationContext
		expected   bool
	}{
		// AND operator &&
		{
			name:       "AND - both true",
			expression: `resource.public == true && resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public": true,
					"active": true,
				},
			},
			expected: true,
		},
		{
			name:       "AND - first false",
			expression: `resource.public == true && resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public": false,
					"active": true,
				},
			},
			expected: false,
		},
		{
			name:       "AND - second false",
			expression: `resource.public == true && resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public": true,
					"active": false,
				},
			},
			expected: false,
		},

		// OR operator ||
		{
			name:       "OR - both true",
			expression: `resource.public == true || resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public": true,
					"active": true,
				},
			},
			expected: true,
		},
		{
			name:       "OR - first true",
			expression: `resource.public == true || resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public": true,
					"active": false,
				},
			},
			expected: true,
		},
		{
			name:       "OR - second true",
			expression: `resource.public == true || resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public": false,
					"active": true,
				},
			},
			expected: true,
		},
		{
			name:       "OR - both false",
			expression: `resource.public == true || resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public": false,
					"active": false,
				},
			},
			expected: false,
		},

		// NOT operator !
		{
			name:       "NOT - negate true",
			expression: `!(resource.public == true)`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"public": true},
			},
			expected: false,
		},
		{
			name:       "NOT - negate false",
			expression: `!(resource.public == true)`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"public": false},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, tt.context)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCELEngine_InOperator(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		context    *EvaluationContext
		expected   bool
	}{
		{
			name:       "in - string in list - true",
			expression: `"admin" in subject.roles`,
			context: &EvaluationContext{
				Subject: map[string]interface{}{
					"roles": []interface{}{"admin", "user"},
				},
			},
			expected: true,
		},
		{
			name:       "in - string in list - false",
			expression: `"superadmin" in subject.roles`,
			context: &EvaluationContext{
				Subject: map[string]interface{}{
					"roles": []interface{}{"admin", "user"},
				},
			},
			expected: false,
		},
		{
			name:       "in - int in list - true",
			expression: `5 in resource.allowed_ids`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"allowed_ids": []interface{}{1, 3, 5, 7},
				},
			},
			expected: true,
		},
		{
			name:       "in - int in list - false",
			expression: `6 in resource.allowed_ids`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"allowed_ids": []interface{}{1, 3, 5, 7},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, tt.context)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCELEngine_ComplexExpressions(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		context    *EvaluationContext
		expected   bool
	}{
		{
			name:       "complex - public OR owner",
			expression: `resource.public == true || resource.owner_id == subject.id`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public":   false,
					"owner_id": "user123",
				},
				Subject: map[string]interface{}{
					"id": "user123",
				},
			},
			expected: true,
		},
		{
			name:       "complex - admin role AND active",
			expression: `"admin" in subject.roles && resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"active": true,
				},
				Subject: map[string]interface{}{
					"roles": []interface{}{"admin", "user"},
				},
			},
			expected: true,
		},
		{
			name:       "complex - age range check",
			expression: `resource.age >= 18 && resource.age <= 65`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"age": 30,
				},
			},
			expected: true,
		},
		{
			name:       "complex - nested logical operators",
			expression: `(resource.public == true || resource.owner_id == subject.id) && resource.active == true`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"public":   true,
					"owner_id": "user456",
					"active":   true,
				},
				Subject: map[string]interface{}{
					"id": "user123",
				},
			},
			expected: true,
		},
		{
			name:       "complex - permission check with multiple conditions",
			expression: `resource.status == "published" && (resource.public == true || resource.owner_id == subject.id || "editor" in subject.roles)`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{
					"status":   "published",
					"public":   false,
					"owner_id": "user456",
				},
				Subject: map[string]interface{}{
					"id":    "user123",
					"roles": []interface{}{"editor"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, tt.context)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCELEngine_EmptyContext(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	// Test with nil context maps
	result, err := engine.Evaluate(`resource.public == true`, &EvaluationContext{})
	if err == nil {
		t.Error("expected error for accessing undefined field, got nil")
	}

	// Test with empty context but valid expression
	result, err = engine.Evaluate(`true == true`, &EvaluationContext{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}
}

func TestCELEngine_ErrorCases(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		context    *EvaluationContext
		wantError  bool
		errorMatch string
	}{
		{
			name:       "invalid syntax",
			expression: `resource.public ==`,
			context:    &EvaluationContext{},
			wantError:  true,
			errorMatch: "compile",
		},
		{
			name:       "non-boolean result",
			expression: `resource.age`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"age": 25},
			},
			wantError:  true,
			errorMatch: "not evaluate to boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.Evaluate(tt.expression, tt.context)
			if (err != nil) != tt.wantError {
				t.Errorf("Evaluate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && tt.errorMatch != "" {
				if !strings.Contains(err.Error(), tt.errorMatch) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errorMatch)
				}
			}
		})
	}
}

func TestCELEngine_ValidateExpression(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		wantError  bool
		errorMatch string
	}{
		{
			name:       "valid boolean expression",
			expression: `resource.public == true`,
			wantError:  false,
		},
		{
			name:       "valid complex expression",
			expression: `resource.public == true && subject.age >= 18`,
			wantError:  false,
		},
		{
			name:       "invalid - non-boolean result",
			expression: `resource.age`,
			wantError:  true,
			errorMatch: "must return boolean",
		},
		{
			name:       "invalid - syntax error",
			expression: `resource.public ==`,
			wantError:  true,
			errorMatch: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateExpression(tt.expression)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateExpression() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && tt.errorMatch != "" {
				if !strings.Contains(err.Error(), tt.errorMatch) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errorMatch)
				}
			}
		})
	}
}

func TestCELEngine_EvaluateWithDefaults(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	result, err := engine.EvaluateWithDefaults(
		`resource.public == true`,
		map[string]interface{}{"public": true},
		nil,
		nil,
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}
}

func TestCELEngine_StringOperations(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		context    *EvaluationContext
		expected   bool
	}{
		{
			name:       "contains - true",
			expression: `resource.title.contains("admin")`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"title": "admin user"},
			},
			expected: true,
		},
		{
			name:       "contains - false",
			expression: `resource.title.contains("superadmin")`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"title": "admin user"},
			},
			expected: false,
		},
		{
			name:       "startsWith - true",
			expression: `resource.email.startsWith("admin")`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"email": "admin@example.com"},
			},
			expected: true,
		},
		{
			name:       "endsWith - true",
			expression: `resource.email.endsWith(".com")`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"email": "admin@example.com"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, tt.context)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCELEngine_MultipleContextVariables(t *testing.T) {
	engine, err := NewCELEngine()
	if err != nil {
		t.Fatalf("failed to create CEL engine: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		context    *EvaluationContext
		expected   bool
	}{
		{
			name:       "resource and subject",
			expression: `resource.owner_id == subject.id`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"owner_id": "user123"},
				Subject:  map[string]interface{}{"id": "user123"},
			},
			expected: true,
		},
		{
			name:       "resource, subject, and request",
			expression: `resource.public == true && subject.role == "admin" && request.ip == "127.0.0.1"`,
			context: &EvaluationContext{
				Resource: map[string]interface{}{"public": true},
				Subject:  map[string]interface{}{"role": "admin"},
				Request:  map[string]interface{}{"ip": "127.0.0.1"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, tt.context)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
