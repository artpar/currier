package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artpar/currier/internal/cookies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_SetAndGet(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	cookie := &cookies.Cookie{
		Domain:   "example.com",
		Path:     "/",
		Name:     "session",
		Value:    "abc123",
		Secure:   true,
		HttpOnly: true,
		SameSite: "Strict",
		Expires:  time.Now().Add(24 * time.Hour),
	}

	// Set cookie
	err = store.Set(ctx, cookie)
	require.NoError(t, err)
	assert.NotEmpty(t, cookie.ID)

	// Get cookie
	got, err := store.Get(ctx, "example.com", "/", "session")
	require.NoError(t, err)
	assert.Equal(t, "session", got.Name)
	assert.Equal(t, "abc123", got.Value)
	assert.True(t, got.Secure)
	assert.True(t, got.HttpOnly)
	assert.Equal(t, "Strict", got.SameSite)
}

func TestStore_SetUpdates(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Set initial cookie
	cookie := &cookies.Cookie{
		Domain: "example.com",
		Path:   "/",
		Name:   "test",
		Value:  "initial",
	}
	err = store.Set(ctx, cookie)
	require.NoError(t, err)

	// Update with same domain/path/name
	cookie2 := &cookies.Cookie{
		Domain: "example.com",
		Path:   "/",
		Name:   "test",
		Value:  "updated",
	}
	err = store.Set(ctx, cookie2)
	require.NoError(t, err)

	// Should have updated value
	got, err := store.Get(ctx, "example.com", "/", "test")
	require.NoError(t, err)
	assert.Equal(t, "updated", got.Value)

	// Should only have one cookie
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStore_List(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add cookies
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "a.com", Path: "/", Name: "c1", Value: "v1"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "a.com", Path: "/api", Name: "c2", Value: "v2"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "b.com", Path: "/", Name: "c3", Value: "v3"}))

	// List all
	all, err := store.List(ctx, cookies.QueryOptions{IncludeExpired: true})
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// List by domain
	aCookies, err := store.List(ctx, cookies.QueryOptions{Domain: "a.com", IncludeExpired: true})
	require.NoError(t, err)
	assert.Len(t, aCookies, 2)

	// List by domain and path
	apiCookies, err := store.List(ctx, cookies.QueryOptions{Domain: "a.com", Path: "/api", IncludeExpired: true})
	require.NoError(t, err)
	assert.Len(t, apiCookies, 1)

	// List with limit
	limited, err := store.List(ctx, cookies.QueryOptions{Limit: 2, IncludeExpired: true})
	require.NoError(t, err)
	assert.Len(t, limited, 2)
}

func TestStore_ListExcludesExpired(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add expired and valid cookies
	require.NoError(t, store.Set(ctx, &cookies.Cookie{
		Domain: "test.com", Path: "/", Name: "valid", Value: "v1",
		Expires: time.Now().Add(1 * time.Hour),
	}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{
		Domain: "test.com", Path: "/", Name: "expired", Value: "v2",
		Expires: time.Now().Add(-1 * time.Hour),
	}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{
		Domain: "test.com", Path: "/", Name: "session", Value: "v3",
		// No expiry - session cookie
	}))

	// List without expired should return 2
	valid, err := store.List(ctx, cookies.QueryOptions{IncludeExpired: false})
	require.NoError(t, err)
	assert.Len(t, valid, 2)

	// List with expired should return 3
	all, err := store.List(ctx, cookies.QueryOptions{IncludeExpired: true})
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestStore_Delete(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add cookie
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "test.com", Path: "/", Name: "c1", Value: "v1"}))

	// Delete it
	err = store.Delete(ctx, "test.com", "/", "c1")
	require.NoError(t, err)

	// Should not exist
	_, err = store.Get(ctx, "test.com", "/", "c1")
	assert.ErrorIs(t, err, cookies.ErrNotFound)
}

