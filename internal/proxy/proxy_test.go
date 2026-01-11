package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
		WithBufferSize(100),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server.Config().ListenAddr != ":0" {
		t.Errorf("Expected listen addr :0, got %s", server.Config().ListenAddr)
	}
	if server.Config().EnableHTTPS {
		t.Error("Expected HTTPS to be disabled")
	}
	if server.Config().BufferSize != 100 {
		t.Errorf("Expected buffer size 100, got %d", server.Config().BufferSize)
	}
}

func TestServerStartStop(t *testing.T) {
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	if !server.IsRunning() {
		t.Error("Server should be running")
	}

	addr := server.ListenAddr()
	if addr == "" || addr == ":0" {
		t.Error("Server should have a real listen address")
	}

	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}

func TestCaptureStore(t *testing.T) {
	store := NewCaptureStore(10)

	// Add some captures
	for i := 0; i < 5; i++ {
		store.Add(&CapturedRequest{
			Method: "GET",
			URL:    "http://example.com",
			Host:   "example.com",
		})
	}

	stats := store.Stats()
	if stats.TotalCount != 5 {
		t.Errorf("Expected 5 captures, got %d", stats.TotalCount)
	}

	// Test list with no filter
	captures := store.List(FilterOptions{})
	if len(captures) != 5 {
		t.Errorf("Expected 5 captures from list, got %d", len(captures))
	}
}

func TestCaptureStoreRingBuffer(t *testing.T) {
	store := NewCaptureStore(3) // Small buffer

	// Add 5 items to a buffer of size 3
	for i := 0; i < 5; i++ {
		store.Add(&CapturedRequest{
			Method: "GET",
			URL:    "http://example.com/" + string(rune('a'+i)),
		})
	}

	// Should only have 3 captures (ring buffer)
	captures := store.List(FilterOptions{})
	if len(captures) != 3 {
		t.Errorf("Expected 3 captures (ring buffer), got %d", len(captures))
	}
}

func TestCaptureStoreFiltering(t *testing.T) {
	store := NewCaptureStore(100)

	// Add mixed captures
	store.Add(&CapturedRequest{Method: "GET", URL: "http://api.example.com/users", Host: "api.example.com", StatusCode: 200})
	store.Add(&CapturedRequest{Method: "POST", URL: "http://api.example.com/users", Host: "api.example.com", StatusCode: 201})
	store.Add(&CapturedRequest{Method: "GET", URL: "http://other.com/page", Host: "other.com", StatusCode: 404})
	store.Add(&CapturedRequest{Method: "DELETE", URL: "http://api.example.com/users/1", Host: "api.example.com", StatusCode: 500})

	// Filter by method
	captures := store.List(FilterOptions{Method: "GET"})
	if len(captures) != 2 {
		t.Errorf("Expected 2 GET requests, got %d", len(captures))
	}

	// Filter by host
	captures = store.List(FilterOptions{Host: "api.example.com"})
	if len(captures) != 3 {
		t.Errorf("Expected 3 requests to api.example.com, got %d", len(captures))
	}

	// Filter by status code range
	captures = store.List(FilterOptions{StatusMin: 200, StatusMax: 299})
	if len(captures) != 2 {
		t.Errorf("Expected 2 2xx requests, got %d", len(captures))
	}

	// Filter by search term
	captures = store.List(FilterOptions{Search: "users"})
	if len(captures) != 3 {
		t.Errorf("Expected 3 requests matching 'users', got %d", len(captures))
	}
}

func TestCaptureStoreClear(t *testing.T) {
	store := NewCaptureStore(10)

	store.Add(&CapturedRequest{Method: "GET", URL: "http://example.com"})
	store.Add(&CapturedRequest{Method: "GET", URL: "http://example.org"})

	if store.Stats().TotalCount != 2 {
		t.Error("Should have 2 captures before clear")
	}

	store.Clear()

	if store.Stats().TotalCount != 0 {
		t.Error("Should have 0 captures after clear")
	}
}

func TestCaptureListener(t *testing.T) {
	store := NewCaptureStore(10)

	received := make(chan *CapturedRequest, 1)
	listener := CaptureListenerFunc(func(capture *CapturedRequest) {
		select {
		case received <- capture:
		default:
			// Channel full, ignore
		}
	})

	store.AddListener(listener)

	// Add a capture
	capture := &CapturedRequest{Method: "GET", URL: "http://example.com"}
	store.Add(capture)

	// Should receive notification
	select {
	case got := <-received:
		if got.URL != capture.URL {
			t.Errorf("Expected URL %s, got %s", capture.URL, got.URL)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for capture notification")
	}

	// Note: RemoveListener with function types is not supported due to Go's
	// function comparison limitations. This is a known limitation.
}

func TestProxyHTTPCapture(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backend.Close()

	// Create proxy server
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Create HTTP client that uses the proxy
	proxyURL, _ := url.Parse("http://" + server.ListenAddr())
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	// Make request through proxy
	resp, err := client.Get(backend.URL + "/test")
	if err != nil {
		t.Fatalf("Request through proxy failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ok") {
		t.Errorf("Unexpected response body: %s", body)
	}

	// Give time for capture to be recorded
	time.Sleep(100 * time.Millisecond)

	// Check captures
	captures := server.GetCaptures(FilterOptions{})
	if len(captures) == 0 {
		t.Error("Expected at least one capture")
		return
	}

	capture := captures[0]
	if capture.Method != "GET" {
		t.Errorf("Expected GET method, got %s", capture.Method)
	}
	if capture.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", capture.StatusCode)
	}
}

func TestConfig(t *testing.T) {
	config := NewConfig(
		WithListenAddr(":9090"),
		WithHTTPS(true),
		WithBufferSize(500),
		WithVerbose(true),
		WithExcludeHosts("tracking.example.com"),
		WithIncludeHosts("api.example.com"),
	)

	if config.ListenAddr != ":9090" {
		t.Errorf("Expected :9090, got %s", config.ListenAddr)
	}
	if !config.EnableHTTPS {
		t.Error("Expected HTTPS enabled")
	}
	if config.BufferSize != 500 {
		t.Errorf("Expected buffer 500, got %d", config.BufferSize)
	}
	if !config.Verbose {
		t.Error("Expected verbose enabled")
	}
	if len(config.ExcludeHosts) != 1 || config.ExcludeHosts[0] != "tracking.example.com" {
		t.Error("ExcludeHosts not set correctly")
	}
	if len(config.IncludeHosts) != 1 || config.IncludeHosts[0] != "api.example.com" {
		t.Error("IncludeHosts not set correctly")
	}
}

func TestHostMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		host    string
		match   bool
	}{
		{"example.com", "example.com", true},
		{"example.com", "other.com", false},
		{"*.example.com", "api.example.com", true},
		{"*.example.com", "example.com", false},
		{"*.example.com", "sub.api.example.com", true},
		{"api.*.com", "api.example.com", true},
		{"*", "anything.com", true},
	}

	for _, tc := range tests {
		got := matchHostPattern(tc.pattern, tc.host)
		if got != tc.match {
			t.Errorf("matchHostPattern(%q, %q) = %v, want %v", tc.pattern, tc.host, got, tc.match)
		}
	}
}

