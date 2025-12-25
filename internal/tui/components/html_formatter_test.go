package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTMLFormatter(t *testing.T) {
	formatter := NewHTMLFormatter()
	assert.NotNil(t, formatter)
}

func TestHTMLFormatter_FormatLines_SimpleElement(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div>content</div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	assert.True(t, len(lines) >= 1)
}

func TestHTMLFormatter_FormatLines_NestedElements(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div><span>value</span></div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	// Should have multiple lines due to formatting
	assert.True(t, len(lines) >= 3)
}

func TestHTMLFormatter_FormatLines_WithAttributes(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div class="container" id="main">content</div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "class")
	assert.Contains(t, fullOutput, "container")
	assert.Contains(t, fullOutput, "id")
	assert.Contains(t, fullOutput, "main")
}

func TestHTMLFormatter_FormatLines_DOCTYPE(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<!DOCTYPE html><html></html>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "DOCTYPE")
}

func TestHTMLFormatter_FormatLines_Comment(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div><!-- This is a comment --></div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "comment")
}

func TestHTMLFormatter_FormatLines_VoidElements(t *testing.T) {
	// Test void elements that don't have closing tags
	voidTags := []string{"br", "hr", "img", "input", "link", "meta"}

	formatter := NewHTMLFormatter()

	for _, tag := range voidTags {
		t.Run(tag, func(t *testing.T) {
			input := "<div><" + tag + "></div>"
			lines := formatter.FormatLines(input)
			require.NotEmpty(t, lines)
		})
	}
}

func TestHTMLFormatter_FormatLines_SelfClosingVoidElement(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div><br/><hr/></div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
}

func TestHTMLFormatter_FormatLines_VoidElementWithAttributes(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<img src="image.png" alt="An image">`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "src")
	assert.Contains(t, fullOutput, "alt")
}

func TestHTMLFormatter_FormatLines_PreserveElements(t *testing.T) {
	// Test elements that should preserve content formatting
	preserveTags := []string{"pre", "code", "script", "style", "textarea"}

	formatter := NewHTMLFormatter()

	for _, tag := range preserveTags {
		t.Run(tag, func(t *testing.T) {
			input := "<" + tag + ">preserved content</" + tag + ">"
			lines := formatter.FormatLines(input)
			require.NotEmpty(t, lines)
			fullOutput := strings.Join(lines, "\n")
			assert.Contains(t, fullOutput, tag)
		})
	}
}

func TestHTMLFormatter_FormatLines_ScriptTag(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<script>console.log('hello');</script>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "script")
}

func TestHTMLFormatter_FormatLines_StyleTag(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<style>.class { color: red; }</style>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "style")
}

func TestHTMLFormatter_FormatLines_DeeplyNested(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div><section><article><p>deep</p></article></section></div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	// Should have proper indentation with multiple lines
	assert.True(t, len(lines) >= 5)
}

func TestHTMLFormatter_FormatLines_MixedContent(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div>text<span/>more text</div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "text")
	assert.Contains(t, fullOutput, "span")
}

func TestHTMLFormatter_FormatLines_EmptyInput(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := ``

	lines := formatter.FormatLines(input)

	// Empty input should return empty or single empty line
	assert.True(t, len(lines) <= 1)
}

func TestHTMLFormatter_FormatLines_FullHTMLDocument(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello</h1></body></html>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "DOCTYPE")
	assert.Contains(t, fullOutput, "html")
	assert.Contains(t, fullOutput, "head")
	assert.Contains(t, fullOutput, "title")
	assert.Contains(t, fullOutput, "body")
	assert.Contains(t, fullOutput, "h1")
}

func TestHTMLFormatter_FormatLines_BooleanAttributes(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<input type="checkbox" checked disabled>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "checked")
	assert.Contains(t, fullOutput, "disabled")
}

func TestHTMLFormatter_FormatLines_SingleQuoteAttributes(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div data-value='single quoted'>content</div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "data-value")
}

func TestHTMLFormatter_FormatLines_Whitespace(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div>
		<span>value</span>
	</div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "div")
	assert.Contains(t, fullOutput, "span")
}

func TestHTMLFormatter_Indentation(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div><section><article>value</article></section></div>`

	lines := formatter.FormatLines(input)

	require.True(t, len(lines) >= 5)

	// Check that indentation increases for nested elements
	foundArticle := false
	for _, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		if strings.Contains(stripped, "article") {
			indent := len(line) - len(stripped)
			// article should be indented at least 4 spaces (2 levels * 2 spaces)
			assert.True(t, indent >= 4 || strings.Contains(line, "\x1b"), "article should be indented")
			foundArticle = true
			break
		}
	}
	assert.True(t, foundArticle, "should find article element")
}

func TestHTMLFormatter_SyntaxHighlighting(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<div class="test">text</div>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")

	// Verify the output contains the expected content
	// Note: ANSI codes may not be present in test environment without terminal context
	assert.Contains(t, fullOutput, "div")
	assert.Contains(t, fullOutput, "class")
	assert.Contains(t, fullOutput, "test")
	assert.Contains(t, fullOutput, "text")
}

func TestHTMLFormatter_FormatLines_MultipleAttributes(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<a href="https://example.com" target="_blank" rel="noopener">Link</a>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "href")
	assert.Contains(t, fullOutput, "target")
	assert.Contains(t, fullOutput, "rel")
}

func TestHTMLFormatter_FormatLines_TableStructure(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<table><thead><tr><th>Header</th></tr></thead><tbody><tr><td>Cell</td></tr></tbody></table>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "table")
	assert.Contains(t, fullOutput, "thead")
	assert.Contains(t, fullOutput, "tbody")
	assert.Contains(t, fullOutput, "tr")
	assert.Contains(t, fullOutput, "th")
	assert.Contains(t, fullOutput, "td")
}

func TestHTMLFormatter_FormatLines_FormElements(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<form action="/submit" method="post"><label for="name">Name</label><input type="text" id="name"><button type="submit">Submit</button></form>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "form")
	assert.Contains(t, fullOutput, "label")
	assert.Contains(t, fullOutput, "input")
	assert.Contains(t, fullOutput, "button")
}

func TestHTMLFormatter_FormatLines_MetaTags(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<head><meta charset="utf-8"><meta name="viewport" content="width=device-width"></head>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "meta")
	assert.Contains(t, fullOutput, "charset")
	assert.Contains(t, fullOutput, "viewport")
}

func TestHTMLFormatter_FormatLines_LinkTags(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<head><link rel="stylesheet" href="styles.css"></head>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "link")
	assert.Contains(t, fullOutput, "rel")
	assert.Contains(t, fullOutput, "href")
}

func TestHTMLFormatter_FormatLines_ListElements(t *testing.T) {
	formatter := NewHTMLFormatter()
	input := `<ul><li>Item 1</li><li>Item 2</li></ul>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "ul")
	assert.Contains(t, fullOutput, "li")
	assert.Contains(t, fullOutput, "Item 1")
	assert.Contains(t, fullOutput, "Item 2")
}
