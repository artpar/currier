package components

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// XMLFormatter provides XML pretty printing with syntax highlighting.
type XMLFormatter struct {
	tagStyle       lipgloss.Style
	attrNameStyle  lipgloss.Style
	attrValueStyle lipgloss.Style
	textStyle      lipgloss.Style
	commentStyle   lipgloss.Style
	punctStyle     lipgloss.Style
}

// NewXMLFormatter creates a new XML formatter with default styling.
func NewXMLFormatter() *XMLFormatter {
	return &XMLFormatter{
		tagStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("33")),  // Blue
		attrNameStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("141")), // Purple
		attrValueStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("34")),  // Green
		textStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("252")), // Light gray
		commentStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")), // Gray
		punctStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")), // Gray
	}
}

// FormatLines pretty-prints XML with indentation and syntax highlighting.
func (f *XMLFormatter) FormatLines(content string) []string {
	// First, try to pretty-print the XML
	formatted := f.prettyPrint(content)

	// Then apply syntax highlighting
	return f.highlightLines(formatted)
}

// prettyPrint reformats XML with proper indentation.
func (f *XMLFormatter) prettyPrint(content string) string {
	decoder := xml.NewDecoder(strings.NewReader(content))
	var buf bytes.Buffer
	var indent int

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// If XML is invalid, return original content
			return content
		}

		switch t := token.(type) {
		case xml.StartElement:
			f.writeIndent(&buf, indent)
			buf.WriteString("<")
			buf.WriteString(t.Name.Local)
			for _, attr := range t.Attr {
				buf.WriteString(" ")
				buf.WriteString(attr.Name.Local)
				buf.WriteString(`="`)
				buf.WriteString(attr.Value)
				buf.WriteString(`"`)
			}
			buf.WriteString(">\n")
			indent++

		case xml.EndElement:
			indent--
			f.writeIndent(&buf, indent)
			buf.WriteString("</")
			buf.WriteString(t.Name.Local)
			buf.WriteString(">\n")

		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" {
				f.writeIndent(&buf, indent)
				buf.WriteString(text)
				buf.WriteString("\n")
			}

		case xml.Comment:
			f.writeIndent(&buf, indent)
			buf.WriteString("<!--")
			buf.WriteString(string(t))
			buf.WriteString("-->\n")

		case xml.ProcInst:
			f.writeIndent(&buf, indent)
			buf.WriteString("<?")
			buf.WriteString(t.Target)
			if len(t.Inst) > 0 {
				buf.WriteString(" ")
				buf.WriteString(string(t.Inst))
			}
			buf.WriteString("?>\n")

		case xml.Directive:
			f.writeIndent(&buf, indent)
			buf.WriteString("<!")
			buf.WriteString(string(t))
			buf.WriteString(">\n")
		}
	}

	return strings.TrimSuffix(buf.String(), "\n")
}

func (f *XMLFormatter) writeIndent(buf *bytes.Buffer, level int) {
	for i := 0; i < level; i++ {
		buf.WriteString("  ")
	}
}

// highlightLines applies syntax highlighting to formatted XML.
func (f *XMLFormatter) highlightLines(content string) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))

	for i, line := range lines {
		result[i] = f.highlightLine(line)
	}

	return result
}

// highlightLine applies syntax highlighting to a single line.
func (f *XMLFormatter) highlightLine(line string) string {
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

	// Processing instruction
	if strings.HasPrefix(trimmed, "<?") {
		result.WriteString(f.highlightProcInst(trimmed))
		return result.String()
	}

	// Directive (DOCTYPE, etc.)
	if strings.HasPrefix(trimmed, "<!") {
		result.WriteString(f.commentStyle.Render(trimmed))
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

// highlightTag highlights an XML tag with attributes.
func (f *XMLFormatter) highlightTag(tag string) string {
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
		if c == ' ' || c == '>' || c == '/' {
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
		for idx < len(rest) && (rest[idx] == ' ' || rest[idx] == '\t') {
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

		// Parse attribute name
		eqIdx := strings.Index(rest, "=")
		if eqIdx == -1 {
			result.WriteString(rest)
			break
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
		}
	}

	return result.String()
}

// highlightProcInst highlights a processing instruction.
func (f *XMLFormatter) highlightProcInst(pi string) string {
	var result strings.Builder
	result.WriteString(f.punctStyle.Render("<?"))

	// Remove <? and ?>
	inner := pi[2:]
	if strings.HasSuffix(inner, "?>") {
		inner = inner[:len(inner)-2]
	}

	// Find target name
	spaceIdx := strings.Index(inner, " ")
	if spaceIdx == -1 {
		result.WriteString(f.tagStyle.Render(inner))
	} else {
		result.WriteString(f.tagStyle.Render(inner[:spaceIdx]))
		// Highlight attributes in the rest
		rest := inner[spaceIdx:]
		result.WriteString(f.highlightProcInstAttrs(rest))
	}

	result.WriteString(f.punctStyle.Render("?>"))
	return result.String()
}

// highlightProcInstAttrs highlights attributes in a processing instruction.
func (f *XMLFormatter) highlightProcInstAttrs(attrs string) string {
	var result strings.Builder

	for len(attrs) > 0 {
		// Skip whitespace
		idx := 0
		for idx < len(attrs) && (attrs[idx] == ' ' || attrs[idx] == '\t') {
			result.WriteString(" ")
			idx++
		}
		attrs = attrs[idx:]

		if len(attrs) == 0 {
			break
		}

		// Parse attribute name
		eqIdx := strings.Index(attrs, "=")
		if eqIdx == -1 {
			result.WriteString(attrs)
			break
		}

		attrName := attrs[:eqIdx]
		result.WriteString(f.attrNameStyle.Render(attrName))
		result.WriteString(f.punctStyle.Render("="))
		attrs = attrs[eqIdx+1:]

		// Parse attribute value
		if len(attrs) > 0 && (attrs[0] == '"' || attrs[0] == '\'') {
			quote := attrs[0]
			endQuote := strings.Index(attrs[1:], string(quote))
			if endQuote == -1 {
				result.WriteString(attrs)
				break
			}
			attrValue := attrs[:endQuote+2]
			result.WriteString(f.attrValueStyle.Render(attrValue))
			attrs = attrs[endQuote+2:]
		}
	}

	return result.String()
}
