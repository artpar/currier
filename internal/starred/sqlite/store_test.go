package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestStore_StarUnstar(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	requestID := "test-request-1"

	// Initially not starred
	starred, err := store.IsStarred(ctx, requestID)
	require.NoError(t, err)
	assert.False(t, starred)

	// Star it
	err = store.Star(ctx, requestID)
	require.NoError(t, err)

	// Now should be starred
	starred, err = store.IsStarred(ctx, requestID)
	require.NoError(t, err)
	assert.True(t, starred)

	// Unstar it
	err = store.Unstar(ctx, requestID)
	require.NoError(t, err)

	// Should not be starred anymore
	starred, err = store.IsStarred(ctx, requestID)
	require.NoError(t, err)
	assert.False(t, starred)
}

func TestStore_Toggle(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	requestID := "toggle-test"

	// Toggle from unstarred to starred
	newState, err := store.Toggle(ctx, requestID)
	require.NoError(t, err)
	assert.True(t, newState)

	// Verify it's starred
	starred, err := store.IsStarred(ctx, requestID)
	require.NoError(t, err)
	assert.True(t, starred)

	// Toggle again to unstar
	newState, err = store.Toggle(ctx, requestID)
	require.NoError(t, err)
	assert.False(t, newState)

	// Verify it's unstarred
	starred, err = store.IsStarred(ctx, requestID)
	require.NoError(t, err)
	assert.False(t, starred)
}

func TestStore_ListStarred(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Initially empty
	ids, err := store.ListStarred(ctx)
	require.NoError(t, err)
	assert.Empty(t, ids)

	// Star several requests
	require.NoError(t, store.Star(ctx, "req-1"))
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	require.NoError(t, store.Star(ctx, "req-2"))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.Star(ctx, "req-3"))

	// List should return all, ordered by starred_at DESC (most recent first)
	ids, err = store.ListStarred(ctx)
	require.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.Equal(t, "req-3", ids[0]) // Most recent first
	assert.Equal(t, "req-2", ids[1])
	assert.Equal(t, "req-1", ids[2])
}

func TestStore_Count(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Initially zero
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Star some requests
	require.NoError(t, store.Star(ctx, "req-1"))
	require.NoError(t, store.Star(ctx, "req-2"))

	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Unstar one
	require.NoError(t, store.Unstar(ctx, "req-1"))

	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStore_Clear(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Star some requests
	require.NoError(t, store.Star(ctx, "req-1"))
	require.NoError(t, store.Star(ctx, "req-2"))
	require.NoError(t, store.Star(ctx, "req-3"))

	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Clear all
	err = store.Clear(ctx)
	require.NoError(t, err)

	// Should be empty
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	ids, err := store.ListStarred(ctx)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestStore_StarIdempotent(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	requestID := "idempotent-test"

	// Star multiple times should not error
	require.NoError(t, store.Star(ctx, requestID))
	require.NoError(t, store.Star(ctx, requestID))
	require.NoError(t, store.Star(ctx, requestID))

	// Should still only have one entry
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStore_UnstarNonExistent(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Unstarring a non-existent request should not error
	err = store.Unstar(ctx, "non-existent")
	require.NoError(t, err)
}

func TestStore_ClosedStore(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)

	// Close the store
	err = store.Close()
	require.NoError(t, err)

	ctx := context.Background()

	// All operations should return ErrStoreClosed
	_, err = store.IsStarred(ctx, "test")
	assert.Error(t, err)

	err = store.Star(ctx, "test")
	assert.Error(t, err)

	err = store.Unstar(ctx, "test")
	assert.Error(t, err)

	_, err = store.Toggle(ctx, "test")
	assert.Error(t, err)

	_, err = store.ListStarred(ctx)
	assert.Error(t, err)

	_, err = store.Count(ctx)
	assert.Error(t, err)

	err = store.Clear(ctx)
	assert.Error(t, err)
}

func TestStore_NewWithFilePath(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "starred-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "starred.db")

	// Create store with file path
	store, err := New(dbPath)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Star a request
	err = store.Star(ctx, "req-1")
	require.NoError(t, err)

	// Verify starred
	starred, err := store.IsStarred(ctx, "req-1")
	require.NoError(t, err)
	assert.True(t, starred)

	// Close and reopen to verify persistence
	store.Close()

	store2, err := New(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	// Should still be starred
	starred, err = store2.IsStarred(ctx, "req-1")
	require.NoError(t, err)
	assert.True(t, starred)
}

func TestStore_NewWithDB(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create store with existing db
	store, err := NewWithDB(db)
	require.NoError(t, err)

	ctx := context.Background()

	// Use the store
	err = store.Star(ctx, "req-1")
	require.NoError(t, err)

	starred, err := store.IsStarred(ctx, "req-1")
	require.NoError(t, err)
	assert.True(t, starred)

	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStore_CloseIdempotent(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)

	// Close multiple times should not error
	err = store.Close()
	require.NoError(t, err)

	err = store.Close()
	require.NoError(t, err)
}
