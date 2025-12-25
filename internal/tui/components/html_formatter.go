package components

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HTMLFormatter provides HTML pretty printing with syntax highlighting.
type HTMLFormatter struct {
	tagStyle       lipgloss.Style
	attrNameStyle  lipgloss.Style
	attrValueStyle lipgloss.Style
	textStyle      lipgloss.Style
	commentStyle   lipgloss.Style
	punctStyle     lipgloss.Style
	doctypeStyle   lipgloss.Style
}

// NewHTMLFormatter creates a new HTML formatter with default styling.
func NewHTMLFormatter() *HTMLFormatter {
	return &HTMLFormatter{
		tagStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("33")),  // Blue
		attrNameStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("141")), // Purple
		attrValueStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("34")),  // Green
		textStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("252")), // Light gray
		commentStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")), // Gray italic
		punctStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")), // Gray
		doctypeStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // Orange
	}
}

// FormatLines pretty-prints HTML with indentation and syntax highlighting.
func (f *HTMLFormatter) FormatLines(content string) []string {
	// First, pretty-print the HTML
	formatted := f.prettyPrint(content)

	// Then apply syntax highlighting
	return f.highlightLines(formatted)
}

// Void elements that don't have closing tags
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

// Elements that should preserve their content formatting
var preserveElements = map[string]bool{
	"pre": true, "code": true, "script": true, "style": true, "textarea": true,
}

// prettyPrint reformats HTML with proper indentation.
func (f *HTMLFormatter) prettyPrint(content string) string {
	var result strings.Builder
	var indent int
	var inPreserve bool
	var preserveTag string

	// Simple regex-based parsing
	tagRegex := regexp.MustCompile(`(?s)(<[^>]+>|[^<]+)`)
	tokens := tagRegex.FindAllString(content, -1)

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		// Check if it's a tag
		if strings.HasPrefix(token, "<") {
			// Comment
			if strings.HasPrefix(token, "<!--") {
				f.writeIndent(&result, indent)
				result.WriteString(token)
				result.WriteString("\n")
				continue
			}

			// DOCTYPE
			if strings.HasPrefix(strings.ToUpper(token), "<!DOCTYPE") {
				result.WriteString(token)
				result.WriteString("\n")
				continue
			}

			// Extract tag name
			tagName := f.extractTagName(token)
			tagLower := strings.ToLower(tagName)

			// Check if entering or exiting preserve mode
			if preserveElements[tagLower] {
				if strings.HasPrefix(token, "</") {
					inPreserve = false
					preserveTag = ""
				} else if !strings.HasSuffix(token, "/>") {
					inPreserve = true
					preserveTag = tagLower
				}
			}

			// End tag
			if strings.HasPrefix(token, "</") {
				if !inPreserve || tagLower == preserveTag {
					indent--
					if indent < 0 {
						indent = 0
					}
				}
				f.writeIndent(&result, indent)
				result.WriteString(token)
				result.WriteString("\n")
				continue
			}

			// Start tag
			f.writeIndent(&result, indent)
			result.WriteString(token)
			result.WriteString("\n")

			// Increase indent for non-void, non-self-closing tags
			if !voidElements[tagLower] && !strings.HasSuffix(token, "/>") {
				indent++
			}
		} else {
			// Text content
			text := strings.TrimSpace(token)
			if text != "" {
				if inPreserve {
					// Preserve original formatting in pre/code/etc
					result.WriteString(token)
				} else {
					f.writeIndent(&result, indent)
					result.WriteString(text)
					result.WriteString("\n")
				}
			}
		}
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (f *HTMLFormatter) extractTagName(tag string) string {
	// Remove < and potential /
	start := 1
	if len(tag) > 1 && tag[1] == '/' {
		start = 2
	}

	// Find end of tag name
	for i := start; i < len(tag); i++ {
		c := tag[i]
		if c == ' ' || c == '>' || c == '/' || c == '\t' || c == '\n' {
			return tag[start:i]
		}
	}

	// Remove trailing > if present
	end := len(tag)
	if tag[end-1] == '>' {
		end--
	}
	return tag[start:end]
}

func (f *HTMLFormatter) writeIndent(buf *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		buf.WriteString("  ")
	}
}

// highlightLines applies syntax highlighting to formatted HTML.
func (f *HTMLFormatter) highlightLines(content string) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))

	for i, line := range lines {
		result[i] = f.highlightLine(line)
	}

	return result
}

