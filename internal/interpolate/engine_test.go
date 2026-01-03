package interpolate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	t.Run("creates engine", func(t *testing.T) {
		engine := NewEngine()
		assert.NotNil(t, engine)
	})

	t.Run("starts with empty variables", func(t *testing.T) {
		engine := NewEngine()
		assert.Equal(t, 0, len(engine.Variables()))
	})
}

func TestEngine_SetVariable(t *testing.T) {
	t.Run("sets a variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "John")

		assert.Equal(t, "John", engine.GetVariable("name"))
	})

	t.Run("overwrites existing variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "John")
		engine.SetVariable("name", "Jane")

		assert.Equal(t, "Jane", engine.GetVariable("name"))
	})

	t.Run("returns empty for non-existent variable", func(t *testing.T) {
		engine := NewEngine()
		assert.Equal(t, "", engine.GetVariable("unknown"))
	})
}

func TestEngine_SetVariables(t *testing.T) {
	t.Run("sets multiple variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariables(map[string]string{
			"host": "api.example.com",
			"port": "8080",
		})

		assert.Equal(t, "api.example.com", engine.GetVariable("host"))
		assert.Equal(t, "8080", engine.GetVariable("port"))
	})

	t.Run("merges with existing variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("existing", "value")
		engine.SetVariables(map[string]string{
			"new": "newvalue",
		})

		assert.Equal(t, "value", engine.GetVariable("existing"))
		assert.Equal(t, "newvalue", engine.GetVariable("new"))
	})
}

func TestEngine_Interpolate(t *testing.T) {
	t.Run("interpolates single variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "John")

		result, err := engine.Interpolate("Hello, {{name}}!")

		require.NoError(t, err)
		assert.Equal(t, "Hello, John!", result)
	})

	t.Run("interpolates multiple variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("host", "api.example.com")
		engine.SetVariable("port", "8080")

		result, err := engine.Interpolate("https://{{host}}:{{port}}/api")

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com:8080/api", result)
	})

	t.Run("handles spaces in variable syntax", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "John")

		result, err := engine.Interpolate("Hello, {{ name }}!")

		require.NoError(t, err)
		assert.Equal(t, "Hello, John!", result)
	})

	t.Run("preserves text without variables", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("Hello, World!")

		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", result)
	})

	t.Run("returns error for undefined variable", func(t *testing.T) {
		engine := NewEngine()

		_, err := engine.Interpolate("Hello, {{undefined}}!")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("allows undefined variables with option", func(t *testing.T) {
		engine := NewEngine()
		engine.SetOption(OptionAllowUndefined, true)

		result, err := engine.Interpolate("Hello, {{undefined}}!")

		require.NoError(t, err)
		assert.Equal(t, "Hello, !", result)
	})

	t.Run("keeps placeholder for undefined with option", func(t *testing.T) {
		engine := NewEngine()
		engine.SetOption(OptionKeepUndefined, true)

		result, err := engine.Interpolate("Hello, {{undefined}}!")

		require.NoError(t, err)
		assert.Equal(t, "Hello, {{undefined}}!", result)
	})

	t.Run("handles nested braces", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("json", `{"key": "value"}`)

		result, err := engine.Interpolate("Data: {{json}}")

		require.NoError(t, err)
		assert.Equal(t, `Data: {"key": "value"}`, result)
	})

	t.Run("handles empty string variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("empty", "")

		result, err := engine.Interpolate("Value: {{empty}}")

		require.NoError(t, err)
		assert.Equal(t, "Value: ", result)
	})

	t.Run("handles variable at start", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("greeting", "Hello")

		result, err := engine.Interpolate("{{greeting}}, World!")

		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", result)
	})

	t.Run("handles variable at end", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "World")

		result, err := engine.Interpolate("Hello, {{name}}")

		require.NoError(t, err)
		assert.Equal(t, "Hello, World", result)
	})

	t.Run("handles only variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("value", "test")

		result, err := engine.Interpolate("{{value}}")

		require.NoError(t, err)
		assert.Equal(t, "test", result)
	})

	t.Run("handles special characters in variable value", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("special", "a&b=c?d#e")

		result, err := engine.Interpolate("url?param={{special}}")

		require.NoError(t, err)
		assert.Equal(t, "url?param=a&b=c?d#e", result)
	})

	t.Run("handles underscore in variable name", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("user_name", "John")

		result, err := engine.Interpolate("User: {{user_name}}")

		require.NoError(t, err)
		assert.Equal(t, "User: John", result)
	})

	t.Run("handles hyphen in variable name", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("user-name", "John")

		result, err := engine.Interpolate("User: {{user-name}}")

		require.NoError(t, err)
		assert.Equal(t, "User: John", result)
	})

	t.Run("handles numbers in variable name", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("var1", "value1")

		result, err := engine.Interpolate("Value: {{var1}}")

		require.NoError(t, err)
		assert.Equal(t, "Value: value1", result)
	})
}

