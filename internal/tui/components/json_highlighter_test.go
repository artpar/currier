package components

import (
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
