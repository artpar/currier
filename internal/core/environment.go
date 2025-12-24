package core

import (
	"time"

	"github.com/google/uuid"
)

// Environment represents a set of variables for a specific context (e.g., Production, Staging).
type Environment struct {
	id          string
	name        string
	description string
	variables   map[string]string
	secrets     map[string]string
	isActive    bool
	isGlobal    bool
	createdAt   time.Time
	updatedAt   time.Time
}

// NewEnvironment creates a new environment with the given name.
func NewEnvironment(name string) *Environment {
	now := time.Now()
	return &Environment{
		id:        uuid.New().String(),
		name:      name,
		variables: make(map[string]string),
		secrets:   make(map[string]string),
		createdAt: now,
		updatedAt: now,
	}
}

func (e *Environment) ID() string          { return e.id }
func (e *Environment) Name() string        { return e.name }
func (e *Environment) Description() string { return e.description }
func (e *Environment) CreatedAt() time.Time { return e.createdAt }
func (e *Environment) UpdatedAt() time.Time { return e.updatedAt }
func (e *Environment) IsActive() bool      { return e.isActive }
func (e *Environment) IsGlobal() bool      { return e.isGlobal }

func (e *Environment) SetDescription(desc string) {
	e.description = desc
	e.touch()
}

func (e *Environment) SetActive(active bool) {
	e.isActive = active
	e.touch()
}

func (e *Environment) SetGlobal(global bool) {
	e.isGlobal = global
	e.touch()
}

func (e *Environment) touch() {
	e.updatedAt = time.Now()
}

// Variables returns a copy of all variables.
func (e *Environment) Variables() map[string]string {
	result := make(map[string]string)
	for k, v := range e.variables {
		result[k] = v
	}
	return result
}

// GetVariable returns a variable value.
func (e *Environment) GetVariable(key string) string {
	return e.variables[key]
}

// SetVariable sets a variable value.
func (e *Environment) SetVariable(key, value string) {
	e.variables[key] = value
	e.touch()
}

// DeleteVariable removes a variable.
func (e *Environment) DeleteVariable(key string) {
	delete(e.variables, key)
	e.touch()
}

// GetSecret returns a secret value.
func (e *Environment) GetSecret(key string) string {
	return e.secrets[key]
}

// SetSecret sets a secret value.
func (e *Environment) SetSecret(key, value string) {
	e.secrets[key] = value
	e.touch()
}

// DeleteSecret removes a secret.
func (e *Environment) DeleteSecret(key string) {
	delete(e.secrets, key)
	e.touch()
}

// HasSecret checks if a secret exists.
func (e *Environment) HasSecret(key string) bool {
	_, exists := e.secrets[key]
	return exists
}

// SecretNames returns a list of secret names (not values).
func (e *Environment) SecretNames() []string {
	names := make([]string, 0, len(e.secrets))
	for k := range e.secrets {
		names = append(names, k)
	}
	return names
}

// Clone creates a deep copy of the environment.
func (e *Environment) Clone() *Environment {
	clone := NewEnvironment(e.name)
	clone.description = e.description
	clone.isActive = e.isActive
	clone.isGlobal = e.isGlobal

	for k, v := range e.variables {
		clone.variables[k] = v
	}

	for k, v := range e.secrets {
		clone.secrets[k] = v
	}

	return clone
}

// Merge merges variables and secrets from another environment.
// Values from the other environment take precedence.
func (e *Environment) Merge(other *Environment) {
	for k, v := range other.variables {
		e.variables[k] = v
	}

	for k, v := range other.secrets {
		e.secrets[k] = v
	}

	e.touch()
}

// ExportAll returns all variables and secrets combined for interpolation.
func (e *Environment) ExportAll() map[string]string {
	result := make(map[string]string)
	for k, v := range e.variables {
		result[k] = v
	}
	for k, v := range e.secrets {
		result[k] = v
	}
	return result
}

// ExportVariablesOnly returns only variables (no secrets).
func (e *Environment) ExportVariablesOnly() map[string]string {
	result := make(map[string]string)
	for k, v := range e.variables {
		result[k] = v
	}
	return result
}

// NewEnvironmentWithID creates an environment with a specific ID (for loading from storage).
func NewEnvironmentWithID(id, name string) *Environment {
	now := time.Now()
	return &Environment{
		id:        id,
		name:      name,
		variables: make(map[string]string),
		secrets:   make(map[string]string),
		createdAt: now,
		updatedAt: now,
	}
}

// SetTimestamps sets created and updated timestamps (for loading from storage).
func (e *Environment) SetTimestamps(created, updated time.Time) {
	e.createdAt = created
	e.updatedAt = updated
}
