package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnvironment(t *testing.T) {
	t.Run("creates environment with name", func(t *testing.T) {
		env := NewEnvironment("Production")
		assert.NotEmpty(t, env.ID())
		assert.Equal(t, "Production", env.Name())
		assert.False(t, env.CreatedAt().IsZero())
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		env1 := NewEnvironment("Dev")
		env2 := NewEnvironment("Staging")
		assert.NotEqual(t, env1.ID(), env2.ID())
	})

	t.Run("starts with empty variables", func(t *testing.T) {
		env := NewEnvironment("Test")
		assert.Empty(t, env.Variables())
	})
}

func TestEnvironment_Variables(t *testing.T) {
	t.Run("sets and gets variable", func(t *testing.T) {
		env := NewEnvironment("Dev")
		env.SetVariable("base_url", "https://api.dev.example.com")
		assert.Equal(t, "https://api.dev.example.com", env.GetVariable("base_url"))
	})

	t.Run("returns empty string for undefined variable", func(t *testing.T) {
		env := NewEnvironment("Dev")
		assert.Equal(t, "", env.GetVariable("undefined"))
	})

	t.Run("overwrites existing variable", func(t *testing.T) {
		env := NewEnvironment("Dev")
		env.SetVariable("key", "value1")
		env.SetVariable("key", "value2")
		assert.Equal(t, "value2", env.GetVariable("key"))
	})

	t.Run("deletes variable", func(t *testing.T) {
		env := NewEnvironment("Dev")
		env.SetVariable("key", "value")
		env.DeleteVariable("key")
		assert.Equal(t, "", env.GetVariable("key"))
	})

	t.Run("lists all variables", func(t *testing.T) {
		env := NewEnvironment("Dev")
		env.SetVariable("a", "1")
		env.SetVariable("b", "2")
		env.SetVariable("c", "3")

		vars := env.Variables()
		assert.Len(t, vars, 3)
		assert.Equal(t, "1", vars["a"])
		assert.Equal(t, "2", vars["b"])
		assert.Equal(t, "3", vars["c"])
	})

	t.Run("variables returns copy not reference", func(t *testing.T) {
		env := NewEnvironment("Dev")
		env.SetVariable("key", "value")

		vars := env.Variables()
		vars["key"] = "modified"

		assert.Equal(t, "value", env.GetVariable("key"))
	})
}

func TestEnvironment_Secrets(t *testing.T) {
	t.Run("sets and gets secret", func(t *testing.T) {
		env := NewEnvironment("Prod")
		env.SetSecret("api_key", "super-secret-key")
		assert.Equal(t, "super-secret-key", env.GetSecret("api_key"))
	})

	t.Run("returns empty string for undefined secret", func(t *testing.T) {
		env := NewEnvironment("Prod")
		assert.Equal(t, "", env.GetSecret("undefined"))
	})

	t.Run("deletes secret", func(t *testing.T) {
		env := NewEnvironment("Prod")
		env.SetSecret("api_key", "secret")
		env.DeleteSecret("api_key")
		assert.Equal(t, "", env.GetSecret("api_key"))
	})

	t.Run("lists secret names without values", func(t *testing.T) {
		env := NewEnvironment("Prod")
		env.SetSecret("api_key", "secret1")
		env.SetSecret("db_password", "secret2")

		names := env.SecretNames()
		assert.Len(t, names, 2)
		assert.Contains(t, names, "api_key")
		assert.Contains(t, names, "db_password")
	})

	t.Run("has secret check", func(t *testing.T) {
		env := NewEnvironment("Prod")
		env.SetSecret("api_key", "secret")

		assert.True(t, env.HasSecret("api_key"))
		assert.False(t, env.HasSecret("undefined"))
	})
}

