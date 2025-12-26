package interpolate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScope(t *testing.T) {
	t.Run("creates empty scope", func(t *testing.T) {
		scope := NewScope()
		assert.NotNil(t, scope)
	})

	t.Run("has global level by default", func(t *testing.T) {
		scope := NewScope()
		assert.NotNil(t, scope.Global())
	})
}

func TestScope_Levels(t *testing.T) {
	t.Run("sets environment level", func(t *testing.T) {
		scope := NewScope()
		env := NewVariableSet()
		env.Set("api_key", "env_key")

		scope.SetEnvironment(env)

		assert.Equal(t, "env_key", scope.Get("api_key"))
	})

	t.Run("sets collection level", func(t *testing.T) {
		scope := NewScope()
		coll := NewVariableSet()
		coll.Set("base_url", "https://api.example.com")

		scope.SetCollection(coll)

		assert.Equal(t, "https://api.example.com", scope.Get("base_url"))
	})

	t.Run("sets request level", func(t *testing.T) {
		scope := NewScope()
		req := NewVariableSet()
		req.Set("user_id", "123")

		scope.SetRequest(req)

		assert.Equal(t, "123", scope.Get("user_id"))
	})
}

func TestScope_Precedence(t *testing.T) {
	t.Run("request overrides collection", func(t *testing.T) {
		scope := NewScope()

		coll := NewVariableSet()
		coll.Set("name", "collection_value")
		scope.SetCollection(coll)

		req := NewVariableSet()
		req.Set("name", "request_value")
		scope.SetRequest(req)

		assert.Equal(t, "request_value", scope.Get("name"))
	})

	t.Run("collection overrides environment", func(t *testing.T) {
		scope := NewScope()

		env := NewVariableSet()
		env.Set("name", "env_value")
		scope.SetEnvironment(env)

		coll := NewVariableSet()
		coll.Set("name", "collection_value")
		scope.SetCollection(coll)

		assert.Equal(t, "collection_value", scope.Get("name"))
	})

	t.Run("environment overrides global", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("name", "global_value")

		env := NewVariableSet()
		env.Set("name", "env_value")
		scope.SetEnvironment(env)

		assert.Equal(t, "env_value", scope.Get("name"))
	})

	t.Run("request overrides all levels", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("name", "global")

		env := NewVariableSet()
		env.Set("name", "env")
		scope.SetEnvironment(env)

		coll := NewVariableSet()
		coll.Set("name", "collection")
		scope.SetCollection(coll)

		req := NewVariableSet()
		req.Set("name", "request")
		scope.SetRequest(req)

		assert.Equal(t, "request", scope.Get("name"))
	})

	t.Run("falls back to lower precedence levels", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("global_var", "global_value")

		env := NewVariableSet()
		env.Set("env_var", "env_value")
		scope.SetEnvironment(env)

		coll := NewVariableSet()
		coll.Set("coll_var", "coll_value")
		scope.SetCollection(coll)

		req := NewVariableSet()
		req.Set("req_var", "req_value")
		scope.SetRequest(req)

		assert.Equal(t, "global_value", scope.Get("global_var"))
		assert.Equal(t, "env_value", scope.Get("env_var"))
		assert.Equal(t, "coll_value", scope.Get("coll_var"))
		assert.Equal(t, "req_value", scope.Get("req_var"))
	})
}

func TestScope_All(t *testing.T) {
	t.Run("returns merged variables with correct precedence", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("a", "global_a")
		scope.Global().Set("b", "global_b")

		env := NewVariableSet()
		env.Set("b", "env_b")
		env.Set("c", "env_c")
		scope.SetEnvironment(env)

		coll := NewVariableSet()
		coll.Set("c", "coll_c")
		coll.Set("d", "coll_d")
		scope.SetCollection(coll)

		all := scope.All()

		assert.Equal(t, "global_a", all["a"])
		assert.Equal(t, "env_b", all["b"])
		assert.Equal(t, "coll_c", all["c"])
		assert.Equal(t, "coll_d", all["d"])
	})
}

func TestScope_Has(t *testing.T) {
	t.Run("returns true for existing variable at any level", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("global_var", "value")

		assert.True(t, scope.Has("global_var"))
	})

	t.Run("returns false for non-existent variable", func(t *testing.T) {
		scope := NewScope()

		assert.False(t, scope.Has("unknown"))
	})
}

func TestScope_Set(t *testing.T) {
	t.Run("sets variable at request level by default", func(t *testing.T) {
		scope := NewScope()

		scope.Set("name", "value")

		// Should be at request level (highest precedence for user-set values)
		assert.Equal(t, "value", scope.Get("name"))
	})

	t.Run("set variable overrides lower levels", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("name", "global")

		scope.Set("name", "override")

		assert.Equal(t, "override", scope.Get("name"))
	})
}

