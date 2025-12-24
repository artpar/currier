package script

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	t.Run("creates engine", func(t *testing.T) {
		engine := NewEngine()
		assert.NotNil(t, engine)
	})

	t.Run("engine is reusable", func(t *testing.T) {
		engine := NewEngine()

		result1, err := engine.Execute(context.Background(), "1 + 1")
		require.NoError(t, err)
		assert.Equal(t, int64(2), result1)

		result2, err := engine.Execute(context.Background(), "2 + 2")
		require.NoError(t, err)
		assert.Equal(t, int64(4), result2)
	})
}

func TestEngine_Execute(t *testing.T) {
	t.Run("executes simple expression", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), "1 + 1")

		require.NoError(t, err)
		assert.Equal(t, int64(2), result)
	})

	t.Run("executes string concatenation", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), "'hello' + ' ' + 'world'")

		require.NoError(t, err)
		assert.Equal(t, "hello world", result)
	})

	t.Run("executes function definition and call", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), `
			function add(a, b) {
				return a + b;
			}
			add(3, 4);
		`)

		require.NoError(t, err)
		assert.Equal(t, int64(7), result)
	})

	t.Run("executes arrow function", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), `
			const multiply = (a, b) => a * b;
			multiply(3, 4);
		`)

		require.NoError(t, err)
		assert.Equal(t, int64(12), result)
	})

	t.Run("returns undefined for statements", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), "var x = 5;")

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error for syntax error", func(t *testing.T) {
		engine := NewEngine()

		_, err := engine.Execute(context.Background(), "function {")

		assert.Error(t, err)
	})

	t.Run("returns error for runtime error", func(t *testing.T) {
		engine := NewEngine()

		_, err := engine.Execute(context.Background(), "undefinedVariable.property")

		assert.Error(t, err)
	})

	t.Run("handles JSON operations", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), `
			var obj = {name: "John", age: 30};
			JSON.stringify(obj);
		`)

		require.NoError(t, err)
		assert.Contains(t, result.(string), "John")
	})

	t.Run("handles array operations", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), `
			var arr = [1, 2, 3, 4, 5];
			arr.filter(x => x > 2).map(x => x * 2);
		`)

		require.NoError(t, err)
		// Result should be [6, 8, 10]
		assert.NotNil(t, result)
	})

	t.Run("handles object destructuring", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), `
			const obj = {a: 1, b: 2};
			const {a, b} = obj;
			a + b;
		`)

		require.NoError(t, err)
		assert.Equal(t, int64(3), result)
	})

	t.Run("handles template literals", func(t *testing.T) {
		engine := NewEngine()

		result, err := engine.Execute(context.Background(), "const name = 'World'; `Hello ${name}!`")

		require.NoError(t, err)
		assert.Equal(t, "Hello World!", result)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		engine := NewEngine()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := engine.Execute(ctx, "while(true) {}")

		assert.Error(t, err)
	})

	t.Run("respects context timeout", func(t *testing.T) {
		engine := NewEngine()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := engine.Execute(ctx, "while(true) {}")

		assert.Error(t, err)
	})
}

func TestEngine_SetGlobal(t *testing.T) {
	t.Run("sets string global", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("name", "John")

		result, err := engine.Execute(context.Background(), "name")

		require.NoError(t, err)
		assert.Equal(t, "John", result)
	})

	t.Run("sets number global", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("count", 42)

		result, err := engine.Execute(context.Background(), "count * 2")

		require.NoError(t, err)
		assert.Equal(t, int64(84), result)
	})

	t.Run("sets boolean global", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("enabled", true)

		result, err := engine.Execute(context.Background(), "enabled ? 'yes' : 'no'")

		require.NoError(t, err)
		assert.Equal(t, "yes", result)
	})

	t.Run("sets object global", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("user", map[string]interface{}{
			"name": "John",
			"age":  30,
		})

		result, err := engine.Execute(context.Background(), "user.name + ' is ' + user.age")

		require.NoError(t, err)
		assert.Equal(t, "John is 30", result)
	})

	t.Run("sets array global", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("items", []interface{}{1, 2, 3})

		result, err := engine.Execute(context.Background(), "items.length")

		require.NoError(t, err)
		assert.Equal(t, int64(3), result)
	})

	t.Run("sets function global", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("greet", func(name string) string {
			return "Hello, " + name + "!"
		})

		result, err := engine.Execute(context.Background(), "greet('World')")

		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", result)
	})

	t.Run("overwrites existing global", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("value", 1)
		engine.SetGlobal("value", 2)

		result, err := engine.Execute(context.Background(), "value")

		require.NoError(t, err)
		assert.Equal(t, int64(2), result)
	})
}

