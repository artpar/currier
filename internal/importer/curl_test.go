package importer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurlImporter_Name(t *testing.T) {
	imp := NewCurlImporter()
	assert.Equal(t, "curl command", imp.Name())
}

func TestCurlImporter_Format(t *testing.T) {
	imp := NewCurlImporter()
	assert.Equal(t, FormatCurl, imp.Format())
}

func TestCurlImporter_FileExtensions(t *testing.T) {
	imp := NewCurlImporter()
	extensions := imp.FileExtensions()
	assert.Contains(t, extensions, ".sh")
	assert.Contains(t, extensions, ".curl")
	assert.Contains(t, extensions, ".txt")
}

func TestCurlImporter_DetectFormat(t *testing.T) {
	imp := NewCurlImporter()

	t.Run("detects curl command", func(t *testing.T) {
		assert.True(t, imp.DetectFormat([]byte("curl https://example.com")))
		assert.True(t, imp.DetectFormat([]byte("curl -X GET https://example.com")))
		assert.True(t, imp.DetectFormat([]byte("  curl https://example.com")))
	})

	t.Run("rejects non-curl", func(t *testing.T) {
		assert.False(t, imp.DetectFormat([]byte("wget https://example.com")))
		assert.False(t, imp.DetectFormat([]byte(`{"openapi": "3.0.0"}`)))
	})
}

func TestCurlImporter_Import_SimpleGET(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	content := []byte(`curl https://api.example.com/users`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)

	req := requests[0]
	assert.Equal(t, "GET", req.Method())
	assert.Equal(t, "https://api.example.com/users", req.URL())
	assert.Equal(t, "users", req.Name())
}

func TestCurlImporter_Import_ExplicitMethod(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	t.Run("POST with -X", func(t *testing.T) {
		content := []byte(`curl -X POST https://api.example.com/users`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "POST", req.Method())
	})

	t.Run("DELETE with --request", func(t *testing.T) {
		content := []byte(`curl --request DELETE https://api.example.com/users/1`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "DELETE", req.Method())
	})

	t.Run("HEAD with -I", func(t *testing.T) {
		content := []byte(`curl -I https://api.example.com/`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "HEAD", req.Method())
	})
}

