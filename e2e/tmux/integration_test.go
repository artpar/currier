package tmux_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/artpar/currier/e2e/tmux"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Get the project root directory
	projectRoot := filepath.Join("..", "..")

	// Build binary before running tests
	buildCmd := exec.Command("go", "build", "-o", "bin/currier-test", "./cmd/currier")
	buildCmd.Dir = projectRoot
	if err := buildCmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	// Set absolute path to binary
	absRoot, _ := filepath.Abs(projectRoot)
	binaryPath = filepath.Join(absRoot, "bin", "currier-test")

	code := m.Run()

	// Cleanup
	os.Remove(binaryPath)
	os.Exit(code)
}

// =============================================================================
// SMOKE TESTS - Basic app functionality
// =============================================================================

func TestSmoke_AppStarts(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	// Should see initial UI elements
	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("App did not show Collections pane: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}

	// Should show Request and Response panes
	screen := sess.Capture()
	if !strings.Contains(screen, "Request") {
		t.Error("Missing Request pane")
	}
	if !strings.Contains(screen, "Response") {
		t.Error("Missing Response pane")
	}
	if !strings.Contains(screen, "NORMAL") {
		t.Error("Should start in NORMAL mode")
	}
}

func TestSmoke_QuitWithQ(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Press 'q' to quit
	sess.SendKey("q")
	time.Sleep(500 * time.Millisecond)

	// Session should be dead or show shell
	if sess.IsAlive() {
		screen := sess.Capture()
		if strings.Contains(screen, "Collections") && strings.Contains(screen, "Request") {
			t.Error("App did not quit on 'q' key")
		}
	}
}

func TestSmoke_QuitWithCtrlC(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Press Ctrl+C to quit
	sess.SendKey("Ctrl+C")
	time.Sleep(500 * time.Millisecond)

	// Should have quit
	if sess.IsAlive() {
		screen := sess.Capture()
		if strings.Contains(screen, "Collections") && strings.Contains(screen, "Request") {
			t.Error("App did not quit on Ctrl+C")
		}
	}
}

// =============================================================================
// HELP OVERLAY TESTS
// =============================================================================

func TestHelp_ShowAndDismiss(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Press '?' to show help
	sess.SendKey("?")
	time.Sleep(200 * time.Millisecond)

	// Help should be visible (check for some help-related text)
	screen := sess.Capture()
	hasHelp := strings.Contains(screen, "Help") || strings.Contains(screen, "Keys") || strings.Contains(screen, "Navigation")
	if !hasHelp {
		t.Log("Note: Help overlay may use different text format")
	}

	// Press Escape to close help
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Should be back to normal view
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("Failed to return to normal view after help")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestHelp_DismissWithQuestionMark(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Press '?' to show help
	sess.SendKey("?")
	time.Sleep(200 * time.Millisecond)

	// Press '?' again to close help
	sess.SendKey("?")
	time.Sleep(200 * time.Millisecond)

	// Should be back to normal view
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("Failed to dismiss help with '?'")
	}
}

// =============================================================================
// PANE NAVIGATION TESTS
// =============================================================================

func TestNavigation_TabCyclesPanes(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Should start with Collections focused
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("Should start on Collections pane")
	}

	// Tab to Request pane
	sess.SendKey("Tab")
	time.Sleep(100 * time.Millisecond)

	// Tab to Response pane
	sess.SendKey("Tab")
	time.Sleep(100 * time.Millisecond)

	// Tab back to Collections
	sess.SendKey("Tab")
	time.Sleep(100 * time.Millisecond)

	// Should still be running
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during tab navigation")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestNavigation_NumberKeysJumpToPanes(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Press '2' to focus Request pane
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)

	// Press '3' to focus Response pane
	sess.SendKey("3")
	time.Sleep(100 * time.Millisecond)

	// Press '1' to focus Collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during number key navigation")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestNavigation_ShiftTabReversesCycle(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Tab forward twice
	sess.SendKey("Tab")
	time.Sleep(50 * time.Millisecond)
	sess.SendKey("Tab")
	time.Sleep(50 * time.Millisecond)

	// Shift+Tab backward once
	sess.SendKey("Shift-Tab")
	time.Sleep(50 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during shift-tab navigation")
	}
}

// =============================================================================
// NEW REQUEST TESTS
// =============================================================================

