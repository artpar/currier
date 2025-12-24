package interfaces

import (
	"context"
)

// ScriptEngine executes JavaScript scripts.
type ScriptEngine interface {
	// Execute runs a script with the given scope.
	Execute(ctx context.Context, script string, scope ScriptScope) (any, error)

	// RegisterFunction registers a Go function for use in scripts.
	RegisterFunction(name string, fn any) error

	// RegisterObject registers a Go object for use in scripts.
	RegisterObject(name string, obj any) error

	// Validate checks if a script is syntactically valid.
	Validate(script string) error
}

// ScriptScope provides context to scripts during execution.
type ScriptScope interface {
	// Request returns the current request (may be nil in some contexts).
	Request() Request

	// Response returns the current response (may be nil in pre-request scripts).
	Response() Response

	// Variables returns the variable store.
	Variables() VariableStore

	// Environment returns the current environment.
	Environment() Environment

	// Log writes a message to the script log.
	Log(msg string)

	// SetRequest allows modifying the request (for pre-request scripts).
	SetRequest(req Request)

	// AddTestResult adds a test result.
	AddTestResult(name string, passed bool, message string)

	// GetTestResults returns all test results.
	GetTestResults() []TestResult
}

// VariableStore manages variables across scopes.
type VariableStore interface {
	// Get retrieves a variable value, checking all scopes.
	Get(key string) (string, bool)

	// Set sets a variable in the current scope.
	Set(key, value string)

	// SetLocal sets a request-scoped variable.
	SetLocal(key, value string)

	// SetEnvironment sets an environment-scoped variable.
	SetEnvironment(key, value string)

	// SetCollection sets a collection-scoped variable.
	SetCollection(key, value string)

	// Delete removes a variable.
	Delete(key string)

	// All returns all variables with their scopes.
	All() map[string]VariableInfo

	// Interpolate replaces variables in a string.
	Interpolate(s string) (string, error)
}

// VariableInfo contains variable metadata.
type VariableInfo struct {
	Value string
	Scope VariableScope
}

// VariableScope indicates where a variable is defined.
type VariableScope int

const (
	VariableScopeLocal VariableScope = iota
	VariableScopeRequest
	VariableScopeCollection
	VariableScopeEnvironment
	VariableScopeGlobal
	VariableScopeBuiltin
)

func (s VariableScope) String() string {
	switch s {
	case VariableScopeLocal:
		return "local"
	case VariableScopeRequest:
		return "request"
	case VariableScopeCollection:
		return "collection"
	case VariableScopeEnvironment:
		return "environment"
	case VariableScopeGlobal:
		return "global"
	case VariableScopeBuiltin:
		return "builtin"
	default:
		return "unknown"
	}
}

// TestResult represents the result of a test assertion.
type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

// ScriptContext provides the currier.* API to scripts.
type ScriptContext struct {
	// Request provides access to request data and modification.
	Request ScriptRequestContext

	// Response provides access to response data.
	Response ScriptResponseContext

	// Environment provides access to environment variables.
	Environment ScriptEnvironmentContext

	// Variables provides access to all variables.
	Variables map[string]any
}

// ScriptRequestContext provides the currier.request.* API.
type ScriptRequestContext struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    any

	SetHeader func(key, value string)
	SetBody   func(body any)
	SetURL    func(url string)
}

// ScriptResponseContext provides the currier.response.* API.
type ScriptResponseContext struct {
	Status     int
	StatusText string
	Headers    map[string]string
	Body       string
	Time       int64 // milliseconds
	Size       int64 // bytes

	JSON func() (any, error)
}

// ScriptEnvironmentContext provides the currier.environment.* API.
type ScriptEnvironmentContext struct {
	Name string

	Get func(key string) string
	Set func(key, value string)
}