func TestScope_SetAt(t *testing.T) {
	t.Run("sets variable at global level", func(t *testing.T) {
		scope := NewScope()

		scope.SetAt(LevelGlobal, "name", "value")

		assert.Equal(t, "value", scope.Global().Get("name"))
	})

	t.Run("sets variable at environment level", func(t *testing.T) {
		scope := NewScope()
		scope.SetEnvironment(NewVariableSet())

		scope.SetAt(LevelEnvironment, "name", "value")

		assert.Equal(t, "value", scope.Get("name"))
	})

	t.Run("sets variable at collection level", func(t *testing.T) {
		scope := NewScope()
		scope.SetCollection(NewVariableSet())

		scope.SetAt(LevelCollection, "name", "value")

		assert.Equal(t, "value", scope.Get("name"))
	})

	t.Run("sets variable at request level", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequest(NewVariableSet())

		scope.SetAt(LevelRequest, "name", "value")

		assert.Equal(t, "value", scope.Get("name"))
	})
}

func TestScope_Clear(t *testing.T) {
	t.Run("clears all levels", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("global", "value")

		env := NewVariableSet()
		env.Set("env", "value")
		scope.SetEnvironment(env)

		scope.Clear()

		assert.False(t, scope.Has("global"))
		assert.False(t, scope.Has("env"))
	})
}

func TestScope_ClearLevel(t *testing.T) {
	t.Run("clears specific level", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("global", "value")

		req := NewVariableSet()
		req.Set("req", "value")
		scope.SetRequest(req)

		scope.ClearLevel(LevelRequest)

		assert.True(t, scope.Has("global"))
		assert.False(t, scope.Has("req"))
	})
}

func TestScope_Interpolate(t *testing.T) {
	t.Run("interpolates with scoped variables", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("host", "api.example.com")

		env := NewVariableSet()
		env.Set("token", "secret123")
		scope.SetEnvironment(env)

		result, err := scope.Interpolate("https://{{host}}?token={{token}}")

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com?token=secret123", result)
	})

	t.Run("uses correct precedence during interpolation", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("value", "global")

		env := NewVariableSet()
		env.Set("value", "env")
		scope.SetEnvironment(env)

		result, err := scope.Interpolate("{{value}}")

		require.NoError(t, err)
		assert.Equal(t, "env", result)
	})

	t.Run("interpolates builtins", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Interpolate("UUID: {{$uuid}}")

		require.NoError(t, err)
		assert.Regexp(t, `UUID: [0-9a-f-]{36}`, result)
	})
}

func TestScope_Clone(t *testing.T) {
	t.Run("clones scope with all levels", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("global", "value")

		env := NewVariableSet()
		env.Set("env", "value")
		scope.SetEnvironment(env)

		clone := scope.Clone()
		clone.Set("new", "value")

		assert.True(t, clone.Has("global"))
		assert.True(t, clone.Has("env"))
		assert.True(t, clone.Has("new"))
		assert.False(t, scope.Has("new"))
	})
}

func TestVariableSet(t *testing.T) {
	t.Run("creates empty set", func(t *testing.T) {
		vs := NewVariableSet()
		assert.NotNil(t, vs)
		assert.Equal(t, 0, vs.Len())
	})

	t.Run("sets and gets values", func(t *testing.T) {
		vs := NewVariableSet()
		vs.Set("key", "value")

		assert.Equal(t, "value", vs.Get("key"))
	})

	t.Run("has returns correct result", func(t *testing.T) {
		vs := NewVariableSet()
		vs.Set("exists", "value")

		assert.True(t, vs.Has("exists"))
		assert.False(t, vs.Has("unknown"))
	})

	t.Run("delete removes variable", func(t *testing.T) {
		vs := NewVariableSet()
		vs.Set("key", "value")

		vs.Delete("key")

		assert.False(t, vs.Has("key"))
	})

	t.Run("all returns copy of variables", func(t *testing.T) {
		vs := NewVariableSet()
		vs.Set("a", "1")
		vs.Set("b", "2")

		all := vs.All()
		all["c"] = "3" // Modify the copy

		assert.False(t, vs.Has("c")) // Original unchanged
	})

	t.Run("clear removes all variables", func(t *testing.T) {
		vs := NewVariableSet()
		vs.Set("a", "1")
		vs.Set("b", "2")

		vs.Clear()

		assert.Equal(t, 0, vs.Len())
	})

	t.Run("clone creates independent copy", func(t *testing.T) {
		vs := NewVariableSet()
		vs.Set("key", "value")

		clone := vs.Clone()
		clone.Set("new", "value")

		assert.False(t, vs.Has("new"))
		assert.True(t, clone.Has("key"))
	})

	t.Run("merge combines variable sets", func(t *testing.T) {
		vs1 := NewVariableSet()
		vs1.Set("a", "1")
		vs1.Set("b", "2")

		vs2 := NewVariableSet()
		vs2.Set("b", "override")
		vs2.Set("c", "3")

		vs1.Merge(vs2)

		assert.Equal(t, "1", vs1.Get("a"))
		assert.Equal(t, "override", vs1.Get("b"))
		assert.Equal(t, "3", vs1.Get("c"))
	})
}

