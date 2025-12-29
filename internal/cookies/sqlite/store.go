package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/artpar/currier/internal/cookies"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store implements cookies.Store using SQLite.
type Store struct {
	mu     sync.RWMutex
	db     *sql.DB
	closed bool
}

// New creates a new SQLite-based cookie store.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize cookie database: %w", err)
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
		CREATE TABLE IF NOT EXISTS cookies (
			id TEXT PRIMARY KEY,
			domain TEXT NOT NULL,
			path TEXT NOT NULL,
			name TEXT NOT NULL,
			value TEXT NOT NULL,
			secure INTEGER NOT NULL DEFAULT 0,
			http_only INTEGER NOT NULL DEFAULT 0,
			same_site TEXT,
			expires DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(domain, path, name)
		);

		CREATE INDEX IF NOT EXISTS idx_cookies_domain ON cookies(domain);
		CREATE INDEX IF NOT EXISTS idx_cookies_expires ON cookies(expires);
		CREATE INDEX IF NOT EXISTS idx_cookies_domain_path ON cookies(domain, path);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Set stores or updates a cookie.
func (s *Store) Set(ctx context.Context, cookie *cookies.Cookie) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return cookies.ErrStoreClosed
	}

	if cookie.ID == "" {
		cookie.ID = uuid.New().String()
	}
	cookie.UpdatedAt = time.Now()
	if cookie.CreatedAt.IsZero() {
		cookie.CreatedAt = cookie.UpdatedAt
	}

	// Use INSERT OR REPLACE for upsert behavior
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO cookies
		(id, domain, path, name, value, secure, http_only, same_site, expires, created_at, updated_at)
		VALUES (
			COALESCE((SELECT id FROM cookies WHERE domain = ? AND path = ? AND name = ?), ?),
			?, ?, ?, ?, ?, ?, ?, ?,
			COALESCE((SELECT created_at FROM cookies WHERE domain = ? AND path = ? AND name = ?), ?),
			?
		)
	`,
		cookie.Domain, cookie.Path, cookie.Name, cookie.ID,
		cookie.Domain, cookie.Path, cookie.Name, cookie.Value,
		boolToInt(cookie.Secure), boolToInt(cookie.HttpOnly),
		cookie.SameSite, nullTime(cookie.Expires),
		cookie.Domain, cookie.Path, cookie.Name, cookie.CreatedAt,
		cookie.UpdatedAt,
	)
	return err
}

// Get retrieves a cookie by domain, path, and name.
func (s *Store) Get(ctx context.Context, domain, path, name string) (*cookies.Cookie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, cookies.ErrStoreClosed
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, domain, path, name, value, secure, http_only, same_site, expires, created_at, updated_at
		FROM cookies
		WHERE domain = ? AND path = ? AND name = ?
	`, domain, path, name)

	return scanCookie(row)
}

// List returns cookies matching the query options.
func (s *Store) List(ctx context.Context, opts cookies.QueryOptions) ([]*cookies.Cookie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, cookies.ErrStoreClosed
	}

	var conditions []string
	var args []interface{}

	if opts.Domain != "" {
		conditions = append(conditions, "domain = ?")
		args = append(args, opts.Domain)
	}

	if opts.Path != "" {
		conditions = append(conditions, "path = ?")
		args = append(args, opts.Path)
	}

	if opts.Name != "" {
		conditions = append(conditions, "name = ?")
		args = append(args, opts.Name)
	}

	if !opts.IncludeExpired {
		conditions = append(conditions, "(expires IS NULL OR expires > ?)")
		args = append(args, time.Now())
	}

	query := "SELECT id, domain, path, name, value, secure, http_only, same_site, expires, created_at, updated_at FROM cookies"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY domain, path, name"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCookies(rows)
}

// Delete removes a specific cookie.
func (s *Store) Delete(ctx context.Context, domain, path, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return cookies.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cookies WHERE domain = ? AND path = ? AND name = ?
	`, domain, path, name)
	return err
}

// DeleteByDomain removes all cookies for a domain.
func (s *Store) DeleteByDomain(ctx context.Context, domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return cookies.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM cookies WHERE domain = ?`, domain)
	return err
}

// DeleteExpired removes all expired cookies and returns count.
func (s *Store) DeleteExpired(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, cookies.ErrStoreClosed
	}

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM cookies WHERE expires IS NOT NULL AND expires <= ?
	`, time.Now())
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Clear removes all cookies.
func (s *Store) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return cookies.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM cookies`)
	return err
}

// Count returns total number of cookies.
func (s *Store) Count(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return 0, cookies.ErrStoreClosed
	}

	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cookies`).Scan(&count)
	return count, err
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

// Helper functions

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanCookie(row scannable) (*cookies.Cookie, error) {
	var c cookies.Cookie
	var secure, httpOnly int
	var sameSite sql.NullString
	var expires sql.NullTime

	err := row.Scan(
		&c.ID, &c.Domain, &c.Path, &c.Name, &c.Value,
		&secure, &httpOnly, &sameSite, &expires,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, cookies.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	c.Secure = intToBool(secure)
	c.HttpOnly = intToBool(httpOnly)
	if sameSite.Valid {
		c.SameSite = sameSite.String
	}
	if expires.Valid {
		c.Expires = expires.Time
	}

	return &c, nil
}

func scanCookies(rows *sql.Rows) ([]*cookies.Cookie, error) {
	var result []*cookies.Cookie
	for rows.Next() {
		c, err := scanCookie(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
