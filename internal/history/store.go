package history

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrNotFound      = errors.New("history entry not found")
	ErrInvalidID     = errors.New("invalid history entry ID")
	ErrStoreClosed   = errors.New("history store is closed")
	ErrInvalidOption = errors.New("invalid query option")
)

// Store defines the interface for history storage operations.
type Store interface {
	// Add adds a new history entry and returns its ID.
	Add(ctx context.Context, entry Entry) (string, error)

	// Get retrieves a single history entry by ID.
	Get(ctx context.Context, id string) (Entry, error)

	// List retrieves history entries matching the query options.
	List(ctx context.Context, opts QueryOptions) ([]Entry, error)

	// Count returns the number of entries matching the query options.
	Count(ctx context.Context, opts QueryOptions) (int64, error)

	// Update updates an existing history entry.
	Update(ctx context.Context, entry Entry) error

	// Delete removes a history entry by ID.
	Delete(ctx context.Context, id string) error

	// DeleteMany removes multiple history entries matching the query options.
	DeleteMany(ctx context.Context, opts QueryOptions) (int64, error)

	// Search performs a full-text search on history entries.
	Search(ctx context.Context, query string, opts QueryOptions) ([]Entry, error)

	// Prune removes old entries based on the prune options.
	Prune(ctx context.Context, opts PruneOptions) (PruneResult, error)

	// Stats returns aggregate statistics about the history.
	Stats(ctx context.Context) (Stats, error)

	// Clear removes all history entries.
	Clear(ctx context.Context) error

	// Close closes the store and releases resources.
	Close() error
}

// CacheStore extends Store with caching capabilities for response bodies.
type CacheStore interface {
	Store

	// GetCachedResponse retrieves a cached response body by its hash.
	GetCachedResponse(ctx context.Context, hash string) (string, error)

	// CacheResponse stores a response body and returns its hash.
	CacheResponse(ctx context.Context, body string) (string, error)

	// PruneCache removes unused cached responses.
	PruneCache(ctx context.Context) (int64, error)

	// CacheStats returns statistics about the response cache.
	CacheStats(ctx context.Context) (CacheStats, error)
}

// CacheStats provides statistics about the response cache.
type CacheStats struct {
	TotalEntries int64 `json:"total_entries"`
	TotalSize    int64 `json:"total_size"`
	HitCount     int64 `json:"hit_count"`
	MissCount    int64 `json:"miss_count"`
	HitRate      float64 `json:"hit_rate"`
}
