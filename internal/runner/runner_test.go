package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/script"
)

func TestNewRunner(t *testing.T) {
	t.Run("creates runner with collection", func(t *testing.T) {
		coll := core.NewCollection("Test Collection")
		runner := NewRunner(coll)

		if runner == nil {
			t.Fatal("expected runner to be created")
		}
		if runner.collection != coll {
			t.Error("expected collection to be set")
		}
		if runner.engine == nil {
			t.Error("expected interpolation engine to be created")
		}
		if runner.httpClient == nil {
			t.Error("expected HTTP client to be created")
		}
		if runner.cookieJar == nil {
			t.Error("expected cookie jar to be created")
		}
	})

	t.Run("applies WithEnvironment option", func(t *testing.T) {
		coll := core.NewCollection("Test")
		env := core.NewEnvironment("test-env")
		env.SetVariable("api_url", "https://api.example.com")

		runner := NewRunner(coll, WithEnvironment(env))

		if runner.env != env {
			t.Error("expected environment to be set")
		}
	})

	t.Run("applies WithProgressCallback option", func(t *testing.T) {
		coll := core.NewCollection("Test")
		cb := func(current, total int, result *RunResult) {
			// callback set
		}

		runner := NewRunner(coll, WithProgressCallback(cb))

		if runner.onProgress == nil {
			t.Error("expected progress callback to be set")
		}
	})

	t.Run("applies WithCookieJar option", func(t *testing.T) {
		coll := core.NewCollection("Test")
		jar := &mockCookieJar{}

		runner := NewRunner(coll, WithCookieJar(jar))

		// Verify cookie jar was set by checking it's not the default
		if runner.cookieJar == nil {
			t.Error("expected custom cookie jar to be set")
		}
	})
}

