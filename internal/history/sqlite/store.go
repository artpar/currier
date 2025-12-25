package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/artpar/currier/internal/history"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store implements history.Store using SQLite.
type Store struct {
	mu     sync.RWMutex
	db     *sql.DB
	closed bool
}

// New creates a new SQLite-based history store.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
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
	// Core schema - always created
	coreSchema := `
		CREATE TABLE IF NOT EXISTS history (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			request_method TEXT NOT NULL,
			request_url TEXT NOT NULL,
			request_headers TEXT,
			request_body TEXT,
			response_status INTEGER NOT NULL,
			response_status_text TEXT,
			response_headers TEXT,
			response_body TEXT,
			response_time INTEGER,
			response_size INTEGER,
			collection_id TEXT,
			collection_name TEXT,
			request_id TEXT,
			request_name TEXT,
			environment TEXT,
			tags TEXT,
			notes TEXT,
			metadata TEXT,
			tests_passed INTEGER DEFAULT 0,
			tests_failed INTEGER DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_history_method ON history(request_method);
		CREATE INDEX IF NOT EXISTS idx_history_status ON history(response_status);
		CREATE INDEX IF NOT EXISTS idx_history_collection ON history(collection_id);
		CREATE INDEX IF NOT EXISTS idx_history_request ON history(request_id);
		CREATE INDEX IF NOT EXISTS idx_history_environment ON history(environment);
	`

	_, err := s.db.Exec(coreSchema)
	return err
}

// Add adds a new history entry and returns its ID.
func (s *Store) Add(ctx context.Context, entry history.Entry) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", history.ErrStoreClosed
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	headersJSON, _ := json.Marshal(entry.RequestHeaders)
	respHeadersJSON, _ := json.Marshal(entry.ResponseHeaders)
	tagsJSON, _ := json.Marshal(entry.Tags)
	metadataJSON, _ := json.Marshal(entry.Metadata)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO history (
			id, timestamp, request_method, request_url, request_headers, request_body,
			response_status, response_status_text, response_headers, response_body,
			response_time, response_size, collection_id, collection_name, request_id,
			request_name, environment, tags, notes, metadata, tests_passed, tests_failed
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.ID, entry.Timestamp, entry.RequestMethod, entry.RequestURL,
		string(headersJSON), entry.RequestBody, entry.ResponseStatus, entry.ResponseStatusText,
		string(respHeadersJSON), entry.ResponseBody, entry.ResponseTime, entry.ResponseSize,
		entry.CollectionID, entry.CollectionName, entry.RequestID, entry.RequestName,
		entry.Environment, string(tagsJSON), entry.Notes, string(metadataJSON),
		entry.TestsPassed, entry.TestsFailed,
	)

	if err != nil {
		return "", fmt.Errorf("failed to insert history entry: %w", err)
	}

	return entry.ID, nil
}

