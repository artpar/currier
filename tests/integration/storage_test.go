package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/storage/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStorageIntegration_CollectionRoundTrip tests full save/load cycle for collections
func TestStorageIntegration_CollectionRoundTrip(t *testing.T) {
	t.Run("preserves all collection data through save/load cycle", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		// Create a complex collection
		c := core.NewCollection("My API")
		c.SetDescription("A comprehensive API collection")
		c.SetVersion("2.1.0")
		c.SetVariable("base_url", "https://api.example.com")
		c.SetVariable("api_version", "v2")
		c.SetAuth(core.AuthConfig{
			Type:  "bearer",
			Token: "{{access_token}}",
		})
		c.SetPreScript("console.log('Pre-request');")
		c.SetPostScript("console.log('Post-response');")

		// Add folders with nested structure
		usersFolder := c.AddFolder("Users")
		usersFolder.SetDescription("User management endpoints")

		adminFolder := usersFolder.AddFolder("Admin")
		adminFolder.SetDescription("Admin-only endpoints")

		// Add requests
		listUsers := core.NewRequestDefinition("List Users", "GET", "{{base_url}}/users")
		listUsers.SetHeader("Accept", "application/json")
		listUsers.SetHeader("Authorization", "Bearer {{token}}")
		usersFolder.AddRequest(listUsers)

		createUser := core.NewRequestDefinition("Create User", "POST", "{{base_url}}/users")
		createUser.SetHeader("Content-Type", "application/json")
		createUser.SetBodyRaw(`{"name": "{{username}}"}`, "json")
		createUser.SetPreScript("currier.setVariable('timestamp', Date.now());")
		usersFolder.AddRequest(createUser)

		deleteAdmin := core.NewRequestDefinition("Delete Admin", "DELETE", "{{base_url}}/admin/{{id}}")
		adminFolder.AddRequest(deleteAdmin)

		// Root-level request
		healthCheck := core.NewRequestDefinition("Health Check", "GET", "{{base_url}}/health")
		c.AddRequest(healthCheck)

		// Save
		err := store.Save(ctx, c)
		require.NoError(t, err)

		// Load
		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)

		// Verify all data
		assert.Equal(t, c.ID(), loaded.ID())
		assert.Equal(t, "My API", loaded.Name())
		assert.Equal(t, "A comprehensive API collection", loaded.Description())
		assert.Equal(t, "2.1.0", loaded.Version())
		assert.Equal(t, "https://api.example.com", loaded.GetVariable("base_url"))
		assert.Equal(t, "v2", loaded.GetVariable("api_version"))
		assert.Equal(t, "bearer", loaded.Auth().Type)
		assert.Equal(t, "{{access_token}}", loaded.Auth().Token)
		assert.Equal(t, "console.log('Pre-request');", loaded.PreScript())
		assert.Equal(t, "console.log('Post-response');", loaded.PostScript())

		// Verify folder structure
		require.Len(t, loaded.Folders(), 1)
		loadedUsers := loaded.Folders()[0]
		assert.Equal(t, "Users", loadedUsers.Name())
		assert.Equal(t, "User management endpoints", loadedUsers.Description())
		require.Len(t, loadedUsers.Requests(), 2)
		require.Len(t, loadedUsers.Folders(), 1)

		loadedAdmin := loadedUsers.Folders()[0]
		assert.Equal(t, "Admin", loadedAdmin.Name())
		require.Len(t, loadedAdmin.Requests(), 1)

		// Verify requests
		require.Len(t, loaded.Requests(), 1)
		assert.Equal(t, "Health Check", loaded.Requests()[0].Name())
	})

	t.Run("handles multiple collections", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		// Create multiple collections
		collections := make([]*core.Collection, 5)
		for i := 0; i < 5; i++ {
			c := core.NewCollection("API " + string(rune('A'+i)))
			c.SetVariable("index", string(rune('0'+i)))
			collections[i] = c
			require.NoError(t, store.Save(ctx, c))
		}

		// List and verify
		list, err := store.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 5)

		// Load each and verify
		for _, c := range collections {
			loaded, err := store.Get(ctx, c.ID())
			require.NoError(t, err)
			assert.Equal(t, c.Name(), loaded.Name())
		}
	})

	t.Run("search finds collections by name and description", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		c1 := core.NewCollection("User Service API")
		c1.SetDescription("Handles user authentication")

		c2 := core.NewCollection("Payment Gateway")
		c2.SetDescription("Processes payments")

		c3 := core.NewCollection("Notification API")
		c3.SetDescription("Sends user notifications")

		require.NoError(t, store.Save(ctx, c1))
		require.NoError(t, store.Save(ctx, c2))
		require.NoError(t, store.Save(ctx, c3))

		// Search by name
		results, err := store.Search(ctx, "API")
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// Search by description
		results, err = store.Search(ctx, "user")
		require.NoError(t, err)
		assert.Len(t, results, 2) // User Service API and Notification API (user notifications)
	})
}

