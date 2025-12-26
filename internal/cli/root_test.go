package cli

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/tui/views"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommand(t *testing.T) {
	t.Run("creates root command", func(t *testing.T) {
		cmd := NewRootCommand("1.0.0")
		assert.NotNil(t, cmd)
		assert.Equal(t, "currier", cmd.Use)
		assert.Equal(t, "1.0.0", cmd.Version)
	})

	t.Run("has import flag", func(t *testing.T) {
		cmd := NewRootCommand("1.0.0")
		flag := cmd.Flags().Lookup("import")
		require.NotNil(t, flag)
		assert.Equal(t, "i", flag.Shorthand)
	})

	t.Run("has env flag", func(t *testing.T) {
		cmd := NewRootCommand("1.0.0")
		flag := cmd.Flags().Lookup("env")
		require.NotNil(t, flag)
		assert.Equal(t, "e", flag.Shorthand)
	})

	t.Run("has send subcommand", func(t *testing.T) {
		cmd := NewRootCommand("1.0.0")
		sendCmd, _, err := cmd.Find([]string{"send"})
		require.NoError(t, err)
		assert.Contains(t, sendCmd.Use, "send")
	})
}

func TestNewImporterRegistry(t *testing.T) {
	t.Run("creates registry with all importers", func(t *testing.T) {
		registry := newImporterRegistry()
		assert.NotNil(t, registry)
	})
}

func TestLoadCollections(t *testing.T) {
	t.Run("returns nil for empty paths", func(t *testing.T) {
		collections, err := loadCollections(nil)
		assert.NoError(t, err)
		assert.Nil(t, collections)
	})

	t.Run("returns nil for empty slice", func(t *testing.T) {
		collections, err := loadCollections([]string{})
		assert.NoError(t, err)
		assert.Nil(t, collections)
	})

	t.Run("loads valid collection file", func(t *testing.T) {
		// Create a temp Postman collection file
		tmpDir := t.TempDir()
		collectionPath := filepath.Join(tmpDir, "collection.json")

		content := `{
			"info": {
				"name": "Test Collection",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			},
			"item": []
		}`

		err := os.WriteFile(collectionPath, []byte(content), 0644)
		require.NoError(t, err)

		collections, err := loadCollections([]string{collectionPath})
		require.NoError(t, err)
		require.Len(t, collections, 1)
		assert.Equal(t, "Test Collection", collections[0].Name())
	})

	t.Run("fails for non-existent file", func(t *testing.T) {
		_, err := loadCollections([]string{"/non/existent/file.json"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("fails for invalid content", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.json")

		err := os.WriteFile(invalidPath, []byte("not valid json or collection"), 0644)
		require.NoError(t, err)

		_, err = loadCollections([]string{invalidPath})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to import")
	})
}

func TestTuiModel(t *testing.T) {
	t.Run("Init returns view init", func(t *testing.T) {
		view := views.NewMainView()
		model := tuiModel{view: view}
		cmd := model.Init()
		assert.Nil(t, cmd)
	})

	t.Run("Update handles messages", func(t *testing.T) {
		view := views.NewMainView()
		model := tuiModel{view: view}

		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		updated, _ := model.Update(msg)

		assert.NotNil(t, updated)
	})

	t.Run("View returns string", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(120, 40)
		model := tuiModel{view: view}

		output := model.View()
		assert.NotEmpty(t, output)
	})
}

func TestNewRootCommand_HasCurlSubcommand(t *testing.T) {
	cmd := NewRootCommand("1.0.0")
	curlCmd, _, err := cmd.Find([]string{"curl"})
	require.NoError(t, err)
	assert.Contains(t, curlCmd.Use, "curl")
}

func TestLoadCollections_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two collection files
	collection1 := filepath.Join(tmpDir, "collection1.json")
	content1 := `{
		"info": {
			"name": "Collection 1",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": []
	}`
	err := os.WriteFile(collection1, []byte(content1), 0644)
	require.NoError(t, err)

	collection2 := filepath.Join(tmpDir, "collection2.json")
	content2 := `{
		"info": {
			"name": "Collection 2",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": []
	}`
	err = os.WriteFile(collection2, []byte(content2), 0644)
	require.NoError(t, err)

	collections, err := loadCollections([]string{collection1, collection2})
	require.NoError(t, err)
	require.Len(t, collections, 2)

	names := []string{collections[0].Name(), collections[1].Name()}
	assert.Contains(t, names, "Collection 1")
	assert.Contains(t, names, "Collection 2")
}