func TestEngine_BuiltinVariables(t *testing.T) {
	t.Run("interpolates $uuid", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("ID: {{$uuid}}")

		require.NoError(t, err)
		assert.Contains(t, result, "ID: ")
		// UUID format: 8-4-4-4-12 hex chars
		assert.Regexp(t, `ID: [0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, result)
	})

	t.Run("interpolates $timestamp", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("Time: {{$timestamp}}")

		require.NoError(t, err)
		assert.Contains(t, result, "Time: ")
		// Should be a Unix timestamp (numeric)
		assert.Regexp(t, `Time: \d+`, result)
	})

	t.Run("interpolates $isoTimestamp", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("Time: {{$isoTimestamp}}")

		require.NoError(t, err)
		// ISO 8601 format
		assert.Regexp(t, `Time: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, result)
	})

	t.Run("interpolates $randomInt", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("Num: {{$randomInt}}")

		require.NoError(t, err)
		assert.Regexp(t, `Num: \d+`, result)
	})

	t.Run("generates different UUIDs each time", func(t *testing.T) {
		engine := NewEngine()

		result1, _ := engine.Interpolate("{{$uuid}}")
		result2, _ := engine.Interpolate("{{$uuid}}")

		assert.NotEqual(t, result1, result2)
	})

	t.Run("interpolates $randomEmail", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("Email: {{$randomEmail}}")

		require.NoError(t, err)
		assert.Contains(t, result, "@")
		assert.Contains(t, result, ".")
	})

	t.Run("interpolates $randomName", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("Name: {{$randomName}}")

		require.NoError(t, err)
		assert.NotEqual(t, "Name: ", result)
	})

	t.Run("interpolates $date", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Interpolate("Date: {{$date}}")

		require.NoError(t, err)
		// Date format: YYYY-MM-DD
		assert.Regexp(t, `Date: \d{4}-\d{2}-\d{2}`, result)
	})
}

func TestEngine_RegisterBuiltin(t *testing.T) {
	t.Run("registers custom builtin", func(t *testing.T) {
		engine := NewEngine()
		engine.RegisterBuiltin("$custom", func() string {
			return "custom_value"
		})

		result, err := engine.Interpolate("Value: {{$custom}}")

		require.NoError(t, err)
		assert.Equal(t, "Value: custom_value", result)
	})

	t.Run("overwrites existing builtin", func(t *testing.T) {
		engine := NewEngine()
		engine.RegisterBuiltin("$uuid", func() string {
			return "fixed-uuid"
		})

		result, err := engine.Interpolate("{{$uuid}}")

		require.NoError(t, err)
		assert.Equal(t, "fixed-uuid", result)
	})
}

func TestEngine_InterpolateMap(t *testing.T) {
	t.Run("interpolates all values in map", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("host", "api.example.com")
		engine.SetVariable("token", "secret123")

		input := map[string]string{
			"url":           "https://{{host}}/api",
			"Authorization": "Bearer {{token}}",
		}

		result, err := engine.InterpolateMap(input)

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com/api", result["url"])
		assert.Equal(t, "Bearer secret123", result["Authorization"])
	})

	t.Run("returns error if any interpolation fails", func(t *testing.T) {
		engine := NewEngine()

		input := map[string]string{
			"valid":   "no variables",
			"invalid": "{{undefined}}",
		}

		_, err := engine.InterpolateMap(input)

		assert.Error(t, err)
	})
}

func TestEngine_Clear(t *testing.T) {
	t.Run("clears all variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("var1", "value1")
		engine.SetVariable("var2", "value2")

		engine.Clear()

		assert.Equal(t, 0, len(engine.Variables()))
	})
}

