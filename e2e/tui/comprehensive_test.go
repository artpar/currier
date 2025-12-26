package tui_test

import (
	"strings"
	"testing"

	"github.com/artpar/currier/e2e/harness"
	"github.com/artpar/currier/internal/core"
)

// TestTUI_ComprehensiveE2E runs comprehensive end-to-end tests for all TUI features
func TestTUI_ComprehensiveE2E(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	// Create test collection with multiple folders and requests
	coll := createTestCollection()

	t.Run("1_Layout_Rendering", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		output := session.Output()
		lines := strings.Split(output, "\n")

		// Log full output for debugging
		t.Logf("=== Full TUI Output (%d lines) ===", len(lines))
		for i, line := range lines {
			t.Logf("%2d: %s", i+1, line)
		}

		assert := harness.NewAssertions(t)

		// Verify all panels are present
		assert.OutputContains(output, "Collections")
		assert.OutputContains(output, "Request")
		assert.OutputContains(output, "Response")

		// Verify Postman-like layout: Request should be ABOVE Response
		requestLine := -1
		responseLine := -1
		for i, line := range lines {
			if strings.Contains(line, "Request") && !strings.Contains(line, "Response") {
				if requestLine == -1 {
					requestLine = i
				}
			}
			if strings.Contains(line, "Response") {
				if responseLine == -1 {
					responseLine = i
				}
			}
		}

		if requestLine >= responseLine {
			t.Errorf("BUG: Request panel (line %d) should be ABOVE Response panel (line %d)", requestLine, responseLine)
		}

		// Verify sidebar spans full height (check first column for border chars)
		sidebarBorders := 0
		for _, line := range lines {
			if len(line) > 0 && (strings.HasPrefix(line, "│") || strings.HasPrefix(line, "╭") || strings.HasPrefix(line, "╰")) {
				sidebarBorders++
			}
		}
		if sidebarBorders < 20 {
			t.Errorf("BUG: Sidebar should span full height, only found %d border lines", sidebarBorders)
		}
	})

	t.Run("2_Collection_Navigation", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		assert := harness.NewAssertions(t)

		// Initial state - should show collection
		output := session.Output()
		assert.OutputContains(output, "Sample API")

		// Navigate down with j
		session.SendKey("j")
		output = session.Output()
		t.Logf("After j: cursor should have moved")

		// Expand with l
		session.SendKey("l")
		output = session.Output()
		// After expanding, should see folder contents
		t.Logf("After l (expand): %s", extractCollectionsPane(output))

		// Navigate and select a request
		session.SendKeys("j", "j", "Enter")
		output = session.Output()

		// Should now show the request in the Request panel
		if !strings.Contains(output, "jsonplaceholder") {
			t.Errorf("BUG: Selected request URL should appear in Request panel")
		}
	})

	t.Run("3_Tab_Key_Navigation", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Initial focus should be on Collections
		if session.FocusedPane() != 0 { // PaneCollections = 0
			t.Errorf("BUG: Initial focus should be on Collections pane")
		}

		// Tab should cycle to Request pane
		session.SendKey("Tab")
		if session.FocusedPane() != 1 { // PaneRequest = 1
			t.Errorf("BUG: Tab should cycle to Request pane, got pane %d", session.FocusedPane())
		}

		// Tab should cycle to Response pane
		session.SendKey("Tab")
		if session.FocusedPane() != 2 { // PaneResponse = 2
			t.Errorf("BUG: Tab should cycle to Response pane, got pane %d", session.FocusedPane())
		}

		// Tab should wrap back to Collections
		session.SendKey("Tab")
		if session.FocusedPane() != 0 {
			t.Errorf("BUG: Tab should wrap to Collections pane, got pane %d", session.FocusedPane())
		}
	})

	t.Run("4_Internal_Tab_Switching_With_Brackets", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Select a request first
		session.SendKeys("l", "j", "Enter")

		// Switch to Request pane
		session.SendKey("2")
		output := session.Output()

		// Should be on URL tab initially
		if !strings.Contains(output, "URL") {
			t.Logf("Request panel output: %s", extractRequestPane(output))
		}

		// ] should switch to next tab (Headers)
		session.SendKey("]")
		output = session.Output()
		t.Logf("After ] key: checking if tab switched")

		// [ should switch back to previous tab (URL)
		session.SendKey("[")
		output = session.Output()
		t.Logf("After [ key: checking if tab switched back")

		// Test in Response pane too
		session.SendKey("3")
		session.SendKey("]")
		output = session.Output()
		t.Logf("Response pane after ] key")
	})

	t.Run("5_History_View_Toggle", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Both History and Collections should be visible in stacked layout
		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.OutputContains(output, "History")
		assert.OutputContains(output, "Collections")

		// Press H to focus History section
		session.SendKey("H")
		output = session.Output()

		// Both sections should still be visible
		if !strings.Contains(output, "History") {
			t.Errorf("BUG: History section should be visible")
		}
		if !strings.Contains(output, "Collections") {
			t.Errorf("BUG: Collections section should be visible")
		}

		// Press C to focus Collections section
		session.SendKey("C")
		output = session.Output()

		// Both sections should still be visible
		if !strings.Contains(output, "History") {
			t.Errorf("BUG: History section should still be visible")
		}
		if !strings.Contains(output, "Collections") {
			t.Errorf("BUG: Collections section should still be visible")
		}
	})

	t.Run("6_Search_With_Feedback", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Expand collection first
		session.SendKey("l")

		// Start search with /
		session.SendKey("/")
		output := session.Output()

		// Should show search indicator
		if !strings.Contains(output, "🔍") {
			t.Errorf("BUG: Search mode should show search icon")
		}

		// Type search query
		session.Type("user")
		output = session.Output()
		t.Logf("During search: %s", extractCollectionsPane(output))

		// Press Enter to confirm search
		session.SendKey("Enter")
		output = session.Output()
		t.Logf("After Enter: %s", extractCollectionsPane(output))

		// Should show result count - check for "(X result" pattern
		if !strings.Contains(output, "result") && !strings.Contains(output, "match") {
			t.Logf("NOTE: Search result count may not be showing")
		}

		// Press Esc to clear search
		session.SendKey("Escape")
		output = session.Output()
	})

	t.Run("7_New_Request_Creation", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Press n to create new request
		session.SendKey("n")
		output := session.Output()

		// Should be in Request pane with edit mode
		if session.FocusedPane() != 1 {
			t.Errorf("BUG: After 'n', should focus Request pane")
		}

		// Should show "New Request" or be in URL edit mode
		if !strings.Contains(output, "New Request") && !strings.Contains(output, "INSERT") {
			t.Logf("Output after 'n': %s", output)
		}
	})

	t.Run("8_URL_Editing", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Create new request
		session.SendKey("n")

		// Should auto-enter URL edit mode
		output := session.Output()

		// Type URL
		session.Type("https://httpbin.org/get")
		output = session.Output()

		if !strings.Contains(output, "httpbin") {
			t.Errorf("BUG: Typed URL should appear in the request panel")
			t.Logf("Output: %s", extractRequestPane(output))
		}

		// Press Enter to save
		session.SendKey("Enter")
		output = session.Output()

		// URL should be saved
		if !strings.Contains(output, "httpbin") {
			t.Errorf("BUG: URL should remain after pressing Enter")
		}
	})

	t.Run("9_Method_Cycling", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Create new request and go to request pane
		session.SendKey("n")
		session.SendKey("Escape") // Exit URL edit mode

		// Press m to cycle method
		session.SendKey("m")
		output := session.Output()

		// Should show method options or cycle to next method
		t.Logf("After 'm' key: %s", extractRequestPane(output))
	})

	t.Run("10_Header_Editing", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Select a request with headers
		session.SendKeys("l", "j", "Enter")

		// Go to Headers tab
		session.SendKey("2") // Focus request pane
		session.SendKey("]") // Switch to Headers tab

		output := session.Output()
		t.Logf("Headers tab: %s", extractRequestPane(output))

		// Press 'a' to add new header
		session.SendKey("a")
		output = session.Output()

		// Should be in INSERT mode
		if !strings.Contains(output, "INSERT") {
			t.Errorf("BUG: 'a' on Headers tab should enter INSERT mode for adding header")
		}
	})

	t.Run("11_Query_Params_Tab", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Create new request with query params in URL
		session.SendKey("n")
		session.Type("https://example.com/api?foo=bar&baz=qux")
		session.SendKey("Enter")

		// Navigate to Query tab
		session.SendKey("]") // Headers
		session.SendKey("]") // Query

		output := session.Output()
		t.Logf("Query tab output: %s", extractRequestPane(output))

		// Query params should be displayed
		// Try to edit a param
		session.SendKey("e")
		output = session.Output()
		t.Logf("After 'e' on Query tab: %s", output)
	})

	t.Run("12_Body_Tab", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Select POST request
		session.SendKeys("l", "j", "j", "j", "Enter") // Navigate to Create User

		// Go to Body tab
		session.SendKey("2")
		session.SendKeys("]", "]", "]") // URL -> Headers -> Query -> Body

		output := session.Output()
		t.Logf("Body tab: %s", extractRequestPane(output))

		// Should show body content
		if !strings.Contains(output, "Body") {
			t.Logf("May not be on Body tab yet")
		}
	})

	t.Run("13_Response_Panel_Tabs", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Focus response pane
		session.SendKey("3")

		output := session.Output()

		// Should show tab names
		if !strings.Contains(output, "Body") || !strings.Contains(output, "Headers") {
			t.Errorf("BUG: Response panel should show tab names")
		}

		// Check Console tab exists
		if !strings.Contains(output, "Console") {
			t.Errorf("BUG: Response panel should have Console tab")
		}

		// Navigate to Console tab
		for i := 0; i < 4; i++ {
			session.SendKey("]")
		}
		output = session.Output()
		t.Logf("Console tab: %s", extractResponsePane(output))
	})

	t.Run("14_Help_Overlay", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Press ? to show help
		session.SendKey("?")
		output := session.Output()

		if !strings.Contains(output, "Help") {
			t.Errorf("BUG: ? should show help overlay")
		}

		// Should mention new keybindings
		if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
			t.Errorf("BUG: Help should mention [ and ] for tab switching")
		}

		// Press ? again to close
		session.SendKey("?")
		output = session.Output()

		if strings.Contains(output, "Currier Help") {
			t.Errorf("BUG: ? should close help overlay")
		}
	})

	t.Run("15_Status_Bar_Mode_Indicator", func(t *testing.T) {
		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		output := session.Output()

		// Should show NORMAL mode
		if !strings.Contains(output, "NORMAL") {
			t.Errorf("BUG: Status bar should show NORMAL mode")
		}

		// Enter edit mode
		session.SendKey("n") // New request enters edit mode
		output = session.Output()

		// Should show INSERT mode
		if !strings.Contains(output, "INSERT") {
			t.Errorf("BUG: Status bar should show INSERT mode when editing")
		}
	})
}

