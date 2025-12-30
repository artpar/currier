package components

import (
	"testing"

	"github.com/artpar/currier/internal/core"
	"github.com/stretchr/testify/assert"
)

// Tests for pure functions - trivial input → output, no mocks needed.

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		delta    int
		count    int
		expected int
	}{
		{"move down from start", 0, 1, 10, 1},
		{"move down in middle", 5, 1, 10, 6},
		{"move up in middle", 5, -1, 10, 4},
		{"clamp at end", 9, 1, 10, 9},
		{"clamp at start", 0, -1, 10, 0},
		{"large jump down", 5, 100, 10, 9},
		{"large jump up", 5, -100, 10, 0},
		{"empty list", 0, 1, 0, 0},
		{"single item list", 0, 1, 1, 0},
		{"move multiple down", 2, 3, 10, 5},
		{"move multiple up", 5, -3, 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MoveCursor(tt.cursor, tt.delta, tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdjustOffset(t *testing.T) {
	tests := []struct {
		name          string
		cursor        int
		offset        int
		visibleHeight int
		expected      int
	}{
		{"cursor in view", 5, 3, 10, 3},
		{"cursor above view - scroll up", 2, 5, 10, 2},
		{"cursor below view - scroll down", 15, 3, 10, 6},
		{"cursor at top of view", 3, 3, 10, 3},
		{"cursor at bottom of view", 12, 3, 10, 3},
		{"cursor just below view", 13, 3, 10, 4},
		{"zero visible height", 5, 0, 0, 5},
		{"negative visible height", 5, 0, -1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AdjustOffset(tt.cursor, tt.offset, tt.visibleHeight)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToggleExpand(t *testing.T) {
	t.Run("expand new item", func(t *testing.T) {
		original := map[string]bool{"a": true, "b": false}

		result := ToggleExpand(original, "c", true)

		// Original unchanged (immutability)
		assert.Len(t, original, 2)
		_, exists := original["c"]
		assert.False(t, exists, "original should not have new key")

		// Result has new value
		assert.True(t, result["c"])
		assert.True(t, result["a"])
		assert.False(t, result["b"])
	})

	t.Run("collapse existing item", func(t *testing.T) {
		original := map[string]bool{"a": true, "b": true}

		result := ToggleExpand(original, "a", false)

		// Original unchanged
		assert.True(t, original["a"], "original should be unchanged")

		// Result has collapsed item
		assert.False(t, result["a"])
		assert.True(t, result["b"])
	})

	t.Run("empty map", func(t *testing.T) {
		original := map[string]bool{}

		result := ToggleExpand(original, "x", true)

		assert.Len(t, original, 0)
		assert.True(t, result["x"])
	})

	t.Run("nil safety", func(t *testing.T) {
		result := ToggleExpand(nil, "x", true)

		assert.True(t, result["x"])
	})
}

func TestFilterItemsBySearch(t *testing.T) {
	items := []TreeItem{
		{Name: "Users API", Type: ItemRequest, Method: "GET"},
		{Name: "Products", Type: ItemCollection},
		{Name: "Create User", Type: ItemRequest, Method: "POST"},
		{Name: "Delete Item", Type: ItemRequest, Method: "DELETE"},
	}

	t.Run("empty search returns all", func(t *testing.T) {
		result := FilterItemsBySearch(items, "")
		assert.Len(t, result, 4)
	})

	t.Run("filter by name", func(t *testing.T) {
		result := FilterItemsBySearch(items, "user")
		assert.Len(t, result, 2)
		assert.Equal(t, "Users API", result[0].Name)
		assert.Equal(t, "Create User", result[1].Name)
	})

	t.Run("filter by method", func(t *testing.T) {
		result := FilterItemsBySearch(items, "POST")
		assert.Len(t, result, 1)
		assert.Equal(t, "Create User", result[0].Name)
	})

	t.Run("case insensitive", func(t *testing.T) {
		result := FilterItemsBySearch(items, "DELETE")
		assert.Len(t, result, 1)

		result = FilterItemsBySearch(items, "delete")
		assert.Len(t, result, 1)
	})

	t.Run("no matches", func(t *testing.T) {
		result := FilterItemsBySearch(items, "xyz123")
		assert.Len(t, result, 0)
	})

	t.Run("empty items", func(t *testing.T) {
		result := FilterItemsBySearch([]TreeItem{}, "test")
		assert.Len(t, result, 0)
	})
}

func TestFilterItemsBySearch_Extended(t *testing.T) {
	// Create request with body
	reqWithBody := core.NewRequestDefinition("Simple Request", "GET", "https://api.example.com/users")
	reqWithBody.SetBody(`{"username": "john_doe", "email": "john@example.com"}`)

	// Create request with headers
	reqWithHeaders := core.NewRequestDefinition("Auth Request", "POST", "https://api.example.com/login")
	reqWithHeaders.SetHeader("Authorization", "Bearer secret-token-abc123")
	reqWithHeaders.SetHeader("X-Custom-Header", "custom-value-xyz")

	// Create request with specific URL
	reqWithURL := core.NewRequestDefinition("Specific Endpoint", "GET", "https://api.myservice.io/v2/products")

	// Create WebSocket with endpoint
	wsItem := &core.WebSocketDefinition{
		Name:     "WS Connection",
		Endpoint: "wss://websocket.example.org/stream",
	}

	items := []TreeItem{
		{Name: "Simple Request", Type: ItemRequest, Method: "GET", Request: reqWithBody},
		{Name: "Auth Request", Type: ItemRequest, Method: "POST", Request: reqWithHeaders},
		{Name: "Specific Endpoint", Type: ItemRequest, Method: "GET", Request: reqWithURL},
		{Name: "WS Connection", Type: ItemWebSocket, WebSocket: wsItem},
		{Name: "Folder", Type: ItemFolder},
	}

	t.Run("search by body content", func(t *testing.T) {
		result := FilterItemsBySearch(items, "john_doe")
		assert.Len(t, result, 1)
		assert.Equal(t, "Simple Request", result[0].Name)
	})

	t.Run("search by body email", func(t *testing.T) {
		result := FilterItemsBySearch(items, "john@example.com")
		assert.Len(t, result, 1)
		assert.Equal(t, "Simple Request", result[0].Name)
	})

	t.Run("search by header key", func(t *testing.T) {
		result := FilterItemsBySearch(items, "authorization")
		assert.Len(t, result, 1)
		assert.Equal(t, "Auth Request", result[0].Name)
	})

	t.Run("search by header value", func(t *testing.T) {
		result := FilterItemsBySearch(items, "secret-token")
		assert.Len(t, result, 1)
		assert.Equal(t, "Auth Request", result[0].Name)
	})

	t.Run("search by custom header", func(t *testing.T) {
		result := FilterItemsBySearch(items, "x-custom")
		assert.Len(t, result, 1)
		assert.Equal(t, "Auth Request", result[0].Name)
	})

	t.Run("search by URL domain", func(t *testing.T) {
		result := FilterItemsBySearch(items, "myservice.io")
		assert.Len(t, result, 1)
		assert.Equal(t, "Specific Endpoint", result[0].Name)
	})

	t.Run("search by URL path", func(t *testing.T) {
		result := FilterItemsBySearch(items, "/v2/products")
		assert.Len(t, result, 1)
		assert.Equal(t, "Specific Endpoint", result[0].Name)
	})

	t.Run("search by URL partial", func(t *testing.T) {
		result := FilterItemsBySearch(items, "api.example.com")
		assert.Len(t, result, 2) // Both reqWithBody and reqWithHeaders
	})

	t.Run("search by WebSocket endpoint", func(t *testing.T) {
		result := FilterItemsBySearch(items, "websocket.example.org")
		assert.Len(t, result, 1)
		assert.Equal(t, "WS Connection", result[0].Name)
	})

	t.Run("search by WebSocket wss protocol", func(t *testing.T) {
		result := FilterItemsBySearch(items, "wss://")
		assert.Len(t, result, 1)
		assert.Equal(t, "WS Connection", result[0].Name)
	})

	t.Run("folders do not match body search", func(t *testing.T) {
		result := FilterItemsBySearch(items, "john_doe")
		for _, item := range result {
			assert.NotEqual(t, ItemFolder, item.Type)
		}
	})

	t.Run("case insensitive URL search", func(t *testing.T) {
		result := FilterItemsBySearch(items, "MYSERVICE.IO")
		assert.Len(t, result, 1)
		assert.Equal(t, "Specific Endpoint", result[0].Name)
	})

	t.Run("case insensitive header search", func(t *testing.T) {
		result := FilterItemsBySearch(items, "BEARER")
		assert.Len(t, result, 1)
		assert.Equal(t, "Auth Request", result[0].Name)
	})
}

func TestMatchesSearch_NilRequest(t *testing.T) {
	item := TreeItem{
		Name:    "Nil Request",
		Type:    ItemRequest,
		Method:  "GET",
		Request: nil,
	}

	result := matchesSearch(item, "test")
	assert.False(t, result)

	result = matchesSearch(item, "nil")
	assert.True(t, result) // Matches name
}

func TestMatchesSearch_NilWebSocket(t *testing.T) {
	item := TreeItem{
		Name:      "Nil WebSocket",
		Type:      ItemWebSocket,
		WebSocket: nil,
	}

	result := matchesSearch(item, "websocket")
	assert.True(t, result) // Matches name

	result = matchesSearch(item, "endpoint")
	assert.False(t, result) // Doesn't crash on nil
}
