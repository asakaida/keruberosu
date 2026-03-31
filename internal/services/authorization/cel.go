package authorization

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const celEvalTimeout = 5 * time.Second

// CELEngine provides CEL expression evaluation for ABAC rules
type CELEngine struct {
	env *cel.Env

	// Compiled program cache
	programCache sync.Map // expression string -> *cel.Program
}

// EvaluationContext contains the context data for CEL evaluation
type EvaluationContext struct {
	Resource map[string]interface{} // Resource attributes (e.g., resource.public, resource.owner_id)
	Subject  map[string]interface{} // Subject attributes (e.g., subject.role, subject.id)
	Request  map[string]interface{} // Request context (e.g., request.ip, request.time)
}

// NewCELEngine creates a new CEL engine with predefined declarations
func NewCELEngine() (*CELEngine, error) {
	// Create CEL environment with standard library and common declarations
	env, err := cel.NewEnv(
		// Declare variables that will be available in CEL expressions
		cel.Variable("resource", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("subject", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("request", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &CELEngine{
		env: env,
	}, nil
}

// getOrCompileProgram returns a cached CEL program or compiles and caches a new one.
func (e *CELEngine) getOrCompileProgram(expression string) (cel.Program, error) {
	if cached, ok := e.programCache.Load(expression); ok {
		return cached.(cel.Program), nil
	}

	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	program, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	e.programCache.Store(expression, program)
	return program, nil
}

// Evaluate evaluates a CEL expression with the given context
func (e *CELEngine) Evaluate(expression string, evalCtx *EvaluationContext) (bool, error) {
	program, err := e.getOrCompileProgram(expression)
	if err != nil {
		return false, err
	}

	// Prepare the evaluation variables
	vars := make(map[string]interface{})
	if evalCtx.Resource != nil {
		vars["resource"] = evalCtx.Resource
	} else {
		vars["resource"] = map[string]interface{}{}
	}
	if evalCtx.Subject != nil {
		vars["subject"] = evalCtx.Subject
	} else {
		vars["subject"] = map[string]interface{}{}
	}
	if evalCtx.Request != nil {
		vars["request"] = evalCtx.Request
	} else {
		vars["request"] = map[string]interface{}{}
	}

	// Evaluate with timeout
	ctx, cancel := context.WithTimeout(context.Background(), celEvalTimeout)
	defer cancel()

	resultCh := make(chan ref.Val, 1)
	errCh := make(chan error, 1)
	go func() {
		result, _, err := program.Eval(vars)
		if err != nil {
			errCh <- err
		} else {
			resultCh <- result
		}
	}()

	select {
	case result := <-resultCh:
		boolResult, ok := result.Value().(bool)
		if !ok {
			return false, fmt.Errorf("CEL expression did not evaluate to boolean, got: %T", result.Value())
		}
		return boolResult, nil
	case err := <-errCh:
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	case <-ctx.Done():
		return false, fmt.Errorf("CEL expression evaluation timed out after %v", celEvalTimeout)
	}
}

// ValidateExpression validates a CEL expression without evaluating it
func (e *CELEngine) ValidateExpression(expression string) error {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("invalid CEL expression: %w", issues.Err())
	}

	// Check that the expression returns a boolean
	if ast.OutputType() != cel.BoolType {
		return fmt.Errorf("CEL expression must return boolean, got: %s", ast.OutputType())
	}

	return nil
}

// EvaluateWithDefaults evaluates a CEL expression with empty context maps if not provided
func (e *CELEngine) EvaluateWithDefaults(expression string, resource, subject, request map[string]interface{}) (bool, error) {
	context := &EvaluationContext{
		Resource: resource,
		Subject:  subject,
		Request:  request,
	}
	return e.Evaluate(expression, context)
}

// GetAvailableFunctions returns a list of available CEL functions
func (e *CELEngine) GetAvailableFunctions() []string {
	return []string{
		// Comparison operators
		"==", "!=", "<", "<=", ">", ">=",
		// Logical operators
		"&&", "||", "!",
		// Membership
		"in",
		// String operations
		"contains", "startsWith", "endsWith", "matches",
		// Collection operations
		"size", "all", "exists", "exists_one", "filter", "map",
		// Type conversions
		"int", "uint", "double", "bool", "string", "bytes", "timestamp", "duration",
		// Arithmetic
		"+", "-", "*", "/", "%",
	}
}

// EvaluateWithParams evaluates a CEL expression with "this" (parent attributes)
// and dynamically named parameters. Used for HierarchicalRuleCallRule evaluation.
func (e *CELEngine) EvaluateWithParams(expression string, thisAttrs map[string]interface{}, params map[string]interface{}) (bool, error) {
	if thisAttrs == nil {
		thisAttrs = map[string]interface{}{}
	}
	if params == nil {
		params = map[string]interface{}{}
	}

	// Build CEL environment options with "this" and each parameter
	envOpts := []cel.EnvOption{
		cel.Variable("this", cel.MapType(cel.StringType, cel.DynType)),
	}
	for paramName := range params {
		envOpts = append(envOpts, cel.Variable(paramName, cel.DynType))
	}

	env, err := cel.NewEnv(envOpts...)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL environment for params: %w", err)
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	program, err := env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL program: %w", err)
	}

	vars := map[string]interface{}{
		"this": thisAttrs,
	}
	for k, v := range params {
		vars[k] = v
	}

	result, _, err := program.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	boolResult, ok := result.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL expression did not evaluate to boolean, got: %T", result.Value())
	}

	return boolResult, nil
}

// ConvertGoValueToCEL converts a Go value to a CEL ref.Val
func ConvertGoValueToCEL(value interface{}) ref.Val {
	switch v := value.(type) {
	case bool:
		return types.Bool(v)
	case int:
		return types.Int(v)
	case int64:
		return types.Int(v)
	case float64:
		return types.Double(v)
	case string:
		return types.String(v)
	case []interface{}:
		vals := make([]ref.Val, len(v))
		for i, item := range v {
			vals[i] = ConvertGoValueToCEL(item)
		}
		return types.NewDynamicList(types.DefaultTypeAdapter, vals)
	case map[string]interface{}:
		return types.NewDynamicMap(types.DefaultTypeAdapter, v)
	default:
		return types.DefaultTypeAdapter.NativeToValue(value)
	}
}