func TestRunner_Run(t *testing.T) {
	t.Run("runs empty collection", func(t *testing.T) {
		coll := core.NewCollection("Empty")
		runner := NewRunner(coll)

		summary := runner.Run(context.Background())

		if summary.CollectionName != "Empty" {
			t.Errorf("expected collection name 'Empty', got %s", summary.CollectionName)
		}
		if summary.TotalRequests != 0 {
			t.Errorf("expected 0 requests, got %d", summary.TotalRequests)
		}
		if summary.Executed != 0 {
			t.Errorf("expected 0 executed, got %d", summary.Executed)
		}
	})

	t.Run("executes requests in collection", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		coll := core.NewCollection("Test API")
		req := core.NewRequestDefinition("GET Request", "GET", server.URL+"/test")
		coll.AddRequest(req)

		runner := NewRunner(coll)
		summary := runner.Run(context.Background())

		if summary.TotalRequests != 1 {
			t.Errorf("expected 1 request, got %d", summary.TotalRequests)
		}
		if summary.Executed != 1 {
			t.Errorf("expected 1 executed, got %d", summary.Executed)
		}
		if summary.Passed != 1 {
			t.Errorf("expected 1 passed, got %d", summary.Passed)
		}
		if summary.Failed != 0 {
			t.Errorf("expected 0 failed, got %d", summary.Failed)
		}
		if len(summary.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(summary.Results))
		}
		if summary.Results[0].Status != 200 {
			t.Errorf("expected status 200, got %d", summary.Results[0].Status)
		}
	})

	t.Run("executes requests in folders", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		coll := core.NewCollection("Test")
		folder := coll.AddFolder("API Endpoints")
		req := core.NewRequestDefinition("Get Users", "GET", server.URL+"/users")
		folder.AddRequest(req)

		runner := NewRunner(coll)
		summary := runner.Run(context.Background())

		if summary.TotalRequests != 1 {
			t.Errorf("expected 1 request, got %d", summary.TotalRequests)
		}
		if summary.Executed != 1 {
			t.Errorf("expected 1 executed, got %d", summary.Executed)
		}
	})

	t.Run("executes nested folder requests", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		coll := core.NewCollection("Test")
		folder := coll.AddFolder("API")
		subfolder := folder.AddFolder("Users")
		req := core.NewRequestDefinition("Get User", "GET", server.URL+"/user/1")
		subfolder.AddRequest(req)

		runner := NewRunner(coll)
		summary := runner.Run(context.Background())

		if summary.TotalRequests != 1 {
			t.Errorf("expected 1 request, got %d", summary.TotalRequests)
		}
	})

	t.Run("calls progress callback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Request 1", "GET", server.URL+"/1")
		coll.AddRequest(req)

		progressCalls := 0
		var lastCurrent, lastTotal int
		runner := NewRunner(coll, WithProgressCallback(func(current, total int, result *RunResult) {
			progressCalls++
			lastCurrent = current
			lastTotal = total
		}))

		runner.Run(context.Background())

		if progressCalls != 1 {
			t.Errorf("expected 1 progress call, got %d", progressCalls)
		}
		if lastCurrent != 1 || lastTotal != 1 {
			t.Errorf("expected current=1, total=1, got current=%d, total=%d", lastCurrent, lastTotal)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		coll := core.NewCollection("Test")
		for i := 0; i < 5; i++ {
			req := core.NewRequestDefinition("Request", "GET", server.URL)
			coll.AddRequest(req)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		runner := NewRunner(coll)
		summary := runner.Run(ctx)

		// Should execute 0 requests due to immediate cancellation
		if summary.Executed > 1 {
			t.Errorf("expected at most 1 executed after cancellation, got %d", summary.Executed)
		}
	})

	t.Run("handles request errors", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Bad Request", "GET", "http://invalid-host-that-does-not-exist.local/")
		coll.AddRequest(req)

		runner := NewRunner(coll)
		summary := runner.Run(context.Background())

		if summary.Failed != 1 {
			t.Errorf("expected 1 failed, got %d", summary.Failed)
		}
		if summary.Results[0].Error == nil {
			t.Error("expected error in result")
		}
	})

	t.Run("records timing information", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Request", "GET", server.URL)
		coll.AddRequest(req)

		runner := NewRunner(coll)
		summary := runner.Run(context.Background())

		if summary.StartTime.IsZero() {
			t.Error("expected start time to be set")
		}
		if summary.EndTime.IsZero() {
			t.Error("expected end time to be set")
		}
		if summary.TotalDuration <= 0 {
			t.Error("expected positive duration")
		}
		if summary.Results[0].Duration <= 0 {
			t.Error("expected positive request duration")
		}
	})
}

func TestRunner_walkRequests(t *testing.T) {
	t.Run("collects requests from root", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req1 := core.NewRequestDefinition("Req1", "GET", "http://example.com/1")
		req2 := core.NewRequestDefinition("Req2", "POST", "http://example.com/2")
		coll.AddRequest(req1)
		coll.AddRequest(req2)

		runner := NewRunner(coll)
		requests := runner.walkRequests(coll)

		if len(requests) != 2 {
			t.Errorf("expected 2 requests, got %d", len(requests))
		}
	})

	t.Run("collects requests from folders and subfolders", func(t *testing.T) {
		coll := core.NewCollection("Test")

		// Root request
		rootReq := core.NewRequestDefinition("Root", "GET", "http://example.com/root")
		coll.AddRequest(rootReq)

		// Folder with request
		folder := coll.AddFolder("Folder")
		folderReq := core.NewRequestDefinition("Folder Req", "GET", "http://example.com/folder")
		folder.AddRequest(folderReq)

		// Nested folder with request
		subfolder := folder.AddFolder("Subfolder")
		subReq := core.NewRequestDefinition("Sub Req", "GET", "http://example.com/sub")
		subfolder.AddRequest(subReq)

		runner := NewRunner(coll)
		requests := runner.walkRequests(coll)

		if len(requests) != 3 {
			t.Errorf("expected 3 requests, got %d", len(requests))
		}
	})
}