func TestProxyConcurrentRequests(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	// Create proxy server
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
		WithBufferSize(1000),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Create HTTP client that uses the proxy
	proxyURL, _ := url.Parse("http://" + server.ListenAddr())
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(proxyURL),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
		Timeout: 10 * time.Second,
	}

	// Make concurrent requests
	numRequests := 50
	numWorkers := 10
	errors := make(chan error, numRequests)
	var wg sync.WaitGroup

	requestsPerWorker := numRequests / numWorkers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < requestsPerWorker; i++ {
				resp, err := client.Get(backend.URL + "/test")
				if err != nil {
					errors <- err
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errCount int
	for err := range errors {
		t.Logf("Request error: %v", err)
		errCount++
	}

	if errCount > 0 {
		t.Errorf("%d/%d requests failed", errCount, numRequests)
	}

	// Verify captures were recorded
	time.Sleep(100 * time.Millisecond)
	stats := server.Stats()
	if stats.TotalCount < numRequests/2 {
		t.Errorf("Expected at least %d captures, got %d", numRequests/2, stats.TotalCount)
	}
	t.Logf("Successfully captured %d/%d requests", stats.TotalCount, numRequests)
}

func TestCaptureStoreStress(t *testing.T) {
	store := NewCaptureStore(100)

	// Concurrent writers
	var wg sync.WaitGroup
	numWriters := 10
	writesPerWriter := 100

	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < writesPerWriter; i++ {
				store.Add(&CapturedRequest{
					Method: "GET",
					URL:    "http://example.com/" + string(rune('a'+workerID)),
					Host:   "example.com",
				})
			}
		}(w)
	}

	// Concurrent readers
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = store.List(FilterOptions{})
				_ = store.Stats()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Ring buffer should have capped at 100
	stats := store.Stats()
	if stats.TotalCount > 100 {
		t.Errorf("Ring buffer overflow: got %d, max should be 100", stats.TotalCount)
	}
	t.Logf("Final capture count: %d (max 100)", stats.TotalCount)
}

func TestCapturedRequestMethods(t *testing.T) {
	t.Run("ContentType returns empty for nil headers", func(t *testing.T) {
		c := &CapturedRequest{}
		if c.ContentType() != "" {
			t.Error("Expected empty content type for nil headers")
		}
	})

	t.Run("ContentType returns value when present", func(t *testing.T) {
		c := &CapturedRequest{
			ResponseHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
		}
		if c.ContentType() != "application/json" {
			t.Errorf("Expected application/json, got %s", c.ContentType())
		}
	})

	t.Run("ContentType returns empty for empty header values", func(t *testing.T) {
		c := &CapturedRequest{
			ResponseHeaders: map[string][]string{
				"Content-Type": {},
			},
		}
		if c.ContentType() != "" {
			t.Error("Expected empty content type for empty header values")
		}
	})

	t.Run("IsSuccess returns true for 2xx", func(t *testing.T) {
		tests := []struct {
			status int
			expect bool
		}{
			{199, false},
			{200, true},
			{201, true},
			{299, true},
			{300, false},
		}
		for _, tc := range tests {
			c := &CapturedRequest{StatusCode: tc.status}
			if c.IsSuccess() != tc.expect {
				t.Errorf("IsSuccess() for %d: got %v, want %v", tc.status, c.IsSuccess(), tc.expect)
			}
		}
	})

	t.Run("IsRedirect returns true for 3xx", func(t *testing.T) {
		tests := []struct {
			status int
			expect bool
		}{
			{299, false},
			{300, true},
			{301, true},
			{399, true},
			{400, false},
		}
		for _, tc := range tests {
			c := &CapturedRequest{StatusCode: tc.status}
			if c.IsRedirect() != tc.expect {
				t.Errorf("IsRedirect() for %d: got %v, want %v", tc.status, c.IsRedirect(), tc.expect)
			}
		}
	})

	t.Run("IsClientError returns true for 4xx", func(t *testing.T) {
		tests := []struct {
			status int
			expect bool
		}{
			{399, false},
			{400, true},
			{404, true},
			{499, true},
			{500, false},
		}
		for _, tc := range tests {
			c := &CapturedRequest{StatusCode: tc.status}
			if c.IsClientError() != tc.expect {
				t.Errorf("IsClientError() for %d: got %v, want %v", tc.status, c.IsClientError(), tc.expect)
			}
		}
	})

	t.Run("IsServerError returns true for 5xx", func(t *testing.T) {
		tests := []struct {
			status int
			expect bool
		}{
			{499, false},
			{500, true},
			{503, true},
			{599, true},
			{600, false},
		}
		for _, tc := range tests {
			c := &CapturedRequest{StatusCode: tc.status}
			if c.IsServerError() != tc.expect {
				t.Errorf("IsServerError() for %d: got %v, want %v", tc.status, c.IsServerError(), tc.expect)
			}
		}
	})
}

