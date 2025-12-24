package interpolate

import (
	"sync"
)

// Level represents the precedence level of variables.
type Level int

const (
	LevelNone Level = iota
	LevelGlobal
	LevelEnvironment
	LevelCollection
	LevelRequest
)

// String returns the string representation of a level.
func (l Level) String() string {
	switch l {
	case LevelGlobal:
		return "global"
	case LevelEnvironment:
		return "environment"
	case LevelCollection:
		return "collection"
	case LevelRequest:
		return "request"
	default:
		return "none"
	}
}

// VariableSet holds a set of variables at a single level.
type VariableSet struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewVariableSet creates a new empty variable set.
func NewVariableSet() *VariableSet {
	return &VariableSet{
		data: make(map[string]string),
	}
}

// Set sets a variable value.
func (vs *VariableSet) Set(key, value string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.data[key] = value
}

// Get gets a variable value.
func (vs *VariableSet) Get(key string) string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.data[key]
}

// Has checks if a variable exists.
func (vs *VariableSet) Has(key string) bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	_, exists := vs.data[key]
	return exists
}

// Delete removes a variable.
func (vs *VariableSet) Delete(key string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	delete(vs.data, key)
}

// All returns a copy of all variables.
func (vs *VariableSet) All() map[string]string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	result := make(map[string]string, len(vs.data))
	for k, v := range vs.data {
		result[k] = v
	}
	return result
}

// Len returns the number of variables.
func (vs *VariableSet) Len() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.data)
}

// Clear removes all variables.
func (vs *VariableSet) Clear() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.data = make(map[string]string)
}

// Clone creates a copy of the variable set.
func (vs *VariableSet) Clone() *VariableSet {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	clone := NewVariableSet()
	for k, v := range vs.data {
		clone.data[k] = v
	}
	return clone
}

// Merge merges another variable set into this one.
// Variables from the other set override existing variables.
func (vs *VariableSet) Merge(other *VariableSet) {
	if other == nil {
		return
	}
	other.mu.RLock()
	defer other.mu.RUnlock()
	vs.mu.Lock()
	defer vs.mu.Unlock()
	for k, v := range other.data {
		vs.data[k] = v
	}
}

// Scope manages variables across multiple precedence levels.
// Precedence order (highest to lowest): Request > Collection > Environment > Global
type Scope struct {
	mu          sync.RWMutex
	global      *VariableSet
	environment *VariableSet
	collection  *VariableSet
	request     *VariableSet
	engine      *Engine
}

// NewScope creates a new scope with an empty global level.
func NewScope() *Scope {
	return &Scope{
		global: NewVariableSet(),
		engine: NewEngine(),
	}
}

// Global returns the global variable set.
func (s *Scope) Global() *VariableSet {
	return s.global
}

// SetEnvironment sets the environment-level variables.
func (s *Scope) SetEnvironment(vs *VariableSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.environment = vs
}

// SetCollection sets the collection-level variables.
func (s *Scope) SetCollection(vs *VariableSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collection = vs
}

// SetRequest sets the request-level variables.
func (s *Scope) SetRequest(vs *VariableSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.request = vs
}

// Get gets a variable value using precedence order.
func (s *Scope) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check in precedence order (highest first)
	if s.request != nil && s.request.Has(key) {
		return s.request.Get(key)
	}
	if s.collection != nil && s.collection.Has(key) {
		return s.collection.Get(key)
	}
	if s.environment != nil && s.environment.Has(key) {
		return s.environment.Get(key)
	}
	if s.global != nil && s.global.Has(key) {
		return s.global.Get(key)
	}
	return ""
}

// Has checks if a variable exists at any level.
func (s *Scope) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.request != nil && s.request.Has(key) {
		return true
	}
	if s.collection != nil && s.collection.Has(key) {
		return true
	}
	if s.environment != nil && s.environment.Has(key) {
		return true
	}
	if s.global != nil && s.global.Has(key) {
		return true
	}
	return false
}

// Set sets a variable at the request level (or creates request level if needed).
func (s *Scope) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.request == nil {
		s.request = NewVariableSet()
	}
	s.request.Set(key, value)
}

// SetAt sets a variable at a specific level.
func (s *Scope) SetAt(level Level, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch level {
	case LevelGlobal:
		if s.global == nil {
			s.global = NewVariableSet()
		}
		s.global.Set(key, value)
	case LevelEnvironment:
		if s.environment == nil {
			s.environment = NewVariableSet()
		}
		s.environment.Set(key, value)
	case LevelCollection:
		if s.collection == nil {
			s.collection = NewVariableSet()
		}
		s.collection.Set(key, value)
	case LevelRequest:
		if s.request == nil {
			s.request = NewVariableSet()
		}
		s.request.Set(key, value)
	}
}

// GetSource returns the level where a variable is defined (highest precedence).
func (s *Scope) GetSource(key string) Level {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.request != nil && s.request.Has(key) {
		return LevelRequest
	}
	if s.collection != nil && s.collection.Has(key) {
		return LevelCollection
	}
	if s.environment != nil && s.environment.Has(key) {
		return LevelEnvironment
	}
	if s.global != nil && s.global.Has(key) {
		return LevelGlobal
	}
	return LevelNone
}

// All returns all variables merged with correct precedence.
func (s *Scope) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)

	// Add in order of lowest to highest precedence (so higher overwrites)
	if s.global != nil {
		for k, v := range s.global.All() {
			result[k] = v
		}
	}
	if s.environment != nil {
		for k, v := range s.environment.All() {
			result[k] = v
		}
	}
	if s.collection != nil {
		for k, v := range s.collection.All() {
			result[k] = v
		}
	}
	if s.request != nil {
		for k, v := range s.request.All() {
			result[k] = v
		}
	}

	return result
}

// Clear clears all levels.
func (s *Scope) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global != nil {
		s.global.Clear()
	}
	s.environment = nil
	s.collection = nil
	s.request = nil
}

// ClearLevel clears a specific level.
func (s *Scope) ClearLevel(level Level) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch level {
	case LevelGlobal:
		if s.global != nil {
			s.global.Clear()
		}
	case LevelEnvironment:
		s.environment = nil
	case LevelCollection:
		s.collection = nil
	case LevelRequest:
		s.request = nil
	}
}

// Interpolate interpolates a string using scoped variables.
func (s *Scope) Interpolate(input string) (string, error) {
	// Create engine with all scoped variables
	engine := NewEngine()
	engine.SetVariables(s.All())
	return engine.Interpolate(input)
}

// InterpolateMap interpolates all values in a string map.
func (s *Scope) InterpolateMap(input map[string]string) (map[string]string, error) {
	engine := NewEngine()
	engine.SetVariables(s.All())
	return engine.InterpolateMap(input)
}

// Clone creates a copy of the scope.
func (s *Scope) Clone() *Scope {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := NewScope()

	if s.global != nil {
		clone.global = s.global.Clone()
	}
	if s.environment != nil {
		clone.environment = s.environment.Clone()
	}
	if s.collection != nil {
		clone.collection = s.collection.Clone()
	}
	if s.request != nil {
		clone.request = s.request.Clone()
	}

	return clone
}
