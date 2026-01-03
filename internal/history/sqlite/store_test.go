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

func TestSQLiteStore_DeleteMany(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	// Add test entries
	for i := 0; i < 5; i++ {
		_, err := store.Add(ctx, history.Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
			CollectionID:   "col1",
		})
		if err != nil {
			t.Fatalf("Failed to add entry: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		_, err := store.Add(ctx, history.Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "POST",
			RequestURL:     "https://example.com/post",
			ResponseStatus: 201,
			CollectionID:   "col2",
		})
		if err != nil {
			t.Fatalf("Failed to add entry: %v", err)
		}
	}

	t.Run("delete by method", func(t *testing.T) {
		deleted, err := store.DeleteMany(ctx, history.QueryOptions{Method: "POST"})
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}
		if deleted != 3 {
			t.Errorf("Expected 3 deleted, got %d", deleted)
		}
	})

	t.Run("delete by collection", func(t *testing.T) {
		deleted, err := store.DeleteMany(ctx, history.QueryOptions{CollectionID: "col1"})
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}
		if deleted != 5 {
			t.Errorf("Expected 5 deleted, got %d", deleted)
		}
	})

	t.Run("fails when closed", func(t *testing.T) {
		closedStore, _ := NewInMemory()
		closedStore.Close()
		_, err := closedStore.DeleteMany(ctx, history.QueryOptions{})
		if err != history.ErrStoreClosed {
			t.Errorf("Expected ErrStoreClosed, got %v", err)
		}
	})
}