func TestConfigOptions(t *testing.T) {
	t.Run("WithCACert sets certificate paths", func(t *testing.T) {
		config := NewConfig(WithCACert("/path/to/cert.crt", "/path/to/key.pem"))
		if config.CACertPath != "/path/to/cert.crt" {
			t.Errorf("Expected cert path /path/to/cert.crt, got %s", config.CACertPath)
		}
		if config.CAKeyPath != "/path/to/key.pem" {
			t.Errorf("Expected key path /path/to/key.pem, got %s", config.CAKeyPath)
		}
	})

	t.Run("WithAutoGenerateCA sets auto generate flag", func(t *testing.T) {
		config := NewConfig(WithAutoGenerateCA(true))
		if !config.AutoGenerateCA {
			t.Error("Expected AutoGenerateCA to be true")
		}
	})

	t.Run("WithMaxBodySize sets max body size", func(t *testing.T) {
		config := NewConfig(WithMaxBodySize(1024 * 1024))
		if config.MaxBodySize != 1024*1024 {
			t.Errorf("Expected max body size 1048576, got %d", config.MaxBodySize)
		}
	})
}

// matchHostPattern checks if a host matches a pattern with wildcards
func matchHostPattern(pattern, host string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == host
	}

	// Convert pattern to parts
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]
		return strings.HasPrefix(host, prefix) && strings.HasSuffix(host, suffix)
	}

	return false
}

