package interfaces

import (
	"context"
)

// Importer reads external formats and converts them to Currier format.
type Importer interface {
	// Name returns the importer name (e.g., "postman", "openapi").
	Name() string

	// DisplayName returns a human-readable name.
	DisplayName() string

	// FileExtensions returns supported file extensions.
	FileExtensions() []string

	// DetectFormat checks if the content matches this format.
	DetectFormat(content []byte) bool

	// Import converts the content to a Currier collection.
	Import(ctx context.Context, content []byte, opts ImportOptions) (ImportResult, error)
}

// ImportOptions configures import behavior.
type ImportOptions struct {
	// Name overrides the imported collection name.
	Name string

	// BasePath is the directory to save imported files.
	BasePath string

	// IncludeEnvironments imports environments if available.
	IncludeEnvironments bool

	// MergeWithExisting merges with an existing collection.
	MergeWithExisting bool

	// ExistingCollectionID is the ID of the collection to merge with.
	ExistingCollectionID string
}

// ImportResult contains the results of an import operation.
type ImportResult struct {
	// Collection is the imported collection.
	Collection Collection

	// Environments are the imported environments.
	Environments []Environment

	// RequestCount is the number of requests imported.
	RequestCount int

	// Warnings contains non-fatal issues encountered during import.
	Warnings []string

	// Errors contains errors that prevented some items from being imported.
	Errors []ImportError
}

// ImportError describes an import error for a specific item.
type ImportError struct {
	Item    string
	Message string
	Details string
}

// Exporter writes Currier format to external formats.
type Exporter interface {
	// Name returns the exporter name.
	Name() string

	// DisplayName returns a human-readable name.
	DisplayName() string

	// FileExtension returns the file extension for exported files.
	FileExtension() string

	// Export converts a collection to the external format.
	Export(ctx context.Context, c Collection, opts ExportOptions) ([]byte, error)
}

// ExportOptions configures export behavior.
type ExportOptions struct {
	// IncludeEnvironment includes environment variables.
	IncludeEnvironment bool

	// Environment is the environment to include.
	Environment Environment

	// IncludeScripts includes pre/post scripts.
	IncludeScripts bool

	// IncludeTests includes test definitions.
	IncludeTests bool

	// PrettyPrint formats output for readability.
	PrettyPrint bool

	// IndentSize is the indentation size for pretty printing.
	IndentSize int
}

// ImporterRegistry manages available importers.
type ImporterRegistry interface {
	// Register adds an importer.
	Register(importer Importer)

	// Get retrieves an importer by name.
	Get(name string) (Importer, bool)

	// List returns all registered importers.
	List() []Importer

	// DetectFormat finds the best importer for the given content.
	DetectFormat(content []byte) (Importer, bool)

	// GetByExtension finds importers supporting the given extension.
	GetByExtension(ext string) []Importer
}

// ExporterRegistry manages available exporters.
type ExporterRegistry interface {
	// Register adds an exporter.
	Register(exporter Exporter)

	// Get retrieves an exporter by name.
	Get(name string) (Exporter, bool)

	// List returns all registered exporters.
	List() []Exporter
}
