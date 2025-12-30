package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/artpar/currier/internal/starred"
	_ "modernc.org/sqlite"
)

// Store implements starred.Store using SQLite.
type Store struct {
	mu     sync.RWMutex
	db     *sql.DB
	closed bool
}

// New creates a new SQLite-based starred store.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open starred database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize starred database: %w", err)
	}

	return store, nil
}

// NewWithDB creates a store using an existing database connection.
// This allows sharing a database file with other stores (e.g., history.db).
func NewWithDB(db *sql.DB) (*Store, error) {
	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize starred tables: %w", err)
	}
	return store, nil
}

// NewInMemory creates a new in-memory SQLite store (useful for testing).
func NewInMemory() (*Store, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initialize creates the necessary tables and indexes.
func (s *Store) initialize() error {
	schema := `
		CREATE TABLE IF NOT EXISTS starred_requests (
			request_id TEXT PRIMARY KEY,
			starred_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_starred_at ON starred_requests(starred_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// IsStarred checks if a request is starred.
func (s *Store) IsStarred(ctx context.Context, requestID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return false, starred.ErrStoreClosed
	}

	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM starred_requests WHERE request_id = ?",
		requestID,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check starred status: %w", err)
	}

	return count > 0, nil
}

// Star marks a request as starred.
func (s *Store) Star(ctx context.Context, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return starred.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO starred_requests (request_id, starred_at) VALUES (?, ?)",
		requestID, time.Now().Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to star request: %w", err)
	}

	return nil
}

// Unstar removes the starred status from a request.
func (s *Store) Unstar(ctx context.Context, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return starred.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx,
		"DELETE FROM starred_requests WHERE request_id = ?",
		requestID,
	)

	if err != nil {
		return fmt.Errorf("failed to unstar request: %w", err)
	}

	return nil
}

// Toggle toggles the starred status and returns the new state.
func (s *Store) Toggle(ctx context.Context, requestID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false, starred.ErrStoreClosed
	}

	// Check current state
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM starred_requests WHERE request_id = ?",
		requestID,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check starred status: %w", err)
	}

	if count > 0 {
		// Currently starred, unstar it
		_, err = s.db.ExecContext(ctx,
			"DELETE FROM starred_requests WHERE request_id = ?",
			requestID,
		)
		if err != nil {
			return false, fmt.Errorf("failed to unstar request: %w", err)
		}
		return false, nil
	}

	// Not starred, star it
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO starred_requests (request_id, starred_at) VALUES (?, ?)",
		requestID, time.Now().Unix(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to star request: %w", err)
	}
	return true, nil
}

// ListStarred returns all starred request IDs.
func (s *Store) ListStarred(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, starred.ErrStoreClosed
	}

	rows, err := s.db.QueryContext(ctx,
		"SELECT request_id FROM starred_requests ORDER BY starred_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list starred requests: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan starred request: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// Clear removes all starred entries.
func (s *Store) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return starred.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx, "DELETE FROM starred_requests")
	if err != nil {
		return fmt.Errorf("failed to clear starred requests: %w", err)
	}

	return nil
}

// Count returns total number of starred requests.
func (s *Store) Count(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return 0, starred.ErrStoreClosed
	}

	var count int64
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM starred_requests",
	).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count starred requests: %w", err)
	}

	return count, nil
}

// Close closes the store.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return s.db.Close()
}
