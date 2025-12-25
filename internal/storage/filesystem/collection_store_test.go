package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/currier/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollectionStore(t *testing.T) {
	t.Run("creates store with base path", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewCollectionStore(tmpDir)

		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("creates collections directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		collectionsDir := filepath.Join(tmpDir, "collections")

		_, err := NewCollectionStore(collectionsDir)
		require.NoError(t, err)

		info, err := os.Stat(collectionsDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestCollectionStore_Save(t *testing.T) {
	t.Run("saves new collection", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		c.SetDescription("Test API collection")
		c.SetVariable("base_url", "https://api.example.com")

		err := store.Save(ctx, c)
		require.NoError(t, err)

		// Verify file was created
		path := store.collectionPath(c.ID())
		_, err = os.Stat(path)
		assert.NoError(t, err)
	})

	t.Run("updates existing collection", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		err := store.Save(ctx, c)
		require.NoError(t, err)

		c.SetDescription("Updated description")
		err = store.Save(ctx, c)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, "Updated description", loaded.Description())
	})

	t.Run("saves collection with folders and requests", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		folder := c.AddFolder("Users")
		req := core.NewRequestDefinition("Get User", "GET", "https://api.example.com/users/1")
		folder.AddRequest(req)

		err := store.Save(ctx, c)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Len(t, loaded.Folders(), 1)
		assert.Len(t, loaded.Folders()[0].Requests(), 1)
	})
}

func TestCollectionStore_Get(t *testing.T) {
	t.Run("gets collection by ID", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		c.SetDescription("Test description")
		c.SetVersion("1.0.0")
		err := store.Save(ctx, c)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, c.ID(), loaded.ID())
		assert.Equal(t, "My API", loaded.Name())
		assert.Equal(t, "Test description", loaded.Description())
		assert.Equal(t, "1.0.0", loaded.Version())
	})

	t.Run("returns error for non-existent collection", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		_, err := store.Get(ctx, "non-existent-id")
		assert.Error(t, err)
	})

	t.Run("loads collection variables", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		c.SetVariable("base_url", "https://api.example.com")
		c.SetVariable("api_key", "secret123")
		err := store.Save(ctx, c)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com", loaded.GetVariable("base_url"))
		assert.Equal(t, "secret123", loaded.GetVariable("api_key"))
	})

	t.Run("loads collection auth config", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		c.SetAuth(core.AuthConfig{
			Type:  "bearer",
			Token: "{{access_token}}",
		})
		err := store.Save(ctx, c)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, "bearer", loaded.Auth().Type)
		assert.Equal(t, "{{access_token}}", loaded.Auth().Token)
	})
}

func TestCollectionStore_List(t *testing.T) {
	t.Run("lists all collections", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c1 := core.NewCollection("API 1")
		c2 := core.NewCollection("API 2")
		c3 := core.NewCollection("API 3")

		require.NoError(t, store.Save(ctx, c1))
		require.NoError(t, store.Save(ctx, c2))
		require.NoError(t, store.Save(ctx, c3))

		list, err := store.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})

	t.Run("returns empty list when no collections", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		list, err := store.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("list contains metadata", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		c.SetDescription("A test API")
		req := core.NewRequestDefinition("Get Users", "GET", "/users")
		c.AddRequest(req)
		require.NoError(t, store.Save(ctx, c))

		list, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)

		meta := list[0]
		assert.Equal(t, c.ID(), meta.ID)
		assert.Equal(t, "My API", meta.Name)
		assert.Equal(t, "A test API", meta.Description)
		assert.Equal(t, 1, meta.RequestCount)
	})
}

func TestCollectionStore_Delete(t *testing.T) {
	t.Run("deletes collection", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		require.NoError(t, store.Save(ctx, c))

		err := store.Delete(ctx, c.ID())
		require.NoError(t, err)

		_, err = store.Get(ctx, c.ID())
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent collection", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		err := store.Delete(ctx, "non-existent-id")
		assert.Error(t, err)
	})
}

func TestCollectionStore_Search(t *testing.T) {
	t.Run("searches by name", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c1 := core.NewCollection("User API")
		c2 := core.NewCollection("Product API")
		c3 := core.NewCollection("Order Service")

		require.NoError(t, store.Save(ctx, c1))
		require.NoError(t, store.Save(ctx, c2))
		require.NoError(t, store.Save(ctx, c3))

		results, err := store.Search(ctx, "API")
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("searches by description", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c1 := core.NewCollection("API 1")
		c1.SetDescription("Handles user authentication")
		c2 := core.NewCollection("API 2")
		c2.SetDescription("Handles payments")

		require.NoError(t, store.Save(ctx, c1))
		require.NoError(t, store.Save(ctx, c2))

		results, err := store.Search(ctx, "authentication")
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "API 1", results[0].Name)
	})

	t.Run("case insensitive search", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		require.NoError(t, store.Save(ctx, c))

		results, err := store.Search(ctx, "my api")
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		require.NoError(t, store.Save(ctx, c))

		results, err := store.Search(ctx, "xyz")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestCollectionStore_GetByPath(t *testing.T) {
	t.Run("gets collection by path", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")
		require.NoError(t, store.Save(ctx, c))

		path := store.collectionPath(c.ID())
		loaded, err := store.GetByPath(ctx, path)
		require.NoError(t, err)
		assert.Equal(t, c.ID(), loaded.ID())
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		_, err := store.GetByPath(ctx, "/nonexistent/path/collection.yaml")
		assert.Error(t, err)
	})
}

