package exporter

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/artpar/currier/internal/core"
)

// CurlExporter exports collections and requests to curl commands.
type CurlExporter struct {
	// Options
	Pretty      bool // Use line continuations for readability
	IncludeAuth bool // Include auth in output
}

// NewCurlExporter creates a new curl exporter.
func NewCurlExporter() *CurlExporter {
	return &CurlExporter{
		Pretty:      true,
		IncludeAuth: true,
	}
}

func (c *CurlExporter) Name() string {
	return "curl command"
}

func (c *CurlExporter) Format() Format {
	return FormatCurl
}

func (c *CurlExporter) FileExtension() string {
	return ".sh"
}

func (c *CurlExporter) Export(ctx context.Context, coll *core.Collection) ([]byte, error) {
	if coll == nil {
		return nil, ErrInvalidCollection
	}

	var sb strings.Builder

	// Write header comment
	sb.WriteString(fmt.Sprintf("#!/bin/bash\n# Collection: %s\n", coll.Name()))
	if coll.Description() != "" {
		sb.WriteString(fmt.Sprintf("# %s\n", coll.Description()))
	}
	sb.WriteString("\n")

	// Export root-level requests
	for _, req := range coll.Requests() {
		cmd, err := c.ExportRequest(ctx, req)
		if err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("# %s\n", req.Name()))
		sb.Write(cmd)
		sb.WriteString("\n\n")
	}

	// Export folders recursively
	c.exportFolders(&sb, coll.Folders(), ctx)

	return []byte(sb.String()), nil
}

func (c *CurlExporter) exportFolders(sb *strings.Builder, folders []*core.Folder, ctx context.Context) {
	for _, folder := range folders {
		sb.WriteString(fmt.Sprintf("# === %s ===\n", folder.Name()))
		if folder.Description() != "" {
			sb.WriteString(fmt.Sprintf("# %s\n", folder.Description()))
		}
		sb.WriteString("\n")

		for _, req := range folder.Requests() {
			cmd, err := c.ExportRequest(ctx, req)
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("# %s\n", req.Name()))
			sb.Write(cmd)
			sb.WriteString("\n\n")
		}

		// Recurse into subfolders
		c.exportFolders(sb, folder.Folders(), ctx)
	}
}

// ExportRequest exports a single request to a curl command.
func (c *CurlExporter) ExportRequest(ctx context.Context, req *core.RequestDefinition) ([]byte, error) {
	if req == nil {
		return nil, ErrInvalidCollection
	}

	var parts []string
	parts = append(parts, "curl")

	// Method (only if not GET)
	if req.Method() != "GET" {
		parts = append(parts, "-X", req.Method())
	}

	// Headers
	headers := req.Headers()
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Deterministic order

	for _, key := range keys {
		value := headers[key]
		parts = append(parts, "-H", fmt.Sprintf("%s: %s", key, value))
	}

	// Body
	body := req.Body()
	if body != "" {
		// Use --data-raw for safety
		parts = append(parts, "--data-raw", body)
	}

	// Auth
	if c.IncludeAuth && req.Auth() != nil {
		auth := req.Auth()
		switch auth.Type {
		case "basic":
			if auth.Password != "" {
				parts = append(parts, "-u", fmt.Sprintf("%s:%s", auth.Username, auth.Password))
			} else {
				parts = append(parts, "-u", auth.Username)
			}
		case "bearer":
			parts = append(parts, "-H", fmt.Sprintf("Authorization: Bearer %s", auth.Token))
		}
	}

	// URL (always last)
	parts = append(parts, req.URL())

	// Format output
	if c.Pretty {
		return []byte(formatPrettyCurl(parts)), nil
	}

	return []byte(formatInlineCurl(parts)), nil
}

func formatInlineCurl(parts []string) string {
	var result strings.Builder
	for i, part := range parts {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(shellQuote(part))
	}
	return result.String()
}

func formatPrettyCurl(parts []string) string {
	var result strings.Builder
	result.WriteString("curl")

	for i := 1; i < len(parts); i++ {
		part := parts[i]

		// Check if this is an option with a value
		if strings.HasPrefix(part, "-") && i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
			result.WriteString(" \\\n  ")
			result.WriteString(shellQuote(part))
			result.WriteString(" ")
			i++
			result.WriteString(shellQuote(parts[i]))
		} else if strings.HasPrefix(part, "http") {
			// URL - put on its own line
			result.WriteString(" \\\n  ")
			result.WriteString(shellQuote(part))
		} else {
			result.WriteString(" ")
			result.WriteString(shellQuote(part))
		}
	}

	return result.String()
}

func shellQuote(s string) string {
	// If string contains special characters, wrap in single quotes
	needsQuote := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '\'' ||
			r == '$' || r == '`' || r == '\\' || r == '!' || r == '*' ||
			r == '?' || r == '[' || r == ']' || r == '{' || r == '}' ||
			r == '(' || r == ')' || r == '<' || r == '>' || r == '|' ||
			r == '&' || r == ';' {
			needsQuote = true
			break
		}
	}

	if !needsQuote {
		return s
	}

	// Use single quotes and escape any single quotes in the string
	escaped := strings.ReplaceAll(s, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// Verify CurlExporter implements Exporter and RequestExporter interfaces
var _ Exporter = (*CurlExporter)(nil)
var _ RequestExporter = (*CurlExporter)(nil)