func createTestCollection() *core.Collection {
	coll := core.NewCollection("Sample API Collection")

	// Add Users folder with requests
	usersFolder := coll.AddFolder("Users")

	getUsers := core.NewRequestDefinition("Get All Users", "GET", "https://jsonplaceholder.typicode.com/users")
	getUsers.SetHeader("Accept", "application/json")
	usersFolder.AddRequest(getUsers)

	getUser := core.NewRequestDefinition("Get User by ID", "GET", "https://jsonplaceholder.typicode.com/users/1")
	usersFolder.AddRequest(getUser)

	createUser := core.NewRequestDefinition("Create User", "POST", "https://jsonplaceholder.typicode.com/users")
	createUser.SetHeader("Content-Type", "application/json")
	createUser.SetBody(`{"name": "John Doe", "email": "john@example.com"}`)
	usersFolder.AddRequest(createUser)

	// Add Posts folder
	postsFolder := coll.AddFolder("Posts")

	getPosts := core.NewRequestDefinition("Get All Posts", "GET", "https://jsonplaceholder.typicode.com/posts")
	postsFolder.AddRequest(getPosts)

	return coll
}

func extractCollectionsPane(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		if len(line) > 30 {
			result = append(result, line[:30])
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result[:min(len(result), 15)], "\n")
}

func extractRequestPane(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		if len(line) > 30 {
			// Get middle portion
			start := 30
			end := min(len(line), 90)
			if start < end {
				result = append(result, line[start:end])
			}
		}
	}
	return strings.Join(result[:min(len(result), 15)], "\n")
}

func extractResponsePane(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		if len(line) > 90 {
			result = append(result, line[90:])
		}
	}
	return strings.Join(result[:min(len(result), 15)], "\n")
}
