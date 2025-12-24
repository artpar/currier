// Package testserver provides a configurable HTTP test server for E2E tests.
package testserver

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// Server wraps httptest.Server with additional utilities.
type Server struct {
	*httptest.Server
	mu       sync.Mutex
	requests []*RecordedRequest
}

// RecordedRequest stores request details for verification.
type RecordedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
	Time    time.Time
}

// New creates a test server with the given routes.
func New(routes map[string]http.HandlerFunc) *Server {
	s := &Server{
		requests: make([]*RecordedRequest, 0),
	}

	mux := http.NewServeMux()
	for pattern, handler := range routes {
		mux.HandleFunc(pattern, s.recordingWrapper(handler))
	}

	s.Server = httptest.NewServer(mux)
	return s
}

// recordingWrapper wraps handlers to record requests.
func (s *Server) recordingWrapper(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, &RecordedRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: r.Header.Clone(),
			Time:    time.Now(),
		})
		s.mu.Unlock()
		h(w, r)
	}
}

// LastRequest returns the last recorded request.
func (s *Server) LastRequest() *RecordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		return nil
	}
	return s.requests[len(s.requests)-1]
}

// Requests returns all recorded requests.
func (s *Server) Requests() []*RecordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*RecordedRequest, len(s.requests))
	copy(result, s.requests)
	return result
}

// RequestCount returns the number of recorded requests.
func (s *Server) RequestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

// ClearRequests clears recorded requests.
func (s *Server) ClearRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = s.requests[:0]
}