func TestStore_DeleteByDomain(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add cookies
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "a.com", Path: "/", Name: "c1", Value: "v1"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "a.com", Path: "/api", Name: "c2", Value: "v2"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "b.com", Path: "/", Name: "c3", Value: "v3"}))

	// Delete domain
	err = store.DeleteByDomain(ctx, "a.com")
	require.NoError(t, err)

	// Check counts
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// b.com cookie should remain
	got, err := store.Get(ctx, "b.com", "/", "c3")
	require.NoError(t, err)
	assert.Equal(t, "v3", got.Value)
}

func TestStore_DeleteExpired(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add cookies
	require.NoError(t, store.Set(ctx, &cookies.Cookie{
		Domain: "test.com", Path: "/", Name: "valid", Value: "v1",
		Expires: time.Now().Add(1 * time.Hour),
	}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{
		Domain: "test.com", Path: "/", Name: "expired1", Value: "v2",
		Expires: time.Now().Add(-1 * time.Hour),
	}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{
		Domain: "test.com", Path: "/", Name: "expired2", Value: "v3",
		Expires: time.Now().Add(-2 * time.Hour),
	}))

	// Delete expired
	deleted, err := store.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Should have 1 remaining
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStore_Clear(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add cookies
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "a.com", Path: "/", Name: "c1", Value: "v1"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "b.com", Path: "/", Name: "c2", Value: "v2"}))

	// Clear
	err = store.Clear(ctx)
	require.NoError(t, err)

	// Should be empty
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestStore_Count(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Initially empty
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add cookies
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "a.com", Path: "/", Name: "c1", Value: "v1"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "b.com", Path: "/", Name: "c2", Value: "v2"}))

	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestStore_ClosedStore(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)

	// Close the store
	err = store.Close()
	require.NoError(t, err)

	ctx := context.Background()

	// All operations should return ErrStoreClosed
	_, err = store.Get(ctx, "test.com", "/", "test")
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)

	err = store.Set(ctx, &cookies.Cookie{Domain: "test.com", Path: "/", Name: "test", Value: "v"})
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)

	_, err = store.List(ctx, cookies.QueryOptions{})
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)

	err = store.Delete(ctx, "test.com", "/", "test")
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)

	err = store.DeleteByDomain(ctx, "test.com")
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)

	_, err = store.DeleteExpired(ctx)
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)

	err = store.Clear(ctx)
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)

	_, err = store.Count(ctx)
	assert.ErrorIs(t, err, cookies.ErrStoreClosed)
}

func TestStore_GetNotFound(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	_, err = store.Get(ctx, "nonexistent.com", "/", "missing")
	assert.ErrorIs(t, err, cookies.ErrNotFound)
}

func TestStore_ListByName(t *testing.T) {
	store, err := NewInMemory()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add cookies with same name on different domains
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "a.com", Path: "/", Name: "session", Value: "a"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "b.com", Path: "/", Name: "session", Value: "b"}))
	require.NoError(t, store.Set(ctx, &cookies.Cookie{Domain: "c.com", Path: "/", Name: "other", Value: "c"}))

	// List by name
	sessionCookies, err := store.List(ctx, cookies.QueryOptions{Name: "session", IncludeExpired: true})
	require.NoError(t, err)
	assert.Len(t, sessionCookies, 2)
}

func TestHelperFunctions(t *testing.T) {
	// Test boolToInt
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))

	// Test intToBool
	assert.True(t, intToBool(1))
	assert.True(t, intToBool(42))
	assert.False(t, intToBool(0))

	// Test nullTime
	now := time.Now()
	assert.Equal(t, now, nullTime(now))
	assert.Nil(t, nullTime(time.Time{}))
}

func TestStore_NewWithFilePath(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "cookies-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "cookies.db")

	// Create store with file path
	store, err := New(dbPath)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Set a cookie
	err = store.Set(ctx, &cookies.Cookie{
		Domain: "test.com",
		Path:   "/",
		Name:   "session",
		Value:  "abc123",
	})
	require.NoError(t, err)

	// Close and reopen to verify persistence
	store.Close()

	store2, err := New(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	// Cookie should persist
	got, err := store2.Get(ctx, "test.com", "/", "session")
	require.NoError(t, err)
	assert.Equal(t, "abc123", got.Value)
}
