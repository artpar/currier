package components

import (
	"strings"
)

// ContentFormat represents the detected format of response content.
type ContentFormat string

const (
	FormatJSON ContentFormat = "json"
	FormatXML  ContentFormat = "xml"
	FormatHTML ContentFormat = "html"
	FormatText ContentFormat = "text"
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

// detectFromContent attempts to detect the format from the content itself.
func detectFromContent(body string) ContentFormat {
	trimmed := strings.TrimSpace(body)
	if len(trimmed) == 0 {
		return FormatText
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