func TestCaptureStoreMatchesFilter(t *testing.T) {
	store := NewCaptureStore(100)

	// Add captures with different methods and status codes
	store.Add(&CapturedRequest{Method: "GET", URL: "http://api.example.com/users", Host: "api.example.com", StatusCode: 200})
	store.Add(&CapturedRequest{Method: "POST", URL: "http://api.example.com/users", Host: "api.example.com", StatusCode: 201})
	store.Add(&CapturedRequest{Method: "GET", URL: "http://api.example.com/error", Host: "api.example.com", StatusCode: 500})
	store.Add(&CapturedRequest{Method: "GET", URL: "http://other.com/page", Host: "other.com", StatusCode: 404})

	tests := []struct {
		name   string
		filter FilterOptions
		count  int
	}{
		{"no filter", FilterOptions{}, 4},
		{"filter by method", FilterOptions{Method: "POST"}, 1},
		{"filter by status 2xx", FilterOptions{StatusMin: 200, StatusMax: 299}, 2},
		{"filter by status 4xx", FilterOptions{StatusMin: 400, StatusMax: 499}, 1},
		{"filter by status 5xx", FilterOptions{StatusMin: 500, StatusMax: 599}, 1},
		{"filter by host", FilterOptions{Host: "other.com"}, 1},
		{"filter by search in URL", FilterOptions{Search: "users"}, 2},
		{"filter by search in host", FilterOptions{Search: "other"}, 1},
		{"combined filters", FilterOptions{Method: "GET", StatusMin: 200, StatusMax: 299}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captures := store.List(tc.filter)
			if len(captures) != tc.count {
				t.Errorf("Expected %d captures, got %d", tc.count, len(captures))
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ListenAddr != ":8080" {
		t.Errorf("Expected :8080, got %s", config.ListenAddr)
	}
	if config.BufferSize != 1000 {
		t.Errorf("Expected buffer 1000, got %d", config.BufferSize)
	}
	if config.MaxBodySize != 10*1024*1024 {
		t.Errorf("Expected max body 10MB, got %d", config.MaxBodySize)
	}
}

func TestCaptureStoreGet(t *testing.T) {
	store := NewCaptureStore(10)

	capture := &CapturedRequest{ID: "test-123", Method: "GET", URL: "http://example.com"}
	store.Add(capture)

	got := store.Get(capture.ID)
	if got == nil {
		t.Error("Expected to get capture, got nil")
	}
	if got.ID != capture.ID {
		t.Errorf("Expected ID %s, got %s", capture.ID, got.ID)
	}

	// Test getting non-existent capture
	got = store.Get("non-existent")
	if got != nil {
		t.Error("Expected nil for non-existent capture")
	}
}

func TestCaptureStoreAll(t *testing.T) {
	store := NewCaptureStore(10)

	store.Add(&CapturedRequest{Method: "GET", URL: "http://example.com/1"})
	store.Add(&CapturedRequest{Method: "POST", URL: "http://example.com/2"})
	store.Add(&CapturedRequest{Method: "PUT", URL: "http://example.com/3"})

	all := store.All()
	if len(all) != 3 {
		t.Errorf("Expected 3 captures, got %d", len(all))
	}
}

func TestCaptureStoreCount(t *testing.T) {
	store := NewCaptureStore(10)

	if store.Count() != 0 {
		t.Errorf("Expected 0 captures, got %d", store.Count())
	}

	store.Add(&CapturedRequest{Method: "GET", URL: "http://example.com/1"})
	store.Add(&CapturedRequest{Method: "POST", URL: "http://example.com/2"})

	if store.Count() != 2 {
		t.Errorf("Expected 2 captures, got %d", store.Count())
	}
}

func TestCaptureStoreRemoveListener(t *testing.T) {
	store := NewCaptureStore(10)

	listener := &testCaptureListener{}
	store.AddListener(listener)

	// Remove listener
	store.RemoveListener(listener)

	// Add capture - listener should not be notified
	store.Add(&CapturedRequest{Method: "GET", URL: "http://example.com"})

	// Give time for any notifications
	time.Sleep(10 * time.Millisecond)

	if listener.captureCount != 0 {
		t.Errorf("Expected 0 captures, listener received %d", listener.captureCount)
	}
}

type testCaptureListener struct {
	captureCount int
	mu           sync.Mutex
}

func (l *testCaptureListener) OnCapture(capture *CapturedRequest) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.captureCount++
}

func TestServerAddListener(t *testing.T) {
	server, err := NewServer(WithListenAddr(":0"), WithHTTPS(false))
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	listener := &testCaptureListener{}
	server.AddListener(listener)
	server.RemoveListener(listener)

	// Verify listener was added and removed without panic
}

func TestServerStore(t *testing.T) {
	server, err := NewServer(WithListenAddr(":0"), WithHTTPS(false))
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	store := server.Store()
	if store == nil {
		t.Error("Expected store to be non-nil")
	}
}

func TestServerGetCapture(t *testing.T) {
	server, err := NewServer(WithListenAddr(":0"), WithHTTPS(false))
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Get non-existent capture
	capture := server.GetCapture("non-existent")
	if capture != nil {
		t.Error("Expected nil for non-existent capture")
	}
}

func TestServerConfig(t *testing.T) {
	server, err := NewServer(
		WithListenAddr(":9090"),
		WithHTTPS(true),
		WithBufferSize(500),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	config := server.Config()
	if config.ListenAddr != ":9090" {
		t.Errorf("Expected :9090, got %s", config.ListenAddr)
	}
	if !config.EnableHTTPS {
		t.Error("Expected HTTPS to be enabled")
	}
	if config.BufferSize != 500 {
		t.Errorf("Expected buffer 500, got %d", config.BufferSize)
	}
}

func TestServerClearCaptures(t *testing.T) {
	server, err := NewServer(WithListenAddr(":0"), WithHTTPS(false))
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Add some captures to store
	store := server.Store()
	store.Add(&CapturedRequest{
		ID:         "test-1",
		Method:     "GET",
		URL:        "http://example.com",
		StatusCode: 200,
	})
	store.Add(&CapturedRequest{
		ID:         "test-2",
		Method:     "POST",
		URL:        "http://example.com/api",
		StatusCode: 201,
	})

	if store.Count() != 2 {
		t.Errorf("Expected 2 captures, got %d", store.Count())
	}

	server.ClearCaptures()

	if store.Count() != 0 {
		t.Errorf("Expected 0 captures after clear, got %d", store.Count())
	}
}

func TestServerGetCaptures(t *testing.T) {
	server, err := NewServer(WithListenAddr(":0"), WithHTTPS(false))
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	store := server.Store()
	store.Add(&CapturedRequest{
		ID:         "test-1",
		Method:     "GET",
		URL:        "http://example.com",
		StatusCode: 200,
	})
	store.Add(&CapturedRequest{
		ID:         "test-2",
		Method:     "POST",
		URL:        "http://example.com/api",
		StatusCode: 201,
	})

	captures := server.GetCaptures(FilterOptions{})
	if len(captures) != 2 {
		t.Errorf("Expected 2 captures, got %d", len(captures))
	}
}

func TestServerStats(t *testing.T) {
	server, err := NewServer(WithListenAddr(":0"), WithHTTPS(false))
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	stats := server.Stats()
	if stats.TotalCount != 0 {
		t.Errorf("Expected 0 captured requests, got %d", stats.TotalCount)
	}

	store := server.Store()
	store.Add(&CapturedRequest{
		ID:         "test-1",
		Method:     "GET",
		URL:        "http://example.com",
		StatusCode: 200,
	})

	stats = server.Stats()
	if stats.TotalCount != 1 {
		t.Errorf("Expected 1 captured request, got %d", stats.TotalCount)
	}
}

func TestCapturedRequestIsSuccess(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{299, true},
		{100, false},
		{300, false},
		{400, false},
		{500, false},
	}

	for _, tc := range testCases {
		capture := &CapturedRequest{StatusCode: tc.statusCode}
		if capture.IsSuccess() != tc.expected {
			t.Errorf("StatusCode %d: expected IsSuccess=%v, got %v", tc.statusCode, tc.expected, capture.IsSuccess())
		}
	}
}

func TestCapturedRequestIsRedirect(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
	}{
		{300, true},
		{301, true},
		{302, true},
		{307, true},
		{399, true},
		{200, false},
		{400, false},
	}

	for _, tc := range testCases {
		capture := &CapturedRequest{StatusCode: tc.statusCode}
		if capture.IsRedirect() != tc.expected {
			t.Errorf("StatusCode %d: expected IsRedirect=%v, got %v", tc.statusCode, tc.expected, capture.IsRedirect())
		}
	}
}

func TestCapturedRequestIsClientError(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
	}{
		{400, true},
		{401, true},
		{404, true},
		{499, true},
		{200, false},
		{500, false},
	}

	for _, tc := range testCases {
		capture := &CapturedRequest{StatusCode: tc.statusCode}
		if capture.IsClientError() != tc.expected {
			t.Errorf("StatusCode %d: expected IsClientError=%v, got %v", tc.statusCode, tc.expected, capture.IsClientError())
		}
	}
}

func TestCapturedRequestIsServerError(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
	}{
		{500, true},
		{502, true},
		{503, true},
		{599, true},
		{200, false},
		{400, false},
	}

	for _, tc := range testCases {
		capture := &CapturedRequest{StatusCode: tc.statusCode}
		if capture.IsServerError() != tc.expected {
			t.Errorf("StatusCode %d: expected IsServerError=%v, got %v", tc.statusCode, tc.expected, capture.IsServerError())
		}
	}
}

