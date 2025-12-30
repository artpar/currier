package history

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the Store interface are in testing.go and can be run against
// any Store implementation using RunStoreTests.

func TestStoreInterface(t *testing.T) {
	// This test verifies the interface compiles correctly.
	// Actual Store tests are run via RunStoreTests against implementations.
	var _ Store = (*mockStore)(nil)
}

func TestCacheStoreInterface(t *testing.T) {
	// Verify CacheStore extends Store
	var _ CacheStore = (*mockCacheStore)(nil)
}

// mockStore is a minimal mock for compile-time interface checking.
type mockStore struct{}

func (m *mockStore) Add(ctx context.Context, entry Entry) (string, error)           { return "", nil }
func (m *mockStore) Get(ctx context.Context, id string) (Entry, error)              { return Entry{}, nil }
func (m *mockStore) List(ctx context.Context, opts QueryOptions) ([]Entry, error)   { return nil, nil }
func (m *mockStore) Count(ctx context.Context, opts QueryOptions) (int64, error)    { return 0, nil }
func (m *mockStore) Update(ctx context.Context, entry Entry) error                  { return nil }
func (m *mockStore) Delete(ctx context.Context, id string) error                    { return nil }
func (m *mockStore) DeleteMany(ctx context.Context, opts QueryOptions) (int64, error) { return 0, nil }
func (m *mockStore) Search(ctx context.Context, query string, opts QueryOptions) ([]Entry, error) { return nil, nil }
func (m *mockStore) Prune(ctx context.Context, opts PruneOptions) (PruneResult, error) { return PruneResult{}, nil }
func (m *mockStore) Stats(ctx context.Context) (Stats, error)                       { return Stats{}, nil }
func (m *mockStore) Clear(ctx context.Context) error                                { return nil }
func (m *mockStore) Close() error                                                    { return nil }

// mockCacheStore for CacheStore interface checking
type mockCacheStore struct {
	mockStore
}

func (m *mockCacheStore) GetCachedResponse(ctx context.Context, hash string) (string, error) { return "", nil }
func (m *mockCacheStore) CacheResponse(ctx context.Context, body string) (string, error)    { return "", nil }
func (m *mockCacheStore) PruneCache(ctx context.Context) (int64, error)                     { return 0, nil }
func (m *mockCacheStore) CacheStats(ctx context.Context) (CacheStats, error)                { return CacheStats{}, nil }

// ============================================================================
// Error Tests
// ============================================================================

func TestErrors(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		assert.Error(t, ErrNotFound)
		assert.Equal(t, "history entry not found", ErrNotFound.Error())
	})

	t.Run("ErrInvalidID", func(t *testing.T) {
		assert.Error(t, ErrInvalidID)
		assert.Equal(t, "invalid history entry ID", ErrInvalidID.Error())
	})

	t.Run("ErrStoreClosed", func(t *testing.T) {
		assert.Error(t, ErrStoreClosed)
		assert.Equal(t, "history store is closed", ErrStoreClosed.Error())
	})

	t.Run("ErrInvalidOption", func(t *testing.T) {
		assert.Error(t, ErrInvalidOption)
		assert.Equal(t, "invalid query option", ErrInvalidOption.Error())
	})

	t.Run("errors are comparable", func(t *testing.T) {
		err := ErrNotFound
		assert.True(t, errors.Is(err, ErrNotFound))
		assert.False(t, errors.Is(err, ErrInvalidID))
	})
}

// ============================================================================
// Entry Model Tests
// ============================================================================

func TestEntry_JSONSerialization(t *testing.T) {
	t.Run("serializes all fields", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		entry := Entry{
			ID:                 "test-id",
			Timestamp:          now,
			RequestMethod:      "POST",
			RequestURL:         "https://api.example.com/users",
			RequestHeaders:     map[string]string{"Content-Type": "application/json"},
			RequestBody:        `{"name":"John"}`,
			ResponseStatus:     201,
			ResponseStatusText: "Created",
			ResponseHeaders:    map[string]string{"X-Request-ID": "abc123"},
			ResponseBody:       `{"id":1,"name":"John"}`,
			ResponseTime:       150,
			ResponseSize:       1024,
			CollectionID:       "coll-1",
			CollectionName:     "My API",
			RequestID:          "req-1",
			RequestName:        "Create User",
			Environment:        "production",
			Tags:               []string{"user", "create"},
			Notes:              "Test note",
			Metadata:           map[string]string{"key": "value"},
			TestsPassed:        3,
			TestsFailed:        1,
		}

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		var decoded Entry
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, entry.ID, decoded.ID)
		assert.Equal(t, entry.RequestMethod, decoded.RequestMethod)
		assert.Equal(t, entry.RequestURL, decoded.RequestURL)
		assert.Equal(t, entry.RequestBody, decoded.RequestBody)
		assert.Equal(t, entry.ResponseStatus, decoded.ResponseStatus)
		assert.Equal(t, entry.ResponseBody, decoded.ResponseBody)
		assert.Equal(t, entry.ResponseTime, decoded.ResponseTime)
		assert.Equal(t, entry.CollectionID, decoded.CollectionID)
		assert.Equal(t, entry.Environment, decoded.Environment)
		assert.Equal(t, entry.Tags, decoded.Tags)
		assert.Equal(t, entry.TestsPassed, decoded.TestsPassed)
		assert.Equal(t, entry.TestsFailed, decoded.TestsFailed)
	})

	t.Run("omits empty fields", func(t *testing.T) {
		entry := Entry{
			ID:             "test-id",
			Timestamp:      time.Now(),
			RequestMethod:  "GET",
			RequestURL:     "https://api.example.com",
			ResponseStatus: 200,
		}

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		// Check that empty fields are omitted
		jsonStr := string(data)
		assert.NotContains(t, jsonStr, "request_headers")
		assert.NotContains(t, jsonStr, "request_body")
		assert.NotContains(t, jsonStr, "tags")
		assert.NotContains(t, jsonStr, "notes")
	})
}

