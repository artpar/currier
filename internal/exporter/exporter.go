package exporter

import (
	"context"
	"errors"

	"github.com/artpar/currier/internal/core"
)

// Common errors
var (
	ErrInvalidCollection = errors.New("invalid collection")
	ErrExportFailed      = errors.New("export failed")
)

// Format represents a supported export format.
type Format string

const (
	FormatPostman Format = "postman"
	FormatCurl    Format = "curl"
	FormatHAR     Format = "har"
	FormatOpenAPI Format = "openapi"
)

// Exporter defines the interface for exporting collections to external formats.
type Exporter interface {
	// Name returns the name of this exporter.
	Name() string

	// Format returns the format this exporter produces.
	Format() Format

	// FileExtension returns the file extension for exported files.
	FileExtension() string

	// Export converts the collection to the target format.
	Export(ctx context.Context, coll *core.Collection) ([]byte, error)
}

// RequestExporter exports individual requests (useful for curl, httpie, etc.)
type RequestExporter interface {
	// ExportRequest exports a single request.
	ExportRequest(ctx context.Context, req *core.RequestDefinition) ([]byte, error)
}

// ExportResult contains the result of an export operation.
type ExportResult struct {
	Content      []byte
	Format       Format
	FileExtension string
}

// Registry holds all registered exporters.
type Registry struct {
	exporters map[Format]Exporter
}

// NewRegistry creates a new exporter registry.
func NewRegistry() *Registry {
	return &Registry{
		exporters: make(map[Format]Exporter),
	}
}

// Register adds an exporter to the registry.
func (r *Registry) Register(exp Exporter) {
	r.exporters[exp.Format()] = exp
}

// Get returns an exporter by format.
func (r *Registry) Get(format Format) (Exporter, bool) {
	exp, ok := r.exporters[format]
	return exp, ok
}

// Export exports the collection using the specified format.
func (r *Registry) Export(ctx context.Context, format Format, coll *core.Collection) (*ExportResult, error) {
	exp, ok := r.exporters[format]
	if !ok {
		return nil, ErrExportFailed
	}

	content, err := exp.Export(ctx, coll)
	if err != nil {
		return nil, err
	}

	return &ExportResult{
		Content:       content,
		Format:        format,
		FileExtension: exp.FileExtension(),
	}, nil
}

// ListFormats returns all registered formats.
func (r *Registry) ListFormats() []Format {
	formats := make([]Format, 0, len(r.exporters))
	for f := range r.exporters {
		formats = append(formats, f)
	}
	return formats
}