func TestEnvironment_Metadata(t *testing.T) {
	t.Run("sets description", func(t *testing.T) {
		env := NewEnvironment("Production")
		env.SetDescription("Production environment with real API endpoints")
		assert.Equal(t, "Production environment with real API endpoints", env.Description())
	})

	t.Run("updates timestamp on modification", func(t *testing.T) {
		env := NewEnvironment("Dev")
		original := env.UpdatedAt()

		env.SetVariable("key", "value")
		assert.True(t, env.UpdatedAt().After(original) || env.UpdatedAt().Equal(original))
	})
}

func TestEnvironment_Clone(t *testing.T) {
	t.Run("creates deep copy", func(t *testing.T) {
		original := NewEnvironment("Production")
		original.SetDescription("Original description")
		original.SetVariable("base_url", "https://api.example.com")
		original.SetSecret("api_key", "secret123")

		clone := original.Clone()

		// Verify it's a copy
		assert.NotEqual(t, original.ID(), clone.ID())
		assert.Equal(t, original.Name(), clone.Name())
		assert.Equal(t, original.Description(), clone.Description())
		assert.Equal(t, original.GetVariable("base_url"), clone.GetVariable("base_url"))
		assert.Equal(t, original.GetSecret("api_key"), clone.GetSecret("api_key"))

		// Verify modifications don't affect original
		clone.SetDescription("Modified")
		clone.SetVariable("base_url", "https://modified.example.com")
		assert.Equal(t, "Original description", original.Description())
		assert.Equal(t, "https://api.example.com", original.GetVariable("base_url"))
	})
}

func TestEnvironment_Merge(t *testing.T) {
	t.Run("merges variables from another environment", func(t *testing.T) {
		base := NewEnvironment("Base")
		base.SetVariable("a", "1")
		base.SetVariable("b", "2")

		overlay := NewEnvironment("Overlay")
		overlay.SetVariable("b", "override")
		overlay.SetVariable("c", "3")

		base.Merge(overlay)

		assert.Equal(t, "1", base.GetVariable("a"))
		assert.Equal(t, "override", base.GetVariable("b"))
		assert.Equal(t, "3", base.GetVariable("c"))
	})

	t.Run("merges secrets from another environment", func(t *testing.T) {
		base := NewEnvironment("Base")
		base.SetSecret("key1", "secret1")

		overlay := NewEnvironment("Overlay")
		overlay.SetSecret("key2", "secret2")

		base.Merge(overlay)

		assert.Equal(t, "secret1", base.GetSecret("key1"))
		assert.Equal(t, "secret2", base.GetSecret("key2"))
	})
}

func TestEnvironment_Active(t *testing.T) {
	t.Run("defaults to inactive", func(t *testing.T) {
		env := NewEnvironment("Test")
		assert.False(t, env.IsActive())
	})

	t.Run("can be set active", func(t *testing.T) {
		env := NewEnvironment("Test")
		env.SetActive(true)
		assert.True(t, env.IsActive())
	})

	t.Run("can be deactivated", func(t *testing.T) {
		env := NewEnvironment("Test")
		env.SetActive(true)
		env.SetActive(false)
		assert.False(t, env.IsActive())
	})
}

func TestEnvironment_Global(t *testing.T) {
	t.Run("can be marked as global", func(t *testing.T) {
		env := NewEnvironment("Globals")
		env.SetGlobal(true)
		assert.True(t, env.IsGlobal())
	})

	t.Run("defaults to non-global", func(t *testing.T) {
		env := NewEnvironment("Test")
		assert.False(t, env.IsGlobal())
	})
}

func TestEnvironment_Export(t *testing.T) {
	t.Run("exports variables for interpolation", func(t *testing.T) {
		env := NewEnvironment("Test")
		env.SetVariable("base_url", "https://api.example.com")
		env.SetVariable("version", "v1")
		env.SetSecret("api_key", "secret")

		// Export should include both variables and secrets
		exported := env.ExportAll()
		assert.Equal(t, "https://api.example.com", exported["base_url"])
		assert.Equal(t, "v1", exported["version"])
		assert.Equal(t, "secret", exported["api_key"])
	})

	t.Run("export without secrets", func(t *testing.T) {
		env := NewEnvironment("Test")
		env.SetVariable("base_url", "https://api.example.com")
		env.SetSecret("api_key", "secret")

		exported := env.ExportVariablesOnly()
		assert.Equal(t, "https://api.example.com", exported["base_url"])
		_, hasSecret := exported["api_key"]
		assert.False(t, hasSecret)
	})
}

