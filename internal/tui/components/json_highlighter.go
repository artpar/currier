package components

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// JSONHighlighter provides syntax highlighting for JSON content.
type JSONHighlighter struct {
	keyStyle     lipgloss.Style
	stringStyle  lipgloss.Style
	numberStyle  lipgloss.Style
	boolStyle    lipgloss.Style
	nullStyle    lipgloss.Style
	bracketStyle lipgloss.Style
	colonStyle   lipgloss.Style
}

// NewJSONHighlighter creates a new JSON highlighter with default styles.
func NewJSONHighlighter() *JSONHighlighter {
	return &JSONHighlighter{
		keyStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("141")), // Purple
		stringStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("34")),  // Green
		numberStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // Orange
		boolStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("33")),  // Blue
		nullStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")), // Gray
		bracketStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("252")), // Light gray
		colonStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")), // Gray
	}
}

// Highlight applies syntax highlighting to JSON content.
func (h *JSONHighlighter) Highlight(json string) string {
	lines := strings.Split(json, "\n")
	var result []string

	for _, line := range lines {
		result = append(result, h.highlightLine(line))
	}

	return strings.Join(result, "\n")
}

// HighlightLines applies syntax highlighting and returns lines.
func (h *JSONHighlighter) HighlightLines(jsonContent string) []string {
	lines := strings.Split(jsonContent, "\n")
	var result []string

	for _, line := range lines {
		result = append(result, h.highlightLine(line))
	}

	return result
}

// FormatLines pretty-prints JSON with indentation and syntax highlighting.
func (h *JSONHighlighter) FormatLines(content string) []string {
	// First, try to pretty-print the JSON
	formatted := h.prettyPrint(content)

	// Then apply syntax highlighting
	return h.HighlightLines(formatted)
}

// prettyPrint reformats JSON with proper indentation.
func (h *JSONHighlighter) prettyPrint(content string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(content), "", "  ")
	if err != nil {
		// If JSON is invalid, return original content
		return content
	}
	return out.String()
}

func (h *JSONHighlighter) highlightLine(line string) string {
	// Preserve leading whitespace
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	if trimmed == "" {
		return line
	}

	// Process the line character by character
	var result strings.Builder
	result.WriteString(indent)

	i := 0
	chars := []rune(trimmed)

	for i < len(chars) {
		ch := chars[i]

		switch {
		case ch == '"':
			// Check if this is a key (followed by :) or a value
			str, end := h.extractString(chars, i)
			if end < len(chars) {
				// Look ahead for colon (skip whitespace)
				j := end
				for j < len(chars) && (chars[j] == ' ' || chars[j] == '\t') {
					j++
				}
				if j < len(chars) && chars[j] == ':' {
					// It's a key
					result.WriteString(h.keyStyle.Render(str))
				} else {
					// It's a string value
					result.WriteString(h.stringStyle.Render(str))
				}
			} else {
				result.WriteString(h.stringStyle.Render(str))
			}
			i = end

		case ch == ':':
			result.WriteString(h.colonStyle.Render(":"))
			i++

		case ch == '{' || ch == '}' || ch == '[' || ch == ']':
			result.WriteString(h.bracketStyle.Render(string(ch)))
			i++

		case ch == 't' || ch == 'f':
			// Check for true/false
			word := h.extractWord(chars, i)
			if word == "true" || word == "false" {
				result.WriteString(h.boolStyle.Render(word))
				i += len(word)
			} else {
				result.WriteRune(ch)
				i++
			}

		case ch == 'n':
			// Check for null
			word := h.extractWord(chars, i)
			if word == "null" {
				result.WriteString(h.nullStyle.Render(word))
				i += len(word)
			} else {
				result.WriteRune(ch)
				i++
			}

		case ch == '-' || (ch >= '0' && ch <= '9'):
			// Number
			num := h.extractNumber(chars, i)
			result.WriteString(h.numberStyle.Render(num))
			i += len(num)

		default:
			result.WriteRune(ch)
			i++
		}
	}

	return result.String()
}

func (h *JSONHighlighter) extractString(chars []rune, start int) (string, int) {
	if start >= len(chars) || chars[start] != '"' {
		return "", start
	}

	var sb strings.Builder
	sb.WriteRune('"')
	i := start + 1

	for i < len(chars) {
		ch := chars[i]
		sb.WriteRune(ch)
		if ch == '\\' && i+1 < len(chars) {
			i++
			sb.WriteRune(chars[i])
		} else if ch == '"' {
			i++
			break
		}
		i++
	}

	return sb.String(), i
}

func (h *JSONHighlighter) extractWord(chars []rune, start int) string {
	var sb strings.Builder
	i := start

	for i < len(chars) && ((chars[i] >= 'a' && chars[i] <= 'z') || (chars[i] >= 'A' && chars[i] <= 'Z')) {
		sb.WriteRune(chars[i])
		i++
	}

	return sb.String()
}

func (h *JSONHighlighter) extractNumber(chars []rune, start int) string {
	var sb strings.Builder
	i := start

	// Optional minus
	if i < len(chars) && chars[i] == '-' {
		sb.WriteRune(chars[i])
		i++
	}

	// Integer part
	for i < len(chars) && chars[i] >= '0' && chars[i] <= '9' {
		sb.WriteRune(chars[i])
		i++
	}

	// Decimal part
	if i < len(chars) && chars[i] == '.' {
		sb.WriteRune(chars[i])
		i++
		for i < len(chars) && chars[i] >= '0' && chars[i] <= '9' {
			sb.WriteRune(chars[i])
			i++
		}
	}

	// Exponent
	if i < len(chars) && (chars[i] == 'e' || chars[i] == 'E') {
		sb.WriteRune(chars[i])
		i++
		if i < len(chars) && (chars[i] == '+' || chars[i] == '-') {
			sb.WriteRune(chars[i])
			i++
		}
		for i < len(chars) && chars[i] >= '0' && chars[i] <= '9' {
			sb.WriteRune(chars[i])
			i++
		}
	}

	return sb.String()
}

// IsJSON checks if content looks like JSON.
func IsJSON(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}

	// Check if it starts with { or [
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return true
	}

	// Check common JSON patterns
	jsonPattern := regexp.MustCompile(`^\s*[\[{]`)
	return jsonPattern.MatchString(content)
}
