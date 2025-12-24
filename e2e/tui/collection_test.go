package tui_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/artpar/currier/e2e/harness"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTUI_CollectionLoading(t *testing.T) {
	h := harness.New(t, harness.Config{
		Timeout: 5 * time.Second,
	})

	t.Run("loads Postman collection from file", func(t *testing.T) {
		// Load the test collection file
		content, err := os.ReadFile("../../testdata/imports/sample_postman.json")
		require.NoError(t, err, "failed to read test collection file")

		// Import it using the registry
		registry := importer.NewRegistry()
		registry.Register(importer.NewPostmanImporter())

		result, err := registry.DetectAndImport(context.Background(), content)
		require.NoError(t, err, "failed to import collection")
		require.NotNil(t, result.Collection)

		// Verify collection name
		assert.Equal(t, "Sample API Collection", result.Collection.Name())
		assert.Equal(t, "postman", string(result.SourceFormat))
	})

	t.Run("TUI displays loaded collection", func(t *testing.T) {
		// Create a test collection with requests
		collection := core.NewCollection("Test Collection")
		folder := collection.AddFolder("API Endpoints")
		folder.AddRequest(core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users"))
		folder.AddRequest(core.NewRequestDefinition("Create User", "POST", "https://api.example.com/users"))

		// Start TUI with the collection
		session := h.TUI().StartWithCollections(t, []*core.Collection{collection})
		defer session.Quit()

		// Verify collection appears in output
		output := session.Output()
		assert.Contains(t, output, "Test Collection")
	})

	t.Run("TUI displays multiple collections", func(t *testing.T) {
		// Create multiple collections
		collection1 := core.NewCollection("Users API")
		collection1.AddFolder("Users").AddRequest(
			core.NewRequestDefinition("List Users", "GET", "https://api.example.com/users"),
		)

		collection2 := core.NewCollection("Posts API")
		collection2.AddFolder("Posts").AddRequest(
			core.NewRequestDefinition("List Posts", "GET", "https://api.example.com/posts"),
		)

		// Start TUI with multiple collections
		session := h.TUI().StartWithCollections(t, []*core.Collection{collection1, collection2})
		defer session.Quit()

		// Verify both collections appear
		output := session.Output()
		assert.Contains(t, output, "Users API")
		assert.Contains(t, output, "Posts API")
	})

	t.Run("can navigate to collection item", func(t *testing.T) {
		// Create a collection
		collection := core.NewCollection("Navigation Test")
		folder := collection.AddFolder("Requests")
		folder.AddRequest(core.NewRequestDefinition("Test Request", "GET", "https://api.example.com/test"))

		session := h.TUI().StartWithCollections(t, []*core.Collection{collection})
		defer session.Quit()

		// Navigate down in the collection tree
		session.SendKey("j")
		session.SendKey("j")

		// Verify no error in output
		output := session.Output()
		assert.NotContains(t, output, "Error")
	})

	t.Run("loads collection with nested folders", func(t *testing.T) {
		// Load the test collection file
		content, err := os.ReadFile("../../testdata/imports/sample_postman.json")
		require.NoError(t, err)

		registry := importer.NewRegistry()
		registry.Register(importer.NewPostmanImporter())

		result, err := registry.DetectAndImport(context.Background(), content)
		require.NoError(t, err)

		// Start TUI with loaded collection
		session := h.TUI().StartWithCollections(t, []*core.Collection{result.Collection})
		defer session.Quit()

		// Collection should show in output (may be truncated due to panel width)
		output := session.Output()
		assert.Contains(t, output, "Sample API")
	})
}

func TestTUI_CollectionInteraction(t *testing.T) {
	h := harness.New(t, harness.Config{
		Timeout: 5 * time.Second,
	})

	t.Run("expand folder with l key", func(t *testing.T) {
		collection := core.NewCollection("Test API")
		folder := collection.AddFolder("Endpoints")
		folder.AddRequest(core.NewRequestDefinition("Get Data", "GET", "https://api.example.com/data"))

		session := h.TUI().StartWithCollections(t, []*core.Collection{collection})
		defer session.Quit()

		// Try expanding with l
		session.SendKey("l")

		output := session.Output()
		assert.NotContains(t, output, "Error")
	})

	t.Run("collapse folder with h key", func(t *testing.T) {
		collection := core.NewCollection("Test API")
		folder := collection.AddFolder("Endpoints")
		folder.AddRequest(core.NewRequestDefinition("Get Data", "GET", "https://api.example.com/data"))

		session := h.TUI().StartWithCollections(t, []*core.Collection{collection})
		defer session.Quit()

		// Expand then collapse
		session.SendKey("l")
		session.SendKey("h")

		output := session.Output()
		assert.NotContains(t, output, "Error")
	})
}
