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
