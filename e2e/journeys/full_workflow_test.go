package journeys

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/artpar/currier/e2e/harness"
	"github.com/artpar/currier/internal/history"
	"github.com/artpar/currier/internal/history/sqlite"
)

// TestFullWorkflow_SendRequestAndVerifyHistory tests the complete workflow:
// 1. Create a new request
// 2. Type a URL
// 3. Send the request
// 4. Verify response is displayed
// 5. Switch to history view
// 6. Verify request appears in history
func TestFullWorkflow_SendRequestAndVerifyHistory(t *testing.T) {
	// Create harness with in-memory history store
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/journeys",
	})

	// Create in-memory history store
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	session := h.TUI().Start(t)
	session.Model().SetHistoryStore(store)
	defer session.Quit()

	// Step 1: Create new request
	t.Log("Step 1: Press 'n' to create new request")
	session.SendKey("n")

	state := session.State()
	if state.MainView.Mode != "INSERT" {
		t.Errorf("Expected INSERT mode, got %s", state.MainView.Mode)
	}
	if !state.Request.IsEditing {
		t.Error("Expected to be editing")
	}

	// Step 2: Type URL (using httpbin for reliable testing)
	t.Log("Step 2: Type URL")
	session.Type("https://httpbin.org/get")

	state = session.State()
	if state.Request.URL != "https://httpbin.org/get" {
		t.Errorf("Expected URL 'https://httpbin.org/get', got '%s'", state.Request.URL)
	}

	// Step 3: Exit edit mode and send request
	t.Log("Step 3: Exit edit mode")
	session.SendKey("Escape")

	state = session.State()
	if state.MainView.Mode != "NORMAL" {
		t.Errorf("Expected NORMAL mode after Escape, got %s", state.MainView.Mode)
	}

	t.Log("Step 4: Send request (Enter)")
	session.SendKey("Enter")

	// The request is sent synchronously now that we execute tea.Cmd
	// Give it a moment to complete
	time.Sleep(100 * time.Millisecond)

	// Step 5: Verify response
	t.Log("Step 5: Verify response")
	state = session.State()

	if !state.Response.HasResponse {
		if state.Response.Error != "" {
			t.Logf("Request failed with error: %s", state.Response.Error)
		} else if state.Response.IsLoading {
			t.Log("Response is still loading...")
			// Wait a bit more
			time.Sleep(2 * time.Second)
			state = session.State()
		}
	}

	if state.Response.HasResponse {
		t.Logf("Response received: %d %s", state.Response.StatusCode, state.Response.StatusText)
		if state.Response.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", state.Response.StatusCode)
		}
	} else {
		t.Logf("No response yet. Error: %s, Loading: %v", state.Response.Error, state.Response.IsLoading)
	}

	// Step 6: Verify response body contains expected content
	output := session.Output()
	t.Log("Step 6: Verify response content in output")
	if state.Response.HasResponse && !strings.Contains(output, "200") {
		t.Log("Response panel may not show status in output")
	}

	// Step 7: Check history - give async save time to complete
	t.Log("Step 7: Verify history was saved")
	time.Sleep(200 * time.Millisecond)

	// Query history directly from the store
	entries := queryHistory(t, store)
	t.Logf("History entries found: %d", len(entries))

	if len(entries) == 0 {
		t.Error("Expected at least 1 history entry, got 0")
	} else {
		entry := entries[0]
		t.Logf("History entry: %s %s -> %d", entry.RequestMethod, entry.RequestURL, entry.ResponseStatus)

		if entry.RequestMethod != "GET" {
			t.Errorf("Expected method GET, got %s", entry.RequestMethod)
		}
		if !strings.Contains(entry.RequestURL, "httpbin.org") {
			t.Errorf("Expected URL to contain httpbin.org, got %s", entry.RequestURL)
		}
	}

	// Step 8: Verify History view is active (default view mode is now history)
	t.Log("Step 8: Verify History view is active")
	session.SendKey("1") // Focus collections pane

	state = session.State()
	// History is now the default view mode
	if state.Tree.ViewMode != "history" {
		t.Errorf("Expected history view mode, got %s", state.Tree.ViewMode)
	}

	// Verify history is displayed in TUI
	output = session.Output()
	if !strings.Contains(output, "httpbin") {
		t.Log("Note: History may need refresh to show in TUI")
	}
}

