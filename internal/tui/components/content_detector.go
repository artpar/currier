package components

import (
	"strings"
	"unicode"
)

// ContentFormat represents the detected format of response content.
type ContentFormat string

const (
	FormatJSON   ContentFormat = "json"
	FormatXML    ContentFormat = "xml"
	FormatHTML   ContentFormat = "html"
	FormatText   ContentFormat = "text"
	FormatBinary ContentFormat = "binary"
)

// String returns the string representation of the content format.
func (f ContentFormat) String() string {
	return string(f)
}

// Upper returns the uppercase string for display.
func (f ContentFormat) Upper() string {
	return strings.ToUpper(string(f))
}

// DetectContentFormat checks Content-Type header first, then falls back to content detection.
func DetectContentFormat(contentType string, body string) ContentFormat {
	// Check Content-Type header first (most reliable)
	ct := strings.ToLower(contentType)

	// Binary types - check these first to avoid processing binary as text
	if isBinaryContentType(ct) {
		return FormatBinary
	}

	// JSON types
	if strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "text/json") ||
		strings.Contains(ct, "+json") {
		return FormatJSON
	}

	// XML types
	if strings.Contains(ct, "application/xml") ||
		strings.Contains(ct, "text/xml") ||
		strings.Contains(ct, "+xml") {
		return FormatXML
	}

	// HTML types
	if strings.Contains(ct, "text/html") {
		return FormatHTML
	}

	// Fall back to content-based detection
	return detectFromContent(body)
}

// isBinaryContentType checks if the content type indicates binary data.
func isBinaryContentType(ct string) bool {
	binaryTypes := []string{
		"application/octet-stream",
		"application/gzip",
		"application/x-gzip",
		"application/x-tar",
		"application/zip",
		"application/x-zip",
		"application/x-compressed",
		"application/x-bzip",
		"application/x-bzip2",
		"application/x-7z-compressed",
		"application/x-rar-compressed",
		"application/pdf",
		"application/x-executable",
		"application/x-sharedlib",
		"application/x-mach-binary",
		"application/wasm",
	}

	for _, bt := range binaryTypes {
		if strings.Contains(ct, bt) {
			return true
		}
	}

	// Check for binary media types
	if strings.HasPrefix(ct, "image/") ||
		strings.HasPrefix(ct, "audio/") ||
		strings.HasPrefix(ct, "video/") ||
		strings.HasPrefix(ct, "font/") {
		return true
	}

	return false
}

// detectFromContent attempts to detect the format from the content itself.
func detectFromContent(body string) ContentFormat {
	trimmed := strings.TrimSpace(body)
	if len(trimmed) == 0 {
		return FormatText
	}

	// Check for binary content first (non-printable characters)
	if isBinaryContent(body) {
		return FormatBinary
	}

	// JSON: starts with { or [
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return FormatJSON
	}

	// XML/HTML: starts with <
	if trimmed[0] == '<' {
		lower := strings.ToLower(trimmed)

		// Check for HTML doctype or html tag
		if strings.HasPrefix(lower, "<!doctype html") ||
			strings.HasPrefix(lower, "<html") {
			return FormatHTML
		}

		// Check for XML declaration
		if strings.HasPrefix(trimmed, "<?xml") {
			// Could still be XHTML
			if strings.Contains(lower, "<html") {
				return FormatHTML
			}
			return FormatXML
		}

		// Check for common HTML tags at the start
		htmlTags := []string{"<head", "<body", "<div", "<span", "<p>", "<a ", "<script", "<style", "<meta", "<link", "<title"}
		for _, tag := range htmlTags {
			if strings.HasPrefix(lower, tag) {
				return FormatHTML
			}
		}

		// Generic XML tag (anything starting with <tagname)
		return FormatXML
	}

	return FormatText
}

// isBinaryContent checks if the content contains non-printable characters
// that indicate binary data. We sample the first 8KB for efficiency.
func isBinaryContent(body string) bool {
	// Sample first 8KB for efficiency
	sample := body
	if len(sample) > 8192 {
		sample = sample[:8192]
	}

	// Count non-printable characters
	nonPrintable := 0
	total := 0

	for _, r := range sample {
		total++
		// Allow common whitespace: space, tab, newline, carriage return
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		// Check if character is printable
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			nonPrintable++
		}
		// Early exit if we find enough non-printable chars
		if nonPrintable > 10 {
			return true
		}
	}

	// If more than 5% non-printable, consider it binary
	if total > 0 && float64(nonPrintable)/float64(total) > 0.05 {
		return true
	}

	return false
}
