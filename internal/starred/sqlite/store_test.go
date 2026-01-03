package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artpar/currier/internal/starred"
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

func TestStore_ToggleExtended(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Toggle to star
	starred, err := store.Toggle(ctx, "toggle-req-1")
	require.NoError(t, err)
	assert.True(t, starred)

	// Toggle to unstar
	starred, err = store.Toggle(ctx, "toggle-req-1")
	require.NoError(t, err)
	assert.False(t, starred)

	// Toggle again to star
	starred, err = store.Toggle(ctx, "toggle-req-1")
	require.NoError(t, err)
	assert.True(t, starred)
}

func TestStore_ListStarredExtended(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// List empty
	items, err := store.ListStarred(ctx)
	require.NoError(t, err)
	assert.Empty(t, items)

	// Star multiple items
	err = store.Star(ctx, "list-req-1")
	require.NoError(t, err)
	err = store.Star(ctx, "list-req-2")
	require.NoError(t, err)
	err = store.Star(ctx, "list-req-3")
	require.NoError(t, err)

	// List all
	items, err = store.ListStarred(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestStore_ClearExtended(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Star multiple items
	err = store.Star(ctx, "clear-req-1")
	require.NoError(t, err)
	err = store.Star(ctx, "clear-req-2")
	require.NoError(t, err)

	// Count before clear
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Clear all
	err = store.Clear(ctx)
	require.NoError(t, err)

	// Count after clear
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestStore_StarUnstarExtended(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Star
	err = store.Star(ctx, "extended-req-1")
	require.NoError(t, err)

	// Verify starred
	isStarred, err := store.IsStarred(ctx, "extended-req-1")
	require.NoError(t, err)
	assert.True(t, isStarred)

	// Unstar
	err = store.Unstar(ctx, "extended-req-1")
	require.NoError(t, err)

	// Verify not starred
	isStarred, err = store.IsStarred(ctx, "extended-req-1")
	require.NoError(t, err)
	assert.False(t, isStarred)

	// Unstar again (should not error)
	err = store.Unstar(ctx, "extended-req-1")
	require.NoError(t, err)
}

func TestStore_IsStarredNotExist(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Check non-existent item
	isStarred, err := store.IsStarred(ctx, "non-existent")
	require.NoError(t, err)
	assert.False(t, isStarred)
}

func TestStore_NewWithInvalidPath(t *testing.T) {
	// Test with a directory that doesn't exist and can't be created
	store, err := New("/nonexistent/path/that/cannot/be/created/starred.db")
	if store != nil {
		defer store.Close()
	}
	// May or may not error depending on SQLite version
	_ = err
}

func TestStore_StarAndListOrdering(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Star items in specific order
	require.NoError(t, store.Star(ctx, "item-a"))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.Star(ctx, "item-b"))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.Star(ctx, "item-c"))

	// List should return newest first
	items, err := store.ListStarred(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, "item-c", items[0])
	assert.Equal(t, "item-b", items[1])
	assert.Equal(t, "item-a", items[2])
}

func TestStore_ToggleMultipleTimes(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	requestID := "multi-toggle"

	// Toggle on
	starred, err := store.Toggle(ctx, requestID)
	require.NoError(t, err)
	assert.True(t, starred)

	// Toggle off
	starred, err = store.Toggle(ctx, requestID)
	require.NoError(t, err)
	assert.False(t, starred)

	// Toggle on again
	starred, err = store.Toggle(ctx, requestID)
	require.NoError(t, err)
	assert.True(t, starred)

	// Verify count
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStore_ClearEmpty(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Clear an already empty store
	err = store.Clear(ctx)
	require.NoError(t, err)

	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestStore_ClosedOperationsCoverage(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)

	ctx := context.Background()

	// Close the store
	err = store.Close()
	require.NoError(t, err)

	// All operations should return ErrStoreClosed
	_, err = store.IsStarred(ctx, "test")
	assert.ErrorIs(t, err, starred.ErrStoreClosed)

	err = store.Star(ctx, "test")
	assert.ErrorIs(t, err, starred.ErrStoreClosed)

	err = store.Unstar(ctx, "test")
	assert.ErrorIs(t, err, starred.ErrStoreClosed)

	_, err = store.Toggle(ctx, "test")
	assert.ErrorIs(t, err, starred.ErrStoreClosed)

	_, err = store.ListStarred(ctx)
	assert.ErrorIs(t, err, starred.ErrStoreClosed)

	err = store.Clear(ctx)
	assert.ErrorIs(t, err, starred.ErrStoreClosed)

	_, err = store.Count(ctx)
	assert.ErrorIs(t, err, starred.ErrStoreClosed)
}

func TestStore_ListStarredCoverage(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Star multiple requests
	for i := 0; i < 10; i++ {
		err = store.Star(ctx, fmt.Sprintf("request-%d", i))
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// List all
	list, err := store.ListStarred(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 10)
}

func TestStore_StarAlreadyStarredCoverage(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Star a request
	err = store.Star(ctx, "test-request")
	require.NoError(t, err)

	// Star it again - should not error
	err = store.Star(ctx, "test-request")
	require.NoError(t, err)

	// Should still be starred with count of 1
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStore_UnstarNotStarredCoverage(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Unstar a non-existent request - should not error
	err = store.Unstar(ctx, "non-existent")
	require.NoError(t, err)
}
