package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PostmanEnvironment represents a Postman environment file format.
type PostmanEnvironment struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Values []PostmanEnvValue   `json:"values"`
}

// PostmanEnvValue represents a single variable in Postman format.
type PostmanEnvValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"` // "secret" or "default"
	Enabled *bool  `json:"enabled,omitempty"`
}

// SimpleEnvironment represents a simple key-value environment format.
type SimpleEnvironment struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables,omitempty"`
	Secrets   map[string]string `json:"secrets,omitempty"`
}

// LoadEnvironmentFromFile loads an environment from a file path.
func LoadEnvironmentFromFile(path string) (*Environment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %w", err)
	}

	return LoadEnvironmentFromJSON(data)
}

// LoadEnvironmentFromJSON loads an environment from JSON data.
// Supports both Postman format and simple key-value format.
func LoadEnvironmentFromJSON(data []byte) (*Environment, error) {
	// Try to detect format by checking for "values" array (Postman) or "variables" object (simple)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Check if it's Postman format (has "values" array)
	if _, hasValues := raw["values"]; hasValues {
		return loadPostmanEnvironment(data)
	}

	// Check if it's simple format (has "variables" or "secrets" object)
	if _, hasVariables := raw["variables"]; hasVariables {
		return loadSimpleEnvironment(data)
	}
	if _, hasSecrets := raw["secrets"]; hasSecrets {
		return loadSimpleEnvironment(data)
	}

	// Try to treat it as a flat key-value object
	return loadFlatEnvironment(data, raw)
}

// loadPostmanEnvironment loads a Postman-format environment.
func loadPostmanEnvironment(data []byte) (*Environment, error) {
	var pm PostmanEnvironment
	if err := json.Unmarshal(data, &pm); err != nil {
		return nil, fmt.Errorf("failed to parse Postman environment: %w", err)
	}

	name := pm.Name
	if name == "" {
		name = "Imported Environment"
	}

	env := NewEnvironment(name)

	for _, v := range pm.Values {
		// Skip disabled variables
		if v.Enabled != nil && !*v.Enabled {
			continue
		}

		if strings.ToLower(v.Type) == "secret" {
			env.SetSecret(v.Key, v.Value)
		} else {
			env.SetVariable(v.Key, v.Value)
		}
	}

	return env, nil
}

// loadSimpleEnvironment loads a simple key-value format environment.
func loadSimpleEnvironment(data []byte) (*Environment, error) {
	var simple SimpleEnvironment
	if err := json.Unmarshal(data, &simple); err != nil {
		return nil, fmt.Errorf("failed to parse simple environment: %w", err)
	}

	name := simple.Name
	if name == "" {
		name = "Environment"
	}

	env := NewEnvironment(name)

	for k, v := range simple.Variables {
		env.SetVariable(k, v)
	}

	for k, v := range simple.Secrets {
		env.SetSecret(k, v)
	}

	return env, nil
}

// loadFlatEnvironment loads a flat key-value JSON object as an environment.
func loadFlatEnvironment(data []byte, raw map[string]json.RawMessage) (*Environment, error) {
	env := NewEnvironment("Environment")

	// Extract name if present
	if nameRaw, hasName := raw["name"]; hasName {
		var name string
		if err := json.Unmarshal(nameRaw, &name); err == nil {
			env = NewEnvironment(name)
		}
	}

	// Load all string values as variables
	var flat map[string]interface{}
	if err := json.Unmarshal(data, &flat); err != nil {
		return nil, fmt.Errorf("failed to parse flat environment: %w", err)
	}

	for k, v := range flat {
		if k == "name" {
			continue // Skip name field
		}
		if str, ok := v.(string); ok {
			env.SetVariable(k, str)
		}
	}

	return env, nil
}

// LoadMultipleEnvironments loads and merges multiple environment files.
// Later files take precedence over earlier ones.
func LoadMultipleEnvironments(paths []string) (*Environment, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	var merged *Environment

	for _, path := range paths {
		env, err := LoadEnvironmentFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}

		if merged == nil {
			merged = env
		} else {
			merged.Merge(env)
			// Update name to reflect merge
			if env.Name() != "" && env.Name() != "Environment" {
				merged = NewEnvironmentWithID(merged.ID(), env.Name())
				merged.Merge(env)
			}
		}
	}

	return merged, nil
}

// MergeCollectionVariables merges collection variables into an environment.
// Collection variables have lower precedence (environment overrides them).
func MergeCollectionVariables(env *Environment, collections []*Collection) *Environment {
	// If no environment provided, create one with collection name
	if env == nil {
		name := "Collection Variables"
		if len(collections) == 1 {
			name = collections[0].Name() + " (vars)"
		}
		env = NewEnvironment(name)
	}

	// Create a new environment with collection variables first
	merged := NewEnvironment(env.Name())

	// Add collection variables (lower precedence)
	for _, coll := range collections {
		for k, v := range coll.Variables() {
			merged.SetVariable(k, v)
		}
	}

	// Merge environment variables on top (higher precedence)
	merged.Merge(env)

	return merged
}
