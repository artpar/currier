package history

import (
	"context"
	"testing"
)

// Tests for the Store interface are in testing.go and can be run against
// any Store implementation using RunStoreTests.

func TestStoreInterface(t *testing.T) {
	// This test verifies the interface compiles correctly.
	// Actual Store tests are run via RunStoreTests against implementations.
	var _ Store = (*mockStore)(nil)
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