func TestEngine_Delete(t *testing.T) {
	t.Run("deletes a variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("var1", "value1")
		engine.SetVariable("var2", "value2")

		engine.DeleteVariable("var1")

		assert.Equal(t, "", engine.GetVariable("var1"))
		assert.Equal(t, "value2", engine.GetVariable("var2"))
	})

	t.Run("does nothing for non-existent variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("var1", "value1")

		engine.DeleteVariable("unknown")

		assert.Equal(t, "value1", engine.GetVariable("var1"))
	})
}

func TestEngine_HasVariable(t *testing.T) {
	t.Run("returns true for existing variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("exists", "value")

		assert.True(t, engine.HasVariable("exists"))
	})

	t.Run("returns false for non-existent variable", func(t *testing.T) {
		engine := NewEngine()

		assert.False(t, engine.HasVariable("unknown"))
	})

	t.Run("returns true for empty string variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("empty", "")

		assert.True(t, engine.HasVariable("empty"))
	})
}

func TestEngine_ExtractVariables(t *testing.T) {
	t.Run("extracts variable names from string", func(t *testing.T) {
		engine := NewEngine()

		vars := engine.ExtractVariables("Hello {{name}}, your ID is {{id}}")

		assert.Contains(t, vars, "name")
		assert.Contains(t, vars, "id")
		assert.Len(t, vars, 2)
	})

	t.Run("extracts builtin variable names", func(t *testing.T) {
		engine := NewEngine()

		vars := engine.ExtractVariables("UUID: {{$uuid}}, Time: {{$timestamp}}")

		assert.Contains(t, vars, "$uuid")
		assert.Contains(t, vars, "$timestamp")
	})

	t.Run("returns empty for no variables", func(t *testing.T) {
		engine := NewEngine()

		vars := engine.ExtractVariables("No variables here")

		assert.Empty(t, vars)
	})

	t.Run("handles duplicate variables", func(t *testing.T) {
		engine := NewEngine()

		vars := engine.ExtractVariables("{{name}} and {{name}} again")

		// Should deduplicate
		count := 0
		for _, v := range vars {
			if v == "name" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})
}

func TestEngine_Validate(t *testing.T) {
	t.Run("validates all variables exist", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "John")
		engine.SetVariable("id", "123")

		err := engine.Validate("Hello {{name}}, your ID is {{id}}")

		assert.NoError(t, err)
	})

	t.Run("returns error for missing variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "John")

		err := engine.Validate("Hello {{name}}, your ID is {{id}}")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("validates builtins always exist", func(t *testing.T) {
		engine := NewEngine()

		err := engine.Validate("UUID: {{$uuid}}")

		assert.NoError(t, err)
	})
}

func TestEngine_Clone(t *testing.T) {
	t.Run("clones engine with variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("var1", "value1")

		clone := engine.Clone()
		clone.SetVariable("var2", "value2")

		// Original should not have var2
		assert.False(t, engine.HasVariable("var2"))
		// Clone should have both
		assert.True(t, clone.HasVariable("var1"))
		assert.True(t, clone.HasVariable("var2"))
	})
}

func TestEngine_Performance(t *testing.T) {
	t.Run("handles large number of variables", func(t *testing.T) {
		engine := NewEngine()

		// Set 1000 variables
		for i := 0; i < 1000; i++ {
			engine.SetVariable(strings.Repeat("v", i%10+1)+string(rune('0'+i%10)), "value")
		}

		// Should still work efficiently
		engine.SetVariable("target", "found")
		result, err := engine.Interpolate("Result: {{target}}")

		require.NoError(t, err)
		assert.Equal(t, "Result: found", result)
	})

	t.Run("handles string with many variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("a", "1")
		engine.SetVariable("b", "2")
		engine.SetVariable("c", "3")

		// String with repeated variables
		input := strings.Repeat("{{a}}{{b}}{{c}}", 100)
		result, err := engine.Interpolate(input)

		require.NoError(t, err)
		assert.Equal(t, strings.Repeat("123", 100), result)
	})
}

func TestEngine_Options(t *testing.T) {
	t.Run("SetOption and GetOption work correctly", func(t *testing.T) {
		engine := NewEngine()

		engine.SetOption("strict", true)
		assert.True(t, engine.GetOption("strict"))

		engine.SetOption("strict", false)
		assert.False(t, engine.GetOption("strict"))
	})

	t.Run("GetOption returns false for unset options", func(t *testing.T) {
		engine := NewEngine()
		assert.False(t, engine.GetOption("nonexistent"))
	})
}

