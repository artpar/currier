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

func TestQuoteArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "simple args no quoting needed",
			args:     []string{"-X", "POST", "https://example.com"},
			expected: `-X POST https://example.com`,
		},
		{
			name:     "header with colon and space needs quoting",
			args:     []string{"-H", "Content-Type: application/json", "https://example.com"},
			expected: `-H "Content-Type: application/json" https://example.com`,
		},
		{
			name:     "json data with braces needs quoting",
			args:     []string{"-d", `{"name":"test"}`, "https://example.com"},
			expected: `-d "{\"name\":\"test\"}" https://example.com`,
		},
		{
			name:     "multiple headers",
			args:     []string{"-H", "Authorization: Bearer token", "-H", "Content-Type: application/json", "https://example.com"},
			expected: `-H "Authorization: Bearer token" -H "Content-Type: application/json" https://example.com`,
		},
		{
			name:     "URL with query params",
			args:     []string{"https://example.com/api?foo=bar&baz=qux"},
			expected: `"https://example.com/api?foo=bar&baz=qux"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteArgs(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCurlCommand_QuotedArgsParseCorrectly(t *testing.T) {
	curlImporter := importer.NewCurlImporter()

	// Simulate what happens when CLI receives args from shell
	// The shell has already parsed: currier curl -H "Content-Type: application/json" https://example.com
	// Into args: ["-H", "Content-Type: application/json", "https://example.com"]
	args := []string{"-H", "Content-Type: application/json", "https://example.com/api"}

	// Reconstruct with quoting
	curlCmd := "curl " + quoteArgs(args)

	// This should parse correctly now
	collection, err := curlImporter.Import(context.Background(), []byte(curlCmd))
	require.NoError(t, err)

	requests := collection.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "https://example.com/api", requests[0].URL())
	assert.Equal(t, "application/json", requests[0].Headers()["Content-Type"])
}

func TestCurlCommand_ComplexArgsParseCorrectly(t *testing.T) {
	curlImporter := importer.NewCurlImporter()

	// Simulate: currier curl -X POST -H "Content-Type: application/json" -d '{"name":"test"}' https://httpbin.org/post
	args := []string{"-X", "POST", "-H", "Content-Type: application/json", "-d", `{"name":"test"}`, "https://httpbin.org/post"}

	curlCmd := "curl " + quoteArgs(args)

	collection, err := curlImporter.Import(context.Background(), []byte(curlCmd))
	require.NoError(t, err)

	requests := collection.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "POST", requests[0].Method())
	assert.Equal(t, "https://httpbin.org/post", requests[0].URL())
	assert.Equal(t, "application/json", requests[0].Headers()["Content-Type"])
	assert.Equal(t, `{"name":"test"}`, requests[0].Body())
}

// Additional comprehensive curl parsing tests
func TestCurlParsing_AdditionalFeatures(t *testing.T) {
	curlImporter := importer.NewCurlImporter()

	t.Run("parses URL with query parameters", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl "https://api.example.com/search?q=test&limit=10&offset=0"`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "https://api.example.com/search?q=test&limit=10&offset=0", requests[0].URL())
	})

	t.Run("parses user-agent header", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -A "Mozilla/5.0 (Macintosh)" https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "Mozilla/5.0 (Macintosh)", requests[0].Headers()["User-Agent"])
	})

	t.Run("parses cookie header", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -b "session=abc123; token=xyz" https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "session=abc123; token=xyz", requests[0].Headers()["Cookie"])
	})

	t.Run("parses referer header", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -e "https://google.com" https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "https://google.com", requests[0].Headers()["Referer"])
	})

	t.Run("parses compressed flag", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl --compressed https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "gzip, deflate, br", requests[0].Headers()["Accept-Encoding"])
	})

	t.Run("parses PUT request", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -X PUT -d '{"id":1}' https://example.com/users/1`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "PUT", requests[0].Method())
		assert.Equal(t, `{"id":1}`, requests[0].Body())
	})

	t.Run("parses DELETE request", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -X DELETE https://example.com/users/1`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "DELETE", requests[0].Method())
	})

	t.Run("parses PATCH request", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -X PATCH -d '{"name":"updated"}' https://example.com/users/1`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "PATCH", requests[0].Method())
	})

	t.Run("parses multiple headers", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -H "Authorization: Bearer token123" -H "X-Custom-Header: value" -H "Accept: application/json" https://api.example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		headers := requests[0].Headers()
		assert.Equal(t, "Bearer token123", headers["Authorization"])
		assert.Equal(t, "value", headers["X-Custom-Header"])
		assert.Equal(t, "application/json", headers["Accept"])
	})

	t.Run("parses data-raw flag", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl --data-raw '{"raw":"data"}' https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method())
		assert.Equal(t, `{"raw":"data"}`, requests[0].Body())
	})

	t.Run("parses long form options", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl --request POST --header "Content-Type: application/json" --data '{"test":true}' https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method())
		assert.Equal(t, "application/json", requests[0].Headers()["Content-Type"])
		assert.Equal(t, `{"test":true}`, requests[0].Body())
	})

	t.Run("ignores silent and verbose flags", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -s -S -v https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "https://example.com", requests[0].URL())
	})

	t.Run("ignores follow redirects flag", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -L https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "https://example.com", requests[0].URL())
	})

	t.Run("ignores insecure flag", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -k https://self-signed.example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "https://self-signed.example.com", requests[0].URL())
	})

	t.Run("parses auth with only username", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -u admin https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		auth := requests[0].Auth()
		assert.Equal(t, "basic", auth.Type)
		assert.Equal(t, "admin", auth.Username)
		assert.Equal(t, "", auth.Password)
	})

	t.Run("parses form data with data-urlencode", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl --data-urlencode "name=John Doe" https://example.com`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method())
		assert.Equal(t, "name=John Doe", requests[0].Body())
	})

	t.Run("parses real-world GitHub API example", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl -H "Accept: application/vnd.github.v3+json" -H "Authorization: token ghp_xxxx" https://api.github.com/user`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "GET", requests[0].Method())
		assert.Equal(t, "https://api.github.com/user", requests[0].URL())
		assert.Equal(t, "application/vnd.github.v3+json", requests[0].Headers()["Accept"])
		assert.Equal(t, "token ghp_xxxx", requests[0].Headers()["Authorization"])
	})

	t.Run("parses multiline curl with backslash continuations", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}' \
  https://api.example.com/endpoint`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "POST", requests[0].Method())
		assert.Equal(t, "application/json", requests[0].Headers()["Content-Type"])
		assert.Equal(t, `{"key":"value"}`, requests[0].Body())
		assert.Equal(t, "https://api.example.com/endpoint", requests[0].URL())
	})

	t.Run("generates name from URL path", func(t *testing.T) {
		collection, err := curlImporter.Import(context.Background(),
			[]byte(`curl https://api.example.com/v1/users/profile`))
		require.NoError(t, err)

		requests := collection.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "profile", requests[0].Name())
	})
}
