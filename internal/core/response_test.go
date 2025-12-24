package core

import (
	"io"
	"testing"
	"time"

	"github.com/artpar/currier/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TimingInfo alias for convenience in tests
type TimingInfo = interfaces.TimingInfo

func TestNewResponse(t *testing.T) {
	t.Run("creates response with required fields", func(t *testing.T) {
		resp := NewResponse(
			"req-123",
			"http",
			NewStatus(200, "OK"),
		)
		assert.NotEmpty(t, resp.ID())
		assert.Equal(t, "req-123", resp.RequestID())
		assert.Equal(t, "http", resp.Protocol())
		assert.Equal(t, 200, resp.Status().Code())
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		resp1 := NewResponse("req-1", "http", NewStatus(200, "OK"))
		resp2 := NewResponse("req-2", "http", NewStatus(200, "OK"))
		assert.NotEqual(t, resp1.ID(), resp2.ID())
	})

	t.Run("defaults to empty headers", func(t *testing.T) {
		resp := NewResponse("req-123", "http", NewStatus(200, "OK"))
		assert.Empty(t, resp.Headers().Keys())
	})

	t.Run("defaults to empty body", func(t *testing.T) {
		resp := NewResponse("req-123", "http", NewStatus(200, "OK"))
		assert.True(t, resp.Body().IsEmpty())
	})

	t.Run("defaults to empty metadata", func(t *testing.T) {
		resp := NewResponse("req-123", "http", NewStatus(200, "OK"))
		assert.Empty(t, resp.Metadata())
	})
}

func TestResponse_WithHeaders(t *testing.T) {
	t.Run("adds headers to response", func(t *testing.T) {
		headers := NewHeaders()
		headers.Set("Content-Type", "application/json")
		headers.Set("X-Request-ID", "abc123")

		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithHeaders(headers)

		assert.Equal(t, "application/json", resp.Headers().Get("Content-Type"))
		assert.Equal(t, "abc123", resp.Headers().Get("X-Request-ID"))
	})
}

func TestResponse_WithBody(t *testing.T) {
	t.Run("sets JSON body", func(t *testing.T) {
		body := NewJSONBody(map[string]string{"name": "John"})
		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithBody(body)

		assert.False(t, resp.Body().IsEmpty())
		assert.Equal(t, "json", resp.Body().Type())
	})

	t.Run("sets raw body", func(t *testing.T) {
		body := NewRawBody([]byte("Hello World"), "text/plain")
		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithBody(body)

		assert.Equal(t, "Hello World", resp.Body().String())
		assert.Equal(t, "text/plain", resp.Body().ContentType())
	})

	t.Run("body reader returns correct content", func(t *testing.T) {
		content := []byte(`{"status":"ok"}`)
		body := NewRawBody(content, "application/json")
		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithBody(body)

		reader := resp.Body().Reader()
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})
}

func TestResponse_WithTiming(t *testing.T) {
	t.Run("sets timing info", func(t *testing.T) {
		start := time.Now()
		end := start.Add(150 * time.Millisecond)

		timing := TimingInfo{
			StartTime:       start,
			EndTime:         end,
			DNSLookup:       10 * time.Millisecond,
			TCPConnection:   20 * time.Millisecond,
			TLSHandshake:    30 * time.Millisecond,
			TimeToFirstByte: 50 * time.Millisecond,
			ContentTransfer: 40 * time.Millisecond,
			Total:           150 * time.Millisecond,
		}

		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithTiming(timing)

		assert.Equal(t, start, resp.Timing().StartTime)
		assert.Equal(t, end, resp.Timing().EndTime)
		assert.Equal(t, 150*time.Millisecond, resp.Timing().Total)
		assert.Equal(t, 10*time.Millisecond, resp.Timing().DNSLookup)
	})
}

func TestResponse_WithMetadata(t *testing.T) {
	t.Run("sets metadata", func(t *testing.T) {
		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithMetadata("http_version", "HTTP/2.0").
			WithMetadata("connection_reused", true)

		meta := resp.Metadata()
		assert.Equal(t, "HTTP/2.0", meta["http_version"])
		assert.Equal(t, true, meta["connection_reused"])
	})
}

func TestResponse_Status(t *testing.T) {
	t.Run("returns correct status", func(t *testing.T) {
		resp := NewResponse("req-123", "http", NewStatus(201, "Created"))
		assert.Equal(t, 201, resp.Status().Code())
		assert.Equal(t, "Created", resp.Status().Text())
		assert.True(t, resp.Status().IsSuccess())
	})

	t.Run("error status", func(t *testing.T) {
		resp := NewResponse("req-123", "http", NewStatus(500, "Internal Server Error"))
		assert.True(t, resp.Status().IsError())
		assert.False(t, resp.Status().IsSuccess())
	})
}

func TestResponse_FullBuilder(t *testing.T) {
	t.Run("builds complete response with chained methods", func(t *testing.T) {
		headers := NewHeaders()
		headers.Set("Content-Type", "application/json")

		body := NewJSONBody(map[string]any{
			"id":   1,
			"name": "Test",
		})

		timing := TimingInfo{
			Total: 100 * time.Millisecond,
		}

		resp := NewResponse("req-456", "http", NewStatus(200, "OK")).
			WithHeaders(headers).
			WithBody(body).
			WithTiming(timing).
			WithMetadata("cached", false)

		// Verify all fields
		assert.NotEmpty(t, resp.ID())
		assert.Equal(t, "req-456", resp.RequestID())
		assert.Equal(t, "http", resp.Protocol())
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "application/json", resp.Headers().Get("Content-Type"))
		assert.False(t, resp.Body().IsEmpty())
		assert.Equal(t, 100*time.Millisecond, resp.Timing().Total)
		assert.Equal(t, false, resp.Metadata()["cached"])
	})
}

func TestTimingInfo(t *testing.T) {
	t.Run("zero value timing", func(t *testing.T) {
		var timing TimingInfo
		assert.True(t, timing.StartTime.IsZero())
		assert.Equal(t, time.Duration(0), timing.Total)
	})

	t.Run("calculates total from start and end", func(t *testing.T) {
		start := time.Now()
		timing := TimingInfo{
			StartTime: start,
			EndTime:   start.Add(250 * time.Millisecond),
			Total:     250 * time.Millisecond,
		}
		assert.Equal(t, 250*time.Millisecond, timing.Total)
	})
}

func TestResponseBodyParsing(t *testing.T) {
	t.Run("parses JSON response body", func(t *testing.T) {
		jsonData := `{"users":[{"id":1,"name":"John"},{"id":2,"name":"Jane"}]}`
		body := NewRawBody([]byte(jsonData), "application/json")
		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithBody(body)

		parsed, err := resp.Body().JSON()
		require.NoError(t, err)

		m := parsed.(map[string]any)
		users := m["users"].([]any)
		assert.Len(t, users, 2)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		body := NewRawBody([]byte("not valid json"), "application/json")
		resp := NewResponse("req-123", "http", NewStatus(200, "OK")).
			WithBody(body)

		_, err := resp.Body().JSON()
		assert.Error(t, err)
	})
}