func TestCapturedRequestContentType(t *testing.T) {
	testCases := []struct {
		headers  http.Header
		expected string
	}{
		{http.Header{"Content-Type": []string{"application/json"}}, "application/json"},
		{http.Header{"Content-Type": []string{"text/html; charset=utf-8"}}, "text/html; charset=utf-8"},
		{http.Header{}, ""},
		{nil, ""},
	}

	for _, tc := range testCases {
		capture := &CapturedRequest{ResponseHeaders: tc.headers}
		if capture.ContentType() != tc.expected {
			t.Errorf("Expected content type %q, got %q", tc.expected, capture.ContentType())
		}
	}
}

func TestCaptureStoreWithFilterOptions(t *testing.T) {
	store := NewCaptureStore(100)

	// Add diverse captures
	store.Add(&CapturedRequest{ID: "1", Method: "GET", URL: "http://api.example.com/users", StatusCode: 200, Host: "api.example.com"})
	store.Add(&CapturedRequest{ID: "2", Method: "POST", URL: "http://api.example.com/users", StatusCode: 201, Host: "api.example.com"})
	store.Add(&CapturedRequest{ID: "3", Method: "GET", URL: "http://cdn.example.com/image.png", StatusCode: 200, Host: "cdn.example.com"})
	store.Add(&CapturedRequest{ID: "4", Method: "GET", URL: "http://api.example.com/error", StatusCode: 500, Host: "api.example.com"})

	// Test get all
	all := store.All()
	if len(all) != 4 {
		t.Errorf("Expected 4 captures, got %d", len(all))
	}

	// Test stats
	stats := store.Stats()
	if stats.TotalCount != 4 {
		t.Errorf("Expected 4 total count, got %d", stats.TotalCount)
	}
	if stats.MethodCounts["GET"] != 3 {
		t.Errorf("Expected 3 GET requests, got %d", stats.MethodCounts["GET"])
	}
	if stats.MethodCounts["POST"] != 1 {
		t.Errorf("Expected 1 POST request, got %d", stats.MethodCounts["POST"])
	}
}

func TestMatchHost(t *testing.T) {
	testCases := []struct {
		hostname string
		pattern  string
		expected bool
	}{
		{"example.com", "example.com", true},
		{"example.com", "other.com", false},
		{"api.example.com", "*.example.com", true},
		{"cdn.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false}, // No subdomain, doesn't match *.example.com
		{"other.com", "*.example.com", false},
		{"sub.api.example.com", "*.example.com", true},
		{"example.com", "*", true},
		{"anything.com", "*", true},
	}

	for _, tc := range testCases {
		result := matchHost(tc.hostname, tc.pattern)
		if result != tc.expected {
			t.Errorf("matchHost(%q, %q): expected %v, got %v", tc.hostname, tc.pattern, tc.expected, result)
		}
	}
}

func TestWriteErrorResponse(t *testing.T) {
	var buf strings.Builder
	writeErrorResponse(&buf, http.StatusBadGateway, "Connection failed")

	response := buf.String()
	if !strings.Contains(response, "502") {
		t.Error("Response should contain status code 502")
	}
	if !strings.Contains(response, "Bad Gateway") {
		t.Error("Response should contain status text")
	}
	if !strings.Contains(response, "Connection failed") {
		t.Error("Response should contain error message")
	}
}

func TestProxyHandlerShouldCapture(t *testing.T) {
	// Test with no filters - should capture all
	handler := NewProxyHandler(NewCaptureStore(10), nil, Config{})
	if !handler.shouldCapture("example.com") {
		t.Error("Should capture when no filters")
	}

	// Test with exclude list
	handler = NewProxyHandler(NewCaptureStore(10), nil, Config{
		ExcludeHosts: []string{"excluded.com", "*.internal.com"},
	})
	if handler.shouldCapture("excluded.com") {
		t.Error("Should not capture excluded host")
	}
	if handler.shouldCapture("api.internal.com") {
		t.Error("Should not capture wildcard excluded host")
	}
	if !handler.shouldCapture("allowed.com") {
		t.Error("Should capture non-excluded host")
	}

	// Test with include list
	handler = NewProxyHandler(NewCaptureStore(10), nil, Config{
		IncludeHosts: []string{"api.example.com", "*.myapp.com"},
	})
	if !handler.shouldCapture("api.example.com") {
		t.Error("Should capture included host")
	}
	if !handler.shouldCapture("sub.myapp.com") {
		t.Error("Should capture wildcard included host")
	}
	if handler.shouldCapture("other.com") {
		t.Error("Should not capture non-included host")
	}

	// Test with host:port format
	if !handler.shouldCapture("api.example.com:8080") {
		t.Error("Should capture included host with port")
	}
}

func TestProxyHandlerShouldExcludeContentType(t *testing.T) {
	handler := NewProxyHandler(NewCaptureStore(10), nil, Config{
		ExcludeContentTypes: []string{"image/", "video/", "audio/"},
	})

	testCases := []struct {
		contentType string
		exclude     bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"video/mp4", true},
		{"audio/mpeg", true},
		{"application/json", false},
		{"text/html", false},
		{"IMAGE/PNG", true}, // case insensitive
	}

	for _, tc := range testCases {
		result := handler.shouldExcludeContentType(tc.contentType)
		if result != tc.exclude {
			t.Errorf("shouldExcludeContentType(%q): expected %v, got %v", tc.contentType, tc.exclude, result)
		}
	}
}

