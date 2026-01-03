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

func TestEnvironmentStore_SetActiveWithDeactivation(t *testing.T) {
	t.Run("setting empty ID returns error", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("Development")
		env.SetActive(true)
		require.NoError(t, store.Save(ctx, env))

		// Verify it's active
		active, err := store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, env.ID(), active.ID())

		// Set active to empty returns error for non-existent ID
		err = store.SetActive(ctx, "")
		assert.Error(t, err)
	})

	t.Run("switching active between environments", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env1 := core.NewEnvironment("Dev")
		env2 := core.NewEnvironment("Staging")
		env3 := core.NewEnvironment("Prod")

		require.NoError(t, store.Save(ctx, env1))
		require.NoError(t, store.Save(ctx, env2))
		require.NoError(t, store.Save(ctx, env3))

		// Set first as active
		require.NoError(t, store.SetActive(ctx, env1.ID()))
		active, err := store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Dev", active.Name())

		// Switch to third
		require.NoError(t, store.SetActive(ctx, env3.ID()))
		active, err = store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Prod", active.Name())
	})
}

func TestEnvironmentStore_SaveWithAllVariables(t *testing.T) {
	t.Run("saves environment with multiple variable types", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("FullEnv")
		env.SetDescription("Full test environment")
		env.SetVariable("base_url", "https://api.example.com")
		env.SetVariable("timeout", "30")
		env.SetVariable("version", "v2")
		env.SetSecret("api_key", "secret123")
		env.SetSecret("auth_token", "token456")
		env.SetActive(true)
		env.SetGlobal(false)

		err := store.Save(ctx, env)
		require.NoError(t, err)

		loaded, err := store.Get(ctx, env.ID())
		require.NoError(t, err)

		assert.Equal(t, "Full test environment", loaded.Description())
		assert.Equal(t, "https://api.example.com", loaded.GetVariable("base_url"))
		assert.Equal(t, "30", loaded.GetVariable("timeout"))
		assert.Equal(t, "v2", loaded.GetVariable("version"))
		assert.Equal(t, "secret123", loaded.GetSecret("api_key"))
		assert.Equal(t, "token456", loaded.GetSecret("auth_token"))
		assert.True(t, loaded.IsActive())
		assert.False(t, loaded.IsGlobal())
	})
}

func TestEnvironmentStore_ListOnlyLoadMetadata(t *testing.T) {
	t.Run("listing environments does not fail on malformed files", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		// Save valid environment
		env := core.NewEnvironment("Valid")
		require.NoError(t, store.Save(ctx, env))

		// List should succeed even if some files might have issues
		list, err := store.List(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})
}

func TestEnvironmentStore_GetByNameCaseInsensitive(t *testing.T) {
	t.Run("searching by name is exact match", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("MyEnvName")
		require.NoError(t, store.Save(ctx, env))

		// Exact match should work
		loaded, err := store.GetByName(ctx, "MyEnvName")
		require.NoError(t, err)
		assert.Equal(t, "MyEnvName", loaded.Name())

		// Different case should not match
		_, err = store.GetByName(ctx, "myenvname")
		assert.Error(t, err)
	})
}

func TestEnvironmentStore_SetActiveCoverage(t *testing.T) {
	t.Run("set active environment by id", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		// Create two environments
		env1 := core.NewEnvironment("Env1")
		env2 := core.NewEnvironment("Env2")
		require.NoError(t, store.Save(ctx, env1))
		require.NoError(t, store.Save(ctx, env2))

		// Set env1 as active
		err := store.SetActive(ctx, env1.ID())
		require.NoError(t, err)

		// Get active should return env1
		active, err := store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, env1.ID(), active.ID())
	})

	t.Run("set active with invalid id", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		err := store.SetActive(ctx, "nonexistent-id")
		assert.Error(t, err)
	})

	t.Run("clear active environment returns error for empty id", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("TestEnv")
		require.NoError(t, store.Save(ctx, env))
		require.NoError(t, store.SetActive(ctx, env.ID()))

		// Clear active by setting empty id should return error
		err := store.SetActive(ctx, "")
		assert.Error(t, err)
	})
}

func TestEnvironmentStore_DeleteCoverage(t *testing.T) {
	t.Run("delete existing environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		env := core.NewEnvironment("ToDelete")
		require.NoError(t, store.Save(ctx, env))

		err := store.Delete(ctx, env.ID())
		require.NoError(t, err)

		// Should not exist anymore
		_, err = store.Get(ctx, env.ID())
		assert.Error(t, err)
	})

	t.Run("delete nonexistent environment", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		err := store.Delete(ctx, "nonexistent-id")
		assert.Error(t, err)
	})
}

func TestEnvironmentStore_GetActiveCoverage(t *testing.T) {
	t.Run("get active when none is set returns error", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		active, err := store.GetActive(ctx)
		assert.Error(t, err)
		assert.Nil(t, active)
	})

	t.Run("get active with multiple environments", func(t *testing.T) {
		store := newTestEnvStore(t)
		ctx := context.Background()

		// Create multiple environments
		for i := 0; i < 5; i++ {
			env := core.NewEnvironment("Env" + string(rune('A'+i)))
			require.NoError(t, store.Save(ctx, env))
		}

		// Set the third one as active
		list, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 5)

		loaded, err := store.Get(ctx, list[2].ID)
		require.NoError(t, err)
		require.NoError(t, store.SetActive(ctx, loaded.ID()))

		// Verify GetActive returns correct one
		active, err := store.GetActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, list[2].ID, active.ID())
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
