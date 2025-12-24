package importer

import (
	"context"
	"errors"

	"github.com/artpar/currier/internal/core"
)

// Common errors
var (
	ErrInvalidFormat   = errors.New("invalid format")
	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrMissingRequired = errors.New("missing required field")
	ErrParseError      = errors.New("parse error")
)

// Format represents a supported import format.
type Format string

const (
	FormatAuto     Format = "auto"
	FormatPostman  Format = "postman"
	FormatOpenAPI  Format = "openapi"
	FormatSwagger  Format = "swagger"
	FormatCurl     Format = "curl"
	FormatHAR      Format = "har"
	FormatInsomnia Format = "insomnia"
)

// Importer defines the interface for importing collections from external formats.
type Importer interface {
	// Name returns the name of this importer.
	Name() string

	// Format returns the format this importer handles.
	Format() Format

	// FileExtensions returns the file extensions this importer can handle.
	FileExtensions() []string

	// DetectFormat checks if the content matches this importer's format.
	DetectFormat(content []byte) bool

	// Import parses the content and returns a collection.
	Import(ctx context.Context, content []byte) (*core.Collection, error)
}

// ImportResult contains the result of an import operation.
type ImportResult struct {
	Collection     *core.Collection
	RequestCount   int
	FolderCount    int
	VariableCount  int
	Warnings       []string
	SourceFormat   Format
	SourceVersion  string
}

// Registry holds all registered importers.
type Registry struct {
	importers map[Format]Importer
}

// NewRegistry creates a new importer registry.
func NewRegistry() *Registry {
	return &Registry{
		importers: make(map[Format]Importer),
	}
}

// Register adds an importer to the registry.
func (r *Registry) Register(imp Importer) {
	r.importers[imp.Format()] = imp
}

// Get returns an importer by format.
func (r *Registry) Get(format Format) (Importer, bool) {
	imp, ok := r.importers[format]
	return imp, ok
}

// DetectAndImport automatically detects the format and imports the content.
func (r *Registry) DetectAndImport(ctx context.Context, content []byte) (*ImportResult, error) {
	for _, imp := range r.importers {
		if imp.DetectFormat(content) {
			coll, err := imp.Import(ctx, content)
			if err != nil {
				return nil, err
			}
			return &ImportResult{
				Collection:   coll,
				SourceFormat: imp.Format(),
			}, nil
		}
	}
	return nil, ErrInvalidFormat
}

// Import imports content using the specified format.
func (r *Registry) Import(ctx context.Context, format Format, content []byte) (*ImportResult, error) {
	if format == FormatAuto {
		return r.DetectAndImport(ctx, content)
	}

	imp, ok := r.importers[format]
	if !ok {
		return nil, ErrInvalidFormat
	}

	coll, err := imp.Import(ctx, content)
	if err != nil {
		return nil, err
	}

	return &ImportResult{
		Collection:   coll,
		SourceFormat: format,
	}, nil
}

// ListFormats returns all registered formats.
func (r *Registry) ListFormats() []Format {
	formats := make([]Format, 0, len(r.importers))
	for f := range r.importers {
		formats = append(formats, f)
	}
	return formats
}
