package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/artpar/currier/internal/history"
)

// CacheStore extends Store with caching capabilities for response bodies.
type CacheStore struct {
	*Store
	cacheMu   sync.RWMutex
	hitCount  int64
	missCount int64
}

// NewCacheStore creates a new SQLite-based cache store.
func NewCacheStore(dbPath string) (*CacheStore, error) {
	store, err := New(dbPath)
	if err != nil {
		return nil, err
	}

	cs := &CacheStore{Store: store}
	if err := cs.initializeCache(); err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	return cs, nil
}

// NewInMemoryCacheStore creates a new in-memory cache store (useful for testing).
func NewInMemoryCacheStore() (*CacheStore, error) {
	store, err := NewInMemory()
	if err != nil {
		return nil, err
	}

	cs := &CacheStore{Store: store}
	if err := cs.initializeCache(); err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	return cs, nil
}

// initializeCache creates the cache table.
func (s *CacheStore) initializeCache() error {
	schema := `
		CREATE TABLE IF NOT EXISTS response_cache (
			hash TEXT PRIMARY KEY,
			body TEXT NOT NULL,
			size INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			access_count INTEGER DEFAULT 1,
			last_accessed DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_cache_created ON response_cache(created_at);
		CREATE INDEX IF NOT EXISTS idx_cache_accessed ON response_cache(last_accessed);
	`

	_, err := s.db.Exec(schema)
	return err
}

// GetCachedResponse retrieves a cached response body by its hash.
func (s *CacheStore) GetCachedResponse(ctx context.Context, hash string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return "", history.ErrStoreClosed
	}

	var body string
	err := s.db.QueryRowContext(ctx, `
		SELECT body FROM response_cache WHERE hash = ?
	`, hash).Scan(&body)

	if err == sql.ErrNoRows {
		s.cacheMu.Lock()
		s.missCount++
		s.cacheMu.Unlock()
		return "", history.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get cached response: %w", err)
	}

	// Update access stats
	s.db.ExecContext(ctx, `
		UPDATE response_cache
		SET access_count = access_count + 1, last_accessed = CURRENT_TIMESTAMP
		WHERE hash = ?
	`, hash)

	s.cacheMu.Lock()
	s.hitCount++
	s.cacheMu.Unlock()

	return body, nil
}

// CacheResponse stores a response body and returns its hash.
func (s *CacheStore) CacheResponse(ctx context.Context, body string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", history.ErrStoreClosed
	}

	// Calculate hash
	hasher := sha256.New()
	hasher.Write([]byte(body))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Check if already cached
	var existing string
	err := s.db.QueryRowContext(ctx, `
		SELECT hash FROM response_cache WHERE hash = ?
	`, hash).Scan(&existing)

	if err == nil {
		// Already cached, just update access
		s.db.ExecContext(ctx, `
			UPDATE response_cache
			SET access_count = access_count + 1, last_accessed = CURRENT_TIMESTAMP
			WHERE hash = ?
		`, hash)
		return hash, nil
	}

	// Insert new cache entry
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO response_cache (hash, body, size)
		VALUES (?, ?, ?)
	`, hash, body, len(body))

	if err != nil {
		return "", fmt.Errorf("failed to cache response: %w", err)
	}

	return hash, nil
}

// PruneCache removes unused cached responses.
func (s *CacheStore) PruneCache(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, history.ErrStoreClosed
	}

	// Remove cache entries that are not referenced by any history entry
	// This uses a subquery to find orphaned cache entries
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM response_cache
		WHERE hash NOT IN (
			SELECT DISTINCT response_body FROM history
			WHERE response_body IS NOT NULL AND response_body != ''
		)
	`)

	if err != nil {
		return 0, fmt.Errorf("failed to prune cache: %w", err)
	}

	return result.RowsAffected()
}

// CacheStats returns statistics about the response cache.
func (s *CacheStore) CacheStats(ctx context.Context) (history.CacheStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return history.CacheStats{}, history.ErrStoreClosed
	}

	var stats history.CacheStats

	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(size), 0) FROM response_cache
	`).Scan(&stats.TotalEntries, &stats.TotalSize)

	if err != nil && err != sql.ErrNoRows {
		return stats, fmt.Errorf("failed to get cache stats: %w", err)
	}

	s.cacheMu.RLock()
	stats.HitCount = s.hitCount
	stats.MissCount = s.missCount
	s.cacheMu.RUnlock()

	if stats.HitCount+stats.MissCount > 0 {
		stats.HitRate = float64(stats.HitCount) / float64(stats.HitCount+stats.MissCount)
	}

	return stats, nil
}

// ClearCache removes all cached responses.
func (s *CacheStore) ClearCache(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return history.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx, "DELETE FROM response_cache")
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	s.cacheMu.Lock()
	s.hitCount = 0
	s.missCount = 0
	s.cacheMu.Unlock()

	return nil
}

// Verify CacheStore implements history.CacheStore interface
var _ history.CacheStore = (*CacheStore)(nil)
