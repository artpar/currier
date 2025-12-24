package history

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RunStoreTests runs the standard store test suite against any Store implementation.
// Use this to verify that a Store implementation correctly implements the interface.
func RunStoreTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("Add", func(t *testing.T) {
		runAddTests(t, newStore)
	})
	t.Run("Get", func(t *testing.T) {
		runGetTests(t, newStore)
	})
	t.Run("List", func(t *testing.T) {
		runListTests(t, newStore)
	})
	t.Run("Update", func(t *testing.T) {
		runUpdateTests(t, newStore)
	})
	t.Run("Delete", func(t *testing.T) {
		runDeleteTests(t, newStore)
	})
	t.Run("Search", func(t *testing.T) {
		runSearchTests(t, newStore)
	})
	t.Run("Prune", func(t *testing.T) {
		runPruneTests(t, newStore)
	})
	t.Run("Stats", func(t *testing.T) {
		runStatsTests(t, newStore)
	})
}

func runAddTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("adds entry and returns ID", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entry := Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com/users",
			ResponseStatus: 200,
			ResponseTime:   150,
		}

		id, err := store.Add(context.Background(), entry)

		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("adds entry with all fields", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entry := Entry{
			Timestamp:          time.Now(),
			RequestMethod:      "POST",
			RequestURL:         "https://api.example.com/users",
			RequestHeaders:     map[string]string{"Content-Type": "application/json"},
			RequestBody:        `{"name": "John"}`,
			ResponseStatus:     201,
			ResponseStatusText: "Created",
			ResponseHeaders:    map[string]string{"X-Request-ID": "abc123"},
			ResponseBody:       `{"id": 1, "name": "John"}`,
			ResponseTime:       234,
			ResponseSize:       128,
			CollectionID:       "coll-1",
			CollectionName:     "My API",
			RequestID:          "req-1",
			RequestName:        "Create User",
			Environment:        "production",
			Tags:               []string{"user", "create"},
			Notes:              "Test request",
			TestsPassed:        3,
			TestsFailed:        0,
		}

		id, err := store.Add(context.Background(), entry)
		require.NoError(t, err)

		retrieved, err := store.Get(context.Background(), id)
		require.NoError(t, err)

		assert.Equal(t, entry.RequestMethod, retrieved.RequestMethod)
		assert.Equal(t, entry.RequestURL, retrieved.RequestURL)
		assert.Equal(t, entry.RequestBody, retrieved.RequestBody)
		assert.Equal(t, entry.ResponseStatus, retrieved.ResponseStatus)
		assert.Equal(t, entry.ResponseBody, retrieved.ResponseBody)
		assert.Equal(t, entry.CollectionID, retrieved.CollectionID)
		assert.Equal(t, entry.Environment, retrieved.Environment)
		assert.Equal(t, entry.TestsPassed, retrieved.TestsPassed)
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entry := Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com",
			ResponseStatus: 200,
		}

		ids := make(map[string]bool)
		for i := 0; i < 10; i++ {
			id, err := store.Add(context.Background(), entry)
			require.NoError(t, err)
			assert.False(t, ids[id], "Duplicate ID generated")
			ids[id] = true
		}
	})
}

func runGetTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("retrieves existing entry", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entry := Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com/users/1",
			ResponseStatus: 200,
			ResponseBody:   `{"id": 1}`,
		}

		id, err := store.Add(context.Background(), entry)
		require.NoError(t, err)

		retrieved, err := store.Get(context.Background(), id)

		require.NoError(t, err)
		assert.Equal(t, id, retrieved.ID)
		assert.Equal(t, entry.RequestMethod, retrieved.RequestMethod)
		assert.Equal(t, entry.RequestURL, retrieved.RequestURL)
		assert.Equal(t, entry.ResponseStatus, retrieved.ResponseStatus)
	})

	t.Run("returns error for non-existent entry", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		_, err := store.Get(context.Background(), "non-existent-id")

		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns error for empty ID", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		_, err := store.Get(context.Background(), "")

		assert.Error(t, err)
	})
}

func runListTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("lists all entries", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		for i := 0; i < 5; i++ {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		entries, err := store.List(context.Background(), QueryOptions{})

		require.NoError(t, err)
		assert.Len(t, entries, 5)
	})

	t.Run("filters by method", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		methods := []string{"GET", "POST", "GET", "PUT", "GET"}
		for _, method := range methods {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  method,
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		entries, err := store.List(context.Background(), QueryOptions{Method: "GET"})

		require.NoError(t, err)
		assert.Len(t, entries, 3)
		for _, e := range entries {
			assert.Equal(t, "GET", e.RequestMethod)
		}
	})

	t.Run("filters by status range", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		statuses := []int{200, 201, 400, 404, 500}
		for _, status := range statuses {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: status,
			})
			require.NoError(t, err)
		}

		entries, err := store.List(context.Background(), QueryOptions{
			StatusMin: 400,
			StatusMax: 499,
		})

		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("filters by collection", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		collections := []string{"coll-1", "coll-1", "coll-2"}
		for _, coll := range collections {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
				CollectionID:   coll,
			})
			require.NoError(t, err)
		}

		entries, err := store.List(context.Background(), QueryOptions{
			CollectionID: "coll-1",
		})

		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("filters by time range", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		now := time.Now()
		times := []time.Time{
			now.Add(-48 * time.Hour),
			now.Add(-24 * time.Hour),
			now.Add(-1 * time.Hour),
			now,
		}
		for _, ts := range times {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      ts,
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		entries, err := store.List(context.Background(), QueryOptions{
			After: now.Add(-25 * time.Hour),
		})

		require.NoError(t, err)
		assert.Len(t, entries, 3)
	})

	t.Run("applies pagination", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		for i := 0; i < 10; i++ {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		page1, err := store.List(context.Background(), QueryOptions{
			Limit:  3,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Len(t, page1, 3)

		page2, err := store.List(context.Background(), QueryOptions{
			Limit:  3,
			Offset: 3,
		})
		require.NoError(t, err)
		assert.Len(t, page2, 3)

		assert.NotEqual(t, page1[0].ID, page2[0].ID)
	})

	t.Run("sorts by timestamp descending by default", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		now := time.Now()
		for i := 0; i < 3; i++ {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      now.Add(time.Duration(i) * time.Hour),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		entries, err := store.List(context.Background(), QueryOptions{})

		require.NoError(t, err)
		require.Len(t, entries, 3)
		assert.True(t, entries[0].Timestamp.After(entries[1].Timestamp))
		assert.True(t, entries[1].Timestamp.After(entries[2].Timestamp))
	})

	t.Run("filters by URL pattern", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		urls := []string{
			"https://api.example.com/users",
			"https://api.example.com/users/1",
			"https://api.example.com/posts",
			"https://other.com/users",
		}
		for _, url := range urls {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     url,
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		entries, err := store.List(context.Background(), QueryOptions{
			URLPattern: "%/users%",
		})

		require.NoError(t, err)
		assert.Len(t, entries, 3)
	})
}

func runUpdateTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("updates existing entry", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entry := Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com",
			ResponseStatus: 200,
			Notes:          "Original note",
		}

		id, err := store.Add(context.Background(), entry)
		require.NoError(t, err)

		entry.ID = id
		entry.Notes = "Updated note"
		entry.Tags = []string{"updated"}

		err = store.Update(context.Background(), entry)
		require.NoError(t, err)

		retrieved, err := store.Get(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, "Updated note", retrieved.Notes)
		assert.Contains(t, retrieved.Tags, "updated")
	})

	t.Run("returns error for non-existent entry", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entry := Entry{
			ID:             "non-existent",
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com",
			ResponseStatus: 200,
		}

		err := store.Update(context.Background(), entry)

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func runDeleteTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("deletes existing entry", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		id, err := store.Add(context.Background(), Entry{
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com",
			ResponseStatus: 200,
		})
		require.NoError(t, err)

		err = store.Delete(context.Background(), id)
		require.NoError(t, err)

		_, err = store.Get(context.Background(), id)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns error for non-existent entry", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		err := store.Delete(context.Background(), "non-existent")

		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("deletes many by query", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		for _, method := range []string{"GET", "GET", "POST"} {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  method,
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		deleted, err := store.DeleteMany(context.Background(), QueryOptions{
			Method: "GET",
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		remaining, err := store.List(context.Background(), QueryOptions{})
		require.NoError(t, err)
		assert.Len(t, remaining, 1)
		assert.Equal(t, "POST", remaining[0].RequestMethod)
	})

	t.Run("clear removes all entries", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		for i := 0; i < 5; i++ {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		err := store.Clear(context.Background())
		require.NoError(t, err)

		entries, err := store.List(context.Background(), QueryOptions{})
		require.NoError(t, err)
		assert.Len(t, entries, 0)
	})
}

func runSearchTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("searches in URL", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		urls := []string{
			"https://api.example.com/users",
			"https://api.example.com/posts",
			"https://api.example.com/comments",
		}
		for _, url := range urls {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     url,
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		results, err := store.Search(context.Background(), "users", QueryOptions{})

		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].RequestURL, "users")
	})

	t.Run("searches in response body", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		bodies := []string{
			`{"name": "John Doe"}`,
			`{"name": "Jane Smith"}`,
			`{"title": "Hello World"}`,
		}
		for _, body := range bodies {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
				ResponseBody:   body,
			})
			require.NoError(t, err)
		}

		results, err := store.Search(context.Background(), "John", QueryOptions{})

		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].ResponseBody, "John")
	})

	t.Run("searches in notes", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		notes := []string{
			"Important authentication test",
			"User creation flow",
			"Error handling check",
		}
		for _, note := range notes {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now(),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
				Notes:          note,
			})
			require.NoError(t, err)
		}

		results, err := store.Search(context.Background(), "authentication", QueryOptions{})

		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].Notes, "authentication")
	})

	t.Run("combines search with filters", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entries := []Entry{
			{Timestamp: time.Now(), RequestMethod: "GET", RequestURL: "https://api.example.com/users", ResponseStatus: 200},
			{Timestamp: time.Now(), RequestMethod: "POST", RequestURL: "https://api.example.com/users", ResponseStatus: 201},
			{Timestamp: time.Now(), RequestMethod: "GET", RequestURL: "https://api.example.com/posts", ResponseStatus: 200},
		}
		for _, e := range entries {
			_, err := store.Add(context.Background(), e)
			require.NoError(t, err)
		}

		results, err := store.Search(context.Background(), "users", QueryOptions{
			Method: "GET",
		})

		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "GET", results[0].RequestMethod)
		assert.Contains(t, results[0].RequestURL, "users")
	})
}

func runPruneTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("prunes entries older than duration", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		now := time.Now()
		times := []time.Time{
			now.Add(-48 * time.Hour),
			now.Add(-47 * time.Hour),
			now.Add(-12 * time.Hour),
			now,
		}
		for _, ts := range times {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      ts,
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		result, err := store.Prune(context.Background(), PruneOptions{
			OlderThan: 24 * time.Hour,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2), result.DeletedCount)

		remaining, err := store.List(context.Background(), QueryOptions{})
		require.NoError(t, err)
		assert.Len(t, remaining, 2)
	})

	t.Run("prunes keeping last N entries", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		for i := 0; i < 10; i++ {
			_, err := store.Add(context.Background(), Entry{
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				RequestMethod:  "GET",
				RequestURL:     "https://api.example.com",
				ResponseStatus: 200,
			})
			require.NoError(t, err)
		}

		result, err := store.Prune(context.Background(), PruneOptions{
			KeepLast: 5,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(5), result.DeletedCount)

		remaining, err := store.List(context.Background(), QueryOptions{})
		require.NoError(t, err)
		assert.Len(t, remaining, 5)
	})

	t.Run("prunes by collection", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		now := time.Now()
		entries := []Entry{
			{Timestamp: now.Add(-48 * time.Hour), CollectionID: "coll-1"},
			{Timestamp: now.Add(-48 * time.Hour), CollectionID: "coll-2"},
			{Timestamp: now, CollectionID: "coll-1"},
		}
		for _, e := range entries {
			e.RequestMethod = "GET"
			e.RequestURL = "https://api.example.com"
			e.ResponseStatus = 200
			_, err := store.Add(context.Background(), e)
			require.NoError(t, err)
		}

		result, err := store.Prune(context.Background(), PruneOptions{
			OlderThan:    24 * time.Hour,
			CollectionID: "coll-1",
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1), result.DeletedCount)

		remaining, err := store.List(context.Background(), QueryOptions{
			CollectionID: "coll-2",
		})
		require.NoError(t, err)
		assert.Len(t, remaining, 1)
	})
}

func runStatsTests(t *testing.T, newStore func() (Store, func())) {
	t.Run("returns correct statistics", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		entries := []Entry{
			{RequestMethod: "GET", ResponseStatus: 200, ResponseTime: 100, ResponseSize: 100},
			{RequestMethod: "GET", ResponseStatus: 200, ResponseTime: 200, ResponseSize: 200},
			{RequestMethod: "POST", ResponseStatus: 201, ResponseTime: 300, ResponseSize: 50},
			{RequestMethod: "GET", ResponseStatus: 404, ResponseTime: 50, ResponseSize: 30},
			{RequestMethod: "GET", ResponseStatus: 500, ResponseTime: 150, ResponseSize: 20},
		}
		for i, e := range entries {
			e.Timestamp = time.Now().Add(time.Duration(i) * time.Second)
			e.RequestURL = "https://api.example.com"
			_, err := store.Add(context.Background(), e)
			require.NoError(t, err)
		}

		stats, err := store.Stats(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(5), stats.TotalEntries)
		assert.Equal(t, int64(4), stats.MethodCounts["GET"])
		assert.Equal(t, int64(1), stats.MethodCounts["POST"])
		assert.Equal(t, int64(2), stats.StatusCounts[200])
		assert.Equal(t, int64(1), stats.StatusCounts[404])
		assert.InDelta(t, 160.0, stats.AverageTime, 1.0)
		assert.InDelta(t, 0.6, stats.SuccessRate, 0.01)
	})

	t.Run("returns empty stats for empty store", func(t *testing.T) {
		store, cleanup := newStore()
		defer cleanup()

		stats, err := store.Stats(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.TotalEntries)
	})
}
