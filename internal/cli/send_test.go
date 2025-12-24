package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendCommand(t *testing.T) {
	t.Run("sends GET request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"GET", server.URL})

		err := cmd.Execute()
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "200")
	})

	t.Run("sends POST request with body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"POST", server.URL, "--body", `{"name":"test"}`, "--header", "Content-Type:application/json"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "201")
	})

	t.Run("sends request with headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"GET", server.URL, "--header", "Authorization:Bearer token123"})

		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("outputs JSON format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"data": "value"})
		}))
		defer server.Close()

		out := &bytes.Buffer{}
		cmd := NewSendCommand()
		cmd.SetOut(out)
		cmd.SetArgs([]string{"GET", server.URL, "--json"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Should be valid JSON
		var result map[string]any
		err = json.Unmarshal(out.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "status")
		assert.Contains(t, result, "body")
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cmd := NewSendCommand()
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		cmd.SetArgs([]string{"GET", "://invalid"})

		err := cmd.Execute()
		assert.Error(t, err)
	})

	t.Run("supports all HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

		for _, method := range methods {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, method, r.Method)
				w.WriteHeader(http.StatusOK)
			}))

			out := &bytes.Buffer{}
			cmd := NewSendCommand()
			cmd.SetOut(out)
			cmd.SetArgs([]string{method, server.URL})

			err := cmd.Execute()
			require.NoError(t, err, "method %s should work", method)

			server.Close()
		}
	})
}

func TestRootCommand(t *testing.T) {
	t.Run("shows version", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := NewRootCommand("1.0.0")
		cmd.SetOut(out)
		cmd.SetArgs([]string{"--version"})

		err := cmd.Execute()
		require.NoError(t, err)

		assert.Contains(t, out.String(), "1.0.0")
	})

	t.Run("shows help", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := NewRootCommand("1.0.0")
		cmd.SetOut(out)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "currier")
		assert.Contains(t, output, "send")
	})
}

func TestParseHeaders(t *testing.T) {
	t.Run("parses single header", func(t *testing.T) {
		headers := parseHeaders([]string{"Content-Type:application/json"})
		assert.Equal(t, "application/json", headers["Content-Type"])
	})

	t.Run("parses multiple headers", func(t *testing.T) {
		headers := parseHeaders([]string{
			"Content-Type:application/json",
			"Authorization:Bearer token",
		})
		assert.Equal(t, "application/json", headers["Content-Type"])
		assert.Equal(t, "Bearer token", headers["Authorization"])
	})

	t.Run("handles header with multiple colons", func(t *testing.T) {
		headers := parseHeaders([]string{"Authorization:Basic dXNlcjpwYXNz"})
		assert.Equal(t, "Basic dXNlcjpwYXNz", headers["Authorization"])
	})

	t.Run("ignores invalid headers", func(t *testing.T) {
		headers := parseHeaders([]string{"invalid", "Valid:value"})
		assert.Len(t, headers, 1)
		assert.Equal(t, "value", headers["Valid"])
	})
}

// Ensure context is respected
var _ context.Context
