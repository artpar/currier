package script

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScope(t *testing.T) {
	t.Run("creates scope", func(t *testing.T) {
		scope := NewScope()
		assert.NotNil(t, scope)
	})

	t.Run("scope has engine", func(t *testing.T) {
		scope := NewScope()
		assert.NotNil(t, scope.Engine())
	})
}

func TestScope_Request(t *testing.T) {
	t.Run("sets and gets request method", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestMethod("POST")

		result, err := scope.Execute(context.Background(), "currier.request.method")

		require.NoError(t, err)
		assert.Equal(t, "POST", result)
	})

	t.Run("sets and gets request URL", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestURL("https://api.example.com/users")

		result, err := scope.Execute(context.Background(), "currier.request.url")

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com/users", result)
	})

	t.Run("sets and gets request headers", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestHeaders(map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token123",
		})

		result, err := scope.Execute(context.Background(), "currier.request.headers['Content-Type']")

		require.NoError(t, err)
		assert.Equal(t, "application/json", result)
	})

	t.Run("sets and gets request body", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestBody(`{"name": "John"}`)

		result, err := scope.Execute(context.Background(), "currier.request.body")

		require.NoError(t, err)
		assert.Equal(t, `{"name": "John"}`, result)
	})

	t.Run("request.setHeader modifies headers", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestHeaders(map[string]string{})

		_, err := scope.Execute(context.Background(), `currier.request.setHeader("X-Custom", "value")`)
		require.NoError(t, err)

		headers := scope.GetRequestHeaders()
		assert.Equal(t, "value", headers["X-Custom"])
	})

	t.Run("request.setBody modifies body", func(t *testing.T) {
		scope := NewScope()

		_, err := scope.Execute(context.Background(), `currier.request.setBody('{"updated": true}')`)
		require.NoError(t, err)

		body := scope.GetRequestBody()
		assert.Equal(t, `{"updated": true}`, body)
	})

	t.Run("request.setUrl modifies url", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestURL("https://old.com")

		_, err := scope.Execute(context.Background(), `currier.request.setUrl("https://new.com/api")`)
		require.NoError(t, err)

		url := scope.GetRequestURL()
		assert.Equal(t, "https://new.com/api", url)
	})
}

func TestScope_Response(t *testing.T) {
	t.Run("sets and gets response status", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseStatus(200)

		result, err := scope.Execute(context.Background(), "currier.response.status")

		require.NoError(t, err)
		assert.Equal(t, int64(200), result)
	})

	t.Run("sets and gets response status text", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseStatusText("OK")

		result, err := scope.Execute(context.Background(), "currier.response.statusText")

		require.NoError(t, err)
		assert.Equal(t, "OK", result)
	})

	t.Run("sets and gets response headers", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseHeaders(map[string]string{
			"Content-Type":   "application/json",
			"Content-Length": "123",
		})

		result, err := scope.Execute(context.Background(), "currier.response.headers['Content-Type']")

		require.NoError(t, err)
		assert.Equal(t, "application/json", result)
	})

	t.Run("sets and gets response body", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseBody(`{"id": 1, "name": "John"}`)

		result, err := scope.Execute(context.Background(), "currier.response.body")

		require.NoError(t, err)
		assert.Equal(t, `{"id": 1, "name": "John"}`, result)
	})

	t.Run("response.json() parses body", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseBody(`{"id": 1, "name": "John"}`)

		result, err := scope.Execute(context.Background(), "currier.response.json().name")

		require.NoError(t, err)
		assert.Equal(t, "John", result)
	})

	t.Run("response.json() returns null for invalid JSON", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseBody("not json")

		result, err := scope.Execute(context.Background(), "currier.response.json()")

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("sets and gets response time", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseTime(150)

		result, err := scope.Execute(context.Background(), "currier.response.time")

		require.NoError(t, err)
		assert.Equal(t, int64(150), result)
	})

	t.Run("sets and gets response size", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseSize(1024)

		result, err := scope.Execute(context.Background(), "currier.response.size")

		require.NoError(t, err)
		assert.Equal(t, int64(1024), result)
	})
}

