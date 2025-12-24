package cli_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/artpar/currier/e2e/harness"
	"github.com/artpar/currier/e2e/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLI_SendCommand(t *testing.T) {
	handlers := testserver.Handlers{}

	h := harness.New(t, harness.Config{
		ServerHandlers: map[string]http.HandlerFunc{
			"/api/users": handlers.JSON(200, map[string]interface{}{
				"message": "Hello from server",
				"users":   []string{"alice", "bob"},
			}),
			"/api/error": handlers.Error(500, "Internal Server Error"),
			"/api/created": handlers.JSON(201, map[string]string{
				"id": "123",
			}),
			"/api/echo": handlers.Echo(),
		},
		Timeout:   5 * time.Second,
		GoldenDir: "../golden/cli",
	})

	t.Run("GET request returns 200", func(t *testing.T) {
		result, err := h.CLI().Send("GET", h.ServerURL()+"/api/users")

		require.NoError(t, err)

		assert := harness.NewAssertions(t)
		assert.OutputContains(result.Stdout, "200", "Hello from server")
		assert.NoError(result.Stdout)
	})

	t.Run("GET request with JSON output mode", func(t *testing.T) {
		result, err := h.CLI().SendJSON("GET", h.ServerURL()+"/api/users")

		require.NoError(t, err)
		assert.Contains(t, result.Stdout, "message")
	})

	t.Run("POST request with body", func(t *testing.T) {
		result, err := h.CLI().SendWithBody(
			"POST",
			h.ServerURL()+"/api/echo",
			`{"name":"test"}`,
			map[string]string{"Content-Type": "application/json"},
		)

		require.NoError(t, err)

		assert := harness.NewAssertions(t)
		assert.StatusCode(result.Stdout, 200)
	})

	t.Run("request with custom headers", func(t *testing.T) {
		result, err := h.CLI().SendWithHeaders(
			"GET",
			h.ServerURL()+"/api/echo",
			map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "custom-value",
			},
		)

		require.NoError(t, err)
		assert.Contains(t, result.Stdout, "Authorization")
	})

	t.Run("handles server error gracefully", func(t *testing.T) {
		result, err := h.CLI().Send("GET", h.ServerURL()+"/api/error")

		// CLI should not error on HTTP errors
		require.NoError(t, err)

		assert := harness.NewAssertions(t)
		assert.StatusCode(result.Stdout, 500)
	})

	t.Run("PUT request", func(t *testing.T) {
		result, err := h.CLI().Send("PUT", h.ServerURL()+"/api/echo")

		require.NoError(t, err)
		assert.Contains(t, result.Stdout, "PUT")
	})

	t.Run("DELETE request", func(t *testing.T) {
		result, err := h.CLI().Send("DELETE", h.ServerURL()+"/api/echo")

		require.NoError(t, err)
		assert.Contains(t, result.Stdout, "DELETE")
	})
}

func TestCLI_SendCommand_AllMethods(t *testing.T) {
	handlers := testserver.Handlers{}

	h := harness.New(t, harness.Config{
		ServerHandlers: map[string]http.HandlerFunc{
			"/api/test": handlers.Echo(),
		},
		Timeout: 5 * time.Second,
	})

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method+" request", func(t *testing.T) {
			result, err := h.CLI().Send(method, h.ServerURL()+"/api/test")

			require.NoError(t, err)
			// HEAD doesn't return body, others should show the method
			if method != "HEAD" {
				assert := harness.NewAssertions(t)
				assert.StatusCode(result.Stdout, 200)
			}
		})
	}
}

func TestCLI_SendCommand_Errors(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		h := harness.New(t, harness.Config{
			Timeout: 5 * time.Second,
		})

		result, err := h.CLI().Send("GET", "http://localhost:59999/nonexistent")

		// Should return an error for connection refused
		if err == nil {
			// If no error, check that output indicates failure
			assert.Contains(t, result.Stdout+result.Stderr, "refused")
		}
	})

	t.Run("missing arguments", func(t *testing.T) {
		h := harness.New(t, harness.Config{
			Timeout: 5 * time.Second,
		})

		_, err := h.CLI().Run("send")

		assert.Error(t, err)
	})

	t.Run("invalid URL", func(t *testing.T) {
		h := harness.New(t, harness.Config{
			Timeout: 5 * time.Second,
		})

		_, err := h.CLI().Send("GET", "not-a-valid-url")

		// Should error or show error message
		assert.Error(t, err)
	})
}
