package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artpar/currier/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIIntegration tests the CLI commands with real HTTP servers
func TestCLIIntegration(t *testing.T) {
	t.Run("send command with real server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Hello from server",
			})
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"GET", server.URL})

		err := cmd.Execute()
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "200")
		assert.Contains(t, output, "Hello from server")
	})

	t.Run("send command with headers", func(t *testing.T) {
		var receivedHeaders http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{
			"GET", server.URL,
			"--header", "Authorization:Bearer token123",
			"--header", "X-Custom:custom-value",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		assert.Equal(t, "Bearer token123", receivedHeaders.Get("Authorization"))
		assert.Equal(t, "custom-value", receivedHeaders.Get("X-Custom"))
	})

	t.Run("send command with POST body", func(t *testing.T) {
		var receivedBody map[string]any

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{
			"POST", server.URL,
			"--body", `{"name":"test","value":123}`,
			"--header", "Content-Type:application/json",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		assert.Equal(t, "test", receivedBody["name"])
		assert.Equal(t, float64(123), receivedBody["value"])
	})

	t.Run("send command JSON output mode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-ID", "req-123")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"result": "success"})
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"GET", server.URL, "--json"})

		err := cmd.Execute()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(out.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, float64(200), result["status"])
		assert.Contains(t, result["body"], "success")

		headers := result["headers"].(map[string]any)
		assert.Contains(t, headers, "X-Request-Id")
	})

	t.Run("send command handles server errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"GET", server.URL})

		err := cmd.Execute()
		require.NoError(t, err) // CLI shouldn't error on HTTP errors

		output := out.String()
		assert.Contains(t, output, "500")
	})

	t.Run("root command lists subcommands", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := cli.NewRootCommand("test-version")
		cmd.SetOut(out)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "send")
		assert.Contains(t, output, "currier")
	})

	t.Run("version flag works", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := cli.NewRootCommand("1.2.3")
		cmd.SetOut(out)
		cmd.SetArgs([]string{"--version"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "1.2.3")
	})
}

// TestCLIIntegration_ErrorCases tests CLI error handling
func TestCLIIntegration_ErrorCases(t *testing.T) {
	t.Run("handles connection refused", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		cmd.SetArgs([]string{"GET", "http://localhost:59999/nonexistent"})

		err := cmd.Execute()
		assert.Error(t, err)
	})

	t.Run("handles invalid method", func(t *testing.T) {
		// The command accepts any method string, validation happens at HTTP layer
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"INVALID", server.URL})

		err := cmd.Execute()
		// Should succeed but get 405 from server
		require.NoError(t, err)
	})

	t.Run("requires method and URL arguments", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		cmd.SetArgs([]string{}) // No arguments

		err := cmd.Execute()
		assert.Error(t, err)
	})

	t.Run("requires URL argument", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cmd := cli.NewSendCommand()
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		cmd.SetArgs([]string{"GET"}) // Only method, no URL

		err := cmd.Execute()
		assert.Error(t, err)
	})
}

// TestCLIIntegration_AllMethods tests all HTTP methods via CLI
func TestCLIIntegration_AllMethods(t *testing.T) {
	methods := []struct {
		method         string
		expectedStatus int
	}{
		{"GET", 200},
		{"POST", 201},
		{"PUT", 200},
		{"PATCH", 200},
		{"DELETE", 204},
		{"HEAD", 200},
		{"OPTIONS", 200},
	}

	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.method, r.Method)
				w.WriteHeader(tc.expectedStatus)
			}))
			defer server.Close()

			out := &bytes.Buffer{}
			cmd := cli.NewSendCommand()
			cmd.SetOut(out)
			cmd.SetArgs([]string{tc.method, server.URL})

			err := cmd.Execute()
			require.NoError(t, err)
		})
	}
}