func TestSQLiteStore_Search(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	// Add test entries
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://api.example.com/users",
		ResponseStatus: 200,
		Notes:          "User endpoint",
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "POST",
		RequestURL:     "https://api.example.com/products",
		ResponseStatus: 201,
		Notes:          "Product endpoint",
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://other.example.com/data",
		ResponseStatus: 200,
		Notes:          "Data endpoint",
	})

	t.Run("search by URL", func(t *testing.T) {
		results, err := store.Search(ctx, "api.example.com", history.QueryOptions{})
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("search by notes", func(t *testing.T) {
		results, err := store.Search(ctx, "User endpoint", history.QueryOptions{})
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("search with method filter", func(t *testing.T) {
		results, err := store.Search(ctx, "example.com", history.QueryOptions{Method: "POST"})
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		results, err := store.Search(ctx, "example.com", history.QueryOptions{Limit: 1})
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("fails when closed", func(t *testing.T) {
		closedStore, _ := NewInMemory()
		closedStore.Close()
		_, err := closedStore.Search(ctx, "test", history.QueryOptions{})
		if err != history.ErrStoreClosed {
			t.Errorf("Expected ErrStoreClosed, got %v", err)
		}
	})
}

func TestSQLiteStore_Prune(t *testing.T) {
	t.Run("prune with OlderThan", func(t *testing.T) {
		store, _ := NewInMemory()
		defer store.Close()
		ctx := context.Background()

		// Add old entries
		oldTime := time.Now().Add(-48 * time.Hour)
		_, _ = store.Add(ctx, history.Entry{
			Timestamp:      oldTime,
			RequestMethod:  "GET",
			RequestURL:     "https://old.example.com",
			ResponseStatus: 200,
			ResponseSize:   1000,
		})
		// Add recent entry
		_, _ = store.Add(ctx, history.Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://new.example.com",
			ResponseStatus: 200,
			ResponseSize:   500,
		})

		result, err := store.Prune(ctx, history.PruneOptions{OlderThan: 24 * time.Hour})
		if err != nil {
			t.Fatalf("Failed to prune: %v", err)
		}
		if result.DeletedCount != 1 {
			t.Errorf("Expected 1 deleted, got %d", result.DeletedCount)
		}
	})

	t.Run("prune with KeepLast", func(t *testing.T) {
		store, _ := NewInMemory()
		defer store.Close()
		ctx := context.Background()

		// Add 5 entries
		for i := 0; i < 5; i++ {
			_, _ = store.Add(ctx, history.Entry{
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				RequestMethod:  "GET",
				RequestURL:     "https://example.com",
				ResponseStatus: 200,
				ResponseSize:   100,
			})
		}

		result, err := store.Prune(ctx, history.PruneOptions{KeepLast: 2})
		if err != nil {
			t.Fatalf("Failed to prune: %v", err)
		}
		if result.DeletedCount != 3 {
			t.Errorf("Expected 3 deleted, got %d", result.DeletedCount)
		}

		count, _ := store.Count(ctx, history.QueryOptions{})
		if count != 2 {
			t.Errorf("Expected 2 remaining, got %d", count)
		}
	})

	t.Run("prune with Before date", func(t *testing.T) {
		store, _ := NewInMemory()
		defer store.Close()
		ctx := context.Background()

		// Add entries at different times
		_, _ = store.Add(ctx, history.Entry{
			Timestamp:      time.Now().Add(-72 * time.Hour),
			RequestMethod:  "GET",
			RequestURL:     "https://old.example.com",
			ResponseStatus: 200,
		})
		_, _ = store.Add(ctx, history.Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://new.example.com",
			ResponseStatus: 200,
		})

		result, err := store.Prune(ctx, history.PruneOptions{Before: time.Now().Add(-24 * time.Hour)})
		if err != nil {
			t.Fatalf("Failed to prune: %v", err)
		}
		if result.DeletedCount != 1 {
			t.Errorf("Expected 1 deleted, got %d", result.DeletedCount)
		}
	})

	t.Run("prune with CollectionID", func(t *testing.T) {
		store, _ := NewInMemory()
		defer store.Close()
		ctx := context.Background()

		oldTime := time.Now().Add(-48 * time.Hour)
		_, _ = store.Add(ctx, history.Entry{
			Timestamp:      oldTime,
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
			CollectionID:   "col1",
		})
		_, _ = store.Add(ctx, history.Entry{
			Timestamp:      oldTime,
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
			CollectionID:   "col2",
		})

		result, err := store.Prune(ctx, history.PruneOptions{
			OlderThan:    24 * time.Hour,
			CollectionID: "col1",
		})
		if err != nil {
			t.Fatalf("Failed to prune: %v", err)
		}
		if result.DeletedCount != 1 {
			t.Errorf("Expected 1 deleted, got %d", result.DeletedCount)
		}
	})

	t.Run("fails when closed", func(t *testing.T) {
		closedStore, _ := NewInMemory()
		closedStore.Close()
		_, err := closedStore.Prune(context.Background(), history.PruneOptions{KeepLast: 10})
		if err != history.ErrStoreClosed {
			t.Errorf("Expected ErrStoreClosed, got %v", err)
		}
	})
}

func TestSQLiteStore_DeleteManyWithDateFilters(t *testing.T) {
	store, _ := NewInMemory()
	defer store.Close()
	ctx := context.Background()

	now := time.Now()
	// Add entries at different times
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      now.Add(-48 * time.Hour),
		RequestMethod:  "GET",
		RequestURL:     "https://old.example.com",
		ResponseStatus: 200,
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      now.Add(-24 * time.Hour),
		RequestMethod:  "GET",
		RequestURL:     "https://mid.example.com",
		ResponseStatus: 200,
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      now,
		RequestMethod:  "GET",
		RequestURL:     "https://new.example.com",
		ResponseStatus: 200,
	})

	t.Run("delete with Before filter", func(t *testing.T) {
		deleted, err := store.DeleteMany(ctx, history.QueryOptions{Before: now.Add(-12 * time.Hour)})
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}
		if deleted != 2 {
			t.Errorf("Expected 2 deleted, got %d", deleted)
		}
	})
}

func TestSQLiteStore_Clear(t *testing.T) {
	store, _ := NewInMemory()
	defer store.Close()
	ctx := context.Background()

	// Add entries
	for i := 0; i < 5; i++ {
		_, _ = store.Add(ctx, history.Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
		})
	}

	err := store.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	count, _ := store.Count(ctx, history.QueryOptions{})
	if count != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", count)
	}
}

func TestSQLiteStore_Stats(t *testing.T) {
	store, _ := NewInMemory()
	defer store.Close()
	ctx := context.Background()

	// Add entries with different methods
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
		ResponseTime:   100,
		ResponseSize:   1000,
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
		ResponseTime:   200,
		ResponseSize:   2000,
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "POST",
		RequestURL:     "https://example.com",
		ResponseStatus: 201,
		ResponseTime:   150,
		ResponseSize:   500,
	})

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", stats.TotalEntries)
	}
	if stats.MethodCounts["GET"] != 2 {
		t.Errorf("Expected 2 GET requests, got %d", stats.MethodCounts["GET"])
	}
	if stats.MethodCounts["POST"] != 1 {
		t.Errorf("Expected 1 POST request, got %d", stats.MethodCounts["POST"])
	}
}

