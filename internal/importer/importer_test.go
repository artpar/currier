package importer

import (
	"context"
	"testing"

	"github.com/artpar/currier/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockImporter is a test importer
type mockImporter struct {
	name       string
	format     Format
	extensions []string
	detectFn   func([]byte) bool
	importFn   func(context.Context, []byte) (*core.Collection, error)
}

func (m *mockImporter) Name() string            { return m.name }
func (m *mockImporter) Format() Format          { return m.format }
func (m *mockImporter) FileExtensions() []string { return m.extensions }
func (m *mockImporter) DetectFormat(content []byte) bool {
	if m.detectFn != nil {
		return m.detectFn(content)
	}
	return false
}
func (m *mockImporter) Import(ctx context.Context, content []byte) (*core.Collection, error) {
	if m.importFn != nil {
		return m.importFn(ctx, content)
	}
	return core.NewCollection("Mock"), nil
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	imp := &mockImporter{
		name:   "Test Importer",
		format: FormatPostman,
	}

	registry.Register(imp)

	got, ok := registry.Get(FormatPostman)
	assert.True(t, ok)
	assert.Equal(t, imp, got)
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	t.Run("returns registered importer", func(t *testing.T) {
		imp := &mockImporter{format: FormatCurl}
		registry.Register(imp)

		got, ok := registry.Get(FormatCurl)
		assert.True(t, ok)
		assert.Equal(t, imp, got)
	})

	t.Run("returns false for unregistered format", func(t *testing.T) {
		_, ok := registry.Get(FormatInsomnia)
		assert.False(t, ok)
	})
}

func TestRegistry_Import(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	postmanImp := &mockImporter{
		format: FormatPostman,
		importFn: func(ctx context.Context, content []byte) (*core.Collection, error) {
			return core.NewCollection("Postman Collection"), nil
		},
	}
	registry.Register(postmanImp)

	t.Run("imports with specified format", func(t *testing.T) {
		result, err := registry.Import(ctx, FormatPostman, []byte(`{}`))
		require.NoError(t, err)
		assert.Equal(t, FormatPostman, result.SourceFormat)
		assert.Equal(t, "Postman Collection", result.Collection.Name())
	})

	t.Run("returns error for unknown format", func(t *testing.T) {
		_, err := registry.Import(ctx, FormatInsomnia, []byte(`{}`))
		assert.ErrorIs(t, err, ErrInvalidFormat)
	})
}

func TestRegistry_DetectAndImport(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	postmanImp := &mockImporter{
		format: FormatPostman,
		detectFn: func(content []byte) bool {
			return len(content) > 0 && content[0] == '{'
		},
		importFn: func(ctx context.Context, content []byte) (*core.Collection, error) {
			return core.NewCollection("Detected Postman"), nil
		},
	}
	registry.Register(postmanImp)

	curlImp := &mockImporter{
		format: FormatCurl,
		detectFn: func(content []byte) bool {
			return len(content) >= 4 && string(content[:4]) == "curl"
		},
		importFn: func(ctx context.Context, content []byte) (*core.Collection, error) {
			return core.NewCollection("Detected Curl"), nil
		},
	}
	registry.Register(curlImp)

	t.Run("detects and imports JSON format", func(t *testing.T) {
		result, err := registry.DetectAndImport(ctx, []byte(`{"info": {}}`))
		require.NoError(t, err)
		assert.Equal(t, FormatPostman, result.SourceFormat)
	})

	t.Run("detects and imports curl format", func(t *testing.T) {
		result, err := registry.DetectAndImport(ctx, []byte(`curl -X GET https://example.com`))
		require.NoError(t, err)
		assert.Equal(t, FormatCurl, result.SourceFormat)
	})

	t.Run("returns error when no format matches", func(t *testing.T) {
		_, err := registry.DetectAndImport(ctx, []byte(`unknown format`))
		assert.ErrorIs(t, err, ErrInvalidFormat)
	})
}

func TestRegistry_ImportAuto(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	imp := &mockImporter{
		format: FormatPostman,
		detectFn: func(content []byte) bool {
			return true
		},
		importFn: func(ctx context.Context, content []byte) (*core.Collection, error) {
			return core.NewCollection("Auto Detected"), nil
		},
	}
	registry.Register(imp)

	result, err := registry.Import(ctx, FormatAuto, []byte(`{}`))
	require.NoError(t, err)
	assert.Equal(t, "Auto Detected", result.Collection.Name())
}

func TestRegistry_ListFormats(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&mockImporter{format: FormatPostman})
	registry.Register(&mockImporter{format: FormatCurl})
	registry.Register(&mockImporter{format: FormatHAR})

	formats := registry.ListFormats()
	assert.Len(t, formats, 3)
	assert.Contains(t, formats, FormatPostman)
	assert.Contains(t, formats, FormatCurl)
	assert.Contains(t, formats, FormatHAR)
}