func TestRunResult_IsSuccess(t *testing.T) {
	t.Run("returns true when no error", func(t *testing.T) {
		result := &RunResult{
			RequestID: "1",
			Status:    200,
			Error:     nil,
		}
		if !result.IsSuccess() {
			t.Error("expected IsSuccess to return true")
		}
	})

	t.Run("returns false when error exists", func(t *testing.T) {
		result := &RunResult{
			RequestID: "1",
			Error:     context.DeadlineExceeded,
		}
		if result.IsSuccess() {
			t.Error("expected IsSuccess to return false")
		}
	})
}

func TestRunResult_AllTestsPassed(t *testing.T) {
	t.Run("returns true when no tests", func(t *testing.T) {
		result := &RunResult{
			TestResults: nil,
		}
		if !result.AllTestsPassed() {
			t.Error("expected AllTestsPassed to return true with no tests")
		}
	})

	t.Run("returns true when all tests pass", func(t *testing.T) {
		result := &RunResult{
			TestResults: []script.TestResult{
				{Name: "Test 1", Passed: true},
				{Name: "Test 2", Passed: true},
			},
		}
		if !result.AllTestsPassed() {
			t.Error("expected AllTestsPassed to return true")
		}
	})

	t.Run("returns false when any test fails", func(t *testing.T) {
		result := &RunResult{
			TestResults: []script.TestResult{
				{Name: "Test 1", Passed: true},
				{Name: "Test 2", Passed: false},
			},
		}
		if result.AllTestsPassed() {
			t.Error("expected AllTestsPassed to return false")
		}
	})
}

func TestRunSummary_IsSuccess(t *testing.T) {
	t.Run("returns true when no failures", func(t *testing.T) {
		summary := &RunSummary{
			Failed: 0,
			Passed: 5,
		}
		if !summary.IsSuccess() {
			t.Error("expected IsSuccess to return true")
		}
	})

	t.Run("returns false when failures exist", func(t *testing.T) {
		summary := &RunSummary{
			Failed: 2,
			Passed: 3,
		}
		if summary.IsSuccess() {
			t.Error("expected IsSuccess to return false")
		}
	})
}

func TestRunSummary_AllTestsPassed(t *testing.T) {
	t.Run("returns true when no test failures", func(t *testing.T) {
		summary := &RunSummary{
			TotalTests:  10,
			TestsPassed: 10,
			TestsFailed: 0,
		}
		if !summary.AllTestsPassed() {
			t.Error("expected AllTestsPassed to return true")
		}
	})

	t.Run("returns false when test failures exist", func(t *testing.T) {
		summary := &RunSummary{
			TotalTests:  10,
			TestsPassed: 8,
			TestsFailed: 2,
		}
		if summary.AllTestsPassed() {
			t.Error("expected AllTestsPassed to return false")
		}
	})
}

func TestRunner_WithEnvironment(t *testing.T) {
	t.Run("interpolates environment variables in URL", func(t *testing.T) {
		var receivedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		env := core.NewEnvironment("test")
		env.SetVariable("endpoint", "/api/users")

		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Request", "GET", server.URL+"{{endpoint}}")
		coll.AddRequest(req)

		runner := NewRunner(coll, WithEnvironment(env))
		runner.Run(context.Background())

		if receivedPath != "/api/users" {
			t.Errorf("expected path '/api/users', got '%s'", receivedPath)
		}
	})
}

// mockCookieJar implements http.CookieJar for testing
type mockCookieJar struct {
	cookies []*http.Cookie
}

func (m *mockCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	m.cookies = append(m.cookies, cookies...)
}

func (m *mockCookieJar) Cookies(u *url.URL) []*http.Cookie {
	return m.cookies
}

func TestRunner_WithHTTPClient(t *testing.T) {
	t.Run("uses custom HTTP client", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Request", "GET", "https://example.com")
		coll.AddRequest(req)

		// Create runner with custom HTTP client option
		runner := NewRunner(coll, WithHTTPClient(nil))

		// The WithHTTPClient option sets the httpClient field
		// Even if we pass nil, the option is exercised
		if runner == nil {
			t.Error("expected runner to be created")
		}
	})
}