func TestEngine_VariablesWithData(t *testing.T) {
	t.Run("Variables returns copy of variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("key1", "value1")
		engine.SetVariable("key2", "value2")
		engine.SetVariable("key3", "value3")

		vars := engine.Variables()
		assert.Equal(t, 3, len(vars))
		assert.Equal(t, "value1", vars["key1"])
		assert.Equal(t, "value2", vars["key2"])
		assert.Equal(t, "value3", vars["key3"])
	})

	t.Run("modifying returned map doesn't affect engine", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("key", "original")

		vars := engine.Variables()
		vars["key"] = "modified"

		assert.Equal(t, "original", engine.GetVariable("key"))
	})
}

func TestEngine_CloneWithData(t *testing.T) {
	t.Run("Clone copies variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("key1", "value1")
		engine.SetVariable("key2", "value2")

		clone := engine.Clone()

		assert.Equal(t, "value1", clone.GetVariable("key1"))
		assert.Equal(t, "value2", clone.GetVariable("key2"))
	})

	t.Run("Clone copies options", func(t *testing.T) {
		engine := NewEngine()
		engine.SetOption("strict", true)
		engine.SetOption("debug", false)

		clone := engine.Clone()

		assert.True(t, clone.GetOption("strict"))
		assert.False(t, clone.GetOption("debug"))
	})

	t.Run("Clone is independent from original", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("key", "original")

		clone := engine.Clone()
		clone.SetVariable("key", "modified")

		assert.Equal(t, "original", engine.GetVariable("key"))
		assert.Equal(t, "modified", clone.GetVariable("key"))
	})
}

func TestEngine_ClearExtended(t *testing.T) {
	t.Run("Clear removes all variables", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("key1", "value1")
		engine.SetVariable("key2", "value2")

		engine.Clear()

		assert.False(t, engine.HasVariable("key1"))
		assert.False(t, engine.HasVariable("key2"))
	})
}

func TestEngine_DeleteVariable(t *testing.T) {
	t.Run("DeleteVariable removes a specific variable", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("key1", "value1")
		engine.SetVariable("key2", "value2")

		engine.DeleteVariable("key1")

		assert.False(t, engine.HasVariable("key1"))
		assert.True(t, engine.HasVariable("key2"))
	})
}

func TestEngine_InterpolateMapExtended(t *testing.T) {
	t.Run("InterpolateMap handles empty map", func(t *testing.T) {
		engine := NewEngine()
		result, err := engine.InterpolateMap(map[string]string{})
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("InterpolateMap processes multiple keys", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "test")
		engine.SetVariable("version", "1.0")

		input := map[string]string{
			"title":   "{{name}}",
			"ver":     "v{{version}}",
			"literal": "no vars",
		}

		result, err := engine.InterpolateMap(input)
		require.NoError(t, err)
		assert.Equal(t, "test", result["title"])
		assert.Equal(t, "v1.0", result["ver"])
		assert.Equal(t, "no vars", result["literal"])
	})
}

func TestEngine_ExtractVariablesExtended(t *testing.T) {
	t.Run("ExtractVariables returns empty for no variables", func(t *testing.T) {
		engine := NewEngine()
		vars := engine.ExtractVariables("no variables here")
		assert.Len(t, vars, 0)
	})

	t.Run("ExtractVariables finds multiple variables", func(t *testing.T) {
		engine := NewEngine()
		vars := engine.ExtractVariables("{{var1}} and {{var2}} and {{var1}}")
		// May contain duplicates depending on implementation
		assert.GreaterOrEqual(t, len(vars), 2)
	})
}

func TestEngine_ValidateExtended(t *testing.T) {
	t.Run("Validate returns no error for valid input", func(t *testing.T) {
		engine := NewEngine()
		engine.SetVariable("name", "test")
		err := engine.Validate("Hello {{name}}")
		assert.NoError(t, err)
	})

	t.Run("Validate returns no error for plain text", func(t *testing.T) {
		engine := NewEngine()
		err := engine.Validate("no variables here")
		assert.NoError(t, err)
	})
}