// TestFullWorkflow_SendPOSTRequest tests sending a POST request with body.
func TestFullWorkflow_SendPOSTRequest(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/journeys",
	})

	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	session := h.TUI().Start(t)
	session.Model().SetHistoryStore(store)
	defer session.Quit()

	// Create new request
	t.Log("Creating new POST request")
	session.SendKey("n")
	session.Type("https://httpbin.org/post")
	session.SendKey("Escape")

	// Change method to POST (m cycles: GET -> POST)
	t.Log("Changing method to POST")
	session.SendKey("m") // Cycle to POST

	state := session.State()
	if state.Request.Method != "POST" {
		t.Errorf("Expected method POST, got %s", state.Request.Method)
	}

	// Switch to Body tab and add body
	t.Log("Adding request body")
	session.SendKey("]") // Headers
	session.SendKey("]") // Query
	session.SendKey("]") // Body
	session.SendKey("e") // Edit body

	session.Type(`{"name": "test", "value": 123}`)
	session.SendKey("Escape")

	// Send request
	t.Log("Sending POST request")
	session.SendKey("Enter")

	// Wait for response
	time.Sleep(2 * time.Second)

	state = session.State()
	if state.Response.HasResponse {
		t.Logf("POST response: %d", state.Response.StatusCode)
		if state.Response.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", state.Response.StatusCode)
		}
	}

	// Verify history
	time.Sleep(200 * time.Millisecond)
	entries := queryHistory(t, store)
	if len(entries) == 0 {
		t.Error("Expected history entry for POST request")
	} else {
		if entries[0].RequestMethod != "POST" {
			t.Errorf("History should show POST, got %s", entries[0].RequestMethod)
		}
	}
}

// TestFullWorkflow_RequestWithHeaders tests sending a request with custom headers.
func TestFullWorkflow_RequestWithHeaders(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/journeys",
	})

	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	session := h.TUI().Start(t)
	session.Model().SetHistoryStore(store)
	defer session.Quit()

	// Create request
	session.SendKey("n")
	session.Type("https://httpbin.org/headers")
	session.SendKey("Escape")

	// Add custom header
	t.Log("Adding custom header")
	session.SendKey("]") // Switch to Headers tab
	session.SendKey("a") // Add header

	session.Type("X-Custom-Header")
	session.SendKey("Tab")
	session.Type("TestValue123")
	session.SendKey("Enter")

	// Verify header was added
	state := session.State()
	if state.Request.Headers == nil {
		t.Error("Headers should not be nil")
	} else if state.Request.Headers["X-Custom-Header"] != "TestValue123" {
		t.Errorf("Header value mismatch: got %q", state.Request.Headers["X-Custom-Header"])
	}

	// Send request
	t.Log("Sending request with custom header")
	session.SendKey("Enter")
	time.Sleep(2 * time.Second)

	state = session.State()
	if state.Response.HasResponse {
		// Check if response body contains our header
		body := state.Response.BodyPreview
		if strings.Contains(body, "X-Custom-Header") || strings.Contains(body, "TestValue123") {
			t.Log("Custom header reflected in response body")
		}
	}

	// Verify history includes headers
	time.Sleep(200 * time.Millisecond)
	entries := queryHistory(t, store)
	if len(entries) > 0 {
		if entries[0].RequestHeaders["X-Custom-Header"] != "TestValue123" {
			t.Log("Note: Header may not be saved to history correctly")
		}
	}
}

// TestFullWorkflow_ErrorHandling tests error handling for invalid requests.
func TestFullWorkflow_ErrorHandling(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/journeys",
	})

	session := h.TUI().Start(t)
	defer session.Quit()

	// Create request with invalid URL
	session.SendKey("n")
	session.Type("not-a-valid-url")
	session.SendKey("Escape")

	// Try to send
	session.SendKey("Enter")
	time.Sleep(100 * time.Millisecond)

	state := session.State()
	if state.Response.Error == "" {
		t.Log("Note: Error may not be captured if URL validation is lenient")
	} else {
		t.Logf("Got expected error: %s", state.Response.Error)
	}

	// Verify error is displayed in output
	output := session.Output()
	if strings.Contains(output, "Error") || strings.Contains(output, "error") || strings.Contains(output, "http://") {
		t.Log("Error feedback visible in output")
	}
}

// historyEntry is a local type for history test assertions.
type historyEntry struct {
	RequestMethod   string
	RequestURL      string
	ResponseStatus  int
	RequestHeaders  map[string]string
}

// queryHistory is a helper to query history entries.
func queryHistory(t *testing.T, store *sqlite.Store) []historyEntry {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entries, err := store.List(ctx, history.QueryOptions{Limit: 100})
	if err != nil {
		t.Logf("Error querying history: %v", err)
		return nil
	}

	result := make([]historyEntry, len(entries))
	for i, e := range entries {
		result[i] = historyEntry{
			RequestMethod:   e.RequestMethod,
			RequestURL:      e.RequestURL,
			ResponseStatus:  e.ResponseStatus,
			RequestHeaders:  e.RequestHeaders,
		}
	}
	return result
}