func TestScope_GetSource(t *testing.T) {
	t.Run("returns correct source level for variable", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("global_var", "value")

		env := NewVariableSet()
		env.Set("env_var", "value")
		scope.SetEnvironment(env)

		coll := NewVariableSet()
		coll.Set("coll_var", "value")
		scope.SetCollection(coll)

		assert.Equal(t, LevelGlobal, scope.GetSource("global_var"))
		assert.Equal(t, LevelEnvironment, scope.GetSource("env_var"))
		assert.Equal(t, LevelCollection, scope.GetSource("coll_var"))
	})

	t.Run("returns highest precedence source when overridden", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("name", "global")

		env := NewVariableSet()
		env.Set("name", "env")
		scope.SetEnvironment(env)

		assert.Equal(t, LevelEnvironment, scope.GetSource("name"))
	})

	t.Run("returns LevelNone for non-existent variable", func(t *testing.T) {
		scope := NewScope()

		assert.Equal(t, LevelNone, scope.GetSource("unknown"))
	})
}

func TestLevel_String(t *testing.T) {
	t.Run("returns string for each level", func(t *testing.T) {
		assert.Equal(t, "none", LevelNone.String())
		assert.Equal(t, "global", LevelGlobal.String())
		assert.Equal(t, "environment", LevelEnvironment.String())
		assert.Equal(t, "collection", LevelCollection.String())
		assert.Equal(t, "request", LevelRequest.String())
	})

	t.Run("returns none for undefined level", func(t *testing.T) {
		unknown := Level(999)
		assert.Equal(t, "none", unknown.String())
	})
}

func TestScope_InterpolateMap(t *testing.T) {
	t.Run("interpolates map values", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("base_url", "https://api.example.com")
		scope.Global().Set("version", "v1")

		input := map[string]string{
			"url":     "{{base_url}}/{{version}}/users",
			"header":  "Content-Type: application/json",
			"dynamic": "API is at {{base_url}}",
		}

		result, err := scope.InterpolateMap(input)
		require.NoError(t, err)

		assert.Equal(t, "https://api.example.com/v1/users", result["url"])
		assert.Equal(t, "Content-Type: application/json", result["header"])
		assert.Equal(t, "API is at https://api.example.com", result["dynamic"])
	})

	t.Run("handles empty map", func(t *testing.T) {
		scope := NewScope()
		result, err := scope.InterpolateMap(map[string]string{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestScope_ClearLevelExtended(t *testing.T) {
	t.Run("clears global level with variable set", func(t *testing.T) {
		scope := NewScope()
		scope.Global().Set("key", "value")
		assert.Equal(t, "value", scope.Get("key"))

		scope.ClearLevel(LevelGlobal)
		assert.Empty(t, scope.Get("key"))
	})

	t.Run("clears environment level with variable set", func(t *testing.T) {
		scope := NewScope()
		vs := NewVariableSet()
		vs.Set("env_key", "env_value")
		scope.SetEnvironment(vs)
		assert.Equal(t, "env_value", scope.Get("env_key"))

		scope.ClearLevel(LevelEnvironment)
		assert.Empty(t, scope.Get("env_key"))
	})

	t.Run("clears collection level with variable set", func(t *testing.T) {
		scope := NewScope()
		vs := NewVariableSet()
		vs.Set("col_key", "col_value")
		scope.SetCollection(vs)
		assert.Equal(t, "col_value", scope.Get("col_key"))

		scope.ClearLevel(LevelCollection)
		assert.Empty(t, scope.Get("col_key"))
	})

	t.Run("clears request level with variable set", func(t *testing.T) {
		scope := NewScope()
		vs := NewVariableSet()
		vs.Set("req_key", "req_value")
		scope.SetRequest(vs)
		assert.Equal(t, "req_value", scope.Get("req_key"))

		scope.ClearLevel(LevelRequest)
		assert.Empty(t, scope.Get("req_key"))
	})
}

func TestScope_SetAtExtended(t *testing.T) {
	t.Run("sets at global level with value", func(t *testing.T) {
		scope := NewScope()
		scope.SetAt(LevelGlobal, "key", "global_value")
		assert.Equal(t, "global_value", scope.Get("key"))
	})

	t.Run("sets at environment level with value", func(t *testing.T) {
		scope := NewScope()
		scope.SetAt(LevelEnvironment, "key", "env_value")
		assert.Equal(t, "env_value", scope.Get("key"))
	})

	t.Run("sets at collection level with value", func(t *testing.T) {
		scope := NewScope()
		scope.SetAt(LevelCollection, "key", "col_value")
		assert.Equal(t, "col_value", scope.Get("key"))
	})

	t.Run("sets at request level with value", func(t *testing.T) {
		scope := NewScope()
		scope.SetAt(LevelRequest, "key", "req_value")
		assert.Equal(t, "req_value", scope.Get("key"))
	})

	t.Run("invalid level does nothing", func(t *testing.T) {
		scope := NewScope()
		scope.SetAt(Level(99), "key", "value")
		assert.Empty(t, scope.Get("key"))
	})
}
