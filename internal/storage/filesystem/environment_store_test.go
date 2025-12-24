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

func TestNewEnvironmentStore(t *testing.T) {
	t.Run("creates store with base path", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewEnvironmentStore(tmpDir)

		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("creates environments directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		envsDir := filepath.Join(tmpDir, "environments")

		_, err := NewEnvironmentStore(envsDir)
		require.NoError(t, err)

		info, err := os.Stat(envsDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestEnvironmentStore_Save(t *testing.T) {
	t.Run("saves new environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Production")
		env.SetVariable("base_url", "https://api.example.com")

		err := store.Save(ctx, env)
		require.NoError(t, err)

		// Verify file was created
		path := store.environmentPath(env.ID())
		_, err = os.Stat(path)
		assert.NoError(t, err)
	})

	t.Run("updates existing environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Production")
		err := store.Save(ctx, env)
		require.NoError(t, err)

		env.SetDescription("Updated description")
		err = store.Save(ctx, env)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, env.ID())
		require.NoError(t, err)
		assert.Equal(t, "Updated description", loaded.Description())
	})

	t.Run("saves environment with secrets", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Production")
		env.SetVariable("base_url", "https://api.example.com")
		env.SetSecret("api_key", "super-secret")

		err := store.Save(ctx, env)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, env.ID())
		require.NoError(t, err)
		assert.Equal(t, "super-secret", loaded.GetSecret("api_key"))
	})
}

func TestEnvironmentStore_Get(t *testing.T) {
	t.Run("gets environment by ID", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Staging")
		env.SetDescription("Staging environment")
		env.SetVariable("base_url", "https://staging.example.com")
		err := store.Save(ctx, env)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, env.ID())
		require.NoError(t, err)
		assert.Equal(t, env.ID(), loaded.ID())
		assert.Equal(t, "Staging", loaded.Name())
		assert.Equal(t, "Staging environment", loaded.Description())
		assert.Equal(t, "https://staging.example.com", loaded.GetVariable("base_url"))
	})

	t.Run("returns error for non-existent environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		_, err := store.Get(ctx, "non-existent-id")
		assert.Error(t, err)
	})
}

func TestEnvironmentStore_GetByName(t *testing.T) {
	t.Run("gets environment by name", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Production")
		err := store.Save(ctx, env)
		require.NoError(t, err)

		loaded, err := store.GetByName(ctx, "Production")
		require.NoError(t, err)
		assert.Equal(t, "Production", loaded.Name())
	})

	t.Run("returns error for non-existent name", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		_, err := store.GetByName(ctx, "NonExistent")
		assert.Error(t, err)
	})
}

func TestEnvironmentStore_List(t *testing.T) {
	t.Run("lists all environments", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env1 := core.NewEnvironment("Development")
		env2 := core.NewEnvironment("Staging")
		env3 := core.NewEnvironment("Production")

		require.NoError(t, store.Save(ctx, env1))
		require.NoError(t, store.Save(ctx, env2))
		require.NoError(t, store.Save(ctx, env3))

		list, err := store.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})

	t.Run("returns empty list when no environments", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		list, err := store.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("list contains metadata", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Production")
		env.SetVariable("a", "1")
		env.SetVariable("b", "2")
		env.SetActive(true)
		require.NoError(t, store.Save(ctx, env))

		list, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)

		meta := list[0]
		assert.Equal(t, env.ID(), meta.ID)
		assert.Equal(t, "Production", meta.Name)
		assert.Equal(t, 2, meta.VarCount)
		assert.True(t, meta.IsActive)
	})
}

func TestEnvironmentStore_Delete(t *testing.T) {
	t.Run("deletes environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("ToDelete")
		require.NoError(t, store.Save(ctx, env))

		err := store.Delete(ctx, env.ID())
		require.NoError(t, err)

		_, err = store.Get(ctx, env.ID())
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		err := store.Delete(ctx, "non-existent-id")
		assert.Error(t, err)
	})
}

func TestEnvironmentStore_ActiveEnvironment(t *testing.T) {
	t.Run("sets and gets active environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env1 := core.NewEnvironment("Development")
		env2 := core.NewEnvironment("Production")

		require.NoError(t, store.Save(ctx, env1))
		require.NoError(t, store.Save(ctx, env2))

		err := store.SetActive(ctx, env2.ID())
		require.NoError(t, err)

		active, err := store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Production", active.Name())
	})

	t.Run("only one environment is active at a time", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env1 := core.NewEnvironment("Development")
		env2 := core.NewEnvironment("Production")
		env1.SetActive(true)

		require.NoError(t, store.Save(ctx, env1))
		require.NoError(t, store.Save(ctx, env2))

		err := store.SetActive(ctx, env2.ID())
		require.NoError(t, err)

		// Reload env1 and check it's no longer active
		loaded1, err := store.Get(ctx, env1.ID())
		require.NoError(t, err)
		assert.False(t, loaded1.IsActive())

		loaded2, err := store.Get(ctx, env2.ID())
		require.NoError(t, err)
		assert.True(t, loaded2.IsActive())
	})

	t.Run("returns error when no active environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Development")
		require.NoError(t, store.Save(ctx, env))

		_, err := store.GetActive(ctx)
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent environment ID", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		err := store.SetActive(ctx, "non-existent-id")
		assert.Error(t, err)
	})
}

func TestEnvironmentStore_GlobalEnvironment(t *testing.T) {
	t.Run("saves and loads global environment flag", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Globals")
		env.SetGlobal(true)
		env.SetVariable("common_var", "value")

		require.NoError(t, store.Save(ctx, env))

		loaded, err := store.Get(ctx, env.ID())
		require.NoError(t, err)
		assert.True(t, loaded.IsGlobal())
	})
}

// Helper functions

func newTestEnvStore(t *testing.T) *EnvironmentStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := NewEnvironmentStore(tmpDir)
	require.NoError(t, err)
	return store
}