// ============================================================================
// QueryOptions Tests
// ============================================================================

func TestQueryOptions(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		opts := QueryOptions{}
		assert.Empty(t, opts.Method)
		assert.Empty(t, opts.URLPattern)
		assert.Equal(t, 0, opts.StatusMin)
		assert.Equal(t, 0, opts.StatusMax)
		assert.Equal(t, 0, opts.Limit)
		assert.Equal(t, 0, opts.Offset)
		assert.False(t, opts.TestsOnly)
		assert.False(t, opts.FailedTestsOnly)
	})

	t.Run("with all filters", func(t *testing.T) {
		now := time.Now()
		opts := QueryOptions{
			Method:          "POST",
			URLPattern:      "%/users%",
			StatusMin:       200,
			StatusMax:       299,
			CollectionID:    "coll-1",
			RequestID:       "req-1",
			Environment:     "production",
			Tags:            []string{"important"},
			After:           now.Add(-24 * time.Hour),
			Before:          now,
			Search:          "test",
			TestsOnly:       true,
			FailedTestsOnly: false,
			Limit:           50,
			Offset:          100,
			SortBy:          "response_time",
			SortOrder:       "asc",
		}

		assert.Equal(t, "POST", opts.Method)
		assert.Equal(t, "%/users%", opts.URLPattern)
		assert.Equal(t, 200, opts.StatusMin)
		assert.Equal(t, 299, opts.StatusMax)
		assert.Equal(t, "coll-1", opts.CollectionID)
		assert.Equal(t, 50, opts.Limit)
		assert.Equal(t, "response_time", opts.SortBy)
		assert.True(t, opts.TestsOnly)
	})
}

// ============================================================================
// Stats Tests
// ============================================================================

func TestStats_JSONSerialization(t *testing.T) {
	t.Run("serializes all fields", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		stats := Stats{
			TotalEntries:     100,
			TotalRequests:    95,
			TotalSize:        1024000,
			OldestEntry:      now.Add(-30 * 24 * time.Hour),
			NewestEntry:      now,
			MethodCounts:     map[string]int64{"GET": 60, "POST": 30, "PUT": 10},
			StatusCounts:     map[int]int64{200: 70, 201: 20, 404: 5, 500: 5},
			AverageTime:      156.5,
			SuccessRate:      0.9,
			CollectionCounts: map[string]int64{"coll-1": 50, "coll-2": 50},
		}

		data, err := json.Marshal(stats)
		require.NoError(t, err)

		var decoded Stats
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, stats.TotalEntries, decoded.TotalEntries)
		assert.Equal(t, stats.TotalRequests, decoded.TotalRequests)
		assert.Equal(t, stats.TotalSize, decoded.TotalSize)
		assert.Equal(t, stats.MethodCounts["GET"], decoded.MethodCounts["GET"])
		assert.Equal(t, stats.StatusCounts[200], decoded.StatusCounts[200])
		assert.InDelta(t, stats.AverageTime, decoded.AverageTime, 0.01)
		assert.InDelta(t, stats.SuccessRate, decoded.SuccessRate, 0.01)
	})
}

// ============================================================================
// PruneOptions Tests
// ============================================================================

func TestPruneOptions(t *testing.T) {
	t.Run("time-based pruning options", func(t *testing.T) {
		opts := PruneOptions{
			OlderThan: 7 * 24 * time.Hour,
		}
		assert.Equal(t, 7*24*time.Hour, opts.OlderThan)
	})

	t.Run("count-based pruning options", func(t *testing.T) {
		opts := PruneOptions{
			KeepLast: 1000,
		}
		assert.Equal(t, 1000, opts.KeepLast)
	})

	t.Run("size-based pruning options", func(t *testing.T) {
		opts := PruneOptions{
			MaxTotalSize: 100 * 1024 * 1024, // 100MB
		}
		assert.Equal(t, int64(100*1024*1024), opts.MaxTotalSize)
	})

	t.Run("selective pruning options", func(t *testing.T) {
		opts := PruneOptions{
			CollectionID: "coll-1",
			Method:       "GET",
			StatusMin:    400,
			StatusMax:    599,
		}
		assert.Equal(t, "coll-1", opts.CollectionID)
		assert.Equal(t, "GET", opts.Method)
		assert.Equal(t, 400, opts.StatusMin)
		assert.Equal(t, 599, opts.StatusMax)
	})
}

// ============================================================================
// PruneResult Tests
// ============================================================================

func TestPruneResult_JSONSerialization(t *testing.T) {
	result := PruneResult{
		DeletedCount: 150,
		FreedBytes:   5 * 1024 * 1024,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded PruneResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.DeletedCount, decoded.DeletedCount)
	assert.Equal(t, result.FreedBytes, decoded.FreedBytes)
}

// ============================================================================
// CacheStats Tests
// ============================================================================

func TestCacheStats_JSONSerialization(t *testing.T) {
	stats := CacheStats{
		TotalEntries: 500,
		TotalSize:    10 * 1024 * 1024,
		HitCount:     1000,
		MissCount:    50,
		HitRate:      0.952,
	}

	data, err := json.Marshal(stats)
	require.NoError(t, err)

	var decoded CacheStats
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, stats.TotalEntries, decoded.TotalEntries)
	assert.Equal(t, stats.TotalSize, decoded.TotalSize)
	assert.Equal(t, stats.HitCount, decoded.HitCount)
	assert.Equal(t, stats.MissCount, decoded.MissCount)
	assert.InDelta(t, stats.HitRate, decoded.HitRate, 0.001)
}
