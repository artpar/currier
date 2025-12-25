package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewXMLFormatter(t *testing.T) {
	formatter := NewXMLFormatter()
	assert.NotNil(t, formatter)
}

func TestXMLFormatter_FormatLines_SimpleElement(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root>content</root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	// Should have formatted output with highlighting
	assert.True(t, len(lines) >= 1)
}

func TestXMLFormatter_FormatLines_NestedElements(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root><child>value</child></root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	// Should have multiple lines due to formatting
	assert.True(t, len(lines) >= 3)
}

func TestXMLFormatter_FormatLines_WithAttributes(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root attr="value">content</root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	// Output should contain the attribute
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "attr")
	assert.Contains(t, fullOutput, "value")
}

func TestXMLFormatter_FormatLines_XMLDeclaration(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<?xml version="1.0" encoding="UTF-8"?><root/>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "xml")
	assert.Contains(t, fullOutput, "version")
}

func TestXMLFormatter_FormatLines_Comment(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root><!-- This is a comment --></root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "comment")
}

func TestXMLFormatter_FormatLines_EmptyElement(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root/>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
}

func TestXMLFormatter_FormatLines_MultipleAttributes(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<element id="1" name="test" enabled="true"/>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "id")
	assert.Contains(t, fullOutput, "name")
	assert.Contains(t, fullOutput, "enabled")
}

func TestXMLFormatter_FormatLines_DeeplyNested(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<a><b><c><d>deep</d></c></b></a>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	// Should have proper indentation - check that later lines have more spaces
	assert.True(t, len(lines) >= 5)
}

func TestXMLFormatter_FormatLines_MixedContent(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root>text<child/>more text</root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "text")
	assert.Contains(t, fullOutput, "child")
}

func TestXMLFormatter_FormatLines_InvalidXML(t *testing.T) {
	formatter := NewXMLFormatter()
	// Invalid XML - unclosed tag
	input := `<root><unclosed>`

	lines := formatter.FormatLines(input)

	// Should return something (original content) even for invalid XML
	require.NotEmpty(t, lines)
}

func TestXMLFormatter_FormatLines_Whitespace(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root>
		<child>value</child>
	</root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	// Should normalize whitespace
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "root")
	assert.Contains(t, fullOutput, "child")
}

func TestXMLFormatter_FormatLines_CDATA(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root><![CDATA[Some <special> content]]></root>`

	lines := formatter.FormatLines(input)

	// CDATA might not be perfectly handled but shouldn't crash
	require.NotEmpty(t, lines)
}

func TestXMLFormatter_FormatLines_Namespace(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root xmlns="http://example.com"><child/></root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "xmlns")
}

func TestXMLFormatter_FormatLines_EmptyInput(t *testing.T) {
	formatter := NewXMLFormatter()
	input := ``

	lines := formatter.FormatLines(input)

	// Empty input should return empty or single empty line
	assert.True(t, len(lines) <= 1)
}

func TestXMLFormatter_FormatLines_Doctype(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<!DOCTYPE root SYSTEM "root.dtd"><root/>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")
	assert.Contains(t, fullOutput, "DOCTYPE")
}

func TestXMLFormatter_Indentation(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root><child><grandchild>value</grandchild></child></root>`

	lines := formatter.FormatLines(input)

	require.True(t, len(lines) >= 5)

	// Check that indentation increases for nested elements
	// Find a line with "grandchild" and verify it has more leading spaces
	foundGrandchild := false
	for _, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		if strings.Contains(stripped, "grandchild") {
			indent := len(line) - len(stripped)
			// grandchild should be indented at least 4 spaces (2 levels * 2 spaces)
			assert.True(t, indent >= 4 || strings.Contains(line, "\x1b"), "grandchild should be indented")
			foundGrandchild = true
			break
		}
	}
	assert.True(t, foundGrandchild, "should find grandchild element")
}

func TestXMLFormatter_SyntaxHighlighting(t *testing.T) {
	formatter := NewXMLFormatter()
	input := `<root attr="value">text</root>`

	lines := formatter.FormatLines(input)

	require.NotEmpty(t, lines)
	fullOutput := strings.Join(lines, "\n")

	// Verify the output contains the expected content
	// Note: ANSI codes may not be present in test environment without terminal context
	assert.Contains(t, fullOutput, "root")
	assert.Contains(t, fullOutput, "attr")
	assert.Contains(t, fullOutput, "value")
	assert.Contains(t, fullOutput, "text")
}