func TestEngine_GetGlobal(t *testing.T) {
	t.Run("gets global set from Go", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("myVar", "test")

		value := engine.GetGlobal("myVar")

		assert.Equal(t, "test", value)
	})

	t.Run("gets global set from JavaScript", func(t *testing.T) {
		engine := NewEngine()
		_, err := engine.Execute(context.Background(), "var myGlobal = 'from js';")
		require.NoError(t, err)

		value := engine.GetGlobal("myGlobal")

		assert.Equal(t, "from js", value)
	})

	t.Run("returns nil for undefined global", func(t *testing.T) {
		engine := NewEngine()

		value := engine.GetGlobal("nonexistent")

		assert.Nil(t, value)
	})
}

func TestEngine_RegisterFunction(t *testing.T) {
	t.Run("registers function with no return", func(t *testing.T) {
		engine := NewEngine()
		var called bool
		engine.RegisterFunction("notify", func() {
			called = true
		})

		_, err := engine.Execute(context.Background(), "notify()")

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("registers function with string return", func(t *testing.T) {
		engine := NewEngine()
		engine.RegisterFunction("getVersion", func() string {
			return "1.0.0"
		})

		result, err := engine.Execute(context.Background(), "getVersion()")

		require.NoError(t, err)
		assert.Equal(t, "1.0.0", result)
	})

	t.Run("registers function with parameters", func(t *testing.T) {
		engine := NewEngine()
		engine.RegisterFunction("sum", func(a, b int) int {
			return a + b
		})

		result, err := engine.Execute(context.Background(), "sum(10, 20)")

		require.NoError(t, err)
		assert.Equal(t, int64(30), result)
	})

	t.Run("registers function with multiple returns", func(t *testing.T) {
		engine := NewEngine()
		engine.RegisterFunction("divide", func(a, b int) (int, error) {
			if b == 0 {
				return 0, assert.AnError
			}
			return a / b, nil
		})

		result, err := engine.Execute(context.Background(), "divide(10, 2)")

		require.NoError(t, err)
		assert.Equal(t, int64(5), result)
	})

	t.Run("function error propagates to JavaScript", func(t *testing.T) {
		engine := NewEngine()
		engine.RegisterFunction("fail", func() error {
			return assert.AnError
		})

		_, err := engine.Execute(context.Background(), `
			try {
				fail();
			} catch(e) {
				throw e;
			}
		`)

		assert.Error(t, err)
	})
}

func TestEngine_RegisterObject(t *testing.T) {
	t.Run("registers object with methods", func(t *testing.T) {
		engine := NewEngine()

		type Calculator struct{}
		calc := &Calculator{}

		engine.RegisterObject("calc", map[string]interface{}{
			"add": func(a, b int) int { return a + b },
			"sub": func(a, b int) int { return a - b },
		})

		result, err := engine.Execute(context.Background(), "calc.add(10, 5) + calc.sub(10, 5)")

		require.NoError(t, err)
		assert.Equal(t, int64(20), result)
		_ = calc // suppress unused warning
	})

	t.Run("registers nested object", func(t *testing.T) {
		engine := NewEngine()

		engine.RegisterObject("app", map[string]interface{}{
			"name":    "Currier",
			"version": "1.0.0",
			"config": map[string]interface{}{
				"timeout": 30,
				"retries": 3,
			},
		})

		result, err := engine.Execute(context.Background(), "app.name + ' v' + app.version")

		require.NoError(t, err)
		assert.Equal(t, "Currier v1.0.0", result)
	})
}

func TestEngine_Console(t *testing.T) {
	t.Run("console.log captures output", func(t *testing.T) {
		engine := NewEngine()
		var logs []string
		engine.SetConsoleHandler(func(level, message string) {
			logs = append(logs, level+": "+message)
		})

		_, err := engine.Execute(context.Background(), `console.log("hello world")`)

		require.NoError(t, err)
		assert.Contains(t, logs, "log: hello world")
	})

	t.Run("console.error captures output", func(t *testing.T) {
		engine := NewEngine()
		var logs []string
		engine.SetConsoleHandler(func(level, message string) {
			logs = append(logs, level+": "+message)
		})

		_, err := engine.Execute(context.Background(), `console.error("something failed")`)

		require.NoError(t, err)
		assert.Contains(t, logs, "error: something failed")
	})

	t.Run("console.warn captures output", func(t *testing.T) {
		engine := NewEngine()
		var logs []string
		engine.SetConsoleHandler(func(level, message string) {
			logs = append(logs, level+": "+message)
		})

		_, err := engine.Execute(context.Background(), `console.warn("warning message")`)

		require.NoError(t, err)
		assert.Contains(t, logs, "warn: warning message")
	})

	t.Run("console.log with multiple arguments", func(t *testing.T) {
		engine := NewEngine()
		var logs []string
		engine.SetConsoleHandler(func(level, message string) {
			logs = append(logs, message)
		})

		_, err := engine.Execute(context.Background(), `console.log("name:", "John", "age:", 30)`)

		require.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Contains(t, logs[0], "name:")
		assert.Contains(t, logs[0], "John")
	})
}

func TestEngine_Reset(t *testing.T) {
	t.Run("reset clears globals", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("myVar", "value")

		engine.Reset()

		value := engine.GetGlobal("myVar")
		assert.Nil(t, value)
	})

	t.Run("reset clears script state", func(t *testing.T) {
		engine := NewEngine()
		_, err := engine.Execute(context.Background(), "var counter = 1;")
		require.NoError(t, err)

		engine.Reset()

		_, err = engine.Execute(context.Background(), "counter")
		assert.Error(t, err) // counter should be undefined
	})
}

func TestEngine_Clone(t *testing.T) {
	t.Run("clone creates independent engine", func(t *testing.T) {
		engine := NewEngine()
		engine.SetGlobal("shared", "original")

		clone := engine.Clone()
		clone.SetGlobal("shared", "cloned")

		originalValue := engine.GetGlobal("shared")
		clonedValue := clone.GetGlobal("shared")

		assert.Equal(t, "original", originalValue)
		assert.Equal(t, "cloned", clonedValue)
	})

	t.Run("clone copies registered functions", func(t *testing.T) {
		engine := NewEngine()
		engine.RegisterFunction("greet", func() string {
			return "hello"
		})

		clone := engine.Clone()

		result, err := clone.Execute(context.Background(), "greet()")

		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})
}

func TestEngine_Concurrent(t *testing.T) {
	t.Run("handles concurrent execution on separate engines", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(n int) {
				engine := NewEngine()
				engine.SetGlobal("n", n)
				result, err := engine.Execute(context.Background(), "n * 2")
				assert.NoError(t, err)
				assert.Equal(t, int64(n*2), result)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestEngine_Performance(t *testing.T) {
	t.Run("handles large scripts", func(t *testing.T) {
		engine := NewEngine()

		// Script that does significant computation
		script := `
			var result = 0;
			for (var i = 0; i < 10000; i++) {
				result += i;
			}
			result;
		`

		result, err := engine.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, int64(49995000), result)
	})

	t.Run("handles many small executions", func(t *testing.T) {
		engine := NewEngine()

		for i := 0; i < 1000; i++ {
			result, err := engine.Execute(context.Background(), "1 + 1")
			require.NoError(t, err)
			assert.Equal(t, int64(2), result)
		}
	})
}
