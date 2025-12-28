package journeys

import (
	"testing"

	"github.com/artpar/currier/e2e/harness"
)

// TestJourney_CreateNewRequest tests the complete flow of creating a new request.
func TestJourney_CreateNewRequest(t *testing.T) {
	harness.NewJourney(t, "Create new request").
		Step("Press n to create new request").
			SendKey("n").
			ExpectMode("INSERT").
			ExpectFocus("request").
			ExpectIsEditing(true).
			ExpectEditingField("url").

		Step("Type URL").
			Type("https://example.com/api").
			ExpectURL("https://example.com/api").

		Step("Exit edit mode").
			SendKey("Escape").
			ExpectMode("NORMAL").
			ExpectIsEditing(false).

		Run()
}

// TestJourney_CreateRequestWithSpaces tests URL input with spaces.
func TestJourney_CreateRequestWithSpaces(t *testing.T) {
	harness.NewJourney(t, "URL with spaces").
		Step("Create new request").
			SendKey("n").
			ExpectMode("INSERT").

		Step("Type URL with spaces").
			Type("https://example.com/search?q=hello").
			SendKey("space").
			Type("world").
			ExpectState(func(t *testing.T, s *harness.State) {
				// URL should contain the space
				if s.Request.URL != "https://example.com/search?q=hello world" {
					t.Errorf("URL should contain space: got %q", s.Request.URL)
				}
			}).

		Run()
}

// TestJourney_ChangeMethod tests changing the HTTP method.
func TestJourney_ChangeMethod(t *testing.T) {
	harness.NewJourney(t, "Change HTTP method").
		Step("Create new request").
			SendKey("n").
			Type("https://example.com").
			SendKey("Escape").
			ExpectMethod("GET").

		Step("Cycle method with m (GET -> POST)").
			SendKey("m").
			ExpectMethod("POST").
			ExpectEditingField(""). // Method cycling is inline, no edit mode

		Step("Cycle method again (POST -> PUT)").
			SendKey("m").
			ExpectMethod("PUT").

		Run()
}

// TestJourney_NavigatePanes tests pane navigation.
func TestJourney_NavigatePanes(t *testing.T) {
	harness.NewJourney(t, "Pane navigation").
		Step("Start at collections pane").
			ExpectFocus("collections").

		Step("Press Tab to go to request pane").
			SendKey("Tab").
			ExpectFocus("request").

		Step("Press Tab to go to response pane").
			SendKey("Tab").
			ExpectFocus("response").

		Step("Press Tab to wrap back to collections").
			SendKey("Tab").
			ExpectFocus("collections").

		Step("Press 2 to jump to request pane").
			SendKey("2").
			ExpectFocus("request").

		Step("Press 1 to jump to collections pane").
			SendKey("1").
			ExpectFocus("collections").

		Run()
}

// TestJourney_SwitchTabs tests switching tabs in request panel.
func TestJourney_SwitchTabs(t *testing.T) {
	harness.NewJourney(t, "Tab switching").
		Step("Create request and focus request pane").
			SendKey("n").
			Type("https://example.com").
			SendKey("Escape").
			ExpectActiveTab("URL").

		Step("Switch to Headers tab").
			SendKey("]").
			ExpectActiveTab("Headers").

		Step("Switch to Query tab").
			SendKey("]").
			ExpectActiveTab("Query").

		Step("Switch to Body tab").
			SendKey("]").
			ExpectActiveTab("Body").

		Step("Switch back to URL tab").
			SendKey("[").
			SendKey("[").
			SendKey("[").
			ExpectActiveTab("URL").

		Run()
}

