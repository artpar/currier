package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/artpar/currier/internal/history"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for the complete history workflow

func TestIntegration_FullWorkflow(t *testing.T) {
	// Create temp directory for database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/history.db"

	// Create store
	store, err := New(dbPath)
	require.NoError(t, err)

	ctx := context.Background()

	// Add several history entries
	entries := []history.Entry{
		{
			Timestamp:      time.Now().Add(-2 * time.Hour),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com/users",
			ResponseStatus: 200,
			ResponseBody:   `{"users": []}`,
			ResponseTime:   150,
			ResponseSize:   100,
			CollectionID:   "col-1",
			Environment:    "production",
		},
		{
			Timestamp:      time.Now().Add(-1 * time.Hour),
			RequestMethod:  "POST",
			RequestURL:     "https://api.example.com/users",
			RequestBody:    `{"name": "John"}`,
			ResponseStatus: 201,
			ResponseBody:   `{"id": 1}`,
			ResponseTime:   200,
			ResponseSize:   50,
			CollectionID:   "col-1",
			Environment:    "production",
		},
		{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com/users/1",
			ResponseStatus: 200,
			ResponseBody:   `{"id": 1, "name": "John"}`,
			ResponseTime:   100,
			ResponseSize:   75,
			CollectionID:   "col-1",
			Environment:    "staging",
		},
	}

	var ids []string
	for _, entry := range entries {
		id, err := store.Add(ctx, entry)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	// Test List all
	all, err := store.List(ctx, history.QueryOptions{})
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// Test Filter by method
	gets, err := store.List(ctx, history.QueryOptions{Method: "GET"})
	require.NoError(t, err)
	assert.Len(t, gets, 2)

	// Test Filter by environment
	prodEntries, err := store.List(ctx, history.QueryOptions{Environment: "production"})
	require.NoError(t, err)
	assert.Len(t, prodEntries, 2)

	// Test Search
	searchResults, err := store.Search(ctx, "John", history.QueryOptions{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(searchResults), 1)

	// Test Stats
	stats, err := store.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalEntries)
	assert.Equal(t, int64(2), stats.MethodCounts["GET"])
	assert.Equal(t, int64(1), stats.MethodCounts["POST"])

	// Test Update
	entries[2].Notes = "Updated note"
	entries[2].ID = ids[2]
	err = store.Update(ctx, entries[2])
	require.NoError(t, err)

	updated, err := store.Get(ctx, ids[2])
	require.NoError(t, err)
	assert.Equal(t, "Updated note", updated.Notes)

	// Close and reopen to test persistence
	store.Close()

	store2, err := New(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	// Verify data persisted
	persisted, err := store2.List(ctx, history.QueryOptions{})
	require.NoError(t, err)
	assert.Len(t, persisted, 3)

	// Test Delete
	err = store2.Delete(ctx, ids[0])
	require.NoError(t, err)

	remaining, err := store2.List(ctx, history.QueryOptions{})
	require.NoError(t, err)
	assert.Len(t, remaining, 2)
}

func TestIntegration_CacheWithHistory(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Cache a response body
	responseBody := `{"result": "success", "data": {"id": 123}}`
	hash, err := store.CacheResponse(ctx, responseBody)
	require.NoError(t, err)

	// Add history entry referencing the cache
	entry := history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://api.example.com/data",
		ResponseStatus: 200,
		ResponseBody:   hash, // Store hash reference instead of full body
		ResponseTime:   100,
	}

	id, err := store.Add(ctx, entry)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := store.Get(ctx, id)
	require.NoError(t, err)

	// Get actual body from cache
	cachedBody, err := store.GetCachedResponse(ctx, retrieved.ResponseBody)
	require.NoError(t, err)
	assert.Equal(t, responseBody, cachedBody)

	// Check cache stats
	cacheStats, err := store.CacheStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cacheStats.TotalEntries)
	assert.GreaterOrEqual(t, cacheStats.HitCount, int64(1))
}

func TestIntegration_PruneWorkflow(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add entries at different times
	for i := 0; i < 10; i++ {
		_, err := store.Add(ctx, history.Entry{
			Timestamp:      time.Now().Add(-time.Duration(i) * time.Hour),
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
		})
		require.NoError(t, err)
	}

	// Verify all added
	count, err := store.Count(ctx, history.QueryOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Prune keeping last 5
	result, err := store.Prune(ctx, history.PruneOptions{KeepLast: 5})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.DeletedCount)

	// Verify remaining
	remaining, err := store.Count(ctx, history.QueryOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(5), remaining)
}

func TestIntegration_FileBasedStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Verify file doesn't exist yet
	_, err := os.Stat(dbPath)
	require.True(t, os.IsNotExist(err))

	// Create store - should create file
	store, err := New(dbPath)
	require.NoError(t, err)

	// Add entry
	ctx := context.Background()
	_, err = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
	})
	require.NoError(t, err)

	// Close store
	store.Close()

	// Verify file was created
	info, err := os.Stat(dbPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestIntegration_ErrorHandling(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("get non-existent returns ErrNotFound", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent-id")
		assert.ErrorIs(t, err, history.ErrNotFound)
	})

	t.Run("get empty ID returns ErrInvalidID", func(t *testing.T) {
		_, err := store.Get(ctx, "")
		assert.ErrorIs(t, err, history.ErrInvalidID)
	})

	t.Run("delete non-existent returns ErrNotFound", func(t *testing.T) {
		err := store.Delete(ctx, "non-existent-id")
		assert.ErrorIs(t, err, history.ErrNotFound)
	})

	t.Run("update non-existent returns ErrNotFound", func(t *testing.T) {
		err := store.Update(ctx, history.Entry{ID: "non-existent"})
		assert.ErrorIs(t, err, history.ErrNotFound)
	})

	// Close and verify operations fail
	store.Close()

	t.Run("add after close returns ErrStoreClosed", func(t *testing.T) {
		_, err := store.Add(ctx, history.Entry{})
		assert.ErrorIs(t, err, history.ErrStoreClosed)
	})

	t.Run("list after close returns ErrStoreClosed", func(t *testing.T) {
		_, err := store.List(ctx, history.QueryOptions{})
		assert.ErrorIs(t, err, history.ErrStoreClosed)
	})
}
