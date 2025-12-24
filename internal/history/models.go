package history

import (
	"time"
)

// Entry represents a single request/response history entry.
type Entry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`

	// Request data
	RequestMethod  string            `json:"request_method"`
	RequestURL     string            `json:"request_url"`
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	RequestBody    string            `json:"request_body,omitempty"`

	// Response data
	ResponseStatus     int               `json:"response_status"`
	ResponseStatusText string            `json:"response_status_text,omitempty"`
	ResponseHeaders    map[string]string `json:"response_headers,omitempty"`
	ResponseBody       string            `json:"response_body,omitempty"`
	ResponseTime       int64             `json:"response_time"` // milliseconds
	ResponseSize       int64             `json:"response_size"` // bytes

	// Metadata
	CollectionID   string            `json:"collection_id,omitempty"`
	CollectionName string            `json:"collection_name,omitempty"`
	RequestID      string            `json:"request_id,omitempty"`
	RequestName    string            `json:"request_name,omitempty"`
	Environment    string            `json:"environment,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`

	// Test results
	TestsPassed int `json:"tests_passed"`
	TestsFailed int `json:"tests_failed"`
}

// QueryOptions specifies filters and pagination for history queries.
type QueryOptions struct {
	// Filters
	Method         string    // Filter by HTTP method
	URLPattern     string    // Filter by URL pattern (supports wildcards)
	StatusMin      int       // Minimum status code
	StatusMax      int       // Maximum status code
	CollectionID   string    // Filter by collection
	RequestID      string    // Filter by request
	Environment    string    // Filter by environment
	Tags           []string  // Filter by tags (any match)
	After          time.Time // Only entries after this time
	Before         time.Time // Only entries before this time
	Search         string    // Full-text search query
	TestsOnly      bool      // Only entries with tests
	FailedTestsOnly bool     // Only entries with failed tests

	// Pagination
	Limit  int // Maximum number of results (0 = no limit)
	Offset int // Number of results to skip

	// Sorting
	SortBy    string // Field to sort by (timestamp, response_time, response_size)
	SortOrder string // "asc" or "desc" (default: desc for timestamp)
}

// Stats provides aggregate statistics about history.
type Stats struct {
	TotalEntries     int64         `json:"total_entries"`
	TotalRequests    int64         `json:"total_requests"`
	TotalSize        int64         `json:"total_size"`
	OldestEntry      time.Time     `json:"oldest_entry"`
	NewestEntry      time.Time     `json:"newest_entry"`
	MethodCounts     map[string]int64 `json:"method_counts"`
	StatusCounts     map[int]int64    `json:"status_counts"`
	AverageTime      float64       `json:"average_time"`
	SuccessRate      float64       `json:"success_rate"` // 2xx responses / total
	CollectionCounts map[string]int64 `json:"collection_counts"`
}

// PruneOptions specifies criteria for pruning old history entries.
type PruneOptions struct {
	// Time-based pruning
	OlderThan time.Duration // Delete entries older than this duration
	Before    time.Time     // Delete entries before this time

	// Count-based pruning
	KeepLast int // Keep only the last N entries

	// Size-based pruning
	MaxTotalSize int64 // Maximum total size in bytes

	// Selective pruning
	CollectionID string // Only prune from this collection
	Method       string // Only prune this method
	StatusMin    int    // Only prune entries with status >= this
	StatusMax    int    // Only prune entries with status <= this
}

// PruneResult contains the result of a prune operation.
type PruneResult struct {
	DeletedCount int64 `json:"deleted_count"`
	FreedBytes   int64 `json:"freed_bytes"`
}