func TestScope_Variables(t *testing.T) {
	t.Run("gets variable via currier.variables", func(t *testing.T) {
		scope := NewScope()
		scope.SetVariable("api_key", "secret123")

		result, err := scope.Execute(context.Background(), "currier.variables.api_key")

		require.NoError(t, err)
		assert.Equal(t, "secret123", result)
	})

	t.Run("gets variable via currier.getVariable", func(t *testing.T) {
		scope := NewScope()
		scope.SetVariable("base_url", "https://api.example.com")

		result, err := scope.Execute(context.Background(), `currier.getVariable("base_url")`)

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com", result)
	})

	t.Run("sets variable via currier.setVariable", func(t *testing.T) {
		scope := NewScope()

		_, err := scope.Execute(context.Background(), `currier.setVariable("new_var", "new_value")`)
		require.NoError(t, err)

		value := scope.GetVariable("new_var")
		assert.Equal(t, "new_value", value)
	})

	t.Run("setVariable persists across executions", func(t *testing.T) {
		scope := NewScope()

		_, err := scope.Execute(context.Background(), `currier.setVariable("persistent", "value1")`)
		require.NoError(t, err)

		result, err := scope.Execute(context.Background(), `currier.getVariable("persistent")`)
		require.NoError(t, err)
		assert.Equal(t, "value1", result)
	})

	t.Run("sets local variable via currier.setLocalVariable", func(t *testing.T) {
		scope := NewScope()

		_, err := scope.Execute(context.Background(), `currier.setLocalVariable("local_var", "local_value")`)
		require.NoError(t, err)

		result, err := scope.Execute(context.Background(), `currier.getVariable("local_var")`)
		require.NoError(t, err)
		assert.Equal(t, "local_value", result)
	})

	t.Run("getVariable returns empty string for undefined", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `currier.getVariable("undefined_var")`)

		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

func TestScope_Environment(t *testing.T) {
	t.Run("gets environment name", func(t *testing.T) {
		scope := NewScope()
		scope.SetEnvironmentName("Production")

		result, err := scope.Execute(context.Background(), "currier.environment.name")

		require.NoError(t, err)
		assert.Equal(t, "Production", result)
	})

	t.Run("gets environment variable", func(t *testing.T) {
		scope := NewScope()
		scope.SetEnvironmentVariable("api_url", "https://prod.api.com")

		result, err := scope.Execute(context.Background(), `currier.environment.get("api_url")`)

		require.NoError(t, err)
		assert.Equal(t, "https://prod.api.com", result)
	})

	t.Run("sets environment variable", func(t *testing.T) {
		scope := NewScope()

		_, err := scope.Execute(context.Background(), `currier.environment.set("new_env_var", "env_value")`)
		require.NoError(t, err)

		value := scope.GetEnvironmentVariable("new_env_var")
		assert.Equal(t, "env_value", value)
	})
}

func TestScope_Logging(t *testing.T) {
	t.Run("currier.log captures output", func(t *testing.T) {
		scope := NewScope()
		var logs []string
		scope.SetLogHandler(func(message string) {
			logs = append(logs, message)
		})

		_, err := scope.Execute(context.Background(), `currier.log("test message")`)

		require.NoError(t, err)
		assert.Contains(t, logs, "test message")
	})

	t.Run("currier.log with multiple arguments", func(t *testing.T) {
		scope := NewScope()
		var logs []string
		scope.SetLogHandler(func(message string) {
			logs = append(logs, message)
		})

		_, err := scope.Execute(context.Background(), `currier.log("user:", "John", "age:", 30)`)

		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Contains(t, logs[0], "user:")
		assert.Contains(t, logs[0], "John")
	})
}

func TestScope_Utilities(t *testing.T) {
	t.Run("currier.base64.encode", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `currier.base64.encode("hello")`)

		require.NoError(t, err)
		assert.Equal(t, "aGVsbG8=", result)
	})

	t.Run("currier.base64.decode", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `currier.base64.decode("aGVsbG8=")`)

		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("currier.crypto.md5", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `currier.crypto.md5("hello")`)

		require.NoError(t, err)
		assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", result)
	})

	t.Run("currier.crypto.sha256", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `currier.crypto.sha256("hello")`)

		require.NoError(t, err)
		assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", result)
	})

	t.Run("currier.crypto.hmac", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `currier.crypto.hmac("sha256", "secret", "message")`)

		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