func TestCurlImporter_Import_Headers(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	t.Run("single header", func(t *testing.T) {
		content := []byte(`curl -H "Authorization: Bearer token123" https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "Bearer token123", req.GetHeader("Authorization"))
	})

	t.Run("multiple headers", func(t *testing.T) {
		content := []byte(`curl -H "Accept: application/json" -H "Content-Type: application/json" https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "application/json", req.GetHeader("Accept"))
		assert.Equal(t, "application/json", req.GetHeader("Content-Type"))
	})

	t.Run("--header long form", func(t *testing.T) {
		content := []byte(`curl --header "X-Custom: value" https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "value", req.GetHeader("X-Custom"))
	})
}

func TestCurlImporter_Import_Data(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	t.Run("-d sets POST and body", func(t *testing.T) {
		content := []byte(`curl -d '{"name":"John"}' https://api.example.com/users`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "POST", req.Method())
		assert.Equal(t, `{"name":"John"}`, req.Body())
	})

	t.Run("--data-raw", func(t *testing.T) {
		content := []byte(`curl --data-raw 'test body' https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "test body", req.Body())
	})

	t.Run("--json sets content-type", func(t *testing.T) {
		content := []byte(`curl --json '{"key":"value"}' https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, `{"key":"value"}`, req.Body())
		assert.Equal(t, "application/json", req.GetHeader("Content-Type"))
		assert.Equal(t, "application/json", req.GetHeader("Accept"))
	})

	t.Run("explicit method overrides data default", func(t *testing.T) {
		content := []byte(`curl -X PUT -d '{"name":"John"}' https://api.example.com/users/1`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "PUT", req.Method())
	})
}

func TestCurlImporter_Import_BasicAuth(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	t.Run("-u with password", func(t *testing.T) {
		content := []byte(`curl -u username:password https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		auth := req.Auth()
		require.NotNil(t, auth)
		assert.Equal(t, "basic", auth.Type)
		assert.Equal(t, "username", auth.Username)
		assert.Equal(t, "password", auth.Password)
	})

	t.Run("-u without password", func(t *testing.T) {
		content := []byte(`curl -u username https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		auth := req.Auth()
		require.NotNil(t, auth)
		assert.Equal(t, "username", auth.Username)
		assert.Empty(t, auth.Password)
	})

	t.Run("--user long form", func(t *testing.T) {
		content := []byte(`curl --user admin:secret https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		auth := req.Auth()
		require.NotNil(t, auth)
		assert.Equal(t, "admin", auth.Username)
		assert.Equal(t, "secret", auth.Password)
	})
}

func TestCurlImporter_Import_SpecialOptions(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	t.Run("--compressed adds encoding header", func(t *testing.T) {
		content := []byte(`curl --compressed https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "gzip, deflate, br", req.GetHeader("Accept-Encoding"))
	})

	t.Run("-A sets user agent", func(t *testing.T) {
		content := []byte(`curl -A "MyAgent/1.0" https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "MyAgent/1.0", req.GetHeader("User-Agent"))
	})

	t.Run("-b sets cookie", func(t *testing.T) {
		content := []byte(`curl -b "session=abc123" https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "session=abc123", req.GetHeader("Cookie"))
	})

	t.Run("-e sets referer", func(t *testing.T) {
		content := []byte(`curl -e "https://referrer.com" https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "https://referrer.com", req.GetHeader("Referer"))
	})
}

func TestCurlImporter_Import_LineContinuation(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	content := []byte(`curl \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"John"}' \
  https://api.example.com/users`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Equal(t, "POST", req.Method())
	assert.Equal(t, "https://api.example.com/users", req.URL())
	assert.Equal(t, "application/json", req.GetHeader("Content-Type"))
	assert.Equal(t, `{"name":"John"}`, req.Body())
}

func TestCurlImporter_Import_QuotedStrings(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	t.Run("double quotes", func(t *testing.T) {
		content := []byte(`curl -H "Authorization: Bearer token with spaces" https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, "Bearer token with spaces", req.GetHeader("Authorization"))
	})

	t.Run("single quotes", func(t *testing.T) {
		content := []byte(`curl -d '{"key": "value with spaces"}' https://api.example.com`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		req := coll.Requests()[0]
		assert.Equal(t, `{"key": "value with spaces"}`, req.Body())
	})
}

func TestCurlImporter_Import_NameGeneration(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	tests := []struct {
		url      string
		expected string
	}{
		{"https://api.example.com/users", "users"},
		{"https://api.example.com/users/123", "123"},
		{"https://api.example.com/", "api.example.com"},
		{"https://api.example.com", "api.example.com"},
		{"https://api.example.com:8080/data", "data"},
		{"https://api.example.com/users?page=1", "users"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			content := []byte(`curl ` + tc.url)
			coll, err := imp.Import(ctx, content)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, coll.Requests()[0].Name())
		})
	}
}

func TestCurlImporter_Import_InvalidCommand(t *testing.T) {
	imp := NewCurlImporter()
	ctx := context.Background()

	t.Run("not curl command", func(t *testing.T) {
		content := []byte(`wget https://example.com`)
		_, err := imp.Import(ctx, content)
		assert.ErrorIs(t, err, ErrParseError)
	})

	t.Run("missing URL", func(t *testing.T) {
		content := []byte(`curl -X GET`)
		_, err := imp.Import(ctx, content)
		assert.ErrorIs(t, err, ErrParseError)
	})
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    `curl https://example.com`,
			expected: []string{"curl", "https://example.com"},
		},
		{
			input:    `curl -H "Content-Type: application/json" https://example.com`,
			expected: []string{"curl", "-H", "Content-Type: application/json", "https://example.com"},
		},
		{
			input:    `curl -d '{"key":"value"}' https://example.com`,
			expected: []string{"curl", "-d", `{"key":"value"}`, "https://example.com"},
		},
		{
			input:    `curl -u user:pass https://example.com`,
			expected: []string{"curl", "-u", "user:pass", "https://example.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := tokenize(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