// TestStorageIntegration_EnvironmentRoundTrip tests full save/load cycle for environments
func TestStorageIntegration_EnvironmentRoundTrip(t *testing.T) {
	t.Run("preserves all environment data through save/load cycle", func(t *testing.T) {
		store := newTestEnvironmentStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Production")
		env.SetDescription("Production environment with live endpoints")
		env.SetVariable("base_url", "https://api.example.com")
		env.SetVariable("timeout", "30000")
		env.SetVariable("retry_count", "3")
		env.SetSecret("api_key", "prod-secret-key-123")
		env.SetSecret("db_password", "super-secure-password")
		env.SetGlobal(false)

		// Save
		err := store.Save(ctx, env)
		require.NoError(t, err)

		// Load
		loaded, err := store.Get(ctx, env.ID())
		require.NoError(t, err)

		// Verify
		assert.Equal(t, env.ID(), loaded.ID())
		assert.Equal(t, "Production", loaded.Name())
		assert.Equal(t, "Production environment with live endpoints", loaded.Description())
		assert.Equal(t, "https://api.example.com", loaded.GetVariable("base_url"))
		assert.Equal(t, "30000", loaded.GetVariable("timeout"))
		assert.Equal(t, "prod-secret-key-123", loaded.GetSecret("api_key"))
		assert.Equal(t, "super-secure-password", loaded.GetSecret("db_password"))
		assert.False(t, loaded.IsGlobal())
	})

	t.Run("handles environment switching", func(t *testing.T) {
		store := newTestEnvironmentStore(t)
		ctx := context.Background()

		dev := core.NewEnvironment("Development")
		dev.SetVariable("base_url", "http://localhost:3000")

		staging := core.NewEnvironment("Staging")
		staging.SetVariable("base_url", "https://staging.example.com")

		prod := core.NewEnvironment("Production")
		prod.SetVariable("base_url", "https://api.example.com")

		require.NoError(t, store.Save(ctx, dev))
		require.NoError(t, store.Save(ctx, staging))
		require.NoError(t, store.Save(ctx, prod))

		// Set dev as active
		require.NoError(t, store.SetActive(ctx, dev.ID()))
		active, err := store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Development", active.Name())

		// Switch to staging
		require.NoError(t, store.SetActive(ctx, staging.ID()))
		active, err = store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Staging", active.Name())

		// Verify dev is no longer active
		loadedDev, err := store.Get(ctx, dev.ID())
		require.NoError(t, err)
		assert.False(t, loadedDev.IsActive())

		// Switch to production
		require.NoError(t, store.SetActive(ctx, prod.ID()))
		active, err = store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Production", active.Name())
	})

	t.Run("get by name works correctly", func(t *testing.T) {
		store := newTestEnvironmentStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("MySpecialEnv")
		env.SetVariable("key", "value")
		require.NoError(t, store.Save(ctx, env))

		loaded, err := store.GetByName(ctx, "MySpecialEnv")
		require.NoError(t, err)
		assert.Equal(t, env.ID(), loaded.ID())
		assert.Equal(t, "value", loaded.GetVariable("key"))
	})
}

// TestStorageIntegration_CollectionWithEnvironment tests using collections with environments
func TestStorageIntegration_CollectionWithEnvironment(t *testing.T) {
	t.Run("collection variables can reference environment variables", func(t *testing.T) {
		tmpDir := t.TempDir()
		collStore, err := filesystem.NewCollectionStore(filepath.Join(tmpDir, "collections"))
		require.NoError(t, err)
		envStore, err := filesystem.NewEnvironmentStore(filepath.Join(tmpDir, "environments"))
		require.NoError(t, err)
		ctx := context.Background()

		// Create environment
		env := core.NewEnvironment("Development")
		env.SetVariable("host", "localhost")
		env.SetVariable("port", "3000")
		env.SetSecret("api_key", "dev-key-123")
		require.NoError(t, envStore.Save(ctx, env))
		require.NoError(t, envStore.SetActive(ctx, env.ID()))

		// Create collection that uses environment variables
		coll := core.NewCollection("My API")
		coll.SetVariable("base_url", "http://{{host}}:{{port}}")

		req := core.NewRequestDefinition("Get Data", "GET", "{{base_url}}/api/data")
		req.SetHeader("X-API-Key", "{{api_key}}")
		coll.AddRequest(req)

		require.NoError(t, collStore.Save(ctx, coll))

		// Load both and verify they work together
		loadedEnv, err := envStore.GetActive(ctx)
		require.NoError(t, err)
		loadedColl, err := collStore.Get(ctx, coll.ID())
		require.NoError(t, err)

		// Simulate variable resolution
		allVars := loadedEnv.ExportAll()
		for k, v := range loadedColl.Variables() {
			allVars[k] = v
		}

		assert.Equal(t, "localhost", allVars["host"])
		assert.Equal(t, "3000", allVars["port"])
		assert.Equal(t, "dev-key-123", allVars["api_key"])
		assert.Equal(t, "http://{{host}}:{{port}}", allVars["base_url"])
	})
}

