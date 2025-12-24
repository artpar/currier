package integration

import (
	"strings"
	"testing"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interpolate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpolation_WithEnvironment(t *testing.T) {
	t.Run("interpolates environment variables in request URL", func(t *testing.T) {
		// Setup environment
		env := core.NewEnvironment("Production")
		env.SetVariable("base_url", "https://api.production.com")
		env.SetVariable("api_version", "v2")

		// Setup scope with environment
		scope := interpolate.NewScope()
		envVars := interpolate.NewVariableSet()
		for k, v := range env.Variables() {
			envVars.Set(k, v)
		}
		scope.SetEnvironment(envVars)

		// Interpolate request URL
		urlTemplate := "{{base_url}}/{{api_version}}/users"
		result, err := scope.Interpolate(urlTemplate)

		require.NoError(t, err)
		assert.Equal(t, "https://api.production.com/v2/users", result)
	})

	t.Run("environment variables override global", func(t *testing.T) {
		scope := interpolate.NewScope()
		scope.Global().Set("api_key", "global_key")

		env := interpolate.NewVariableSet()
		env.Set("api_key", "production_key")
		scope.SetEnvironment(env)

		result, err := scope.Interpolate("Key: {{api_key}}")

		require.NoError(t, err)
		assert.Equal(t, "Key: production_key", result)
	})
}

func TestInterpolation_WithCollection(t *testing.T) {
	t.Run("interpolates collection variables in request", func(t *testing.T) {
		// Setup collection
		coll := core.NewCollection("My API")
		coll.SetVariable("collection_id", "coll-123")
		coll.SetVariable("default_timeout", "30")

		// Setup scope with collection
		scope := interpolate.NewScope()
		collVars := interpolate.NewVariableSet()
		for k, v := range coll.Variables() {
			collVars.Set(k, v)
		}
		scope.SetCollection(collVars)

		// Interpolate
		template := "Collection: {{collection_id}}, Timeout: {{default_timeout}}s"
		result, err := scope.Interpolate(template)

		require.NoError(t, err)
		assert.Equal(t, "Collection: coll-123, Timeout: 30s", result)
	})

	t.Run("collection variables override environment", func(t *testing.T) {
		scope := interpolate.NewScope()

		env := interpolate.NewVariableSet()
		env.Set("api_host", "env.example.com")
		scope.SetEnvironment(env)

		coll := interpolate.NewVariableSet()
		coll.Set("api_host", "collection.example.com")
		scope.SetCollection(coll)

		result, err := scope.Interpolate("Host: {{api_host}}")

		require.NoError(t, err)
		assert.Equal(t, "Host: collection.example.com", result)
	})
}

func TestInterpolation_RequestHeaders(t *testing.T) {
	t.Run("interpolates variables in headers", func(t *testing.T) {
		scope := interpolate.NewScope()
		scope.Global().Set("token", "secret123")
		scope.Global().Set("content_type", "application/json")

		headers := map[string]string{
			"Authorization": "Bearer {{token}}",
			"Content-Type":  "{{content_type}}",
			"Accept":        "{{content_type}}",
		}

		result, err := scope.InterpolateMap(headers)

		require.NoError(t, err)
		assert.Equal(t, "Bearer secret123", result["Authorization"])
		assert.Equal(t, "application/json", result["Content-Type"])
		assert.Equal(t, "application/json", result["Accept"])
	})
}

func TestInterpolation_WithBuiltins(t *testing.T) {
	t.Run("interpolates $uuid in request body", func(t *testing.T) {
		scope := interpolate.NewScope()

		body := `{"requestId": "{{$uuid}}"}`
		result, err := scope.Interpolate(body)

		require.NoError(t, err)
		// Should contain a UUID
		assert.Contains(t, result, `"requestId": "`)
		assert.Regexp(t, `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, result)
	})

	t.Run("interpolates $timestamp in header", func(t *testing.T) {
		scope := interpolate.NewScope()

		header := "{{$timestamp}}"
		result, err := scope.Interpolate(header)

		require.NoError(t, err)
		assert.Regexp(t, `^\d+$`, result)
	})

	t.Run("interpolates mixed builtins and variables", func(t *testing.T) {
		scope := interpolate.NewScope()
		scope.Global().Set("user_id", "123")

		template := `{"userId": "{{user_id}}", "requestId": "{{$uuid}}", "timestamp": {{$timestamp}}}`
		result, err := scope.Interpolate(template)

		require.NoError(t, err)
		assert.Contains(t, result, `"userId": "123"`)
		assert.Contains(t, result, `"requestId": "`)
		assert.Contains(t, result, `"timestamp": `)
	})
}

func TestInterpolation_CompleteRequestFlow(t *testing.T) {
	t.Run("interpolates complete request with all levels", func(t *testing.T) {
		// Setup a complete variable hierarchy
		scope := interpolate.NewScope()

		// Global level
		scope.Global().Set("app_name", "Currier")

		// Environment level (simulating production)
		env := interpolate.NewVariableSet()
		env.Set("base_url", "https://api.production.com")
		env.Set("api_key", "prod-key-xxx")
		scope.SetEnvironment(env)

		// Collection level
		coll := interpolate.NewVariableSet()
		coll.Set("api_version", "v2")
		coll.Set("content_type", "application/json")
		scope.SetCollection(coll)

		// Request level
		req := interpolate.NewVariableSet()
		req.Set("user_id", "user-456")
		scope.SetRequest(req)

		// Interpolate URL
		url, err := scope.Interpolate("{{base_url}}/{{api_version}}/users/{{user_id}}")
		require.NoError(t, err)
		assert.Equal(t, "https://api.production.com/v2/users/user-456", url)

		// Interpolate headers
		headers := map[string]string{
			"Authorization": "Bearer {{api_key}}",
			"Content-Type":  "{{content_type}}",
			"X-App-Name":    "{{app_name}}",
		}
		headersResult, err := scope.InterpolateMap(headers)
		require.NoError(t, err)
		assert.Equal(t, "Bearer prod-key-xxx", headersResult["Authorization"])
		assert.Equal(t, "application/json", headersResult["Content-Type"])
		assert.Equal(t, "Currier", headersResult["X-App-Name"])
	})
}

func TestInterpolation_ErrorHandling(t *testing.T) {
	t.Run("returns error for undefined variable", func(t *testing.T) {
		scope := interpolate.NewScope()

		_, err := scope.Interpolate("Hello {{undefined}}")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("validates template before execution", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("defined", "value")

		err := engine.Validate("{{defined}} and {{undefined}}")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})
}

func TestInterpolation_SwitchingEnvironments(t *testing.T) {
	t.Run("switching environments changes interpolated values", func(t *testing.T) {
		scope := interpolate.NewScope()
		scope.Global().Set("app_name", "Test App")

		// Start with development environment
		dev := interpolate.NewVariableSet()
		dev.Set("api_url", "http://localhost:8080")
		dev.Set("api_key", "dev-key")
		scope.SetEnvironment(dev)

		devURL, _ := scope.Interpolate("{{api_url}}")
		assert.Equal(t, "http://localhost:8080", devURL)

		// Switch to production environment
		prod := interpolate.NewVariableSet()
		prod.Set("api_url", "https://api.production.com")
		prod.Set("api_key", "prod-key")
		scope.SetEnvironment(prod)

		prodURL, _ := scope.Interpolate("{{api_url}}")
		assert.Equal(t, "https://api.production.com", prodURL)

		// Global variables remain unchanged
		appName, _ := scope.Interpolate("{{app_name}}")
		assert.Equal(t, "Test App", appName)
	})
}

func TestInterpolation_SpecialCharacters(t *testing.T) {
	t.Run("handles special characters in values", func(t *testing.T) {
		scope := interpolate.NewScope()
		scope.Global().Set("query", "name=John&age=30")
		scope.Global().Set("json", `{"key": "value"}`)

		result1, _ := scope.Interpolate("?{{query}}")
		assert.Equal(t, "?name=John&age=30", result1)

		result2, _ := scope.Interpolate("Data: {{json}}")
		assert.Equal(t, `Data: {"key": "value"}`, result2)
	})

	t.Run("handles multiline values", func(t *testing.T) {
		scope := interpolate.NewScope()
		scope.Global().Set("body", "line1\nline2\nline3")

		result, _ := scope.Interpolate("Body:\n{{body}}")
		assert.Equal(t, "Body:\nline1\nline2\nline3", result)
	})
}

func TestInterpolation_Performance(t *testing.T) {
	t.Run("handles many interpolations efficiently", func(t *testing.T) {
		scope := interpolate.NewScope()
		scope.Global().Set("a", "1")
		scope.Global().Set("b", "2")
		scope.Global().Set("c", "3")

		// Create a string with 1000 variable references
		template := strings.Repeat("{{a}}{{b}}{{c}}", 1000)

		result, err := scope.Interpolate(template)

		require.NoError(t, err)
		assert.Equal(t, strings.Repeat("123", 1000), result)
	})

	t.Run("handles deep scope hierarchy", func(t *testing.T) {
		scope := interpolate.NewScope()

		// Set variables at all levels
		scope.Global().Set("level", "global")

		env := interpolate.NewVariableSet()
		env.Set("level", "environment")
		scope.SetEnvironment(env)

		coll := interpolate.NewVariableSet()
		coll.Set("level", "collection")
		scope.SetCollection(coll)

		req := interpolate.NewVariableSet()
		req.Set("level", "request")
		scope.SetRequest(req)

		// Should use highest precedence (request)
		result, _ := scope.Interpolate("{{level}}")
		assert.Equal(t, "request", result)

		// Clear request, should fall back to collection
		scope.ClearLevel(interpolate.LevelRequest)
		result, _ = scope.Interpolate("{{level}}")
		assert.Equal(t, "collection", result)
	})
}

func TestInterpolation_InterpolateMap(t *testing.T) {
	t.Run("interpolates map helper function", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("host", "api.example.com")
		engine.SetVariable("token", "abc123")

		input := map[string]string{
			"url":           "https://{{host}}/api",
			"authorization": "Bearer {{token}}",
		}

		result, err := engine.InterpolateMap(input)

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com/api", result["url"])
		assert.Equal(t, "Bearer abc123", result["authorization"])
	})
}

func TestInterpolation_Clone(t *testing.T) {
	t.Run("cloned scope is independent", func(t *testing.T) {
		original := interpolate.NewScope()
		original.Global().Set("var1", "original")

		clone := original.Clone()
		clone.Global().Set("var1", "cloned")
		clone.Global().Set("var2", "new")

		// Original unchanged
		assert.Equal(t, "original", original.Get("var1"))
		assert.False(t, original.Has("var2"))

		// Clone has both changes
		assert.Equal(t, "cloned", clone.Get("var1"))
		assert.Equal(t, "new", clone.Get("var2"))
	})
}