// highlightLine applies syntax highlighting to a single line.
func (f *HTMLFormatter) highlightLine(line string) string {
	// Preserve leading whitespace
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	if len(trimmed) == 0 {
		return line
	}

	var result strings.Builder
	result.WriteString(indent)

	// Comment
	if strings.HasPrefix(trimmed, "<!--") {
		result.WriteString(f.commentStyle.Render(trimmed))
		return result.String()
	}

	// DOCTYPE
	if strings.HasPrefix(strings.ToUpper(trimmed), "<!DOCTYPE") {
		result.WriteString(f.doctypeStyle.Render(trimmed))
		return result.String()
	}

	// End tag
	if strings.HasPrefix(trimmed, "</") {
		result.WriteString(f.highlightTag(trimmed))
		return result.String()
	}

	// Start tag
	if strings.HasPrefix(trimmed, "<") {
		result.WriteString(f.highlightTag(trimmed))
		return result.String()
	}

	// Text content
	result.WriteString(f.textStyle.Render(trimmed))
	return result.String()
}

// highlightTag highlights an HTML tag with attributes.
func (f *HTMLFormatter) highlightTag(tag string) string {
	var result strings.Builder

	// Handle end tag
	if strings.HasPrefix(tag, "</") {
		endIdx := strings.Index(tag, ">")
		if endIdx == -1 {
			return f.tagStyle.Render(tag)
		}
		tagName := tag[2:endIdx]
		result.WriteString(f.punctStyle.Render("</"))
		result.WriteString(f.tagStyle.Render(tagName))
		result.WriteString(f.punctStyle.Render(">"))
		if endIdx < len(tag)-1 {
			result.WriteString(tag[endIdx+1:])
		}
		return result.String()
	}

	// Handle start tag with potential attributes
	// Find the tag name
	var i int
	for i = 1; i < len(tag); i++ {
		c := tag[i]
		if c == ' ' || c == '>' || c == '/' || c == '\t' || c == '\n' {
			break
		}
	}

	tagName := tag[1:i]
	result.WriteString(f.punctStyle.Render("<"))
	result.WriteString(f.tagStyle.Render(tagName))

	// Parse attributes
	rest := tag[i:]
	for len(rest) > 0 {
		// Skip whitespace
		idx := 0
		for idx < len(rest) && (rest[idx] == ' ' || rest[idx] == '\t' || rest[idx] == '\n') {
			result.WriteString(" ")
			idx++
		}
		rest = rest[idx:]

		if len(rest) == 0 {
			break
		}

		// Check for end of tag
		if rest[0] == '>' {
			result.WriteString(f.punctStyle.Render(">"))
			rest = rest[1:]
			continue
		}
		if strings.HasPrefix(rest, "/>") {
			result.WriteString(f.punctStyle.Render("/>"))
			rest = rest[2:]
			continue
		}

		// Parse attribute name (HTML attributes can be without value)
		eqIdx := strings.Index(rest, "=")
		spaceIdx := strings.IndexAny(rest, " \t\n>")

		// Attribute without value (e.g., "disabled", "checked")
		if eqIdx == -1 || (spaceIdx != -1 && spaceIdx < eqIdx) {
			if spaceIdx == -1 {
				spaceIdx = len(rest)
			}
			attrName := rest[:spaceIdx]
			if attrName != "" && attrName[0] != '>' && attrName[0] != '/' {
				result.WriteString(f.attrNameStyle.Render(attrName))
			}
			rest = rest[spaceIdx:]
			continue
		}

		attrName := rest[:eqIdx]
		result.WriteString(f.attrNameStyle.Render(attrName))
		result.WriteString(f.punctStyle.Render("="))
		rest = rest[eqIdx+1:]

		// Parse attribute value
		if len(rest) > 0 && (rest[0] == '"' || rest[0] == '\'') {
			quote := rest[0]
			endQuote := strings.Index(rest[1:], string(quote))
			if endQuote == -1 {
				result.WriteString(rest)
				break
			}
			attrValue := rest[:endQuote+2]
			result.WriteString(f.attrValueStyle.Render(attrValue))
			rest = rest[endQuote+2:]
		} else {
			// Unquoted attribute value
			endIdx := strings.IndexAny(rest, " \t\n>")
			if endIdx == -1 {
				endIdx = len(rest)
			}
			attrValue := rest[:endIdx]
			result.WriteString(f.attrValueStyle.Render(attrValue))
			rest = rest[endIdx:]
		}
	}

	return result.String()
}
