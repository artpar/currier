package sqlite

import (
	"context"
	"testing"

	"github.com/artpar/currier/internal/history"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheStore_CacheResponse(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	t.Run("caches response and returns hash", func(t *testing.T) {
		body := `{"id": 1, "name": "Test"}`
		hash, err := store.CacheResponse(ctx, body)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64) // SHA256 hex is 64 chars
	})

	t.Run("same content returns same hash", func(t *testing.T) {
		body := `{"id": 2, "name": "Another"}`
		hash1, err := store.CacheResponse(ctx, body)
		require.NoError(t, err)

		hash2, err := store.CacheResponse(ctx, body)
		require.NoError(t, err)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different content returns different hash", func(t *testing.T) {
		hash1, err := store.CacheResponse(ctx, "body1")
		require.NoError(t, err)

		hash2, err := store.CacheResponse(ctx, "body2")
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestCacheStore_GetCachedResponse(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	t.Run("retrieves cached response", func(t *testing.T) {
		body := `{"data": "test content"}`
		hash, err := store.CacheResponse(ctx, body)
		require.NoError(t, err)

		retrieved, err := store.GetCachedResponse(ctx, hash)
		require.NoError(t, err)
		assert.Equal(t, body, retrieved)
	})

	t.Run("returns error for non-existent hash", func(t *testing.T) {
		_, err := store.GetCachedResponse(ctx, "nonexistent")
		assert.ErrorIs(t, err, history.ErrNotFound)
	})

	t.Run("tracks hit and miss counts", func(t *testing.T) {
		body := "track stats body"
		hash, _ := store.CacheResponse(ctx, body)

		// Get cached (hit)
		store.GetCachedResponse(ctx, hash)
		store.GetCachedResponse(ctx, hash)

		// Get non-existent (miss)
		store.GetCachedResponse(ctx, "missing1")
		store.GetCachedResponse(ctx, "missing2")
		store.GetCachedResponse(ctx, "missing3")

		stats, err := store.CacheStats(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, stats.HitCount, int64(2))
		assert.GreaterOrEqual(t, stats.MissCount, int64(3))
	})
}

func TestCacheStore_CacheStats(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	t.Run("returns empty stats for new store", func(t *testing.T) {
		stats, err := store.CacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.TotalEntries)
		assert.Equal(t, int64(0), stats.TotalSize)
	})

	t.Run("returns correct stats after caching", func(t *testing.T) {
		body1 := "first cached response body"
		body2 := "second cached response"

		store.CacheResponse(ctx, body1)
		store.CacheResponse(ctx, body2)

		stats, err := store.CacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.TotalEntries)
		assert.Equal(t, int64(len(body1)+len(body2)), stats.TotalSize)
	})

	t.Run("calculates hit rate", func(t *testing.T) {
		// Create fresh store for clean stats
		freshStore, _ := NewInMemoryCacheStore()
		defer freshStore.Close()

		body := "hit rate test"
		hash, _ := freshStore.CacheResponse(ctx, body)

		// 3 hits
		freshStore.GetCachedResponse(ctx, hash)
		freshStore.GetCachedResponse(ctx, hash)
		freshStore.GetCachedResponse(ctx, hash)

		// 1 miss
		freshStore.GetCachedResponse(ctx, "missing")

		stats, err := freshStore.CacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), stats.HitCount)
		assert.Equal(t, int64(1), stats.MissCount)
		assert.InDelta(t, 0.75, stats.HitRate, 0.01) // 3/(3+1) = 0.75
	})
}

func TestCacheStore_ClearCache(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add some cache entries
	store.CacheResponse(ctx, "body1")
	store.CacheResponse(ctx, "body2")
	store.CacheResponse(ctx, "body3")

	// Verify entries exist
	stats, _ := store.CacheStats(ctx)
	assert.Equal(t, int64(3), stats.TotalEntries)

	// Clear cache
	err = store.ClearCache(ctx)
	require.NoError(t, err)

	// Verify entries are gone
	stats, err = store.CacheStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEntries)
}

func TestCacheStore_PruneCache(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	t.Run("prunes orphaned cache entries", func(t *testing.T) {
		// Cache some responses
		store.CacheResponse(ctx, "orphan1")
		store.CacheResponse(ctx, "orphan2")

		// Verify they exist
		stats, _ := store.CacheStats(ctx)
		assert.Equal(t, int64(2), stats.TotalEntries)

		// Prune - since no history references them, they should be removed
		deleted, err := store.PruneCache(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		// Verify entries are gone
		stats, _ = store.CacheStats(ctx)
		assert.Equal(t, int64(0), stats.TotalEntries)
	})
}

func TestCacheStore_ClosedStore(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)

	store.Close()
	ctx := context.Background()

	t.Run("CacheResponse fails after close", func(t *testing.T) {
		_, err := store.CacheResponse(ctx, "body")
		assert.ErrorIs(t, err, history.ErrStoreClosed)
	})

	t.Run("GetCachedResponse fails after close", func(t *testing.T) {
		_, err := store.GetCachedResponse(ctx, "hash")
		assert.ErrorIs(t, err, history.ErrStoreClosed)
	})

	t.Run("CacheStats fails after close", func(t *testing.T) {
		_, err := store.CacheStats(ctx)
		assert.ErrorIs(t, err, history.ErrStoreClosed)
	})

	t.Run("PruneCache fails after close", func(t *testing.T) {
		_, err := store.PruneCache(ctx)
		assert.ErrorIs(t, err, history.ErrStoreClosed)
	})

	t.Run("ClearCache fails after close", func(t *testing.T) {
		err := store.ClearCache(ctx)
		assert.ErrorIs(t, err, history.ErrStoreClosed)
	})
}

func TestCacheStore_InheritsStoreInterface(t *testing.T) {
	store, err := NewInMemoryCacheStore()
	require.NoError(t, err)
	defer store.Close()

	// Verify CacheStore can be used as regular Store
	var _ history.Store = store

	// Verify CacheStore implements CacheStore interface
	var _ history.CacheStore = store

	ctx := context.Background()

	// Test basic Store operations work
	entry := history.Entry{
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
	}

	id, err := store.Add(ctx, entry)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	retrieved, err := store.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, entry.RequestURL, retrieved.RequestURL)
}
