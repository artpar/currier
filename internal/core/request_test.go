package core

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequest(t *testing.T) {
	t.Run("creates request with valid HTTP URL", func(t *testing.T) {
		req, err := NewRequest("http", "GET", "https://api.example.com/users")
		require.NoError(t, err)
		assert.NotEmpty(t, req.ID())
		assert.Equal(t, "http", req.Protocol())
		assert.Equal(t, "GET", req.Method())
		assert.Equal(t, "https://api.example.com/users", req.Endpoint())
	})

	t.Run("creates request with POST method", func(t *testing.T) {
		req, err := NewRequest("http", "POST", "https://api.example.com/users")
		require.NoError(t, err)
		assert.Equal(t, "POST", req.Method())
	})

	t.Run("returns error for empty method", func(t *testing.T) {
		_, err := NewRequest("http", "", "https://api.example.com/users")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "method")
	})

	t.Run("returns error for empty endpoint", func(t *testing.T) {
		_, err := NewRequest("http", "GET", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint")
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		req1, _ := NewRequest("http", "GET", "https://example.com")
		req2, _ := NewRequest("http", "GET", "https://example.com")
		assert.NotEqual(t, req1.ID(), req2.ID())
	})

	t.Run("supports all HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
		for _, method := range methods {
			req, err := NewRequest("http", method, "https://example.com")
			require.NoError(t, err)
			assert.Equal(t, method, req.Method())
		}
	})
}

func TestRequest_Headers(t *testing.T) {
	t.Run("returns empty headers for new request", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		headers := req.Headers()
		assert.Empty(t, headers.Keys())
	})

	t.Run("sets and gets header", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		req.SetHeader("Content-Type", "application/json")
		assert.Equal(t, "application/json", req.Headers().Get("Content-Type"))
	})

	t.Run("header keys are case-insensitive", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		req.SetHeader("Content-Type", "application/json")
		assert.Equal(t, "application/json", req.Headers().Get("content-type"))
		assert.Equal(t, "application/json", req.Headers().Get("CONTENT-TYPE"))
	})

	t.Run("sets multiple headers", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		req.SetHeader("Content-Type", "application/json")
		req.SetHeader("Authorization", "Bearer token123")
		req.SetHeader("Accept", "application/json")

		headers := req.Headers()
		assert.Equal(t, "application/json", headers.Get("Content-Type"))
		assert.Equal(t, "Bearer token123", headers.Get("Authorization"))
		assert.Equal(t, "application/json", headers.Get("Accept"))
	})
}

func TestRequest_Body(t *testing.T) {
	t.Run("returns empty body for new request", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		body := req.Body()
		assert.True(t, body.IsEmpty())
		assert.Equal(t, int64(0), body.Size())
	})

	t.Run("sets JSON body", func(t *testing.T) {
		req, _ := NewRequest("http", "POST", "https://example.com")
		body := NewJSONBody(map[string]string{"name": "John"})
		req.SetBody(body)

		assert.False(t, req.Body().IsEmpty())
		assert.Equal(t, "json", req.Body().Type())
		assert.Equal(t, "application/json", req.Body().ContentType())
	})

	t.Run("sets raw body", func(t *testing.T) {
		req, _ := NewRequest("http", "POST", "https://example.com")
		body := NewRawBody([]byte("hello world"), "text/plain")
		req.SetBody(body)

		assert.Equal(t, "hello world", req.Body().String())
		assert.Equal(t, "text/plain", req.Body().ContentType())
	})

	t.Run("body reader returns correct content", func(t *testing.T) {
		req, _ := NewRequest("http", "POST", "https://example.com")
		content := []byte(`{"key":"value"}`)
		body := NewRawBody(content, "application/json")
		req.SetBody(body)

		reader := req.Body().Reader()
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})
}

func TestRequest_Metadata(t *testing.T) {
	t.Run("returns empty metadata for new request", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		assert.Empty(t, req.Metadata())
	})

	t.Run("sets and gets metadata", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		req.SetMetadata("timeout", 30000)
		req.SetMetadata("follow_redirects", true)

		meta := req.Metadata()
		assert.Equal(t, 30000, meta["timeout"])
		assert.Equal(t, true, meta["follow_redirects"])
	})
}

func TestRequest_Clone(t *testing.T) {
	t.Run("creates deep copy", func(t *testing.T) {
		original, _ := NewRequest("http", "POST", "https://example.com")
		original.SetHeader("Authorization", "Bearer token")
		original.SetBody(NewRawBody([]byte("body"), "text/plain"))
		original.SetMetadata("key", "value")

		clone := original.Clone()

		// Verify clone has same values
		assert.Equal(t, original.Protocol(), clone.Protocol())
		assert.Equal(t, original.Method(), clone.Method())
		assert.Equal(t, original.Endpoint(), clone.Endpoint())
		assert.Equal(t, original.Headers().Get("Authorization"), clone.Headers().Get("Authorization"))

		// Verify clone has different ID
		assert.NotEqual(t, original.ID(), clone.ID())

		// Verify modifications don't affect original
		clone.SetHeader("Authorization", "Bearer different")
		assert.Equal(t, "Bearer token", original.Headers().Get("Authorization"))
	})
}

func TestRequest_Validate(t *testing.T) {
	t.Run("valid request passes validation", func(t *testing.T) {
		req, _ := NewRequest("http", "GET", "https://example.com")
		assert.NoError(t, req.Validate())
	})
}