// Get retrieves a single history entry by ID.
func (s *Store) Get(ctx context.Context, id string) (history.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return history.Entry{}, history.ErrStoreClosed
	}

	if id == "" {
		return history.Entry{}, history.ErrInvalidID
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, timestamp, request_method, request_url, request_headers, request_body,
			response_status, response_status_text, response_headers, response_body,
			response_time, response_size, collection_id, collection_name, request_id,
			request_name, environment, tags, notes, metadata, tests_passed, tests_failed
		FROM history WHERE id = ?
	`, id)

	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return history.Entry{}, history.ErrNotFound
	}
	if err != nil {
		return history.Entry{}, fmt.Errorf("failed to get history entry: %w", err)
	}

	return entry, nil
}

// List retrieves history entries matching the query options.
func (s *Store) List(ctx context.Context, opts history.QueryOptions) ([]history.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, history.ErrStoreClosed
	}

	query, args := buildListQuery(opts, false)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list history entries: %w", err)
	}
	defer rows.Close()

	var entries []history.Entry
	for rows.Next() {
		entry, err := scanEntryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Count returns the number of entries matching the query options.
func (s *Store) Count(ctx context.Context, opts history.QueryOptions) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return 0, history.ErrStoreClosed
	}

	query, args := buildListQuery(opts, true)
	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count history entries: %w", err)
	}

	return count, nil
}

// Update updates an existing history entry.
func (s *Store) Update(ctx context.Context, entry history.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return history.ErrStoreClosed
	}

	headersJSON, _ := json.Marshal(entry.RequestHeaders)
	respHeadersJSON, _ := json.Marshal(entry.ResponseHeaders)
	tagsJSON, _ := json.Marshal(entry.Tags)
	metadataJSON, _ := json.Marshal(entry.Metadata)

	result, err := s.db.ExecContext(ctx, `
		UPDATE history SET
			timestamp = ?, request_method = ?, request_url = ?, request_headers = ?,
			request_body = ?, response_status = ?, response_status_text = ?,
			response_headers = ?, response_body = ?, response_time = ?, response_size = ?,
			collection_id = ?, collection_name = ?, request_id = ?, request_name = ?,
			environment = ?, tags = ?, notes = ?, metadata = ?, tests_passed = ?, tests_failed = ?
		WHERE id = ?
	`,
		entry.Timestamp, entry.RequestMethod, entry.RequestURL, string(headersJSON),
		entry.RequestBody, entry.ResponseStatus, entry.ResponseStatusText,
		string(respHeadersJSON), entry.ResponseBody, entry.ResponseTime, entry.ResponseSize,
		entry.CollectionID, entry.CollectionName, entry.RequestID, entry.RequestName,
		entry.Environment, string(tagsJSON), entry.Notes, string(metadataJSON),
		entry.TestsPassed, entry.TestsFailed, entry.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update history entry: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return history.ErrNotFound
	}

	return nil
}

// Delete removes a history entry by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return history.ErrStoreClosed
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM history WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete history entry: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return history.ErrNotFound
	}

	return nil
}

// DeleteMany removes multiple history entries matching the query options.
func (s *Store) DeleteMany(ctx context.Context, opts history.QueryOptions) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, history.ErrStoreClosed
	}

	query, args := buildDeleteQuery(opts)
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete history entries: %w", err)
	}

	return result.RowsAffected()
}

// Search performs a full-text search on history entries.
func (s *Store) Search(ctx context.Context, query string, opts history.QueryOptions) ([]history.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, history.ErrStoreClosed
	}

	// Build the search query
	sqlQuery, args := buildSearchQuery(query, opts)
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search history: %w", err)
	}
	defer rows.Close()

	var entries []history.Entry
	for rows.Next() {
		entry, err := scanEntryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Prune removes old entries based on the prune options.
func (s *Store) Prune(ctx context.Context, opts history.PruneOptions) (history.PruneResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return history.PruneResult{}, history.ErrStoreClosed
	}

	var result history.PruneResult

	// Calculate size before deletion
	var sizeQuery string
	var sizeArgs []interface{}

	if opts.OlderThan > 0 {
		cutoff := time.Now().Add(-opts.OlderThan)
		sizeQuery = "SELECT COALESCE(SUM(response_size), 0) FROM history WHERE timestamp < ?"
		sizeArgs = []interface{}{cutoff}

		if opts.CollectionID != "" {
			sizeQuery += " AND collection_id = ?"
			sizeArgs = append(sizeArgs, opts.CollectionID)
		}

		var size int64
		s.db.QueryRowContext(ctx, sizeQuery, sizeArgs...).Scan(&size)
		result.FreedBytes = size

		// Delete entries
		deleteQuery := "DELETE FROM history WHERE timestamp < ?"
		deleteArgs := []interface{}{cutoff}
		if opts.CollectionID != "" {
			deleteQuery += " AND collection_id = ?"
			deleteArgs = append(deleteArgs, opts.CollectionID)
		}

		res, err := s.db.ExecContext(ctx, deleteQuery, deleteArgs...)
		if err != nil {
			return result, fmt.Errorf("failed to prune history: %w", err)
		}
		result.DeletedCount, _ = res.RowsAffected()

	} else if opts.KeepLast > 0 {
		// Get total count first
		var total int64
		s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM history").Scan(&total)

		if total > int64(opts.KeepLast) {
			toDelete := total - int64(opts.KeepLast)

			// Get size of entries to delete
			s.db.QueryRowContext(ctx, `
				SELECT COALESCE(SUM(response_size), 0) FROM history
				WHERE id IN (
					SELECT id FROM history ORDER BY timestamp ASC LIMIT ?
				)
			`, toDelete).Scan(&result.FreedBytes)

			// Delete oldest entries
			res, err := s.db.ExecContext(ctx, `
				DELETE FROM history WHERE id IN (
					SELECT id FROM history ORDER BY timestamp ASC LIMIT ?
				)
			`, toDelete)
			if err != nil {
				return result, fmt.Errorf("failed to prune history: %w", err)
			}
			result.DeletedCount, _ = res.RowsAffected()
		}

	} else if !opts.Before.IsZero() {
		sizeQuery = "SELECT COALESCE(SUM(response_size), 0) FROM history WHERE timestamp < ?"
		s.db.QueryRowContext(ctx, sizeQuery, opts.Before).Scan(&result.FreedBytes)

		res, err := s.db.ExecContext(ctx, "DELETE FROM history WHERE timestamp < ?", opts.Before)
		if err != nil {
			return result, fmt.Errorf("failed to prune history: %w", err)
		}
		result.DeletedCount, _ = res.RowsAffected()
	}

	return result, nil
}

// Stats returns aggregate statistics about the history.
func (s *Store) Stats(ctx context.Context) (history.Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return history.Stats{}, history.ErrStoreClosed
	}

	var stats history.Stats
	stats.MethodCounts = make(map[string]int64)
	stats.StatusCounts = make(map[int]int64)
	stats.CollectionCounts = make(map[string]int64)

	// Total entries and size (handle timestamps separately due to SQLite string format)
	var oldestStr, newestStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(response_size), 0),
			COALESCE(AVG(response_time), 0),
			MIN(timestamp), MAX(timestamp)
		FROM history
	`).Scan(&stats.TotalEntries, &stats.TotalSize, &stats.AverageTime,
		&oldestStr, &newestStr)
	if err != nil && err != sql.ErrNoRows {
		return stats, fmt.Errorf("failed to get stats: %w", err)
	}

	// Parse timestamps from SQLite format
	if oldestStr.Valid && oldestStr.String != "" {
		if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", oldestStr.String); err == nil {
			stats.OldestEntry = t
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", oldestStr.String); err == nil {
			stats.OldestEntry = t
		} else if t, err := time.Parse(time.RFC3339Nano, oldestStr.String); err == nil {
			stats.OldestEntry = t
		}
	}
	if newestStr.Valid && newestStr.String != "" {
		if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", newestStr.String); err == nil {
			stats.NewestEntry = t
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", newestStr.String); err == nil {
			stats.NewestEntry = t
		} else if t, err := time.Parse(time.RFC3339Nano, newestStr.String); err == nil {
			stats.NewestEntry = t
		}
	}

	stats.TotalRequests = stats.TotalEntries

	// Method counts
	rows, err := s.db.QueryContext(ctx, `
		SELECT request_method, COUNT(*) FROM history GROUP BY request_method
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var method string
			var count int64
			rows.Scan(&method, &count)
			stats.MethodCounts[method] = count
		}
	}

	// Status counts
	rows, err = s.db.QueryContext(ctx, `
		SELECT response_status, COUNT(*) FROM history GROUP BY response_status
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status int
			var count int64
			rows.Scan(&status, &count)
			stats.StatusCounts[status] = count
		}
	}

	// Success rate (2xx responses)
	if stats.TotalEntries > 0 {
		var successCount int64
		s.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM history WHERE response_status >= 200 AND response_status < 300
		`).Scan(&successCount)
		stats.SuccessRate = float64(successCount) / float64(stats.TotalEntries)
	}

	// Collection counts
	rows, err = s.db.QueryContext(ctx, `
		SELECT COALESCE(collection_id, ''), COUNT(*) FROM history
		WHERE collection_id IS NOT NULL AND collection_id != ''
		GROUP BY collection_id
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var coll string
			var count int64
			rows.Scan(&coll, &count)
			if coll != "" {
				stats.CollectionCounts[coll] = count
			}
		}
	}

	return stats, nil
}