func TestProxyHandlerServeHTTPMethod(t *testing.T) {
	store := NewCaptureStore(100)
	handler := NewProxyHandler(store, nil, Config{})

	// Test that CONNECT method is handled differently
	// We can't fully test HTTPS handling without TLS setup, but we can verify the method routing

	// Create a simple test server for HTTP proxying
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	// Test regular HTTP request
	req := httptest.NewRequest("GET", backend.URL+"/test", nil)
	req.Host = backend.Listener.Addr().String()
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should get response from backend
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestProxyHandlerHTTPProxying(t *testing.T) {
	// Create a mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backend.Close()

	store := NewCaptureStore(100)
	handler := NewProxyHandler(store, nil, Config{})

	// Parse backend URL
	backendURL, _ := url.Parse(backend.URL)

	// Create proxy request with absolute URL (as proxy clients do)
	req := httptest.NewRequest("GET", backend.URL+"/api/test", nil)
	req.URL, _ = url.Parse(backend.URL + "/api/test")
	req.Host = backendURL.Host
	req.Header.Set("Accept", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify response - should get response from backend
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Check that request was captured
	stats := store.Stats()
	if stats.TotalCount != 1 {
		t.Errorf("Expected 1 captured request, got %d", stats.TotalCount)
	}
}

func TestProxyHandlerWithPOSTBody(t *testing.T) {
	// Create a mock backend that echoes the body
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		w.Write(body)
	}))
	defer backend.Close()

	store := NewCaptureStore(100)
	handler := NewProxyHandler(store, nil, Config{})

	backendURL, _ := url.Parse(backend.URL)

	// Create POST request with body and absolute URL
	reqBody := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest("POST", backend.URL+"/api/create", reqBody)
	req.URL, _ = url.Parse(backend.URL + "/api/create")
	req.Host = backendURL.Host
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Just verify the request was successful and captured
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rr.Code)
	}

	stats := store.Stats()
	if stats.TotalCount != 1 {
		t.Errorf("Expected 1 captured request, got %d", stats.TotalCount)
	}
}

func TestProxyHandlerExcludesContentType(t *testing.T) {
	// Create a mock backend that returns an image
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-image-data"))
	}))
	defer backend.Close()

	store := NewCaptureStore(100)
	handler := NewProxyHandler(store, nil, Config{
		ExcludeContentTypes: []string{"image/"},
	})

	backendURL, _ := url.Parse(backend.URL)
	req := httptest.NewRequest("GET", backend.URL+"/image.png", nil)
	req.Host = backendURL.Host

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// The request should still be captured, but body should be discarded
	stats := store.Stats()
	if stats.TotalCount != 1 {
		t.Errorf("Expected 1 captured request, got %d", stats.TotalCount)
	}
}

func TestProxyHandlerHostExclusion(t *testing.T) {
	// Create a mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	store := NewCaptureStore(100)
	handler := NewProxyHandler(store, nil, Config{
		ExcludeHosts: []string{"excluded.com"},
	})

	backendURL, _ := url.Parse(backend.URL)

	// Request with excluded host header - should not be captured
	req := httptest.NewRequest("GET", backend.URL+"/test", nil)
	req.Host = "excluded.com"
	// But we need to actually route to the backend
	req.URL.Host = backendURL.Host

	rr := httptest.NewRecorder()
	handler.handleHTTP(rr, req)

	// Request should still work
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// But should NOT be captured because host is excluded
	stats := store.Stats()
	if stats.TotalCount != 0 {
		t.Errorf("Expected 0 captured requests (excluded host), got %d", stats.TotalCount)
	}
}

func TestRemoveHopByHopHeaders(t *testing.T) {
	header := http.Header{
		"Connection":          []string{"keep-alive"},
		"Keep-Alive":          []string{"timeout=5"},
		"Proxy-Authorization": []string{"Basic abc123"},
		"Transfer-Encoding":   []string{"chunked"},
		"Content-Type":        []string{"application/json"},
		"X-Custom-Header":     []string{"should-remain"},
	}

	removeHopByHopHeaders(header)

	// These should be removed
	hopByHopHeaders := []string{"Connection", "Keep-Alive", "Proxy-Authorization", "Transfer-Encoding"}
	for _, h := range hopByHopHeaders {
		if header.Get(h) != "" {
			t.Errorf("Header %s should have been removed", h)
		}
	}

	// These should remain
	if header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should remain")
	}
	if header.Get("X-Custom-Header") != "should-remain" {
		t.Error("X-Custom-Header should remain")
	}
}

func TestCapturedRequestError(t *testing.T) {
	capture := &CapturedRequest{
		ID:         "test-1",
		Method:     "GET",
		URL:        "http://example.com/api",
		StatusCode: 0,
		Error:      "connection refused",
	}

	if capture.Error == "" {
		t.Error("Expected error to be set")
	}
	if capture.StatusCode != 0 {
		t.Error("Failed requests should have status 0")
	}
}

func TestServerDoubleStart(t *testing.T) {
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Try to start again - should fail
	err = server.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running server")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("Expected 'already running' error, got: %v", err)
	}
}

func TestServerStopWhenNotRunning(t *testing.T) {
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Stop without starting - should be fine
	err = server.Stop()
	if err != nil {
		t.Errorf("Stopping non-running server should not error: %v", err)
	}
}

func TestServerExportCACertNoHTTPS(t *testing.T) {
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Try to export CA cert when HTTPS is not enabled
	err = server.ExportCACert("/tmp/test-ca.pem")
	if err == nil {
		t.Error("Expected error when exporting CA cert with HTTPS disabled")
	}

	// CACertPEM should return nil
	if server.CACertPEM() != nil {
		t.Error("CACertPEM should return nil when HTTPS disabled")
	}
}

