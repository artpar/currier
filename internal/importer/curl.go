package importer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/artpar/currier/internal/core"
)

// CurlImporter imports curl commands.
type CurlImporter struct{}

// NewCurlImporter creates a new curl importer.
func NewCurlImporter() *CurlImporter {
	return &CurlImporter{}
}

func (c *CurlImporter) Name() string {
	return "curl command"
}

func (c *CurlImporter) Format() Format {
	return FormatCurl
}

func (c *CurlImporter) FileExtensions() []string {
	return []string{".sh", ".curl", ".txt"}
}

func (c *CurlImporter) DetectFormat(content []byte) bool {
	trimmed := strings.TrimSpace(string(content))
	return strings.HasPrefix(trimmed, "curl ") || strings.HasPrefix(trimmed, "curl\t")
}

func (c *CurlImporter) Import(ctx context.Context, content []byte) (*core.Collection, error) {
	cmd := strings.TrimSpace(string(content))

	// Parse the curl command
	parsed, err := parseCurlCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseError, err)
	}

	// Create collection with single request
	coll := core.NewCollection("Imported from curl")

	req := core.NewRequestDefinition(
		parsed.name,
		parsed.method,
		parsed.url,
	)

	// Set headers
	for key, value := range parsed.headers {
		req.SetHeader(key, value)
	}

	// Set body
	if parsed.body != "" {
		req.SetBody(parsed.body)
	}

	// Set auth
	if parsed.auth.Type != "" {
		req.SetAuth(parsed.auth)
	}

	coll.AddRequest(req)
	return coll, nil
}

type parsedCurl struct {
	name    string
	method  string
	url     string
	headers map[string]string
	body    string
	auth    core.AuthConfig
}

func parseCurlCommand(cmd string) (*parsedCurl, error) {
	result := &parsedCurl{
		method:  "GET",
		headers: make(map[string]string),
	}

	// Normalize: handle line continuations and multiple spaces
	cmd = strings.ReplaceAll(cmd, "\\\n", " ")
	cmd = strings.ReplaceAll(cmd, "\\\r\n", " ")
	cmd = regexp.MustCompile(`\s+`).ReplaceAllString(cmd, " ")

	// Tokenize respecting quotes
	tokens := tokenize(cmd)

	if len(tokens) == 0 || tokens[0] != "curl" {
		return nil, fmt.Errorf("not a curl command")
	}

	i := 1
	for i < len(tokens) {
		token := tokens[i]

		switch token {
		case "-X", "--request":
			if i+1 < len(tokens) {
				result.method = strings.ToUpper(tokens[i+1])
				i += 2
			} else {
				i++
			}

		case "-H", "--header":
			if i+1 < len(tokens) {
				header := tokens[i+1]
				if idx := strings.Index(header, ":"); idx > 0 {
					key := strings.TrimSpace(header[:idx])
					value := strings.TrimSpace(header[idx+1:])
					result.headers[key] = value
				}
				i += 2
			} else {
				i++
			}

		case "-d", "--data", "--data-raw", "--data-binary":
			if i+1 < len(tokens) {
				result.body = tokens[i+1]
				// Data implies POST if not specified
				if result.method == "GET" {
					result.method = "POST"
				}
				i += 2
			} else {
				i++
			}

		case "--data-urlencode":
			if i+1 < len(tokens) {
				if result.body != "" {
					result.body += "&"
				}
				result.body += tokens[i+1]
				if result.method == "GET" {
					result.method = "POST"
				}
				i += 2
			} else {
				i++
			}

		case "-u", "--user":
			if i+1 < len(tokens) {
				userPass := tokens[i+1]
				parts := strings.SplitN(userPass, ":", 2)
				result.auth = core.AuthConfig{
					Type:     "basic",
					Username: parts[0],
				}
				if len(parts) > 1 {
					result.auth.Password = parts[1]
				}
				i += 2
			} else {
				i++
			}

		case "-A", "--user-agent":
			if i+1 < len(tokens) {
				result.headers["User-Agent"] = tokens[i+1]
				i += 2
			} else {
				i++
			}

		case "-e", "--referer":
			if i+1 < len(tokens) {
				result.headers["Referer"] = tokens[i+1]
				i += 2
			} else {
				i++
			}

		case "-b", "--cookie":
			if i+1 < len(tokens) {
				result.headers["Cookie"] = tokens[i+1]
				i += 2
			} else {
				i++
			}

		case "--compressed":
			result.headers["Accept-Encoding"] = "gzip, deflate, br"
			i++

		case "-L", "--location":
			// Follow redirects - note for later
			i++

		case "-k", "--insecure":
			// Skip SSL verification - note for later
			i++

		case "-s", "--silent", "-S", "--show-error", "-v", "--verbose":
			// Output options - ignore
			i++

		case "-o", "--output", "-O", "--remote-name":
			// Output file options
			if token == "-o" || token == "--output" {
				if i+1 < len(tokens) {
					i += 2
				} else {
					i++
				}
			} else {
				i++
			}

		case "-I", "--head":
			result.method = "HEAD"
			i++

		case "-G", "--get":
			result.method = "GET"
			i++

		case "--json":
			if i+1 < len(tokens) {
				result.body = tokens[i+1]
				result.headers["Content-Type"] = "application/json"
				result.headers["Accept"] = "application/json"
				if result.method == "GET" {
					result.method = "POST"
				}
				i += 2
			} else {
				i++
			}

		default:
			// Check if it's an option we should skip
			if strings.HasPrefix(token, "-") {
				// Unknown option, check if it might have a value
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					i += 2
				} else {
					i++
				}
			} else {
				// Assume it's the URL
				if result.url == "" {
					result.url = token
				}
				i++
			}
		}
	}

	if result.url == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	// Generate name from URL
	result.name = generateNameFromURL(result.url)

	return result, nil
}

// tokenize splits the command respecting quotes
func tokenize(cmd string) []string {
	var tokens []string
	var current strings.Builder
	var inQuote rune
	var escaped bool

	for _, r := range cmd {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}

		if r == '"' || r == '\'' {
			inQuote = r
			continue
		}

		if r == ' ' || r == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func generateNameFromURL(url string) string {
	// Remove protocol
	name := url
	if idx := strings.Index(name, "://"); idx >= 0 {
		name = name[idx+3:]
	}

	// Get path
	if idx := strings.Index(name, "/"); idx >= 0 {
		path := name[idx:]
		// Remove query string
		if qIdx := strings.Index(path, "?"); qIdx >= 0 {
			path = path[:qIdx]
		}
		// Get last segment
		segments := strings.Split(strings.Trim(path, "/"), "/")
		if len(segments) > 0 && segments[len(segments)-1] != "" {
			return segments[len(segments)-1]
		}
	}

	// Fallback to host
	if idx := strings.Index(name, "/"); idx >= 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}

	return name
}

// Verify CurlImporter implements Importer interface
var _ Importer = (*CurlImporter)(nil)