// Clear removes all history entries.
func (s *Store) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return history.ErrStoreClosed
	}

	_, err := s.db.ExecContext(ctx, "DELETE FROM history")
	if err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	return nil
}

// Close closes the store and releases resources.
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

func buildListQuery(opts history.QueryOptions, countOnly bool) (string, []interface{}) {
	var query string
	if countOnly {
		query = "SELECT COUNT(*) FROM history WHERE 1=1"
	} else {
		query = `
			SELECT id, timestamp, request_method, request_url, request_headers, request_body,
				response_status, response_status_text, response_headers, response_body,
				response_time, response_size, collection_id, collection_name, request_id,
				request_name, environment, tags, notes, metadata, tests_passed, tests_failed
			FROM history WHERE 1=1
		`
	}

	var args []interface{}

	if opts.Method != "" {
		query += " AND request_method = ?"
		args = append(args, opts.Method)
	}

	if opts.URLPattern != "" {
		query += " AND request_url LIKE ?"
		args = append(args, opts.URLPattern)
	}

	if opts.StatusMin > 0 {
		query += " AND response_status >= ?"
		args = append(args, opts.StatusMin)
	}

	if opts.StatusMax > 0 {
		query += " AND response_status <= ?"
		args = append(args, opts.StatusMax)
	}

	if opts.CollectionID != "" {
		query += " AND collection_id = ?"
		args = append(args, opts.CollectionID)
	}

	if opts.RequestID != "" {
		query += " AND request_id = ?"
		args = append(args, opts.RequestID)
	}

	if opts.Environment != "" {
		query += " AND environment = ?"
		args = append(args, opts.Environment)
	}

	if !opts.After.IsZero() {
		query += " AND timestamp > ?"
		args = append(args, opts.After)
	}

	if !opts.Before.IsZero() {
		query += " AND timestamp < ?"
		args = append(args, opts.Before)
	}

	if opts.TestsOnly {
		query += " AND (tests_passed > 0 OR tests_failed > 0)"
	}

	if opts.FailedTestsOnly {
		query += " AND tests_failed > 0"
	}

	if !countOnly {
		// Sorting
		sortBy := opts.SortBy
		if sortBy == "" {
			sortBy = "timestamp"
		}
		sortOrder := opts.SortOrder
		if sortOrder == "" {
			sortOrder = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", sortBy, strings.ToUpper(sortOrder))

		// Pagination
		if opts.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, opts.Limit)
		}

		if opts.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, opts.Offset)
		}
	}

	return query, args
}