func TestScope_SendRequest(t *testing.T) {
	t.Run("currier.sendRequest is available", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `typeof currier.sendRequest`)

		require.NoError(t, err)
		assert.Equal(t, "function", result)
	})

	t.Run("sendRequest returns nil when no sender set", func(t *testing.T) {
		scope := NewScope()

		result, err := scope.Execute(context.Background(), `currier.sendRequest({url: "https://api.example.com"})`)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("sendRequest calls sender with options", func(t *testing.T) {
		scope := NewScope()
		var capturedOptions map[string]interface{}
		scope.SetRequestSender(func(options map[string]interface{}) (map[string]interface{}, error) {
			capturedOptions = options
			return map[string]interface{}{
				"status": 200,
				"body":   `{"result": "success"}`,
			}, nil
		})

		_, err := scope.Execute(context.Background(), `currier.sendRequest({url: "https://api.example.com", method: "POST"})`)

		require.NoError(t, err)
		assert.NotNil(t, capturedOptions)
		assert.Equal(t, "https://api.example.com", capturedOptions["url"])
		assert.Equal(t, "POST", capturedOptions["method"])
	})

	t.Run("sendRequest returns sender result", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestSender(func(options map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"status": 200,
				"body":   `{"id": 123}`,
			}, nil
		})

		result, err := scope.Execute(context.Background(), `currier.sendRequest({url: "https://api.example.com"}).status`)

		require.NoError(t, err)
		assert.Equal(t, int64(200), result)
	})

	t.Run("sendRequest returns nil on sender error", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestSender(func(options map[string]interface{}) (map[string]interface{}, error) {
			return nil, assert.AnError
		})

		result, err := scope.Execute(context.Background(), `currier.sendRequest({url: "https://api.example.com"})`)

		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestScope_CompleteFlow(t *testing.T) {
	t.Run("pre-request script modifies request", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestMethod("GET")
		scope.SetRequestURL("https://api.example.com/users")
		scope.SetRequestHeaders(map[string]string{})
		scope.SetVariable("token", "abc123")

		script := `
			var token = currier.getVariable("token");
			currier.request.setHeader("Authorization", "Bearer " + token);
			currier.request.setHeader("X-Timestamp", Date.now().toString());
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		headers := scope.GetRequestHeaders()
		assert.Equal(t, "Bearer abc123", headers["Authorization"])
		assert.NotEmpty(t, headers["X-Timestamp"])
	})

	t.Run("post-response script extracts data", func(t *testing.T) {
		scope := NewScope()
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`{"user": {"id": 123, "name": "John"}, "token": "xyz789"}`)

		script := `
			var data = currier.response.json();
			currier.setVariable("user_id", data.user.id.toString());
			currier.setVariable("auth_token", data.token);
			data.user.name;
		`

		result, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "John", result)
		assert.Equal(t, "123", scope.GetVariable("user_id"))
		assert.Equal(t, "xyz789", scope.GetVariable("auth_token"))
	})
}

func TestScope_Clone(t *testing.T) {
	t.Run("clone creates independent scope", func(t *testing.T) {
		scope := NewScope()
		scope.SetVariable("shared", "original")

		clone := scope.Clone()
		clone.SetVariable("shared", "cloned")

		assert.Equal(t, "original", scope.GetVariable("shared"))
		assert.Equal(t, "cloned", clone.GetVariable("shared"))
	})

	t.Run("clone preserves request data", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestURL("https://api.example.com")
		scope.SetRequestMethod("POST")

		clone := scope.Clone()

		result, err := clone.Execute(context.Background(), "currier.request.method")
		require.NoError(t, err)
		assert.Equal(t, "POST", result)
	})
}

func TestScope_Reset(t *testing.T) {
	t.Run("reset clears variables", func(t *testing.T) {
		scope := NewScope()
		scope.SetVariable("test", "value")

		scope.Reset()

		assert.Equal(t, "", scope.GetVariable("test"))
	})

	t.Run("reset clears request data", func(t *testing.T) {
		scope := NewScope()
		scope.SetRequestURL("https://api.example.com")

		scope.Reset()

		assert.Equal(t, "", scope.GetRequestURL())
	})
}