// Helper functions

func TestCollectionStore_CountRequests(t *testing.T) {
	t.Run("counts requests in nested folders", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("My API")

		// Add 2 root-level requests
		c.AddRequest(core.NewRequestDefinition("Root 1", "GET", "/root1"))
		c.AddRequest(core.NewRequestDefinition("Root 2", "POST", "/root2"))

		// Add folder with 3 requests
		folder1 := c.AddFolder("Users")
		folder1.AddRequest(core.NewRequestDefinition("Get Users", "GET", "/users"))
		folder1.AddRequest(core.NewRequestDefinition("Create User", "POST", "/users"))
		folder1.AddRequest(core.NewRequestDefinition("Delete User", "DELETE", "/users/1"))

		// Add nested subfolder with 2 requests
		subfolder := folder1.AddFolder("Admin")
		subfolder.AddRequest(core.NewRequestDefinition("Get Admins", "GET", "/admins"))
		subfolder.AddRequest(core.NewRequestDefinition("Create Admin", "POST", "/admins"))

		require.NoError(t, store.Save(ctx, c))

		list, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)

		// Total: 2 root + 3 folder + 2 subfolder = 7
		assert.Equal(t, 7, list[0].RequestCount)
	})

	t.Run("counts requests in multiple nested levels", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("Deep API")

		// Create deep nesting: folder > subfolder > subsubfolder
		folder := c.AddFolder("Level1")
		folder.AddRequest(core.NewRequestDefinition("L1 Request", "GET", "/l1"))

		subfolder := folder.AddFolder("Level2")
		subfolder.AddRequest(core.NewRequestDefinition("L2 Request", "GET", "/l2"))

		subsubfolder := subfolder.AddFolder("Level3")
		subsubfolder.AddRequest(core.NewRequestDefinition("L3 Request", "GET", "/l3"))

		require.NoError(t, store.Save(ctx, c))

		list, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)

		// Total: 1 + 1 + 1 = 3 requests at different nesting levels
		assert.Equal(t, 3, list[0].RequestCount)
	})

	t.Run("counts requests across multiple folders", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("Multi Folder API")

		// Add multiple folders at root level
		folder1 := c.AddFolder("Users")
		folder1.AddRequest(core.NewRequestDefinition("Get Users", "GET", "/users"))

		folder2 := c.AddFolder("Products")
		folder2.AddRequest(core.NewRequestDefinition("Get Products", "GET", "/products"))
		folder2.AddRequest(core.NewRequestDefinition("Create Product", "POST", "/products"))

		folder3 := c.AddFolder("Orders")
		folder3.AddRequest(core.NewRequestDefinition("Get Orders", "GET", "/orders"))

		require.NoError(t, store.Save(ctx, c))

		list, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)

		// Total: 1 + 2 + 1 = 4
		assert.Equal(t, 4, list[0].RequestCount)
	})
}

func TestCollectionStore_SaveLoadScripts(t *testing.T) {
	t.Run("saves and loads pre/post scripts", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("Scripted API")
		c.SetPreScript("console.log('pre-request');")
		c.SetPostScript("console.log('post-request');")

		require.NoError(t, store.Save(ctx, c))

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		assert.Equal(t, "console.log('pre-request');", loaded.PreScript())
		assert.Equal(t, "console.log('post-request');", loaded.PostScript())
	})

	t.Run("saves and loads request scripts", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("API")
		req := core.NewRequestDefinition("Test", "GET", "/test")
		req.SetPreScript("pm.environment.set('token', 'abc');")
		req.SetPostScript("pm.test('Status is 200', () => pm.response.code === 200);")
		c.AddRequest(req)

		require.NoError(t, store.Save(ctx, c))

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		require.Len(t, loaded.Requests(), 1)
		assert.Equal(t, "pm.environment.set('token', 'abc');", loaded.Requests()[0].PreScript())
		assert.Equal(t, "pm.test('Status is 200', () => pm.response.code === 200);", loaded.Requests()[0].PostScript())
	})
}

func TestCollectionStore_SaveLoadRequestBody(t *testing.T) {
	t.Run("saves and loads request body", func(t *testing.T) {
		store := newTestStore(t)
		ctx := context.Background()

		c := core.NewCollection("API")
		req := core.NewRequestDefinition("Create User", "POST", "/users")
		req.SetBodyRaw(`{"name": "John", "email": "john@example.com"}`, "raw")
		c.AddRequest(req)

		require.NoError(t, store.Save(ctx, c))

		loaded, err := store.Get(ctx, c.ID())
		require.NoError(t, err)
		require.Len(t, loaded.Requests(), 1)
		assert.Equal(t, `{"name": "John", "email": "john@example.com"}`, loaded.Requests()[0].BodyContent())
		assert.Equal(t, "raw", loaded.Requests()[0].BodyType())
	})
}

func newTestStore(t *testing.T) *CollectionStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := NewCollectionStore(tmpDir)
	require.NoError(t, err)
	return store
}