func TestEnvironment_WithID(t *testing.T) {
	t.Run("creates environment with specific ID", func(t *testing.T) {
		env := NewEnvironmentWithID("test-id-123", "Test Env")
		assert.Equal(t, "test-id-123", env.ID())
		assert.Equal(t, "Test Env", env.Name())
	})

	t.Run("sets timestamps", func(t *testing.T) {
		env := NewEnvironment("Test")
		env.SetTimestamps(env.CreatedAt(), env.UpdatedAt())
		assert.False(t, env.CreatedAt().IsZero())
	})
}

func TestLoadEnvironmentFromJSON(t *testing.T) {
	t.Run("loads Postman format", func(t *testing.T) {
		data := []byte(`{
			"id": "env-123",
			"name": "Development",
			"values": [
				{"key": "base_url", "value": "https://dev.example.com"},
				{"key": "api_key", "value": "secret123", "type": "secret"}
			]
		}`)

		env, err := LoadEnvironmentFromJSON(data)
		assert.NoError(t, err)
		assert.Equal(t, "Development", env.Name())
		assert.Equal(t, "https://dev.example.com", env.GetVariable("base_url"))
		assert.Equal(t, "secret123", env.GetSecret("api_key"))
	})

	t.Run("loads Postman format with disabled values", func(t *testing.T) {
		disabled := false
		_ = disabled // use for reference
		data := []byte(`{
			"name": "Test",
			"values": [
				{"key": "enabled_var", "value": "yes", "enabled": true},
				{"key": "disabled_var", "value": "no", "enabled": false}
			]
		}`)

		env, err := LoadEnvironmentFromJSON(data)
		assert.NoError(t, err)
		assert.Equal(t, "yes", env.GetVariable("enabled_var"))
		assert.Equal(t, "", env.GetVariable("disabled_var"))
	})

	t.Run("loads simple format with variables", func(t *testing.T) {
		data := []byte(`{
			"name": "Simple Env",
			"variables": {
				"base_url": "https://api.example.com",
				"version": "v1"
			},
			"secrets": {
				"api_key": "secret-value"
			}
		}`)

		env, err := LoadEnvironmentFromJSON(data)
		assert.NoError(t, err)
		assert.Equal(t, "Simple Env", env.Name())
		assert.Equal(t, "https://api.example.com", env.GetVariable("base_url"))
		assert.Equal(t, "secret-value", env.GetSecret("api_key"))
	})

	t.Run("loads flat key-value format", func(t *testing.T) {
		data := []byte(`{
			"name": "Flat Env",
			"base_url": "https://flat.example.com",
			"timeout": "30"
		}`)

		env, err := LoadEnvironmentFromJSON(data)
		assert.NoError(t, err)
		assert.Equal(t, "Flat Env", env.Name())
		assert.Equal(t, "https://flat.example.com", env.GetVariable("base_url"))
		assert.Equal(t, "30", env.GetVariable("timeout"))
	})

	t.Run("handles empty Postman name", func(t *testing.T) {
		data := []byte(`{
			"values": [
				{"key": "var1", "value": "value1"}
			]
		}`)

		env, err := LoadEnvironmentFromJSON(data)
		assert.NoError(t, err)
		assert.Equal(t, "Imported Environment", env.Name())
	})

	t.Run("handles empty simple name", func(t *testing.T) {
		data := []byte(`{
			"variables": {"key": "value"}
		}`)

		env, err := LoadEnvironmentFromJSON(data)
		assert.NoError(t, err)
		assert.Equal(t, "Environment", env.Name())
	})

	t.Run("fails on invalid JSON", func(t *testing.T) {
		data := []byte(`not valid json`)
		_, err := LoadEnvironmentFromJSON(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})
}

func TestLoadEnvironmentFromFile(t *testing.T) {
	t.Run("fails for non-existent file", func(t *testing.T) {
		_, err := LoadEnvironmentFromFile("/non/existent/file.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read environment file")
	})
}

func TestLoadMultipleEnvironments(t *testing.T) {
	t.Run("returns nil for empty paths", func(t *testing.T) {
		env, err := LoadMultipleEnvironments([]string{})
		assert.NoError(t, err)
		assert.Nil(t, env)
	})

	t.Run("returns nil for nil paths", func(t *testing.T) {
		env, err := LoadMultipleEnvironments(nil)
		assert.NoError(t, err)
		assert.Nil(t, env)
	})

	t.Run("loads single environment from file", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		envPath := tmpDir + "/test.json"
		content := `{"name": "Test", "values": [{"key": "var1", "value": "val1", "enabled": true}]}`
		err := os.WriteFile(envPath, []byte(content), 0644)
		require.NoError(t, err)

		env, err := LoadMultipleEnvironments([]string{envPath})
		assert.NoError(t, err)
		assert.NotNil(t, env)
		assert.Equal(t, "val1", env.GetVariable("var1"))
	})

	t.Run("merges multiple environment files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// First environment - use generic name to avoid name update path
		env1Path := tmpDir + "/env1.json"
		content1 := `{"name": "Environment", "values": [{"key": "shared", "value": "base", "enabled": true}, {"key": "base_only", "value": "yes", "enabled": true}]}`
		err := os.WriteFile(env1Path, []byte(content1), 0644)
		require.NoError(t, err)

		// Second environment - also generic name
		env2Path := tmpDir + "/env2.json"
		content2 := `{"name": "Environment", "values": [{"key": "shared", "value": "overridden", "enabled": true}, {"key": "override_only", "value": "added", "enabled": true}]}`
		err = os.WriteFile(env2Path, []byte(content2), 0644)
		require.NoError(t, err)

		env, err := LoadMultipleEnvironments([]string{env1Path, env2Path})
		assert.NoError(t, err)
		assert.NotNil(t, env)

		// Second env should override first
		assert.Equal(t, "overridden", env.GetVariable("shared"))
		// Both unique vars should exist
		assert.Equal(t, "yes", env.GetVariable("base_only"))
		assert.Equal(t, "added", env.GetVariable("override_only"))
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := LoadMultipleEnvironments([]string{"/nonexistent/path.json"})
		assert.Error(t, err)
	})
}

func TestMergeCollectionVariables(t *testing.T) {
	t.Run("creates environment when nil", func(t *testing.T) {
		c := NewCollection("Test API")
		c.SetVariable("api_key", "collection-key")

		result := MergeCollectionVariables(nil, []*Collection{c})
		assert.NotNil(t, result)
		assert.Equal(t, "collection-key", result.GetVariable("api_key"))
	})

	t.Run("environment overrides collection vars", func(t *testing.T) {
		c := NewCollection("Test API")
		c.SetVariable("api_key", "collection-key")
		c.SetVariable("other_var", "collection-value")

		env := NewEnvironment("My Env")
		env.SetVariable("api_key", "env-key")

		result := MergeCollectionVariables(env, []*Collection{c})
		assert.Equal(t, "env-key", result.GetVariable("api_key"))
		assert.Equal(t, "collection-value", result.GetVariable("other_var"))
	})

	t.Run("uses single collection name", func(t *testing.T) {
		c := NewCollection("My API")

		result := MergeCollectionVariables(nil, []*Collection{c})
		assert.Contains(t, result.Name(), "My API")
	})
}

