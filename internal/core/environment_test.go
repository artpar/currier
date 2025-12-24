package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
