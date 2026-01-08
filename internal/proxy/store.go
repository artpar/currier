package proxy

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CaptureStore stores captured requests in a ring buffer.
type CaptureStore struct {
	captures  []*CapturedRequest
	maxSize   int
	head      int // Next write position
	count     int // Current number of items
	listeners []CaptureListener
	mu        sync.RWMutex
}

// NewCaptureStore creates a new capture store with the given buffer size.
func NewCaptureStore(maxSize int) *CaptureStore {
	if maxSize < 1 {
		maxSize = 1000
	}
	return &CaptureStore{
		captures: make([]*CapturedRequest, maxSize),
		maxSize:  maxSize,
	}
}

// Add adds a new capture to the store.
func (s *CaptureStore) Add(capture *CapturedRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Assign ID if not set
	if capture.ID == "" {
		capture.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if capture.Timestamp.IsZero() {
		capture.Timestamp = time.Now()
	}

	// Add to ring buffer
	s.captures[s.head] = capture
	s.head = (s.head + 1) % s.maxSize
	if s.count < s.maxSize {
		s.count++
	}

	// Notify listeners (in goroutine to not block)
	for _, listener := range s.listeners {
		go listener.OnCapture(capture)
	}
}

// Get returns a capture by ID.
func (s *CaptureStore) Get(id string) *CapturedRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := 0; i < s.count; i++ {
		idx := (s.head - 1 - i + s.maxSize) % s.maxSize
		if s.captures[idx] != nil && s.captures[idx].ID == id {
			return s.captures[idx]
		}
	}
	return nil
}

// List returns captures matching the filter options.
// Results are returned in reverse chronological order (newest first).
func (s *CaptureStore) List(opts FilterOptions) []*CapturedRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*CapturedRequest
	skipped := 0

	for i := 0; i < s.count; i++ {
		// Read in reverse order (newest first)
		idx := (s.head - 1 - i + s.maxSize) % s.maxSize
		capture := s.captures[idx]
		if capture == nil {
			continue
		}

		// Apply filters
		if !matchesFilter(capture, opts) {
			continue
		}

		// Apply offset
		if skipped < opts.Offset {
			skipped++
			continue
		}

		result = append(result, capture)

		// Apply limit
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}

	return result
}

// All returns all captures in reverse chronological order.
func (s *CaptureStore) All() []*CapturedRequest {
	return s.List(FilterOptions{})
}

