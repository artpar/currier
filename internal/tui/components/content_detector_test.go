package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentFormat_String(t *testing.T) {
	tests := []struct {
		format   ContentFormat
		expected string
	}{
		{FormatJSON, "json"},
		{FormatXML, "xml"},
		{FormatHTML, "html"},
		{FormatText, "text"},
		{FormatBinary, "binary"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.format.String())
		})
	}
}

func TestContentFormat_Upper(t *testing.T) {
	tests := []struct {
		format   ContentFormat
		expected string
	}{
		{FormatJSON, "JSON"},
		{FormatXML, "XML"},
		{FormatHTML, "HTML"},
		{FormatText, "TEXT"},
		{FormatBinary, "BINARY"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.format.Upper())
		})
	}
}

func TestDetectContentFormat_FromHeader(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		expected    ContentFormat
	}{
		// JSON content types
		{
			name:        "application/json",
			contentType: "application/json",
			body:        "not json content",
			expected:    FormatJSON,
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			body:        "{}",
			expected:    FormatJSON,
		},
		{
			name:        "text/json",
			contentType: "text/json",
			body:        "[]",
			expected:    FormatJSON,
		},
		{
			name:        "application/vnd.api+json",
			contentType: "application/vnd.api+json",
			body:        "{}",
			expected:    FormatJSON,
		},

		// XML content types
		{
			name:        "application/xml",
			contentType: "application/xml",
			body:        "not xml",
			expected:    FormatXML,
		},
		{
			name:        "text/xml",
			contentType: "text/xml",
			body:        "<root/>",
			expected:    FormatXML,
		},
		{
			name:        "application/soap+xml",
			contentType: "application/soap+xml",
			body:        "<soap/>",
			expected:    FormatXML,
		},

		// HTML content types
		{
			name:        "text/html",
			contentType: "text/html",
			body:        "not html",
			expected:    FormatHTML,
		},
		{
			name:        "text/html with charset",
			contentType: "text/html; charset=utf-8",
			body:        "<html></html>",
			expected:    FormatHTML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentFormat(tt.contentType, tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectContentFormat_FromContent(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected ContentFormat
	}{
		// JSON detection
		{
			name:     "JSON object",
			body:     `{"key": "value"}`,
			expected: FormatJSON,
		},
		{
			name:     "JSON array",
			body:     `[1, 2, 3]`,
			expected: FormatJSON,
		},
		{
			name:     "JSON with whitespace",
			body:     `  { "key": "value" }`,
			expected: FormatJSON,
		},
		{
			name:     "JSON array with whitespace",
			body:     "\n\t[1, 2, 3]",
			expected: FormatJSON,
		},

		// XML detection
		{
			name:     "XML declaration",
			body:     `<?xml version="1.0"?><root/>`,
			expected: FormatXML,
		},
		{
			name:     "XML without declaration",
			body:     `<root><child/></root>`,
			expected: FormatXML,
		},
		{
			name:     "XML with whitespace",
			body:     "  <root/>",
			expected: FormatXML,
		},

		// HTML detection
		{
			name:     "HTML doctype",
			body:     `<!DOCTYPE html><html></html>`,
			expected: FormatHTML,
		},
		{
			name:     "HTML doctype lowercase",
			body:     `<!doctype html><html></html>`,
			expected: FormatHTML,
		},
		{
			name:     "HTML tag",
			body:     `<html><head></head><body></body></html>`,
			expected: FormatHTML,
		},
		{
			name:     "HTML with head tag",
			body:     `<head><title>Test</title></head>`,
			expected: FormatHTML,
		},
		{
			name:     "HTML with body tag",
			body:     `<body><p>Hello</p></body>`,
			expected: FormatHTML,
		},
		{
			name:     "HTML with div tag",
			body:     `<div class="container">Content</div>`,
			expected: FormatHTML,
		},
		{
			name:     "HTML with script tag",
			body:     `<script>console.log('test');</script>`,
			expected: FormatHTML,
		},
		{
			name:     "XHTML with XML declaration",
			body:     `<?xml version="1.0"?><!DOCTYPE html><html></html>`,
			expected: FormatHTML,
		},

		// Plain text
		{
			name:     "plain text",
			body:     "Hello, world!",
			expected: FormatText,
		},
		{
			name:     "empty body",
			body:     "",
			expected: FormatText,
		},
		{
			name:     "whitespace only",
			body:     "   \n\t  ",
			expected: FormatText,
		},
		{
			name:     "numbers",
			body:     "12345",
			expected: FormatText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty content type forces content-based detection
			result := DetectContentFormat("", tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectContentFormat_HeaderTakesPrecedence(t *testing.T) {
	// Even if body looks like JSON, header should take precedence
	result := DetectContentFormat("text/html", `{"key": "value"}`)
	assert.Equal(t, FormatHTML, result)

	// Even if body looks like HTML, header should take precedence
	result = DetectContentFormat("application/json", `<html></html>`)
	assert.Equal(t, FormatJSON, result)
}

func TestDetectContentFormat_CaseInsensitive(t *testing.T) {
	tests := []struct {
		contentType string
		expected    ContentFormat
	}{
		{"APPLICATION/JSON", FormatJSON},
		{"Application/Json", FormatJSON},
		{"TEXT/XML", FormatXML},
		{"Text/Html", FormatHTML},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := DetectContentFormat(tt.contentType, "")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectContentFormat_BinaryContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{"application/octet-stream", "application/octet-stream"},
		{"application/gzip", "application/gzip"},
		{"application/x-gzip", "application/x-gzip"},
		{"application/x-tar", "application/x-tar"},
		{"application/zip", "application/zip"},
		{"application/pdf", "application/pdf"},
		{"image/png", "image/png"},
		{"image/jpeg", "image/jpeg"},
		{"audio/mpeg", "audio/mpeg"},
		{"video/mp4", "video/mp4"},
		{"font/woff2", "font/woff2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentFormat(tt.contentType, "any content")
			assert.Equal(t, FormatBinary, result)
		})
	}
}

func TestDetectContentFormat_BinaryContent(t *testing.T) {
	t.Run("detects binary content with null bytes", func(t *testing.T) {
		binaryContent := "some\x00binary\x00content\x00with\x00nulls"
		result := DetectContentFormat("", binaryContent)
		assert.Equal(t, FormatBinary, result)
	})

	t.Run("detects gzip magic bytes", func(t *testing.T) {
		// Gzip magic bytes: 0x1f 0x8b
		gzipContent := "\x1f\x8b\x08\x00\x00\x00\x00\x00"
		result := DetectContentFormat("", gzipContent)
		assert.Equal(t, FormatBinary, result)
	})

	t.Run("detects content with many non-printable chars", func(t *testing.T) {
		binaryContent := string([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D})
		result := DetectContentFormat("", binaryContent)
		assert.Equal(t, FormatBinary, result)
	})

	t.Run("does not flag text with few non-printable chars", func(t *testing.T) {
		// Normal text should not be flagged as binary
		textContent := "Hello, world! This is normal text with some UTF-8: café"
		result := DetectContentFormat("", textContent)
		assert.NotEqual(t, FormatBinary, result)
	})

	t.Run("does not flag text with newlines and tabs", func(t *testing.T) {
		textContent := "Line 1\nLine 2\tTabbed\r\nWindows line"
		result := DetectContentFormat("", textContent)
		assert.NotEqual(t, FormatBinary, result)
	})
}

func TestIsBinaryContent(t *testing.T) {
	t.Run("returns true for content with many null bytes", func(t *testing.T) {
		content := "test\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00more"
		assert.True(t, isBinaryContent(content))
	})

	t.Run("returns true for tar.gz header", func(t *testing.T) {
		// Typical gzip header
		content := "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\x03"
		assert.True(t, isBinaryContent(content))
	})

	t.Run("returns false for plain text", func(t *testing.T) {
		content := "This is plain text with newlines\nand tabs\t!"
		assert.False(t, isBinaryContent(content))
	})

	t.Run("returns false for JSON", func(t *testing.T) {
		content := `{"key": "value", "number": 123}`
		assert.False(t, isBinaryContent(content))
	})

	t.Run("returns false for HTML", func(t *testing.T) {
		content := "<html><body><h1>Hello</h1></body></html>"
		assert.False(t, isBinaryContent(content))
	})
}
