package interpolate

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Option keys for engine configuration.
const (
	OptionAllowUndefined = "allowUndefined"
	OptionKeepUndefined  = "keepUndefined"
)

// BuiltinFunc is a function that generates a dynamic value.
type BuiltinFunc func() string

// Engine handles variable interpolation.
type Engine struct {
	mu        sync.RWMutex
	variables map[string]string
	builtins  map[string]BuiltinFunc
	options   map[string]bool
}

// variablePattern matches {{variable}} or {{ variable }} syntax.
var variablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z_$][a-zA-Z0-9_\-$]*)\s*\}\}`)

// NewEngine creates a new interpolation engine.
func NewEngine() *Engine {
	e := &Engine{
		variables: make(map[string]string),
		builtins:  make(map[string]BuiltinFunc),
		options:   make(map[string]bool),
	}
	e.registerDefaultBuiltins()
	return e
}

func (e *Engine) registerDefaultBuiltins() {
	e.builtins["$uuid"] = func() string {
		return uuid.New().String()
	}

	e.builtins["$timestamp"] = func() string {
		return fmt.Sprintf("%d", time.Now().Unix())
	}

	e.builtins["$isoTimestamp"] = func() string {
		return time.Now().Format(time.RFC3339)
	}

	e.builtins["$randomInt"] = func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano()%10000)
	}

	e.builtins["$date"] = func() string {
		return time.Now().Format("2006-01-02")
	}

	e.builtins["$randomEmail"] = func() string {
		return fmt.Sprintf("user%d@example.com", time.Now().UnixNano()%10000)
	}

	e.builtins["$randomName"] = func() string {
		names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank"}
		return names[time.Now().UnixNano()%int64(len(names))]
	}
}

// SetVariable sets a variable value.
func (e *Engine) SetVariable(name, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.variables[name] = value
}

// GetVariable gets a variable value.
func (e *Engine) GetVariable(name string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.variables[name]
}

// HasVariable checks if a variable exists.
func (e *Engine) HasVariable(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.variables[name]
	return exists
}

// DeleteVariable removes a variable.
func (e *Engine) DeleteVariable(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.variables, name)
}

// SetVariables sets multiple variables at once.
func (e *Engine) SetVariables(vars map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for k, v := range vars {
		e.variables[k] = v
	}
}

// Variables returns a copy of all variables.
func (e *Engine) Variables() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]string, len(e.variables))
	for k, v := range e.variables {
		result[k] = v
	}
	return result
}

// Clear removes all variables.
func (e *Engine) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.variables = make(map[string]string)
}

// SetOption sets an engine option.
func (e *Engine) SetOption(key string, value bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.options[key] = value
}

// GetOption gets an engine option.
func (e *Engine) GetOption(key string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.options[key]
}

// RegisterBuiltin registers a custom builtin function.
func (e *Engine) RegisterBuiltin(name string, fn BuiltinFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.builtins[name] = fn
}

// Interpolate replaces all {{variable}} placeholders in the input string.
func (e *Engine) Interpolate(input string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var lastErr error
	result := variablePattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name
		submatch := variablePattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		varName := submatch[1]

		// Check builtins first
		if strings.HasPrefix(varName, "$") {
			if fn, ok := e.builtins[varName]; ok {
				return fn()
			}
		}

		// Check user variables
		if value, ok := e.variables[varName]; ok {
			return value
		}

		// Handle undefined variable
		if e.options[OptionKeepUndefined] {
			return match
		}
		if e.options[OptionAllowUndefined] {
			return ""
		}

		lastErr = fmt.Errorf("undefined variable: %s", varName)
		return match
	})

	if lastErr != nil {
		return "", lastErr
	}

	return result, nil
}

// InterpolateMap interpolates all values in a string map.
func (e *Engine) InterpolateMap(input map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(input))
	for k, v := range input {
		interpolated, err := e.Interpolate(v)
		if err != nil {
			return nil, fmt.Errorf("error interpolating key %q: %w", k, err)
		}
		result[k] = interpolated
	}
	return result, nil
}

// ExtractVariables returns all variable names found in the input string.
func (e *Engine) ExtractVariables(input string) []string {
	matches := variablePattern.FindAllStringSubmatch(input, -1)
	seen := make(map[string]bool)
	var result []string

	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			if !seen[varName] {
				seen[varName] = true
				result = append(result, varName)
			}
		}
	}

	return result
}

// Validate checks if all variables in the input string are defined.
func (e *Engine) Validate(input string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	vars := e.ExtractVariables(input)
	var missing []string

	for _, varName := range vars {
		// Builtins are always valid
		if strings.HasPrefix(varName, "$") {
			if _, ok := e.builtins[varName]; ok {
				continue
			}
		}

		// Check user variables
		if _, ok := e.variables[varName]; !ok {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("undefined variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// Clone creates a copy of the engine with the same variables.
func (e *Engine) Clone() *Engine {
	e.mu.RLock()
	defer e.mu.RUnlock()

	clone := NewEngine()
	for k, v := range e.variables {
		clone.variables[k] = v
	}
	for k, v := range e.options {
		clone.options[k] = v
	}

	return clone
}
