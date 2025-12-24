package script

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// ConsoleHandler is a function that handles console output from JavaScript.
type ConsoleHandler func(level, message string)

// Engine wraps the Goja JavaScript runtime for executing scripts.
type Engine struct {
	mu             sync.RWMutex
	runtime        *goja.Runtime
	globals        map[string]interface{}
	functions      map[string]interface{}
	consoleHandler ConsoleHandler
}

// NewEngine creates a new JavaScript execution engine.
func NewEngine() *Engine {
	e := &Engine{
		globals:   make(map[string]interface{}),
		functions: make(map[string]interface{}),
	}
	e.initRuntime()
	return e
}

// initRuntime initializes a new Goja runtime with default globals.
func (e *Engine) initRuntime() {
	e.runtime = goja.New()
	e.runtime.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// Setup console
	e.setupConsole()

	// Re-register all globals
	for name, value := range e.globals {
		e.runtime.Set(name, value)
	}

	// Re-register all functions
	for name, fn := range e.functions {
		e.runtime.Set(name, fn)
	}
}

// setupConsole sets up the console object with log, error, warn, info methods.
func (e *Engine) setupConsole() {
	console := e.runtime.NewObject()

	formatArgs := func(args []goja.Value) string {
		parts := make([]string, len(args))
		for i, arg := range args {
			parts[i] = fmt.Sprintf("%v", arg.Export())
		}
		return strings.Join(parts, " ")
	}

	console.Set("log", func(call goja.FunctionCall) goja.Value {
		msg := formatArgs(call.Arguments)
		if e.consoleHandler != nil {
			e.consoleHandler("log", msg)
		}
		return goja.Undefined()
	})

	console.Set("error", func(call goja.FunctionCall) goja.Value {
		msg := formatArgs(call.Arguments)
		if e.consoleHandler != nil {
			e.consoleHandler("error", msg)
		}
		return goja.Undefined()
	})

	console.Set("warn", func(call goja.FunctionCall) goja.Value {
		msg := formatArgs(call.Arguments)
		if e.consoleHandler != nil {
			e.consoleHandler("warn", msg)
		}
		return goja.Undefined()
	})

	console.Set("info", func(call goja.FunctionCall) goja.Value {
		msg := formatArgs(call.Arguments)
		if e.consoleHandler != nil {
			e.consoleHandler("info", msg)
		}
		return goja.Undefined()
	})

	e.runtime.Set("console", console)
}

// SetConsoleHandler sets the handler for console output.
func (e *Engine) SetConsoleHandler(handler ConsoleHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.consoleHandler = handler
}

// Execute runs a JavaScript script and returns the result.
func (e *Engine) Execute(ctx context.Context, script string) (interface{}, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Set up context cancellation handling
	if ctx.Done() != nil {
		// Check if already cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Set up interrupt for long-running scripts
		done := make(chan struct{})
		defer close(done)

		go func() {
			select {
			case <-ctx.Done():
				e.runtime.Interrupt("context cancelled")
			case <-done:
				// Script completed normally
			}
		}()
	}

	// Clear any previous interrupt
	e.runtime.ClearInterrupt()

	// Compile and run the script
	program, err := goja.Compile("script", script, true)
	if err != nil {
		return nil, fmt.Errorf("syntax error: %w", err)
	}

	value, err := e.runtime.RunProgram(program)
	if err != nil {
		// Check if it was an interrupt
		if exception, ok := err.(*goja.InterruptedError); ok {
			return nil, fmt.Errorf("execution interrupted: %v", exception.Value())
		}
		return nil, fmt.Errorf("runtime error: %w", err)
	}

	// Convert result
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}

	return value.Export(), nil
}

// SetGlobal sets a global variable accessible from JavaScript.
func (e *Engine) SetGlobal(name string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.globals[name] = value
	e.runtime.Set(name, value)
}

// GetGlobal gets a global variable value.
func (e *Engine) GetGlobal(name string) interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	value := e.runtime.Get(name)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	return value.Export()
}

// RegisterFunction registers a Go function to be callable from JavaScript.
func (e *Engine) RegisterFunction(name string, fn interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.functions[name] = fn
	e.runtime.Set(name, fn)
}

// RegisterObject registers an object with properties/methods accessible from JavaScript.
func (e *Engine) RegisterObject(name string, obj map[string]interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.globals[name] = obj
	e.runtime.Set(name, obj)
}

// Reset clears the engine state and creates a fresh runtime.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.globals = make(map[string]interface{})
	e.functions = make(map[string]interface{})
	e.initRuntime()
}

// Clone creates a copy of the engine with the same globals and functions.
func (e *Engine) Clone() *Engine {
	e.mu.RLock()
	defer e.mu.RUnlock()

	clone := &Engine{
		globals:        make(map[string]interface{}),
		functions:      make(map[string]interface{}),
		consoleHandler: e.consoleHandler,
	}

	// Copy globals
	for k, v := range e.globals {
		clone.globals[k] = v
	}

	// Copy functions
	for k, v := range e.functions {
		clone.functions[k] = v
	}

	clone.initRuntime()
	return clone
}

// ExecuteWithTimeout executes a script with a specific timeout.
func (e *Engine) ExecuteWithTimeout(script string, timeout time.Duration) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return e.Execute(ctx, script)
}