// TestStorageIntegration_Persistence tests data survives process restart
func TestStorageIntegration_Persistence(t *testing.T) {
	t.Run("data persists after store recreation", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := context.Background()

		// Create and save with first store instance
		store1, err := filesystem.NewCollectionStore(tmpDir)
		require.NoError(t, err)

		c := core.NewCollection("Persistent API")
		c.SetVariable("key", "value")
		folder := c.AddFolder("Folder")
		req := core.NewRequestDefinition("Request", "GET", "/test")
		folder.AddRequest(req)
		require.NoError(t, store1.Save(ctx, c))

		// Create new store instance (simulating restart)
		store2, err := filesystem.NewCollectionStore(tmpDir)
		require.NoError(t, err)

		// Verify data is still there
		loaded, err := store2.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, "Persistent API", loaded.Name())
		assert.Equal(t, "value", loaded.GetVariable("key"))
		assert.Len(t, loaded.Folders(), 1)
		assert.Len(t, loaded.Folders()[0].Requests(), 1)
	})
}

// TestStorageIntegration_EdgeCases tests edge cases and error handling
func TestStorageIntegration_EdgeCases(t *testing.T) {
	t.Run("handles empty collection", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		c := core.NewCollection("Empty")
		require.NoError(t, store.Save(ctx, c))

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Empty(t, loaded.Folders())
		assert.Empty(t, loaded.Requests())
		assert.Empty(t, loaded.Variables())
	})

	t.Run("handles special characters in names", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		c := core.NewCollection("API with spaces & special-chars_v2.0")
		c.SetDescription("Description with 日本語 and émojis 🚀")
		c.SetVariable("url", "https://example.com/path?query=value&other=123")
		require.NoError(t, store.Save(ctx, c))

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, "API with spaces & special-chars_v2.0", loaded.Name())
		assert.Equal(t, "Description with 日本語 and émojis 🚀", loaded.Description())
		assert.Equal(t, "https://example.com/path?query=value&other=123", loaded.GetVariable("url"))
	})

	t.Run("handles large request body", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		c := core.NewCollection("Large Body API")

		// Create a large body
		largeBody := make([]byte, 100000)
		for i := range largeBody {
			largeBody[i] = byte('a' + (i % 26))
		}

		req := core.NewRequestDefinition("Large Request", "POST", "/upload")
		req.SetBodyRaw(string(largeBody), "text")
		c.AddRequest(req)

		require.NoError(t, store.Save(ctx, c))

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, string(largeBody), loaded.Requests()[0].BodyContent())
	})

	t.Run("handles deeply nested folders", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		c := core.NewCollection("Deep Nesting")
		current := c.AddFolder("Level1")
		for i := 2; i <= 10; i++ {
			current = current.AddFolder("Level" + string(rune('0'+i)))
		}

		req := core.NewRequestDefinition("Deep Request", "GET", "/deep")
		current.AddRequest(req)

		require.NoError(t, store.Save(ctx, c))

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)

		// Navigate to the deepest folder
		loadedCurrent := loaded.Folders()[0]
		for i := 2; i <= 10; i++ {
			require.Len(t, loadedCurrent.Folders(), 1)
			loadedCurrent = loadedCurrent.Folders()[0]
		}
		require.Len(t, loadedCurrent.Requests(), 1)
		assert.Equal(t, "Deep Request", loadedCurrent.Requests()[0].Name())
	})

	t.Run("delete removes file from disk", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := filesystem.NewCollectionStore(tmpDir)
		require.NoError(t, err)
		ctx := context.Background()

		c := core.NewCollection("ToDelete")
		require.NoError(t, store.Save(ctx, c))

		// Verify file exists
		path := filepath.Join(tmpDir, c.ID()+".yaml")
		_, err = os.Stat(path)
		require.NoError(t, err)

		// Delete
		require.NoError(t, store.Delete(ctx, c.ID()))

		// Verify file is gone
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})
}

// TestStorageIntegration_ConcurrentAccess tests concurrent operations
func TestStorageIntegration_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent reads", func(t *testing.T) {
		store := newTestCollectionStore(t)
		ctx := context.Background()

		c := core.NewCollection("Concurrent Read Test")
		c.SetVariable("key", "value")
		require.NoError(t, store.Save(ctx, c))

		// Concurrent reads
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				loaded, err := store.Get(ctx, c.ID())
				assert.NoError(t, err)
				assert.Equal(t, "value", loaded.GetVariable("key"))
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// Helper functions

func newTestCollectionStore(t *testing.T) *filesystem.CollectionStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := filesystem.NewCollectionStore(tmpDir)
	require.NoError(t, err)
	return store
}

func newTestEnvironmentStore(t *testing.T) *filesystem.EnvironmentStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := filesystem.NewEnvironmentStore(tmpDir)
	require.NoError(t, err)
	return store
}
