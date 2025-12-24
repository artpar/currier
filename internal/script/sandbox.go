package script

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

// SandboxedEngine provides a secure JavaScript execution environment.
type SandboxedEngine struct {
	mu             sync.RWMutex
	runtime        *goja.Runtime
	consoleHandler func(level, msg string)
	iterationLimit int64
	memoryLimit    int64
	evalDisabled   bool
}

// NewSandboxedEngine creates a new sandboxed JavaScript engine.
func NewSandboxedEngine() *SandboxedEngine {
	e := &SandboxedEngine{
		runtime:        goja.New(),
		iterationLimit: 0, // 0 means no limit
		memoryLimit:    0, // 0 means no limit
	}
	e.setupSandbox()
	return e
}

// setupSandbox configures the security restrictions.
func (e *SandboxedEngine) setupSandbox() {
	// Remove dangerous globals by setting them to undefined
	dangerousGlobals := []string{
		"require",
		"process",
		"global",
		"__dirname",
		"__filename",
		"module",
		"exports",
		"Buffer",
	}

	for _, name := range dangerousGlobals {
		e.runtime.Set(name, goja.Undefined())
	}

	// Setup console
	e.setupConsole()
}

// setupConsole configures the console object.
func (e *SandboxedEngine) setupConsole() {
	console := make(map[string]interface{})

	console["log"] = func(args ...interface{}) {
		e.log("log", args...)
	}
	console["info"] = func(args ...interface{}) {
		e.log("info", args...)
	}
	console["warn"] = func(args ...interface{}) {
		e.log("warn", args...)
	}
	console["error"] = func(args ...interface{}) {
		e.log("error", args...)
	}
	console["debug"] = func(args ...interface{}) {
		e.log("debug", args...)
	}

	e.runtime.Set("console", console)
}

// log handles console output.
func (e *SandboxedEngine) log(level string, args ...interface{}) {
	e.mu.RLock()
	handler := e.consoleHandler
	e.mu.RUnlock()

	if handler == nil {
		return
	}

	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = fmt.Sprintf("%v", arg)
	}
	handler(level, strings.Join(parts, " "))
}

// SetConsoleHandler sets the handler for console output.
func (e *SandboxedEngine) SetConsoleHandler(handler func(level, msg string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.consoleHandler = handler
}

// SetIterationLimit sets the maximum number of loop iterations.
func (e *SandboxedEngine) SetIterationLimit(limit int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.iterationLimit = limit
}

// SetMemoryLimit sets the memory limit (note: not strictly enforced by Goja).
func (e *SandboxedEngine) SetMemoryLimit(limit int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.memoryLimit = limit
}

// DisableEval disables eval() and Function constructor.
func (e *SandboxedEngine) DisableEval() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evalDisabled = true
}

// Execute runs JavaScript code in the sandbox.
func (e *SandboxedEngine) Execute(ctx context.Context, script string) (interface{}, error) {
	e.mu.RLock()
	evalDisabled := e.evalDisabled
	e.mu.RUnlock()

	// Clear any previous interrupt
	e.runtime.ClearInterrupt()

	// Setup interrupt handler for context cancellation
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			e.runtime.Interrupt("context cancelled")
		case <-done:
		}
	}()

	// Note: True iteration limiting would require Goja hooks.
	// We rely on context timeout for infinite loop protection.
	finalScript := script

	// Wrap script to disable eval if needed
	if evalDisabled {
		wrapperScript := `
			(function() {
				var _eval = eval;
				eval = function() { throw new Error("eval is disabled"); };
				var _Function = Function;
				Function = function() { throw new Error("Function constructor is disabled"); };
				try {
					return (function() {
						` + finalScript + `
					})();
				} finally {
					eval = _eval;
					Function = _Function;
				}
			})()
		`
		finalScript = wrapperScript
	}

	// Compile and run
	program, err := goja.Compile("script", finalScript, false)
	if err != nil {
		return nil, fmt.Errorf("compile error: %w", err)
	}

	result, err := e.runtime.RunProgram(program)
	if err != nil {
		// Check if it was an interrupt
		var interrupt *goja.InterruptedError
		if errors.As(err, &interrupt) {
			return nil, fmt.Errorf("execution interrupted: %v", interrupt.Value())
		}
		// Check for iteration limit error
		if strings.Contains(err.Error(), "iteration limit exceeded") {
			return nil, fmt.Errorf("iteration limit exceeded")
		}
		return nil, err
	}

	return result.Export(), nil
}

// SetGlobal sets a global variable in the sandbox.
func (e *SandboxedEngine) SetGlobal(name string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.runtime.Set(name, value)
}

// RegisterFunction registers a Go function callable from JavaScript.
func (e *SandboxedEngine) RegisterFunction(name string, fn interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.runtime.Set(name, fn)
}

// RegisterObject registers a Go object as a JavaScript global.
func (e *SandboxedEngine) RegisterObject(name string, obj map[string]interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.runtime.Set(name, obj)
}

// SandboxedScope extends ScopeWithAssertions with sandbox protection.
type SandboxedScope struct {
	*ScopeWithAssertions
	sandboxEngine *SandboxedEngine
}

// NewSandboxedScope creates a new scope with sandbox protection.
func NewSandboxedScope() *SandboxedScope {
	// Create a sandboxed engine
	sandboxEngine := NewSandboxedEngine()

	// Create a properly initialized Engine that wraps the sandboxed runtime
	engine := &Engine{
		runtime:   sandboxEngine.runtime,
		globals:   make(map[string]interface{}),
		functions: make(map[string]interface{}),
	}

	// Create the scope with the underlying engine
	s := &SandboxedScope{
		ScopeWithAssertions: &ScopeWithAssertions{
			Scope: &Scope{
				engine:               engine,
				requestHeaders:       make(map[string]string),
				responseHeaders:      make(map[string]string),
				variables:            make(map[string]string),
				localVariables:       make(map[string]string),
				environmentVariables: make(map[string]string),
			},
			testResults: make([]TestResult, 0),
			initialized: false,
		},
		sandboxEngine: sandboxEngine,
	}

	// Setup the currier API
	s.setupCurrierAPI()

	return s
}

// Execute runs a script in the sandboxed scope.
func (s *SandboxedScope) Execute(ctx context.Context, script string) (interface{}, error) {
	// Refresh the currier object
	s.refreshCurrierObject()

	// Initialize assertions
	s.initAssertions()

	// Execute via the sandboxed engine
	return s.sandboxEngine.Execute(ctx, script)
}

// SetIterationLimit sets the iteration limit for the sandbox.
func (s *SandboxedScope) SetIterationLimit(limit int64) {
	s.sandboxEngine.SetIterationLimit(limit)
}

// DisableEval disables eval in the sandbox.
func (s *SandboxedScope) DisableEval() {
	s.sandboxEngine.DisableEval()
}

// SetSandboxConsoleHandler sets the console handler for the sandbox.
func (s *SandboxedScope) SetSandboxConsoleHandler(handler func(level, msg string)) {
	s.sandboxEngine.SetConsoleHandler(handler)
}