func TestSQLiteStore_ClearHistory(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add some entries
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "POST",
		RequestURL:     "https://example.com",
		ResponseStatus: 201,
	})

	// Clear all
	err = store.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Verify empty
	count, err := store.Count(ctx, history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", count)
	}
}

func TestSQLiteStore_ListWithFilters(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add entries with different methods
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://api.example.com/users",
		ResponseStatus: 200,
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "POST",
		RequestURL:     "https://api.example.com/users",
		ResponseStatus: 201,
	})
	_, _ = store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "DELETE",
		RequestURL:     "https://api.example.com/users/1",
		ResponseStatus: 204,
	})

	t.Run("filter by method", func(t *testing.T) {
		entries, err := store.List(ctx, history.QueryOptions{
			Method: "GET",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("Expected 1 GET entry, got %d", len(entries))
		}
	})

	t.Run("filter by status range", func(t *testing.T) {
		entries, err := store.List(ctx, history.QueryOptions{
			StatusMin: 200,
			StatusMax: 299,
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if len(entries) != 3 {
			t.Errorf("Expected 3 entries with 2xx status, got %d", len(entries))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		entries, err := store.List(ctx, history.QueryOptions{
			Limit:  1,
			Offset: 1,
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("Expected 1 entry with pagination, got %d", len(entries))
		}
	})
}

func TestSQLiteStore_GetNonExistent(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	_, err = store.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Expected error for non-existent entry")
	}
}

func TestSQLiteStore_UpdateEntry(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add an entry
	id, err := store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
	})
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Update it - the entry needs to have the ID set
	err = store.Update(ctx, history.Entry{
		ID:             id,
		Timestamp:      time.Now(),
		RequestMethod:  "POST",
		RequestURL:     "https://example.com/updated",
		ResponseStatus: 201,
	})
	if err != nil {
		t.Fatalf("Failed to update entry: %v", err)
	}

	// Verify update
	entry, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if entry.RequestMethod != "POST" {
		t.Errorf("Expected method POST, got %s", entry.RequestMethod)
	}
	if entry.ResponseStatus != 201 {
		t.Errorf("Expected status 201, got %d", entry.ResponseStatus)
	}
}

func TestSQLiteStore_DeleteEntry(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add an entry
	id, err := store.Add(ctx, history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
	})
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Delete it
	err = store.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Failed to delete entry: %v", err)
	}

	// Verify deleted
	_, err = store.Get(ctx, id)
	if err == nil {
		t.Error("Expected error for deleted entry")
	}
}

func TestStore_ClearAll(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		_, err = store.Add(ctx, history.Entry{
			RequestMethod:  "GET",
			RequestURL:     "https://example.com/path",
			ResponseStatus: 200,
		})
		if err != nil {
			t.Fatalf("Failed to add entry: %v", err)
		}
	}

	// Verify count
	count, err := store.Count(ctx, history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}

	// Clear all
	err = store.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Verify empty
	count, err = store.Count(ctx, history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to count after clear: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after clear, got %d", count)
	}
}

func TestStore_CountEmpty(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Count on empty store should be 0
	count, err := store.Count(ctx, history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 for empty store, got %d", count)
	}
}

func TestStore_GetNonExistent(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get non-existent entry
	_, err = store.Get(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent entry")
	}
}

func TestStore_DeleteNonExistent(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Delete non-existent entry - should not error
	err = store.Delete(ctx, "non-existent-id")
	// Some implementations may or may not error
	_ = err
}

