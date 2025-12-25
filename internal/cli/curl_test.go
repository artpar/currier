package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/artpar/currier/internal/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCurlCommand(t *testing.T) {
	t.Run("creates curl command", func(t *testing.T) {
		cmd := NewCurlCommand()
		assert.NotNil(t, cmd)
		assert.Equal(t, "curl [curl command arguments...]", cmd.Use)
		assert.Contains(t, cmd.Short, "curl")
	})

	t.Run("has DisableFlagParsing enabled", func(t *testing.T) {
		cmd := NewCurlCommand()
		assert.True(t, cmd.DisableFlagParsing)
	})

	t.Run("has examples in long description", func(t *testing.T) {
		cmd := NewCurlCommand()
		assert.Contains(t, cmd.Long, "Examples:")
		assert.Contains(t, cmd.Long, "httpbin.org")
	})
}

func TestCurlCommand_Errors(t *testing.T) {
	t.Run("returns error when no arguments provided", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cmd := NewCurlCommand()
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no curl arguments provided")
	})

	t.Run("returns error for invalid curl command", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cmd := NewCurlCommand()
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		// No URL provided - just flags
		cmd.SetArgs([]string{"-X", "POST"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse curl command")
	})
}

func TestCurlCommand_InRootCommand(t *testing.T) {
	t.Run("curl subcommand exists in root", func(t *testing.T) {
		cmd := NewRootCommand("1.0.0")
		curlCmd, _, err := cmd.Find([]string{"curl"})
		require.NoError(t, err)
		assert.Contains(t, curlCmd.Use, "curl")
	})

	t.Run("curl appears in help output", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := NewRootCommand("1.0.0")
		cmd.SetOut(out)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "curl")
	})
}

func TestCurlParsing(t *testing.T) {
	// These tests verify the curl importer works correctly
	// which is what the curl command uses internally
	curlImporter := importer.NewCurlImporter()

	t.Run("parses simple GET request", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte("curl https://example.com/api"))
		require.NoError(t, err)
		require.NotNil(t, collection)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "GET", requests[0].Method())
		assert.Equal(t, "https://example.com/api", requests[0].URL())
	})

	t.Run("parses POST request with method", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte("curl -X POST https://example.com/api"))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method())
	})

	t.Run("parses request with headers", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -H "Content-Type: application/json" -H "Authorization: Bearer token" https://example.com/api`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)

		headers := requests[0].Headers()
		assert.Equal(t, "application/json", headers["Content-Type"])
		assert.Equal(t, "Bearer token", headers["Authorization"])
	})

	t.Run("parses request with data", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -d '{"name":"test"}' https://example.com/api`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method()) // -d implies POST
		assert.Equal(t, `{"name":"test"}`, requests[0].Body())
	})

	t.Run("parses request with basic auth", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -u admin:secret https://example.com/api`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)

		auth := requests[0].Auth()
		assert.Equal(t, "basic", auth.Type)
		assert.Equal(t, "admin", auth.Username)
		assert.Equal(t, "secret", auth.Password)
	})

	t.Run("parses --json flag", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl --json '{"test":true}' https://example.com/api`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method())
		assert.Equal(t, `{"test":true}`, requests[0].Body())

		headers := requests[0].Headers()
		assert.Equal(t, "application/json", headers["Content-Type"])
		assert.Equal(t, "application/json", headers["Accept"])
	})

	t.Run("parses HEAD request", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -I https://example.com/api`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "HEAD", requests[0].Method())
	})

	t.Run("handles line continuations", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte("curl \\\n  -X POST \\\n  https://example.com/api"))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method())
		assert.Equal(t, "https://example.com/api", requests[0].URL())
	})

	t.Run("returns error for missing URL", func(t *testing.T) {
		_, err := curlImporter.Import(context.Background(),
			[]byte("curl -X POST -H 'Content-Type: application/json'"))
		assert.Error(t, err)
	})

	t.Run("returns error for non-curl command", func(t *testing.T) {
		_, err := curlImporter.Import(context.Background(),
			[]byte("wget https://example.com"))
		assert.Error(t, err)
	})
}