func buildDeleteQuery(opts history.QueryOptions) (string, []interface{}) {
	query := "DELETE FROM history WHERE 1=1"
	var args []interface{}

	if opts.Method != "" {
		query += " AND request_method = ?"
		args = append(args, opts.Method)
	}

	if opts.CollectionID != "" {
		query += " AND collection_id = ?"
		args = append(args, opts.CollectionID)
	}

	if !opts.After.IsZero() {
		query += " AND timestamp > ?"
		args = append(args, opts.After)
	}

	if !opts.Before.IsZero() {
		query += " AND timestamp < ?"
		args = append(args, opts.Before)
	}

	return query, args
}

func buildSearchQuery(searchTerm string, opts history.QueryOptions) (string, []interface{}) {
	// Use LIKE-based search across multiple columns
	searchPattern := "%" + searchTerm + "%"

	query := `
		SELECT id, timestamp, request_method, request_url, request_headers, request_body,
			response_status, response_status_text, response_headers, response_body,
			response_time, response_size, collection_id, collection_name, request_id,
			request_name, environment, tags, notes, metadata, tests_passed, tests_failed
		FROM history
		WHERE (
			request_url LIKE ? OR
			request_method LIKE ? OR
			request_body LIKE ? OR
			response_body LIKE ? OR
			notes LIKE ? OR
			collection_name LIKE ? OR
			request_name LIKE ? OR
			environment LIKE ?
		)
	`

	args := []interface{}{
		searchPattern, searchPattern, searchPattern, searchPattern,
		searchPattern, searchPattern, searchPattern, searchPattern,
	}

	if opts.Method != "" {
		query += " AND request_method = ?"
		args = append(args, opts.Method)
	}

	if opts.CollectionID != "" {
		query += " AND collection_id = ?"
		args = append(args, opts.CollectionID)
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	return query, args
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanEntry(row *sql.Row) (history.Entry, error) {
	var entry history.Entry
	var headersJSON, respHeadersJSON, tagsJSON, metadataJSON sql.NullString

	err := row.Scan(
		&entry.ID, &entry.Timestamp, &entry.RequestMethod, &entry.RequestURL,
		&headersJSON, &entry.RequestBody, &entry.ResponseStatus, &entry.ResponseStatusText,
		&respHeadersJSON, &entry.ResponseBody, &entry.ResponseTime, &entry.ResponseSize,
		&entry.CollectionID, &entry.CollectionName, &entry.RequestID, &entry.RequestName,
		&entry.Environment, &tagsJSON, &entry.Notes, &metadataJSON,
		&entry.TestsPassed, &entry.TestsFailed,
	)
	if err != nil {
		return entry, err
	}

	if headersJSON.Valid {
		json.Unmarshal([]byte(headersJSON.String), &entry.RequestHeaders)
	}
	if respHeadersJSON.Valid {
		json.Unmarshal([]byte(respHeadersJSON.String), &entry.ResponseHeaders)
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &entry.Tags)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &entry.Metadata)
	}

	return entry, nil
}

func scanEntryRow(rows *sql.Rows) (history.Entry, error) {
	var entry history.Entry
	var headersJSON, respHeadersJSON, tagsJSON, metadataJSON sql.NullString

	err := rows.Scan(
		&entry.ID, &entry.Timestamp, &entry.RequestMethod, &entry.RequestURL,
		&headersJSON, &entry.RequestBody, &entry.ResponseStatus, &entry.ResponseStatusText,
		&respHeadersJSON, &entry.ResponseBody, &entry.ResponseTime, &entry.ResponseSize,
		&entry.CollectionID, &entry.CollectionName, &entry.RequestID, &entry.RequestName,
		&entry.Environment, &tagsJSON, &entry.Notes, &metadataJSON,
		&entry.TestsPassed, &entry.TestsFailed,
	)
	if err != nil {
		return entry, err
	}

	if headersJSON.Valid {
		json.Unmarshal([]byte(headersJSON.String), &entry.RequestHeaders)
	}
	if respHeadersJSON.Valid {
		json.Unmarshal([]byte(respHeadersJSON.String), &entry.ResponseHeaders)
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &entry.Tags)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &entry.Metadata)
	}

	return entry, nil
}
