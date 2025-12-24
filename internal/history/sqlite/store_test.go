package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/currier/internal/history"
)

// TestSQLiteStore runs the standard store test suite against SQLite.
func TestSQLiteStore(t *testing.T) {
	history.RunStoreTests(t, func() (history.Store, func()) {
		store, err := NewInMemory()
		if err != nil {
			t.Fatalf("Failed to create in-memory store: %v", err)
		}
		return store, func() {
			store.Close()
		}
	})
}

// Additional SQLite-specific tests

func TestSQLiteStore_Persistence(t *testing.T) {
	t.Run("data persists to disk", func(t *testing.T) {
		// Create a temp file
		tmpFile := t.TempDir() + "/test.db"

		// Create store and add entry
		store, err := New(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		entry := history.Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
		}
		id, err := store.Add(context.Background(), entry)
		if err != nil {
			t.Fatalf("Failed to add entry: %v", err)
		}
		store.Close()

		// Reopen and verify
		store2, err := New(tmpFile)
		if err != nil {
			t.Fatalf("Failed to reopen store: %v", err)
		}
		defer store2.Close()

		retrieved, err := store2.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("Failed to get entry: %v", err)
		}

		if retrieved.RequestMethod != entry.RequestMethod {
			t.Errorf("Expected method %s, got %s", entry.RequestMethod, retrieved.RequestMethod)
		}
	})
}

func TestSQLiteStore_Concurrent(t *testing.T) {
	t.Run("handles concurrent writes", func(t *testing.T) {
		store, err := NewInMemory()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 10; j++ {
					_, err := store.Add(ctx, history.Entry{
						Timestamp:      time.Now(),
						RequestMethod:  "GET",
						RequestURL:     "https://example.com",
						ResponseStatus: 200,
					})
					if err != nil {
						t.Errorf("Failed to add entry: %v", err)
					}
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		count, err := store.Count(ctx, history.QueryOptions{})
		if err != nil {
			t.Fatalf("Failed to count: %v", err)
		}

		if count != 100 {
			t.Errorf("Expected 100 entries, got %d", count)
		}
	})
}

func TestSQLiteStore_Close(t *testing.T) {
	t.Run("operations fail after close", func(t *testing.T) {
		store, err := NewInMemory()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		store.Close()

		_, err = store.Add(context.Background(), history.Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
		})

		if err != history.ErrStoreClosed {
			t.Errorf("Expected ErrStoreClosed, got %v", err)
		}
	})
}