// Count returns the current number of captures.
func (s *CaptureStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// Clear removes all captures from the store.
func (s *CaptureStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.captures = make([]*CapturedRequest, s.maxSize)
	s.head = 0
	s.count = 0
}

// AddListener adds a listener for real-time capture events.
func (s *CaptureStore) AddListener(listener CaptureListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

// RemoveListener removes a listener.
func (s *CaptureStore) RemoveListener(listener CaptureListener) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, l := range s.listeners {
		if l == listener {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			return
		}
	}
}

// matchesFilter checks if a capture matches the filter options.
func matchesFilter(capture *CapturedRequest, opts FilterOptions) bool {
	// Method filter
	if opts.Method != "" && !strings.EqualFold(capture.Method, opts.Method) {
		return false
	}

	// Host filter (case-insensitive, supports wildcard prefix *)
	if opts.Host != "" {
		host := strings.ToLower(capture.Host)
		pattern := strings.ToLower(opts.Host)
		if strings.HasPrefix(pattern, "*") {
			suffix := pattern[1:]
			if !strings.HasSuffix(host, suffix) {
				return false
			}
		} else if host != pattern {
			return false
		}
	}

	// Path filter (case-insensitive prefix match)
	if opts.Path != "" {
		path := strings.ToLower(capture.Path)
		pattern := strings.ToLower(opts.Path)
		if !strings.HasPrefix(path, pattern) {
			return false
		}
	}

	// Status code range
	if opts.StatusMin > 0 && capture.StatusCode < opts.StatusMin {
		return false
	}
	if opts.StatusMax > 0 && capture.StatusCode > opts.StatusMax {
		return false
	}

	// Content-type filter (prefix match)
	if opts.ContentType != "" {
		ct := strings.ToLower(capture.ContentType())
		pattern := strings.ToLower(opts.ContentType)
		if !strings.HasPrefix(ct, pattern) {
			return false
		}
	}

	// Full-text search
	if opts.Search != "" {
		search := strings.ToLower(opts.Search)
		found := false

		// Search URL
		if strings.Contains(strings.ToLower(capture.URL), search) {
			found = true
		}

		// Search host
		if !found && strings.Contains(strings.ToLower(capture.Host), search) {
			found = true
		}

		// Search path
		if !found && strings.Contains(strings.ToLower(capture.Path), search) {
			found = true
		}

		// Search request headers
		if !found {
			for key, values := range capture.RequestHeaders {
				if strings.Contains(strings.ToLower(key), search) {
					found = true
					break
				}
				for _, v := range values {
					if strings.Contains(strings.ToLower(v), search) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}

		// Search response headers
		if !found {
			for key, values := range capture.ResponseHeaders {
				if strings.Contains(strings.ToLower(key), search) {
					found = true
					break
				}
				for _, v := range values {
					if strings.Contains(strings.ToLower(v), search) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}

		// Search request body (if text)
		if !found && len(capture.RequestBody) > 0 && len(capture.RequestBody) < 1024*1024 {
			if strings.Contains(strings.ToLower(string(capture.RequestBody)), search) {
				found = true
			}
		}

		// Search response body (if text)
		if !found && len(capture.ResponseBody) > 0 && len(capture.ResponseBody) < 1024*1024 {
			if strings.Contains(strings.ToLower(string(capture.ResponseBody)), search) {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	// Size filters
	if opts.MinSize > 0 && capture.ResponseSize < opts.MinSize {
		return false
	}
	if opts.MaxSize > 0 && capture.ResponseSize > opts.MaxSize {
		return false
	}

	// Time range
	if !opts.After.IsZero() && capture.Timestamp.Before(opts.After) {
		return false
	}
	if !opts.Before.IsZero() && capture.Timestamp.After(opts.Before) {
		return false
	}

	// Protocol filters
	if opts.HTTPSOnly && !capture.IsHTTPS {
		return false
	}
	if opts.HTTPOnly && capture.IsHTTPS {
		return false
	}

	return true
}

// Stats returns statistics about the captured traffic.
type CaptureStats struct {
	TotalCount       int
	TotalRequestSize int64
	TotalResponseSize int64
	MethodCounts     map[string]int
	StatusCounts     map[int]int
	HostCounts       map[string]int
	AvgDuration      time.Duration
	OldestCapture    time.Time
	NewestCapture    time.Time
}

// Stats returns statistics about the captured traffic.
func (s *CaptureStore) Stats() CaptureStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := CaptureStats{
		MethodCounts: make(map[string]int),
		StatusCounts: make(map[int]int),
		HostCounts:   make(map[string]int),
	}

	var totalDuration time.Duration

	for i := 0; i < s.count; i++ {
		idx := (s.head - 1 - i + s.maxSize) % s.maxSize
		capture := s.captures[idx]
		if capture == nil {
			continue
		}

		stats.TotalCount++
		stats.TotalRequestSize += capture.RequestSize
		stats.TotalResponseSize += capture.ResponseSize
		stats.MethodCounts[capture.Method]++
		stats.StatusCounts[capture.StatusCode]++
		stats.HostCounts[capture.Host]++
		totalDuration += capture.Duration

		if stats.OldestCapture.IsZero() || capture.Timestamp.Before(stats.OldestCapture) {
			stats.OldestCapture = capture.Timestamp
		}
		if stats.NewestCapture.IsZero() || capture.Timestamp.After(stats.NewestCapture) {
			stats.NewestCapture = capture.Timestamp
		}
	}

	if stats.TotalCount > 0 {
		stats.AvgDuration = totalDuration / time.Duration(stats.TotalCount)
	}

	return stats
}
