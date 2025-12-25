package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewJSONHighlighter(t *testing.T) {
	t.Run("creates highlighter", func(t *testing.T) {
		h := NewJSONHighlighter()
		assert.NotNil(t, h)
	})
}

func TestJSONHighlighter_Highlight(t *testing.T) {
	h := NewJSONHighlighter()

	t.Run("highlights simple JSON", func(t *testing.T) {
		input := `{"name": "test"}`
		result := h.Highlight(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "test")
	})

	t.Run("highlights JSON with number", func(t *testing.T) {
		input := `{"count": 42}`
		result := h.Highlight(input)
		assert.Contains(t, result, "count")
		assert.Contains(t, result, "42")
	})

	t.Run("highlights boolean values", func(t *testing.T) {
		input := `{"enabled": true, "disabled": false}`
		result := h.Highlight(input)
		assert.Contains(t, result, "true")
		assert.Contains(t, result, "false")
	})

	t.Run("highlights null values", func(t *testing.T) {
		input := `{"value": null}`
		result := h.Highlight(input)
		assert.Contains(t, result, "null")
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := h.Highlight("")
		assert.Empty(t, result)
	})

	t.Run("highlights multiline JSON", func(t *testing.T) {
		input := `{
  "name": "test",
  "count": 123
}`
		result := h.Highlight(input)
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "test")
	})
}

func TestJSONHighlighter_HighlightLines(t *testing.T) {
	h := NewJSONHighlighter()

	t.Run("returns single empty line for empty input", func(t *testing.T) {
		result := h.HighlightLines("")
		assert.Len(t, result, 1)
	})

	t.Run("highlights each line", func(t *testing.T) {
		input := `{
  "key": "value"
}`
		result := h.HighlightLines(input)
		assert.Len(t, result, 3)
	})

	t.Run("handles single line", func(t *testing.T) {
		input := `{"key": "value"}`
		result := h.HighlightLines(input)
		assert.Len(t, result, 1)
	})
}

func TestJSONHighlighter_ExtractWord(t *testing.T) {
	h := NewJSONHighlighter()

	t.Run("extracts word from line", func(t *testing.T) {
		// Test via highlighting which uses extractWord internally
		input := `{"enabled": true}`
		result := h.Highlight(input)
		assert.Contains(t, result, "true")
	})

	t.Run("extracts null keyword", func(t *testing.T) {
		input := `{"value": null}`
		result := h.Highlight(input)
		assert.Contains(t, result, "null")
	})

	t.Run("extracts false keyword", func(t *testing.T) {
		input := `{"value": false}`
		result := h.Highlight(input)
		assert.Contains(t, result, "false")
	})
}

func TestJSONHighlighter_ExtractNumber(t *testing.T) {
	h := NewJSONHighlighter()

	t.Run("extracts integer", func(t *testing.T) {
		input := `{"count": 42}`
		result := h.Highlight(input)
		assert.Contains(t, result, "42")
	})

	t.Run("extracts negative number", func(t *testing.T) {
		input := `{"temp": -10}`
		result := h.Highlight(input)
		assert.Contains(t, result, "-10")
	})

	t.Run("extracts decimal number", func(t *testing.T) {
		input := `{"pi": 3.14}`
		result := h.Highlight(input)
		assert.Contains(t, result, "3.14")
	})
}

func TestJSONHighlighter_ComplexJSON(t *testing.T) {
	h := NewJSONHighlighter()

	t.Run("handles nested objects", func(t *testing.T) {
		input := `{"user": {"name": "John", "age": 30}}`
		result := h.Highlight(input)
		assert.Contains(t, result, "user")
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "John")
		assert.Contains(t, result, "30")
	})

	t.Run("handles arrays", func(t *testing.T) {
		input := `{"items": [1, 2, 3]}`
		result := h.Highlight(input)
		assert.Contains(t, result, "items")
		assert.Contains(t, result, "1")
		assert.Contains(t, result, "2")
		assert.Contains(t, result, "3")
	})

	t.Run("handles string with escaped quotes", func(t *testing.T) {
		input := `{"msg": "hello \"world\""}`
		result := h.Highlight(input)
		assert.NotEmpty(t, result)
	})
}

func TestIsJSON(t *testing.T) {
	t.Run("detects valid JSON object", func(t *testing.T) {
		assert.True(t, IsJSON(`{"key": "value"}`))
	})

	t.Run("detects valid JSON array", func(t *testing.T) {
		assert.True(t, IsJSON(`[1, 2, 3]`))
	})

	t.Run("detects empty JSON object", func(t *testing.T) {
		assert.True(t, IsJSON(`{}`))
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		assert.False(t, IsJSON(`not json`))
	})

	t.Run("rejects HTML", func(t *testing.T) {
		assert.False(t, IsJSON(`<html><body>test</body></html>`))
	})

	t.Run("handles whitespace", func(t *testing.T) {
		assert.True(t, IsJSON(`  { "key": "value" }  `))
	})
}

func TestHighlightJSON_Numbers(t *testing.T) {
	h := NewJSONHighlighter()

	t.Run("highlights integer numbers", func(t *testing.T) {
		result := h.Highlight(`{"count": 42}`)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "42")
	})

	t.Run("highlights floating point numbers", func(t *testing.T) {
		result := h.Highlight(`{"price": 19.99}`)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "19.99")
	})

	t.Run("highlights negative numbers", func(t *testing.T) {
		result := h.Highlight(`{"offset": -10}`)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "-10")
	})

	t.Run("highlights exponential numbers", func(t *testing.T) {
		result := h.Highlight(`{"value": 1.5e10}`)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "1.5e10")
	})

	t.Run("highlights zero", func(t *testing.T) {
		result := h.Highlight(`{"count": 0}`)
		assert.NotEmpty(t, result)
	})
}

func TestHighlightJSON_EdgeCases(t *testing.T) {
	h := NewJSONHighlighter()

	t.Run("handles deeply nested JSON", func(t *testing.T) {
		result := h.Highlight(`{"a":{"b":{"c":{"d":"value"}}}}`)
		assert.NotEmpty(t, result)
	})

	t.Run("handles arrays with mixed types", func(t *testing.T) {
		result := h.Highlight(`[1, "two", true, null, {"key": "value"}]`)
		assert.NotEmpty(t, result)
	})

	t.Run("handles special characters in strings", func(t *testing.T) {
		result := h.Highlight(`{"text": "hello\nworld\ttab"}`)
		assert.NotEmpty(t, result)
	})

	t.Run("handles empty arrays", func(t *testing.T) {
		result := h.Highlight(`{"empty": []}`)
		assert.NotEmpty(t, result)
	})

	t.Run("handles unicode characters", func(t *testing.T) {
		result := h.Highlight(`{"emoji": "😀", "chinese": "中文"}`)
		assert.NotEmpty(t, result)
	})

	t.Run("handles incomplete JSON", func(t *testing.T) {
		result := h.Highlight(`{"key": "value"`)
		assert.NotEmpty(t, result)
	})

	t.Run("handles very long strings", func(t *testing.T) {
		longValue := "a" + strings.Repeat("b", 1000) + "c"
		result := h.Highlight(`{"long": "` + longValue + `"}`)
		assert.NotEmpty(t, result)
	})

	t.Run("handles escaped quotes in strings", func(t *testing.T) {
		result := h.Highlight(`{"text": "hello \"world\""}`)
		assert.NotEmpty(t, result)
	})
}