// TestJourney_AddHeader tests adding a header to a request.
func TestJourney_AddHeader(t *testing.T) {
	harness.NewJourney(t, "Add header").
		Step("Create request").
			SendKey("n").
			Type("https://example.com").
			SendKey("Escape").

		Step("Switch to Headers tab").
			SendKey("]").
			ExpectActiveTab("Headers").

		Step("Add new header").
			SendKey("a").
			ExpectMode("INSERT").
			ExpectEditingField("header_key").

		Step("Type header key").
			Type("Authorization").
			SendKey("Tab").
			ExpectEditingField("header_value").

		Step("Type header value").
			Type("Bearer").
			SendKey("space").
			Type("token123").
			SendKey("Enter").
			ExpectMode("NORMAL").

		Step("Verify header was added").
			ExpectState(func(t *testing.T, s *harness.State) {
				if s.Request.Headers == nil {
					t.Error("Headers should not be nil")
					return
				}
				if s.Request.Headers["Authorization"] != "Bearer token123" {
					t.Errorf("Authorization header should be 'Bearer token123', got %q",
						s.Request.Headers["Authorization"])
				}
			}).

		Run()
}

// TestJourney_EmptyURLError tests that empty URL shows appropriate UI feedback.
// Note: The actual error is returned asynchronously via tea.Cmd, but we can verify
// that the request panel doesn't have a valid URL.
func TestJourney_EmptyURLError(t *testing.T) {
	harness.NewJourney(t, "Empty URL validation").
		Step("Create empty request").
			SendKey("n").
			SendKey("Escape").
			ExpectURL("").

		Step("Verify no request can be sent with empty URL").
			ExpectState(func(t *testing.T, s *harness.State) {
				// The URL should be empty
				if s.Request.URL != "" {
					t.Errorf("Expected empty URL, got %q", s.Request.URL)
				}
				// Request should exist
				if !s.Request.HasRequest {
					t.Error("Request should exist even if URL is empty")
				}
			}).

		Run()
}

// TestJourney_HelpOverlay tests help overlay display.
func TestJourney_HelpOverlay(t *testing.T) {
	harness.NewJourney(t, "Help overlay").
		Step("Initial state - no help").
			ExpectState(func(t *testing.T, s *harness.State) {
				if s.MainView.ShowingHelp {
					t.Error("Help should not be showing initially")
				}
			}).

		Step("Press ? to show help").
			SendKey("?").
			ExpectState(func(t *testing.T, s *harness.State) {
				if !s.MainView.ShowingHelp {
					t.Error("Help should be showing after pressing ?")
				}
			}).

		Step("Press Escape to hide help").
			SendKey("Escape").
			ExpectState(func(t *testing.T, s *harness.State) {
				if s.MainView.ShowingHelp {
					t.Error("Help should be hidden after pressing Escape")
				}
			}).

		Run()
}

// TestJourney_HistoryView tests switching between history and collections view.
func TestJourney_HistoryView(t *testing.T) {
	harness.NewJourney(t, "History view").
		Step("Start in history view (default)").
			ExpectViewMode("history").

		Step("Switch to collections view").
			SendKey("C").
			ExpectViewMode("collections").

		Step("Switch back to history").
			SendKey("H").
			ExpectViewMode("history").

		Run()
}

// TestJourney_SearchCollections tests search functionality.
func TestJourney_SearchCollections(t *testing.T) {
	harness.NewJourney(t, "Search collections").
		Step("Switch to collections mode (search only works in collections)").
			SendKey("C").
			ExpectViewMode("collections").

		Step("Start search").
			SendKey("/").
			ExpectState(func(t *testing.T, s *harness.State) {
				if !s.Tree.IsSearching {
					t.Error("Should be in search mode")
				}
			}).

		Step("Type search query").
			Type("test").
			ExpectState(func(t *testing.T, s *harness.State) {
				if s.Tree.SearchQuery != "test" {
					t.Errorf("Search query should be 'test', got %q", s.Tree.SearchQuery)
				}
			}).

		Step("Clear search with Escape").
			SendKey("Escape").
			ExpectState(func(t *testing.T, s *harness.State) {
				if s.Tree.IsSearching {
					t.Error("Should not be in search mode after Escape")
				}
			}).

		Run()
}