func TestHeaders(t *testing.T) {
	t.Run("creates empty headers", func(t *testing.T) {
		h := NewHeaders()
		assert.Empty(t, h.Keys())
	})

	t.Run("Set replaces existing value", func(t *testing.T) {
		h := NewHeaders()
		h.Set("Key", "value1")
		h.Set("Key", "value2")
		assert.Equal(t, "value2", h.Get("Key"))
		assert.Len(t, h.GetAll("Key"), 1)
	})

	t.Run("Add appends value", func(t *testing.T) {
		h := NewHeaders()
		h.Add("Key", "value1")
		h.Add("Key", "value2")
		assert.Equal(t, "value1", h.Get("Key")) // Get returns first
		assert.Len(t, h.GetAll("Key"), 2)
	})

	t.Run("Del removes all values", func(t *testing.T) {
		h := NewHeaders()
		h.Add("Key", "value1")
		h.Add("Key", "value2")
		h.Del("Key")
		assert.Empty(t, h.Get("Key"))
		assert.Empty(t, h.GetAll("Key"))
	})

	t.Run("Keys returns all header names", func(t *testing.T) {
		h := NewHeaders()
		h.Set("Content-Type", "application/json")
		h.Set("Authorization", "Bearer token")
		keys := h.Keys()
		assert.Contains(t, keys, "Content-Type")
		assert.Contains(t, keys, "Authorization")
	})

	t.Run("Clone creates independent copy", func(t *testing.T) {
		h := NewHeaders()
		h.Set("Key", "value")
		clone := h.Clone()
		clone.Set("Key", "different")
		assert.Equal(t, "value", h.Get("Key"))
	})

	t.Run("ToMap returns map representation", func(t *testing.T) {
		h := NewHeaders()
		h.Add("Key", "value1")
		h.Add("Key", "value2")
		m := h.ToMap()
		assert.Equal(t, []string{"value1", "value2"}, m["Key"])
	})
}

func TestBody(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		b := NewEmptyBody()
		assert.True(t, b.IsEmpty())
		assert.Equal(t, int64(0), b.Size())
		assert.Equal(t, "", b.String())
	})

	t.Run("JSON body from map", func(t *testing.T) {
		data := map[string]any{
			"name":  "John",
			"age":   30,
			"admin": true,
		}
		b := NewJSONBody(data)
		assert.Equal(t, "json", b.Type())
		assert.Equal(t, "application/json", b.ContentType())
		assert.False(t, b.IsEmpty())

		// Parse back
		parsed, err := b.JSON()
		require.NoError(t, err)
		m := parsed.(map[string]any)
		assert.Equal(t, "John", m["name"])
	})

	t.Run("JSON body from struct", func(t *testing.T) {
		type User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		user := User{Name: "Jane", Age: 25}
		b := NewJSONBody(user)

		parsed, err := b.JSON()
		require.NoError(t, err)
		m := parsed.(map[string]any)
		assert.Equal(t, "Jane", m["name"])
		assert.Equal(t, float64(25), m["age"]) // JSON numbers are float64
	})

	t.Run("raw body", func(t *testing.T) {
		content := []byte("raw content here")
		b := NewRawBody(content, "text/plain")
		assert.Equal(t, "raw", b.Type())
		assert.Equal(t, "text/plain", b.ContentType())
		assert.Equal(t, content, b.Bytes())
	})

	t.Run("body reader can be read multiple times", func(t *testing.T) {
		content := []byte("content")
		b := NewRawBody(content, "text/plain")

		// First read
		r1 := b.Reader()
		d1, _ := io.ReadAll(r1)
		assert.Equal(t, content, d1)

		// Second read
		r2 := b.Reader()
		d2, _ := io.ReadAll(r2)
		assert.Equal(t, content, d2)
	})
}

func TestStatus(t *testing.T) {
	t.Run("200 OK is success", func(t *testing.T) {
		s := NewStatus(200, "OK")
		assert.Equal(t, 200, s.Code())
		assert.Equal(t, "OK", s.Text())
		assert.True(t, s.IsSuccess())
		assert.False(t, s.IsError())
	})

	t.Run("201 Created is success", func(t *testing.T) {
		s := NewStatus(201, "Created")
		assert.True(t, s.IsSuccess())
	})

	t.Run("400 Bad Request is error", func(t *testing.T) {
		s := NewStatus(400, "Bad Request")
		assert.False(t, s.IsSuccess())
		assert.True(t, s.IsError())
	})

	t.Run("500 Internal Server Error is error", func(t *testing.T) {
		s := NewStatus(500, "Internal Server Error")
		assert.False(t, s.IsSuccess())
		assert.True(t, s.IsError())
	})

	t.Run("3xx is not error", func(t *testing.T) {
		s := NewStatus(301, "Moved Permanently")
		assert.False(t, s.IsSuccess()) // 3xx is not 2xx
		assert.False(t, s.IsError())   // 3xx is not 4xx/5xx
	})
}

// Benchmark tests
func BenchmarkNewRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewRequest("http", "GET", "https://example.com")
	}
}

func BenchmarkRequest_Clone(b *testing.B) {
	req, _ := NewRequest("http", "POST", "https://example.com")
	req.SetHeader("Authorization", "Bearer token")
	req.SetBody(NewRawBody(bytes.Repeat([]byte("x"), 1000), "text/plain"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Clone()
	}
}

func BenchmarkHeaders_Get(b *testing.B) {
	h := NewHeaders()
	for i := 0; i < 20; i++ {
		h.Set("Header-"+string(rune('A'+i)), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Get("Header-J")
	}
}

func BenchmarkBody_JSON(b *testing.B) {
	data := map[string]any{
		"users": []map[string]any{
			{"name": "John", "age": 30},
			{"name": "Jane", "age": 25},
		},
	}
	body := NewJSONBody(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = body.JSON()
	}
}
