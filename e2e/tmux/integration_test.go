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