func TestServerContextCancellation(t *testing.T) {
	server, err := NewServer(
		WithListenAddr(":0"),
		WithHTTPS(false),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	if !server.IsRunning() {
		t.Error("Server should be running")
	}

	// Cancel context should trigger shutdown
	cancel()

	// Give server time to stop
	time.Sleep(100 * time.Millisecond)

	if server.IsRunning() {
		t.Error("Server should have stopped after context cancellation")
	}
}

func TestCaptureStoreAdd(t *testing.T) {
	t.Run("assigns ID when empty", func(t *testing.T) {
		store := NewCaptureStore(10)
		capture := &CapturedRequest{
			Method: "GET",
			URL:    "https://example.com",
		}
		store.Add(capture)

		if capture.ID == "" {
			t.Error("Expected ID to be assigned")
		}
	})

	t.Run("assigns timestamp when zero", func(t *testing.T) {
		store := NewCaptureStore(10)
		capture := &CapturedRequest{
			ID:     "test-id",
			Method: "GET",
			URL:    "https://example.com",
		}
		store.Add(capture)

		if capture.Timestamp.IsZero() {
			t.Error("Expected timestamp to be assigned")
		}
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		store := NewCaptureStore(10)
		capture := &CapturedRequest{
			ID:     "custom-id",
			Method: "GET",
			URL:    "https://example.com",
		}
		store.Add(capture)

		if capture.ID != "custom-id" {
			t.Errorf("Expected ID 'custom-id', got '%s'", capture.ID)
		}
	})

	t.Run("wraps around in ring buffer", func(t *testing.T) {
		store := NewCaptureStore(3)

		for i := 0; i < 5; i++ {
			store.Add(&CapturedRequest{
				ID:     string(rune('a' + i)),
				Method: "GET",
				URL:    "https://example.com",
			})
		}

		captures := store.List(FilterOptions{})
		if len(captures) != 3 {
			t.Errorf("Expected 3 captures, got %d", len(captures))
		}
	})

	t.Run("notifies listeners", func(t *testing.T) {
		store := NewCaptureStore(10)
		var mu sync.Mutex
		notified := false

		store.AddListener(CaptureListenerFunc(func(cr *CapturedRequest) {
			mu.Lock()
			notified = true
			mu.Unlock()
		}))

		store.Add(&CapturedRequest{
			Method: "GET",
			URL:    "https://example.com",
		})

		// Give listener time to be called
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		wasNotified := notified
		mu.Unlock()
		if !wasNotified {
			t.Error("Expected listener to be notified")
		}
	})
}

func TestCaptureStoreGetByID(t *testing.T) {
	store := NewCaptureStore(10)
	capture := &CapturedRequest{
		ID:     "test-id-123",
		Method: "GET",
		URL:    "https://example.com",
	}
	store.Add(capture)

	t.Run("finds capture by ID", func(t *testing.T) {
		found := store.Get("test-id-123")
		if found == nil {
			t.Error("Expected to find capture")
		}
		if found.ID != "test-id-123" {
			t.Errorf("Expected ID 'test-id-123', got '%s'", found.ID)
		}
	})

	t.Run("returns nil for non-existent ID", func(t *testing.T) {
		found := store.Get("non-existent")
		if found != nil {
			t.Error("Expected not to find capture")
		}
	})
}

func TestCaptureStoreClearAll(t *testing.T) {
	store := NewCaptureStore(10)
	store.Add(&CapturedRequest{Method: "GET", URL: "https://example.com"})
	store.Add(&CapturedRequest{Method: "POST", URL: "https://example.com"})

	captures := store.List(FilterOptions{})
	if len(captures) != 2 {
		t.Errorf("Expected 2 captures, got %d", len(captures))
	}

	store.Clear()
	captures = store.List(FilterOptions{})
	if len(captures) != 0 {
		t.Errorf("Expected 0 captures after clear, got %d", len(captures))
	}
}

func TestNewCaptureStoreMinSize(t *testing.T) {
	store := NewCaptureStore(0)
	// Should use default size of 1000, not panic
	if store == nil {
		t.Error("Expected store to be created")
	}
}

func TestMatchesFilterComprehensive(t *testing.T) {
	store := NewCaptureStore(100)

	// Create test captures with various properties
	capture1 := &CapturedRequest{
		ID:         "1",
		Method:     "GET",
		URL:        "https://api.example.com/users",
		Host:       "api.example.com",
		Path:       "/users",
		StatusCode: 200,
		IsHTTPS:    true,
		Timestamp:  time.Now(),
		RequestHeaders: http.Header{
			"Authorization": []string{"Bearer token123"},
			"Content-Type":  []string{"application/json"},
		},
		ResponseHeaders: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		RequestBody:  []byte(`{"name":"test"}`),
		ResponseBody: []byte(`{"id":1,"name":"test"}`),
		ResponseSize: 500,
	}
	store.Add(capture1)

	capture2 := &CapturedRequest{
		ID:           "2",
		Method:       "POST",
		URL:          "http://other.com/api/data",
		Host:         "other.com",
		Path:         "/api/data",
		StatusCode:   404,
		IsHTTPS:      false,
		Timestamp:    time.Now().Add(-time.Hour),
		ResponseSize: 100,
	}
	store.Add(capture2)

	t.Run("filter by path prefix", func(t *testing.T) {
		captures := store.List(FilterOptions{Path: "/users"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with path /users, got %d", len(captures))
		}
	})

	t.Run("filter by content type", func(t *testing.T) {
		captures := store.List(FilterOptions{ContentType: "application/json"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with content type application/json, got %d", len(captures))
		}
	})

	t.Run("search in request headers key", func(t *testing.T) {
		captures := store.List(FilterOptions{Search: "authorization"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with Authorization header, got %d", len(captures))
		}
	})

	t.Run("search in request headers value", func(t *testing.T) {
		captures := store.List(FilterOptions{Search: "token123"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with token123 in header, got %d", len(captures))
		}
	})

	t.Run("search in response headers key", func(t *testing.T) {
		captures := store.List(FilterOptions{Search: "content-type"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with Content-Type response header, got %d", len(captures))
		}
	})

	t.Run("search in response headers value", func(t *testing.T) {
		captures := store.List(FilterOptions{Search: "charset=utf-8"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with charset in response header, got %d", len(captures))
		}
	})

	t.Run("search in request body", func(t *testing.T) {
		captures := store.List(FilterOptions{Search: "name"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with 'name' in request body, got %d", len(captures))
		}
	})

	t.Run("search in response body", func(t *testing.T) {
		captures := store.List(FilterOptions{Search: "id\":1"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with id:1 in response body, got %d", len(captures))
		}
	})

	t.Run("filter by min size", func(t *testing.T) {
		captures := store.List(FilterOptions{MinSize: 200})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with size >= 200, got %d", len(captures))
		}
	})

	t.Run("filter by max size", func(t *testing.T) {
		captures := store.List(FilterOptions{MaxSize: 150})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with size <= 150, got %d", len(captures))
		}
	})

	t.Run("filter by time range - after", func(t *testing.T) {
		captures := store.List(FilterOptions{After: time.Now().Add(-30 * time.Minute)})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture after 30 minutes ago, got %d", len(captures))
		}
	})

	t.Run("filter by time range - before", func(t *testing.T) {
		captures := store.List(FilterOptions{Before: time.Now().Add(-30 * time.Minute)})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture before 30 minutes ago, got %d", len(captures))
		}
	})

	t.Run("filter HTTPSOnly", func(t *testing.T) {
		captures := store.List(FilterOptions{HTTPSOnly: true})
		if len(captures) != 1 {
			t.Errorf("Expected 1 HTTPS capture, got %d", len(captures))
		}
	})

	t.Run("filter HTTPOnly", func(t *testing.T) {
		captures := store.List(FilterOptions{HTTPOnly: true})
		if len(captures) != 1 {
			t.Errorf("Expected 1 HTTP capture, got %d", len(captures))
		}
	})

	t.Run("filter with wildcard host suffix", func(t *testing.T) {
		captures := store.List(FilterOptions{Host: "*.example.com"})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture matching *.example.com, got %d", len(captures))
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		captures := store.List(FilterOptions{
			Method:    "GET",
			HTTPSOnly: true,
			MinSize:   100,
		})
		if len(captures) != 1 {
			t.Errorf("Expected 1 capture with combined filters, got %d", len(captures))
		}
	})
}

func TestCaptureStoreStats(t *testing.T) {
	store := NewCaptureStore(100)

	// Add captures with various properties
	store.Add(&CapturedRequest{
		ID:           "1",
		Method:       "GET",
		Host:         "api.example.com",
		StatusCode:   200,
		RequestSize:  100,
		ResponseSize: 500,
		Duration:     100 * time.Millisecond,
		Timestamp:    time.Now().Add(-time.Hour),
	})
	store.Add(&CapturedRequest{
		ID:           "2",
		Method:       "POST",
		Host:         "api.example.com",
		StatusCode:   201,
		RequestSize:  200,
		ResponseSize: 100,
		Duration:     50 * time.Millisecond,
		Timestamp:    time.Now(),
	})
	store.Add(&CapturedRequest{
		ID:           "3",
		Method:       "GET",
		Host:         "other.com",
		StatusCode:   404,
		RequestSize:  50,
		ResponseSize: 50,
		Duration:     150 * time.Millisecond,
		Timestamp:    time.Now().Add(-30 * time.Minute),
	})

	stats := store.Stats()

	t.Run("counts captures", func(t *testing.T) {
		if stats.TotalCount != 3 {
			t.Errorf("Expected 3 captures, got %d", stats.TotalCount)
		}
	})

	t.Run("sums request size", func(t *testing.T) {
		if stats.TotalRequestSize != 350 {
			t.Errorf("Expected request size 350, got %d", stats.TotalRequestSize)
		}
	})

	t.Run("sums response size", func(t *testing.T) {
		if stats.TotalResponseSize != 650 {
			t.Errorf("Expected response size 650, got %d", stats.TotalResponseSize)
		}
	})

	t.Run("counts methods", func(t *testing.T) {
		if stats.MethodCounts["GET"] != 2 {
			t.Errorf("Expected 2 GET, got %d", stats.MethodCounts["GET"])
		}
		if stats.MethodCounts["POST"] != 1 {
			t.Errorf("Expected 1 POST, got %d", stats.MethodCounts["POST"])
		}
	})

	t.Run("counts statuses", func(t *testing.T) {
		if stats.StatusCounts[200] != 1 {
			t.Errorf("Expected 1 status 200, got %d", stats.StatusCounts[200])
		}
		if stats.StatusCounts[201] != 1 {
			t.Errorf("Expected 1 status 201, got %d", stats.StatusCounts[201])
		}
		if stats.StatusCounts[404] != 1 {
			t.Errorf("Expected 1 status 404, got %d", stats.StatusCounts[404])
		}
	})

	t.Run("counts hosts", func(t *testing.T) {
		if stats.HostCounts["api.example.com"] != 2 {
			t.Errorf("Expected 2 api.example.com, got %d", stats.HostCounts["api.example.com"])
		}
		if stats.HostCounts["other.com"] != 1 {
			t.Errorf("Expected 1 other.com, got %d", stats.HostCounts["other.com"])
		}
	})

	t.Run("calculates average duration", func(t *testing.T) {
		if stats.AvgDuration != 100*time.Millisecond {
			t.Errorf("Expected avg duration 100ms, got %v", stats.AvgDuration)
		}
	})

	t.Run("tracks oldest capture", func(t *testing.T) {
		if stats.OldestCapture.IsZero() {
			t.Error("OldestCapture should not be zero")
		}
	})

	t.Run("tracks newest capture", func(t *testing.T) {
		if stats.NewestCapture.IsZero() {
			t.Error("NewestCapture should not be zero")
		}
	})
}

func TestCaptureStoreStatsEmpty(t *testing.T) {
	store := NewCaptureStore(100)
	stats := store.Stats()

	if stats.TotalCount != 0 {
		t.Errorf("Expected 0 captures, got %d", stats.TotalCount)
	}
	if stats.AvgDuration != 0 {
		t.Errorf("Expected 0 avg duration, got %v", stats.AvgDuration)
	}
}