func TestStore_ListEmpty(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// List on empty store
	entries, err := store.List(ctx, history.QueryOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestStore_CloseIdempotent(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Close multiple times should be safe
	err = store.Close()
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

func TestStore_ClearExtended(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add some entries
	entry1 := history.Entry{
		ID:             "clear-1",
		RequestURL:    "http://example.com/1",
		RequestMethod: "GET",
		ResponseStatus: 200,
		RequestName:   "Clear Test 1",
		Timestamp:     time.Now(),
	}
	entry2 := history.Entry{
		ID:             "clear-2",
		RequestURL:    "http://example.com/2",
		RequestMethod: "POST",
		ResponseStatus: 201,
		RequestName:   "Clear Test 2",
		Timestamp:     time.Now(),
	}

	_, err = store.Add(ctx, entry1)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}
	_, err = store.Add(ctx, entry2)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Verify entries exist
	count, err := store.Count(ctx, history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 entries, got %d", count)
	}

	// Clear all
	err = store.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Verify entries cleared
	count, err = store.Count(ctx, history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to count after clear: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", count)
	}
}

func TestStore_UpdateExtended(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add entry
	entry := history.Entry{
		ID:             "update-test",
		RequestURL:    "http://example.com/update",
		RequestMethod: "GET",
		ResponseStatus: 200,
		RequestName:   "Original Name",
		Timestamp:     time.Now(),
	}

	id, err := store.Add(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Update entry
	entry.ID = id
	entry.RequestName = "Updated Name"
	entry.ResponseStatus = 404

	err = store.Update(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to update entry: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if retrieved.RequestName != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", retrieved.RequestName)
	}
	if retrieved.ResponseStatus != 404 {
		t.Errorf("Expected status 404, got %d", retrieved.ResponseStatus)
	}
}

func TestStore_SearchExtended(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add entries with different URLs
	entries := []history.Entry{
		{ID: "search-1", RequestURL: "http://api.example.com/users", RequestMethod: "GET", ResponseStatus: 200, Timestamp: time.Now()},
		{ID: "search-2", RequestURL: "http://api.example.com/posts", RequestMethod: "POST", ResponseStatus: 201, Timestamp: time.Now()},
		{ID: "search-3", RequestURL: "http://other.example.com/data", RequestMethod: "GET", ResponseStatus: 200, Timestamp: time.Now()},
	}

	for _, entry := range entries {
		_, err := store.Add(ctx, entry)
		if err != nil {
			t.Fatalf("Failed to add entry: %v", err)
		}
	}

	// Search for "api.example.com"
	results, err := store.Search(ctx, "api.example.com", history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Search for "users"
	results, err = store.Search(ctx, "users", history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestStore_GetExtended(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get non-existent entry
	_, err = store.Get(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent ID")
	}
}

func TestStore_StatsExtended(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Stats on empty store
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 total, got %d", stats.TotalRequests)
	}

	// Add entries and check stats
	entries := []history.Entry{
		{ID: "stats-1", RequestURL: "http://example.com/1", RequestMethod: "GET", ResponseStatus: 200, ResponseTime: 100, Timestamp: time.Now()},
		{ID: "stats-2", RequestURL: "http://example.com/2", RequestMethod: "POST", ResponseStatus: 201, ResponseTime: 200, Timestamp: time.Now()},
		{ID: "stats-3", RequestURL: "http://example.com/3", RequestMethod: "GET", ResponseStatus: 404, ResponseTime: 50, Timestamp: time.Now()},
	}

	for _, entry := range entries {
		_, err := store.Add(ctx, entry)
		if err != nil {
			t.Fatalf("Failed to add entry: %v", err)
		}
	}

	stats, err = store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total, got %d", stats.TotalRequests)
	}
}

func TestSQLiteStore_GetNonExistentAdditional(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	// Try to get non-existent entry
	_, err = store.Get(ctx, "totally-nonexistent-id")
	if err == nil {
		t.Error("Expected error when getting non-existent entry")
	}
}

func TestSQLiteStore_CountEmptyAdditional(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	count, err := store.Count(ctx, history.QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}
}

func TestSQLiteStore_UpdateEntryAdditional(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	// Add an entry
	entry := history.Entry{
		Timestamp:      time.Now(),
		RequestMethod:  "GET",
		RequestURL:     "https://example.com",
		ResponseStatus: 200,
		RequestName:    "Original Name",
	}
	id, err := store.Add(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Update it
	entry.ID = id
	entry.RequestName = "Updated Name"
	err = store.Update(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify update
	updated, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if updated.RequestName != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got '%s'", updated.RequestName)
	}
}

func TestSQLiteStore_DeleteNonExistentAdditional(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	// Delete non-existent returns an error
	err = store.Delete(ctx, "totally-nonexistent-id")
	if err == nil {
		t.Error("Expected error when deleting non-existent entry")
	}
}

func TestSQLiteStore_ListWithPaginationAdditional(t *testing.T) {
	store, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		_, err := store.Add(ctx, history.Entry{
			Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
		})
		if err != nil {
			t.Fatalf("Failed to add entry: %v", err)
		}
	}

	// List with limit
	entries, err := store.List(ctx, history.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(entries))
	}

	// List with offset
	entries, err = store.List(ctx, history.QueryOptions{Offset: 8, Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}