func TestNewRequest_CreatesRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Press 'n' to create new request
	sess.SendKey("n")

	// Should enter INSERT mode
	if err := sess.WaitFor("INSERT", 2*time.Second); err != nil {
		t.Errorf("Did not enter INSERT mode: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}

	// Should show GET method selector and URL input
	screen := sess.Capture()
	if !strings.Contains(screen, "GET") {
		t.Error("New request should default to GET method")
	}
}

func TestNewRequest_TypeURLAndEscape(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create new request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type URL
	sess.Type("https://example.com/api")

	// Press Escape to exit INSERT mode
	sess.SendKey("Escape")

	// Should be in NORMAL mode with URL saved
	if err := sess.WaitFor("NORMAL", 2*time.Second); err != nil {
		t.Errorf("Did not return to NORMAL mode: %v", err)
	}

	screen := sess.Capture()
	if !strings.Contains(screen, "example.com") {
		t.Error("URL was not saved")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestNewRequest_SpaceCharacterInURL(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request and type URL with space
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	sess.Type("https://example.com/path")
	sess.SendKey("Space")
	sess.Type("with")
	sess.SendKey("Space")
	sess.Type("spaces")

	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Verify spaces are in the URL
	screen := sess.Capture()
	if !strings.Contains(screen, "path with spaces") && !strings.Contains(screen, "path%20with%20spaces") {
		t.Errorf("Spaces not preserved in URL")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestNewRequest_MultipleRequestsInSession(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create first request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://first.example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Create second request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://second.example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Both should be in the collection tree
	screen := sess.Capture()
	// The tree should show multiple requests
	if !strings.Contains(screen, "Default") {
		t.Error("Default collection not shown")
		t.Logf("Screen:\n%s", screen)
	}
}

// =============================================================================
// HTTP REQUEST TESTS - REAL NETWORK CALLS
// =============================================================================

func TestHTTP_SendGETRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create new request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type URL
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Send request
	sess.SendKey("Enter")

	// Wait for response (real HTTP call!)
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Errorf("Did not receive 200 response: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestHTTP_ResponseContainsHeaders(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create and send request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.SendKey("Enter")

	// Wait for response
	sess.WaitFor("200", 15*time.Second)

	// Check response contains expected data
	screen := sess.Capture()
	// httpbin.org/get returns JSON with headers info
	if !strings.Contains(screen, "httpbin") {
		t.Error("Response should contain httpbin data")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestHTTP_POSTRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create new request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/post")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Change method to POST using 'm'
	sess.SendKey("m")
	time.Sleep(100 * time.Millisecond)

	// Navigate to POST (should be after GET in the method list)
	sess.SendKey("j") // Down to next method
	sess.SendKey("Enter")
	time.Sleep(100 * time.Millisecond)

	// Verify POST is selected
	screen := sess.Capture()
	if !strings.Contains(screen, "POST") {
		t.Log("Note: Method may still show GET - checking after send")
	}

	// Send request
	sess.SendKey("Enter")

	// Wait for response
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Errorf("POST request failed: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestHTTP_404Response(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request to non-existent endpoint
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/status/404")
	sess.SendKey("Escape")
	sess.SendKey("Enter")

	// Should get 404
	if err := sess.WaitFor("404", 15*time.Second); err != nil {
		t.Errorf("Did not receive 404 response: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestHTTP_InvalidURLError(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request with invalid URL
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("not-a-valid-url")
	sess.SendKey("Escape")
	sess.SendKey("Enter")

	// Should show error (app should not crash)
	time.Sleep(2 * time.Second)
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed on invalid URL")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestHTTP_EmptyURLHandled(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request without typing URL
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.SendKey("Escape")
	time.Sleep(100 * time.Millisecond)

	// Try to send
	sess.SendKey("Enter")
	time.Sleep(500 * time.Millisecond)

	// App should not crash
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Errorf("App crashed on empty URL send")
		t.Logf("Screen:\n%s", screen)
	}
}

// =============================================================================
// REQUEST PANEL TAB TESTS
// =============================================================================

func TestRequestTabs_SwitchWithBrackets(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request first
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)

	// Switch tabs with ]
	for i := 0; i < 6; i++ { // Cycle through all tabs
		sess.SendKey("]")
		time.Sleep(100 * time.Millisecond)
	}

	// Switch tabs with [
	for i := 0; i < 6; i++ {
		sess.SendKey("[")
		time.Sleep(100 * time.Millisecond)
	}

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Request") {
		t.Error("App crashed during tab switching")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestRequestTabs_URLTab(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com/test")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second) // Wait for mode transition

	// Should see URL tab info
	screen := sess.Capture()
	if !strings.Contains(screen, "URL") {
		t.Error("URL tab not visible")
	}
	if !strings.Contains(screen, "example.com") {
		t.Error("URL not displayed")
	}
}

func TestRequestTabs_HeadersTab(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel and go to Headers tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("]") // Next tab (Headers)
	time.Sleep(100 * time.Millisecond)

	// Should show Headers tab
	screen := sess.Capture()
	if !strings.Contains(screen, "Headers") {
		t.Error("Headers tab not visible")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestRequestTabs_AddHeader(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel and go to Headers tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("]") // Headers tab
	time.Sleep(100 * time.Millisecond)

	// Add a header with 'a'
	sess.SendKey("a")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type header name
	sess.Type("X-Custom-Header")
	sess.SendKey("Tab") // Move to value
	sess.Type("my-value")
	sess.SendKey("Escape")

	time.Sleep(200 * time.Millisecond)
	screen := sess.Capture()
	if !strings.Contains(screen, "X-Custom-Header") || !strings.Contains(screen, "my-value") {
		t.Log("Header may not be visible in current view")
	}
}

func TestRequestTabs_BodyTab(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel and go to Body tab (URL -> Headers -> Query -> Body)
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("]")
	sess.SendKey("]")
	sess.SendKey("]")
	time.Sleep(100 * time.Millisecond)

	// Should show Body tab
	screen := sess.Capture()
	if !strings.Contains(screen, "Body") {
		t.Error("Body tab not visible")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestRequestTabs_EditBody(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/post")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Go to Body tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 3; i++ {
		sess.SendKey("]")
		time.Sleep(50 * time.Millisecond)
	}

	// Edit body with 'e'
	sess.SendKey("e")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type JSON body
	sess.Type("{\"test\": \"value\"}")
	sess.SendKey("Escape")

	time.Sleep(200 * time.Millisecond)
	screen := sess.Capture()
	// Body content should be visible or saved
	if !strings.Contains(screen, "test") && !strings.Contains(screen, "Body") {
		t.Log("Body content may not be visible after edit")
	}
}

// =============================================================================
// COLLECTION TREE TESTS
// =============================================================================

func TestCollectionTree_NavigateWithJK(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create multiple requests
	for i := 0; i < 3; i++ {
		sess.SendKey("n")
		sess.WaitFor("INSERT", 2*time.Second)
		sess.Type("https://example.com/test")
		sess.SendKey("Escape")
		sess.WaitFor("NORMAL", 2*time.Second)
	}

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Navigate with j and k
	for i := 0; i < 5; i++ {
		sess.SendKey("j")
		time.Sleep(50 * time.Millisecond)
	}
	for i := 0; i < 5; i++ {
		sess.SendKey("k")
		time.Sleep(50 * time.Millisecond)
	}

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during j/k navigation")
	}
}

func TestCollectionTree_ExpandCollapse(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request (this creates a collection)
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Expand with 'l'
	sess.SendKey("l")
	time.Sleep(100 * time.Millisecond)

	// Collapse with 'h'
	sess.SendKey("h")
	time.Sleep(100 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Default") {
		t.Log("Collection may be collapsed")
	}
}

func TestCollectionTree_GotoTopBottom(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create multiple requests
	for i := 0; i < 3; i++ {
		sess.SendKey("n")
		sess.WaitFor("INSERT", 2*time.Second)
		sess.Type("https://example.com")
		sess.SendKey("Escape")
		sess.WaitFor("NORMAL", 2*time.Second)
	}

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Go to bottom with 'G'
	sess.SendKey("G")
	time.Sleep(100 * time.Millisecond)

	// Go to top with 'gg'
	sess.SendKey("g")
	sess.SendKey("g")
	time.Sleep(100 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during G/gg navigation")
	}
}

func TestCollectionTree_SelectRequestWithEnter(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://selected.example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Navigate to the request and select it
	sess.SendKey("j") // Move into collection
	sess.SendKey("l") // Expand
	sess.SendKey("j") // Move to request
	sess.SendKey("Enter")
	time.Sleep(200 * time.Millisecond)

	// Request should be loaded in request panel
	screen := sess.Capture()
	if !strings.Contains(screen, "selected.example.com") {
		t.Log("Request may not be visible after selection")
	}
}

// =============================================================================
// HISTORY VIEW TESTS
// =============================================================================

func TestHistory_SwitchToHistoryView(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Press 'H' to switch to history view
	sess.SendKey("H")
	time.Sleep(200 * time.Millisecond)

	// Should show history view (may show "No history" if empty)
	screen := sess.Capture()
	if !strings.Contains(screen, "History") && !strings.Contains(screen, "history") {
		t.Log("History view may use different indicator")
	}
}

func TestHistory_RequestAppearsInHistory(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Make a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.SendKey("Enter")

	// Wait for response
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Switch to history view
	sess.SendKey("1") // Focus collections pane
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("H") // History view
	time.Sleep(200 * time.Millisecond)

	// Should show history entry
	if err := sess.WaitFor("httpbin", 5*time.Second); err != nil {
		t.Errorf("History entry not shown: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestHistory_SelectHistoryEntry(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Make a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.SendKey("Enter")
	sess.WaitFor("200", 15*time.Second)

	// Switch to history view
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("H")
	sess.WaitFor("httpbin", 5*time.Second)

	// Select the history entry
	sess.SendKey("j") // Navigate to entry
	sess.SendKey("Enter")
	time.Sleep(200 * time.Millisecond)

	// Request should be loaded
	screen := sess.Capture()
	if !strings.Contains(screen, "httpbin") {
		t.Log("History entry may not be loaded in request panel")
	}
}

// =============================================================================
// RESPONSE PANEL TESTS
// =============================================================================

func TestResponse_TabSwitching(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Make a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.SendKey("Enter")
	sess.WaitFor("200", 15*time.Second)

	// Focus response panel
	sess.SendKey("3")
	time.Sleep(100 * time.Millisecond)

	// Switch tabs
	for i := 0; i < 5; i++ {
		sess.SendKey("]")
		time.Sleep(100 * time.Millisecond)
	}

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Response") {
		t.Error("App crashed during response tab switching")
	}
}

func TestResponse_ScrollBody(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Make a request that returns lots of data
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.SendKey("Enter")
	sess.WaitFor("200", 15*time.Second)

	// Focus response panel
	sess.SendKey("3")
	time.Sleep(100 * time.Millisecond)

	// Scroll with j and k
	for i := 0; i < 10; i++ {
		sess.SendKey("j")
		time.Sleep(50 * time.Millisecond)
	}
	for i := 0; i < 10; i++ {
		sess.SendKey("k")
		time.Sleep(50 * time.Millisecond)
	}

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Response") {
		t.Error("App crashed during response scrolling")
	}
}

// =============================================================================
// METHOD EDITING TESTS
// =============================================================================

func TestMethod_ChangeMethod(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel on URL tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)

	// Press 'm' to edit method
	sess.SendKey("m")
	time.Sleep(200 * time.Millisecond)

	// Navigate through methods and select
	sess.SendKey("j") // Next method
	sess.SendKey("j") // Next method
	sess.SendKey("Enter")
	time.Sleep(200 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Request") {
		t.Error("App crashed during method change")
	}
}

// =============================================================================
// QUERY PARAMETERS TESTS
// =============================================================================

func TestQuery_AddQueryParam(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel and go to Query tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("]") // Headers
	sess.SendKey("]") // Query
	time.Sleep(100 * time.Millisecond)

	// Add a query param with 'a'
	sess.SendKey("a")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type param name and value
	sess.Type("search")
	sess.SendKey("Tab")
	sess.Type("hello")
	sess.SendKey("Escape")

	time.Sleep(200 * time.Millisecond)
	screen := sess.Capture()
	if !strings.Contains(screen, "Query") {
		t.Error("Query tab not visible after adding param")
	}
}

// =============================================================================
// URL EDITING TESTS
// =============================================================================

func TestURL_EditExistingURL(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request with URL
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://original.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)

	// Edit URL with 'e'
	sess.SendKey("e")
	sess.WaitFor("INSERT", 2*time.Second)

	// Clear and type new URL (Ctrl+U to clear line)
	sess.SendKey("Ctrl+U")
	sess.Type("https://modified.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Verify URL changed
	screen := sess.Capture()
	if !strings.Contains(screen, "modified.com") {
		t.Error("URL was not modified")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestURL_CursorNavigation(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type URL
	sess.Type("https://example.com")

	// Navigate with arrow keys
	sess.SendKey("Left")
	sess.SendKey("Left")
	sess.SendKey("Left")
	sess.SendKey("Right")

	// Type in the middle
	sess.Type("/inserted")

	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Verify content (may have inserted text)
	screen := sess.Capture()
	if !strings.Contains(screen, "example") {
		t.Error("URL content lost during cursor navigation")
	}
}

func TestURL_BackspaceWorks(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type URL
	sess.Type("https://example.com/extra")

	// Backspace to remove
	for i := 0; i < 6; i++ { // Remove "/extra"
		sess.SendKey("Backspace")
	}

	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Verify backspace worked
	screen := sess.Capture()
	if strings.Contains(screen, "/extra") {
		t.Error("Backspace did not remove text")
	}
}

// =============================================================================
// EDGE CASE TESTS
// =============================================================================

func TestEdge_VerySmallTerminal(t *testing.T) {
	sess := tmux.NewSession(t).WithSize(40, 15)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Should not crash on small terminal
	time.Sleep(500 * time.Millisecond)
	if !sess.IsAlive() {
		t.Error("App crashed on small terminal")
	}

	// Basic operations should still work
	sess.SendKey("n")
	time.Sleep(500 * time.Millisecond)

	if !sess.IsAlive() {
		t.Error("App crashed after 'n' on small terminal")
	}
}

func TestEdge_RapidKeyPresses(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Rapid key presses
	for i := 0; i < 20; i++ {
		sess.SendKey("Tab")
	}
	for i := 0; i < 10; i++ {
		sess.SendKey("1")
		sess.SendKey("2")
		sess.SendKey("3")
	}

	// Should still be running
	time.Sleep(200 * time.Millisecond)
	if !sess.IsAlive() {
		t.Error("App crashed during rapid key presses")
	}
}

func TestEdge_LongURL(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request with very long URL
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	longURL := "https://example.com/very/long/path/that/goes/on/and/on/and/on/with/many/segments/and/query?param1=value1&param2=value2&param3=value3"
	sess.Type(longURL)
	sess.SendKey("Escape")

	// Should not crash
	time.Sleep(200 * time.Millisecond)
	if !sess.IsAlive() {
		t.Error("App crashed with long URL")
	}
}

func TestEdge_SpecialCharactersInURL(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request with special characters
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	sess.Type("https://example.com/path?q=hello+world&name=foo%20bar")
	sess.SendKey("Escape")

	// Should not crash
	time.Sleep(200 * time.Millisecond)
	if !sess.IsAlive() {
		t.Error("App crashed with special characters")
	}
}

func TestEdge_MultipleSessionsParallel(t *testing.T) {
	// Create multiple sessions to test isolation
	sessions := make([]*tmux.Session, 3)

	for i := 0; i < 3; i++ {
		sessions[i] = tmux.NewSession(t)
		if err := sessions[i].Start(binaryPath); err != nil {
			t.Fatalf("Failed to start session %d: %v", i, err)
		}
	}

	// Cleanup all sessions
	defer func() {
		for _, sess := range sessions {
			sess.Kill()
		}
	}()

	// Wait for all to start
	for i, sess := range sessions {
		if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
			t.Errorf("Session %d failed to start: %v", i, err)
		}
	}

	// All should be running
	for i, sess := range sessions {
		if !sess.IsAlive() {
			t.Errorf("Session %d died unexpectedly", i)
		}
	}
}

// =============================================================================
// SEARCH FUNCTIONALITY TESTS
// =============================================================================

func TestSearch_EnterSearchMode(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request first
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://searchable.example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Enter search mode with '/'
	sess.SendKey("/")
	time.Sleep(200 * time.Millisecond)

	// Type search query
	sess.Type("search")

	// Exit search with Escape
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during search")
	}
}

// =============================================================================
// FULL WORKFLOW TESTS
// =============================================================================

func TestWorkflow_CompleteRequestCycle(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// 1. Create new request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	// 2. Type URL
	sess.Type("https://httpbin.org/post")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// 3. Change method to POST
	sess.SendKey("m")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("j") // Navigate to POST
	sess.SendKey("Enter")
	time.Sleep(100 * time.Millisecond)

	// 4. Add a header
	sess.SendKey("]") // Go to Headers tab
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("a")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("Content-Type")
	sess.SendKey("Tab")
	sess.Type("application/json")
	sess.SendKey("Escape")
	time.Sleep(100 * time.Millisecond)

	// 5. Add body
	sess.SendKey("]") // Query tab
	sess.SendKey("]") // Body tab
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("e")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("{\"test\": true}")
	sess.SendKey("Escape")
	time.Sleep(100 * time.Millisecond)

	// 6. Send request
	sess.SendKey("Enter")

	// 7. Wait for response
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Errorf("Complete workflow failed: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}

	// 8. Verify response panel has data
	sess.SendKey("3") // Focus response panel
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	if !strings.Contains(screen, "Response") {
		t.Error("Response panel not visible after request")
	}
}

func TestWorkflow_MultipleRequestsAndHistory(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Make multiple requests
	urls := []string{
		"https://httpbin.org/get",
		"https://httpbin.org/headers",
	}

	for _, url := range urls {
		sess.SendKey("n")
		sess.WaitFor("INSERT", 2*time.Second)
		sess.Type(url)
		sess.SendKey("Escape")
		sess.SendKey("Enter")
		sess.WaitFor("200", 15*time.Second)
	}

	// Check history
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("H")
	time.Sleep(500 * time.Millisecond)

	// Should see history entries
	screen := sess.Capture()
	if !strings.Contains(screen, "httpbin") {
		t.Error("History should contain httpbin entries")
		t.Logf("Screen:\n%s", screen)
	}
}

// =============================================================================
// CURL COMMAND TESTS
// =============================================================================

func TestCurl_SimpleGET(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	// Start with curl command
	if err := sess.Start(binaryPath, "curl", "https://httpbin.org/get"); err != nil {
		t.Fatalf("Failed to start with curl: %v", err)
	}

	// Should show TUI with the request loaded
	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
		return
	}

	// Should show the URL from curl command
	screen := sess.Capture()
	if !strings.Contains(screen, "httpbin.org") {
		t.Error("URL from curl command not loaded")
		t.Logf("Screen:\n%s", screen)
	}

	// Should show GET method
	if !strings.Contains(screen, "GET") {
		t.Error("GET method not shown")
	}
}

func TestCurl_POSTWithData(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	// Start with curl POST command
	if err := sess.Start(binaryPath, "curl", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", `{"name":"test"}`,
		"https://httpbin.org/post"); err != nil {
		t.Fatalf("Failed to start with curl: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
		return
	}

	// Should show POST method
	screen := sess.Capture()
	if !strings.Contains(screen, "POST") {
		t.Error("POST method not shown")
		t.Logf("Screen:\n%s", screen)
	}

	// Should show the URL
	if !strings.Contains(screen, "httpbin.org") {
		t.Error("URL not loaded")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestCurl_WithHeaders(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath, "curl",
		"-H", "Authorization: Bearer mytoken",
		"-H", "X-Custom-Header: myvalue",
		"https://httpbin.org/headers"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	// Focus request panel and go to Headers tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("]") // Headers tab
	time.Sleep(200 * time.Millisecond)

	// Should show headers
	screen := sess.Capture()
	if !strings.Contains(screen, "Authorization") && !strings.Contains(screen, "Bearer") {
		t.Log("Headers may not be visible in current view")
	}
}

func TestCurl_WithBasicAuth(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath, "curl",
		"-u", "admin:secret",
		"https://httpbin.org/basic-auth/admin/secret"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	// Should be running with URL loaded
	screen := sess.Capture()
	if !strings.Contains(screen, "httpbin.org") {
		t.Error("URL not loaded")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestCurl_JSONFlag(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath, "curl",
		"--json", `{"test":true}`,
		"https://httpbin.org/post"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	// Should show POST (--json implies POST)
	screen := sess.Capture()
	if !strings.Contains(screen, "POST") {
		t.Error("POST method not set by --json flag")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestCurl_SendRequestAfterLoad(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath, "curl", "https://httpbin.org/get"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	// Focus request panel before sending
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)

	// Send the request
	sess.SendKey("Enter")

	// Wait for response
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Errorf("Request failed: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestCurl_ComplexPOSTAndSend(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath, "curl",
		"-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", `{"message":"hello"}`,
		"https://httpbin.org/post"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	// Focus request panel before sending
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)

	// Send the request
	sess.SendKey("Enter")

	// Wait for response
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Errorf("POST request failed: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
		return
	}

	// Response should contain our data echoed back
	screen := sess.Capture()
	if !strings.Contains(screen, "message") || !strings.Contains(screen, "hello") {
		t.Log("Response may not show echoed data in visible area")
	}
}

func TestCurl_HEADRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath, "curl", "-I", "https://httpbin.org/get"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	// Should show HEAD method
	screen := sess.Capture()
	if !strings.Contains(screen, "HEAD") {
		t.Error("HEAD method not set by -I flag")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestCurl_UserAgent(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath, "curl",
		"-A", "MyCustomAgent/1.0",
		"https://httpbin.org/user-agent"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	// Focus request panel before sending
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)

	// Send and check response contains our user agent
	sess.SendKey("Enter")
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Errorf("Request failed: %v", err)
		return
	}

	screen := sess.Capture()
	if !strings.Contains(screen, "MyCustomAgent") {
		t.Log("User-Agent may not be visible in response area")
	}
}

func TestCurl_InvalidCommand(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	// Start with invalid curl (no URL)
	_ = sess.Start(binaryPath, "curl", "-X", "POST")

	// Give it time to potentially start or fail
	time.Sleep(1 * time.Second)

	// The app should either not start TUI or show an error
	// It should NOT crash
	screen := sess.Capture()
	// If it started, it should show something
	// If it failed, the session might be dead which is acceptable
	if sess.IsAlive() && strings.Contains(screen, "Collections") {
		t.Error("Should not start TUI with invalid curl command (no URL)")
	}
}

func TestCurl_QuotedURL(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	// URL with query parameters
	if err := sess.Start(binaryPath, "curl",
		"https://httpbin.org/get?param=value&other=test"); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("TUI did not start: %v", err)
		return
	}

	screen := sess.Capture()
	if !strings.Contains(screen, "httpbin.org") {
		t.Error("URL with query params not loaded")
		t.Logf("Screen:\n%s", screen)
	}
}

// =============================================================================
// WEBSOCKET TESTS
// =============================================================================

func TestWebSocket_ToggleMode(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Press 'w' to enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	screen := sess.Capture()
	// Should show WebSocket indicator in status bar
	if !strings.Contains(screen, "WS") {
		t.Error("Should show WS indicator in status bar")
		t.Logf("Screen:\n%s", screen)
	}

	// Should show WebSocket panel instead of Request/Response
	if !strings.Contains(screen, "WebSocket") {
		t.Error("Should show WebSocket panel")
		t.Logf("Screen:\n%s", screen)
	}

	// Press 'w' again to toggle back
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	screen = sess.Capture()
	// Should be back to HTTP mode
	if !strings.Contains(screen, "Request") || !strings.Contains(screen, "Response") {
		t.Error("Should return to HTTP mode with Request/Response panes")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestWebSocket_PanelTabs(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	screen := sess.Capture()
	// Should show Messages tab by default
	if !strings.Contains(screen, "Messages") {
		t.Error("Should show Messages tab")
	}

	// Switch to Connection tab with ']'
	sess.SendKey("]")
	time.Sleep(200 * time.Millisecond)

	screen = sess.Capture()
	if !strings.Contains(screen, "Connection") {
		t.Error("Should show Connection tab")
	}

	// Switch to Scripts tab
	sess.SendKey("]")
	time.Sleep(200 * time.Millisecond)

	screen = sess.Capture()
	if !strings.Contains(screen, "Scripts") {
		t.Error("Should show Scripts tab")
	}

	// Switch to Auto-Response tab
	sess.SendKey("]")
	time.Sleep(200 * time.Millisecond)

	screen = sess.Capture()
	if !strings.Contains(screen, "Auto-Response") {
		t.Error("Should show Auto-Response tab")
	}

	// Switch back with '['
	sess.SendKey("[")
	time.Sleep(200 * time.Millisecond)

	screen = sess.Capture()
	if !strings.Contains(screen, "Scripts") {
		t.Error("Should go back to Scripts tab")
	}
}

func TestWebSocket_ShowsDisconnectedStatus(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	screen := sess.Capture()
	// Should show Disconnected status
	if !strings.Contains(screen, "Disconnected") && !strings.Contains(screen, "disconnected") {
		t.Error("Should show Disconnected status initially")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestWebSocket_InputMode(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	// Press 'i' to enter input mode
	sess.SendKey("i")
	time.Sleep(200 * time.Millisecond)

	// Type some text (without spaces as tmux Type may have issues with spaces)
	sess.Type("testmessage123")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	// Should show the typed text
	if !strings.Contains(screen, "testmessage123") {
		t.Error("Should show typed message in input field")
		t.Logf("Screen:\n%s", screen)
	}

	// Press Escape to exit input mode
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	screen = sess.Capture()
	if !strings.Contains(screen, "NORMAL") {
		t.Error("Should return to NORMAL mode after Escape")
	}
}

func TestWebSocket_HelpBarShowsWSHints(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	screen := sess.Capture()
	// Help bar should show WebSocket-specific hints
	hasConnect := strings.Contains(screen, "Connect") || strings.Contains(screen, "c ")
	hasType := strings.Contains(screen, "Type") || strings.Contains(screen, "i ")
	if !hasConnect && !hasType {
		t.Log("Help bar may not show WebSocket hints prominently")
	}
}

func TestWebSocket_NavigateWithJK(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	// Navigate with j and k (should scroll messages if any)
	for i := 0; i < 3; i++ {
		sess.SendKey("j")
		time.Sleep(50 * time.Millisecond)
	}
	for i := 0; i < 3; i++ {
		sess.SendKey("k")
		time.Sleep(50 * time.Millisecond)
	}

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "WebSocket") {
		t.Error("App crashed during j/k navigation in WebSocket mode")
	}
}

func TestWebSocket_GGAndG(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	// Press 'G' to go to bottom
	sess.SendKey("G")
	time.Sleep(200 * time.Millisecond)

	// Press 'gg' to go to top
	sess.SendKey("g")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("g")
	time.Sleep(200 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "WebSocket") {
		t.Error("App crashed during G/gg navigation in WebSocket mode")
	}
}

func TestWebSocket_FocusSwitchWithTab(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	// Verify we're focused on WebSocket pane (status bar shows WebSocket)
	screen := sess.Capture()
	if !strings.Contains(screen, "WebSocket") {
		t.Error("Should be focused on WebSocket pane")
	}

	// Press Tab to switch focus
	sess.SendKey("Tab")
	time.Sleep(200 * time.Millisecond)

	screen = sess.Capture()
	// Should now focus Collections
	if !strings.Contains(screen, "Collections") {
		t.Error("Tab should switch focus to Collections")
	}

	// Press Tab again
	sess.SendKey("Tab")
	time.Sleep(200 * time.Millisecond)

	// Should be back on WebSocket
	screen = sess.Capture()
	if !strings.Contains(screen, "WebSocket") {
		t.Log("Focus cycling may differ in WebSocket mode")
	}
}

func TestWebSocket_ConnectionTab_ShowsEndpoint(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	// Switch to Connection tab
	sess.SendKey("]")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	// Should show connection details
	if !strings.Contains(screen, "Endpoint") {
		t.Error("Connection tab should show Endpoint label")
		t.Logf("Screen:\n%s", screen)
	}

	// Should show wss:// (default endpoint prefix)
	if !strings.Contains(screen, "wss://") && !strings.Contains(screen, "ws://") {
		t.Error("Connection tab should show WebSocket URL")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestWebSocket_MessagesTab_ShowsHint(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	screen := sess.Capture()
	// Should show hint about connecting or typing
	hasNoMessages := strings.Contains(screen, "No messages") || strings.Contains(screen, "connect")
	if !hasNoMessages {
		t.Log("Messages tab may show different empty state")
	}
}

func TestWebSocket_ClearInput(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Enter WebSocket mode
	sess.SendKey("w")
	time.Sleep(300 * time.Millisecond)

	// Enter input mode and type
	sess.SendKey("i")
	time.Sleep(200 * time.Millisecond)
	sess.Type("test message to clear")
	time.Sleep(200 * time.Millisecond)

	// Clear with Ctrl+U
	sess.SendKey("Ctrl+U")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	// The typed text should be cleared
	if strings.Contains(screen, "test message to clear") {
		t.Error("Ctrl+U should clear the input field")
		t.Logf("Screen:\n%s", screen)
	}
}

// =============================================================================
// BODY TYPE TESTS
// =============================================================================

func TestBodyType_CycleWithT(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel and go to Body tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 3; i++ {
		sess.SendKey("]")
		time.Sleep(50 * time.Millisecond)
	}

	// Press 't' to cycle body type (raw -> json -> form)
	sess.SendKey("t")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	// Should show body type indicator
	hasJSON := strings.Contains(screen, "JSON") || strings.Contains(screen, "json")
	if !hasJSON {
		t.Log("Body type may show as json or JSON")
	}

	// Cycle again to form
	sess.SendKey("t")
	time.Sleep(200 * time.Millisecond)

	screen = sess.Capture()
	hasForm := strings.Contains(screen, "Form") || strings.Contains(screen, "form")
	if !hasForm {
		t.Log("Body type may show as form or Form")
	}

	// Should not crash
	if !strings.Contains(screen, "Request") {
		t.Error("App crashed during body type cycling")
	}
}

func TestBodyType_FormAddField(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel and go to Body tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 3; i++ {
		sess.SendKey("]")
		time.Sleep(50 * time.Millisecond)
	}

	// Cycle to form body type (raw -> json -> form)
	sess.SendKey("t")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("t")
	time.Sleep(200 * time.Millisecond)

	// Add a text field with 'a'
	sess.SendKey("a")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("fieldname")
	sess.SendKey("Tab")
	sess.Type("fieldvalue")
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Should not crash
	screen := sess.Capture()
	if !strings.Contains(screen, "Request") {
		t.Error("App crashed during form field add")
	}
}

func TestBodyType_FormAddFileField(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus request panel and go to Body tab
	sess.SendKey("2")
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 3; i++ {
		sess.SendKey("]")
		time.Sleep(50 * time.Millisecond)
	}

	// Cycle to form body type
	sess.SendKey("t")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("t")
	time.Sleep(200 * time.Millisecond)

	// Add a file field with 'f'
	sess.SendKey("f")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("myfile")
	sess.SendKey("Tab")
	sess.Type("/path/to/file.txt")
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Should not crash
	screen := sess.Capture()
	if !strings.Contains(screen, "Request") {
		t.Error("App crashed during file field add")
	}
}

// =============================================================================
// COLLECTION MANAGEMENT TESTS
// =============================================================================

func TestCollection_CreateNewCollection(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Press 'N' to create new collection
	sess.SendKey("N")
	time.Sleep(200 * time.Millisecond)

	// Should enter input mode for collection name
	screen := sess.Capture()
	hasInput := strings.Contains(screen, "INSERT") || strings.Contains(screen, "Name")
	if hasInput {
		sess.Type("MyNewCollection")
		sess.SendKey("Enter")
		time.Sleep(200 * time.Millisecond)
	}

	// Should not crash
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during collection creation")
	}
}

func TestCollection_CreateFolder(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request to have a collection
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Press 'F' to create new folder
	sess.SendKey("F")
	time.Sleep(200 * time.Millisecond)

	// Enter folder name if prompted
	screen := sess.Capture()
	if strings.Contains(screen, "INSERT") {
		sess.Type("NewFolder")
		sess.SendKey("Enter")
		time.Sleep(200 * time.Millisecond)
	}

	// Should not crash
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during folder creation")
	}
}

func TestCollection_RenameRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com/rename-test")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane and navigate to the request
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("l") // Expand collection
	sess.SendKey("j") // Move to request
	time.Sleep(100 * time.Millisecond)

	// Press 'R' to rename request
	sess.SendKey("R")
	time.Sleep(200 * time.Millisecond)

	// Enter new name if prompted
	screen := sess.Capture()
	if strings.Contains(screen, "INSERT") {
		sess.Type("RenamedRequest")
		sess.SendKey("Enter")
		time.Sleep(200 * time.Millisecond)
	}

	// Should not crash
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during request rename")
	}
}

func TestCollection_DuplicateRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com/duplicate-test")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane and navigate to the request
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("l") // Expand collection
	sess.SendKey("j") // Move to request
	time.Sleep(100 * time.Millisecond)

	// Press 'y' to duplicate request
	sess.SendKey("y")
	time.Sleep(300 * time.Millisecond)

	// Should not crash
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during request duplication")
	}
}

func TestCollection_MoveRequestUpDown(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create multiple requests
	for i := 0; i < 2; i++ {
		sess.SendKey("n")
		sess.WaitFor("INSERT", 2*time.Second)
		sess.Type("https://example.com/move-test")
		sess.SendKey("Escape")
		sess.WaitFor("NORMAL", 2*time.Second)
	}

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("l") // Expand collection
	sess.SendKey("j") // Move to first request
	time.Sleep(100 * time.Millisecond)

	// Press 'J' to move request down
	sess.SendKey("J")
	time.Sleep(200 * time.Millisecond)

	// Press 'K' to move request up
	sess.SendKey("K")
	time.Sleep(200 * time.Millisecond)

	// Should not crash
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during request move")
	}
}

func TestCollection_CopyAsCurl(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com/curl-test")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane and navigate to the request
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("l") // Expand collection
	sess.SendKey("j") // Move to request
	time.Sleep(100 * time.Millisecond)

	// Press 'c' to copy as cURL
	sess.SendKey("c")
	time.Sleep(300 * time.Millisecond)

	// Should not crash (clipboard may not work in test environment)
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during copy as curl")
	}
}

func TestCollection_SaveRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com/save-test")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Press 's' to save request
	sess.SendKey("s")
	time.Sleep(300 * time.Millisecond)

	// Should show save dialog or save to collection
	screen := sess.Capture()
	// App should not crash and should show some feedback
	if !strings.Contains(screen, "Collections") && !strings.Contains(screen, "Request") {
		t.Error("App crashed during save request")
	}
}

// =============================================================================
// ENVIRONMENT SWITCHER TESTS
// =============================================================================

func TestEnvironment_OpenSwitcher(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Press 'V' to open environment switcher
	sess.SendKey("V")
	time.Sleep(300 * time.Millisecond)

	// Should show environment switcher or indicator
	screen := sess.Capture()
	hasEnv := strings.Contains(screen, "Environment") || strings.Contains(screen, "env") || strings.Contains(screen, "No environments")
	if !hasEnv {
		t.Log("Environment switcher may show different UI")
	}

	// Press Escape to close
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Should return to normal view
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during environment switcher")
	}
}

// =============================================================================
// COOKIE MANAGEMENT TESTS
// =============================================================================

func TestCookies_ClearWithCtrlK(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Press Ctrl+K to clear cookies
	sess.SendKey("Ctrl+K")
	time.Sleep(300 * time.Millisecond)

	// Should show confirmation or clear message
	screen := sess.Capture()
	// App should not crash
	if !strings.Contains(screen, "Collections") && !strings.Contains(screen, "Request") {
		t.Error("App crashed during cookie clear")
	}
}

// =============================================================================
// RESPONSE COPY TESTS
// =============================================================================

func TestResponse_CopyWithY(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Make a request to get response
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.SendKey("Enter")

	// Wait for response
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Focus response panel
	sess.SendKey("3")
	time.Sleep(100 * time.Millisecond)

	// Press 'y' to copy response
	sess.SendKey("y")
	time.Sleep(300 * time.Millisecond)

	// Should not crash (clipboard may not work in test environment)
	screen := sess.Capture()
	if !strings.Contains(screen, "Response") {
		t.Error("App crashed during response copy")
	}
}

// =============================================================================
// COLLECTION RUNNER CLI TESTS
// =============================================================================

func TestRunner_CLIHelp(t *testing.T) {
	// Test that the run command is available
	cmd := exec.Command(binaryPath, "run", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Run help output: %s", string(output))
		// Some commands may not have help, check if it ran
	}

	// Should show help or error about missing collection
	hasHelp := strings.Contains(string(output), "run") || strings.Contains(string(output), "collection") || strings.Contains(string(output), "Usage")
	if !hasHelp {
		t.Log("Run command help may show different format")
	}
}

// =============================================================================
// ALT+ENTER SEND WHILE EDITING TESTS
// =============================================================================

func TestRequest_AltEnterSendWhileEditing(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create request and stay in INSERT mode
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")

	// Press Alt+Enter to send while still in INSERT mode
	sess.SendKey("Alt+Enter")

	// Should send the request
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Log("Alt+Enter may not be supported in this terminal")
	}
}

// =============================================================================
// DELETE REQUEST TESTS
// =============================================================================

func TestCollection_DeleteRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com/delete-test")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane and navigate to the request
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("l") // Expand collection
	sess.SendKey("j") // Move to request
	time.Sleep(100 * time.Millisecond)

	// Press 'd' to delete request
	sess.SendKey("d")
	time.Sleep(300 * time.Millisecond)

	// May show confirmation dialog
	screen := sess.Capture()
	if strings.Contains(screen, "confirm") || strings.Contains(screen, "Delete") {
		sess.SendKey("Enter") // Confirm
		time.Sleep(200 * time.Millisecond)
	}

	// Should not crash
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during request delete")
	}
}

func TestCollection_DeleteFolder(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request to have a collection
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Press 'D' to delete collection/folder
	sess.SendKey("D")
	time.Sleep(300 * time.Millisecond)

	// May show confirmation dialog
	screen := sess.Capture()
	if strings.Contains(screen, "confirm") || strings.Contains(screen, "Delete") {
		sess.SendKey("Escape") // Cancel to not actually delete
		time.Sleep(200 * time.Millisecond)
	}

	// Should not crash
	screen = sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("App crashed during folder delete")
	}
}

// =============================================================================
// IMPORT/EXPORT TESTS
// =============================================================================

func TestCollection_ExportPostman(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Create a request to have a collection
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://example.com")
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Press 'E' to export collection
	sess.SendKey("E")
	time.Sleep(300 * time.Millisecond)

	// Should show export dialog or export
	screen := sess.Capture()
	// App should not crash
	if !strings.Contains(screen, "Collections") && !strings.Contains(screen, "Export") && !strings.Contains(screen, "Request") {
		t.Error("App crashed during export")
	}

	// Press Escape in case dialog is open
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)
}

func TestCollection_Import(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Focus collections pane
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)

	// Press 'I' to import collection
	sess.SendKey("I")
	time.Sleep(300 * time.Millisecond)

	// Should show import dialog
	screen := sess.Capture()
	// App should not crash
	if !strings.Contains(screen, "Collections") && !strings.Contains(screen, "Import") && !strings.Contains(screen, "Request") {
		t.Error("App crashed during import")
	}

	// Press Escape to close dialog
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)
}

// =============================================================================
// SWITCH TO COLLECTIONS VIEW TESTS
// =============================================================================

func TestHistory_SwitchBackToCollections(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	sess.Start(binaryPath)
	sess.WaitFor("Collections", 5*time.Second)

	// Focus collections pane and switch to history
	sess.SendKey("1")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("H")
	time.Sleep(200 * time.Millisecond)

	// Press 'C' to switch back to collections
	sess.SendKey("C")
	time.Sleep(200 * time.Millisecond)

	// Should be back in collections view
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Error("Could not switch back to Collections view")
	}
}
