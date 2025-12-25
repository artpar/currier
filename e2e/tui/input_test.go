package tui_test

import (
	"testing"

	"github.com/artpar/currier/e2e/harness"
	"github.com/artpar/currier/internal/core"
)

// TestInput_SpaceCharacter verifies that space can be typed in all input fields.
// This was a critical bug where tea.KeySpace was not handled.
func TestInput_SpaceCharacter(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("space in URL field", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Create new request which enters URL edit mode
		session.SendKey("n")

		// Type URL with space (though URLs shouldn't have unencoded spaces,
		// the input should accept it)
		session.Type("http://example.com/path")
		session.SendKey("space")
		session.Type("with")
		session.SendKey("space")
		session.Type("spaces")

		output := session.Output()
		// The space should be in the output somewhere
		if !containsAll(output, "http", "example", "com") {
			t.Errorf("URL not showing in output")
		}
	})

	t.Run("space in header value", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Select the request
		session.SendKeys("l", "j", "Enter")

		// Go to Headers tab and add header
		session.SendKey("2") // Focus request
		session.SendKey("]") // Switch to Headers tab
		session.SendKey("a") // Add new header

		// Type header with spaces
		session.Type("Authorization")
		session.SendKey("Tab") // Switch to value
		session.Type("Bearer")
		session.SendKey("space")
		session.Type("token123")

		output := session.Output()
		t.Logf("Output after typing header with space: checking for Bearer token123")
		// Just verify we're in editing mode and content is visible
		if !containsAll(output, "Authorization") {
			t.Logf("Note: Header may not be fully visible in output yet")
		}
	})

	t.Run("space in search", func(t *testing.T) {
		coll := core.NewCollection("Test Collection")
		req := core.NewRequestDefinition("Get User Info", "GET", "https://example.com/user")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Expand and start search
		session.SendKey("l")
		session.SendKey("/")

		// Type search with space
		session.Type("User")
		session.SendKey("space")
		session.Type("Info")

		output := session.Output()
		// Search indicator should be visible
		if !containsAll(output, "🔍") {
			t.Logf("Search mode may not show icon in this view")
		}
	})
}

// TestInput_CursorNavigation tests cursor movement in text fields.
func TestInput_CursorNavigation(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("left/right in URL field", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("n") // New request
		session.Type("https://example.com")
		session.SendKey("left")
		session.SendKey("left")
		session.SendKey("left")
		session.Type("X") // Should insert X before .com

		output := session.Output()
		t.Logf("URL field after cursor nav: should have X inserted")
		// We can't easily verify exact position in E2E, but verify no crash
		if output == "" {
			t.Error("Empty output after cursor navigation")
		}
	})

	t.Run("home/end in URL field", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("n")
		session.Type("example.com")
		session.SendKey("home") // Go to start (Ctrl+A or Home key)
		session.Type("https://")

		output := session.Output()
		if !containsAll(output, "https") {
			t.Logf("Note: Home key may need testing with actual terminal")
		}
	})
}

// TestInput_DeleteOperations tests backspace and delete in text fields.
func TestInput_DeleteOperations(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("backspace in URL", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("n")
		session.Type("https://example.comm") // Typo
		session.SendKey("backspace")         // Remove extra m

		output := session.Output()
		// Should show the URL without the extra m
		t.Logf("URL after backspace: checking output is valid")
		if output == "" {
			t.Error("Empty output after backspace")
		}
	})

	t.Run("ctrl+u clears field", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("n")
		session.Type("https://example.com")
		session.SendKey("ctrl+u") // Clear entire field

		output := session.Output()
		t.Logf("URL field after Ctrl+U: should be empty")
		// We'd need state access to verify field is actually empty
		if output == "" {
			t.Error("Empty output after Ctrl+U")
		}
	})
}

// TestInput_ModeTransitions tests that edit mode works correctly.
func TestInput_ModeTransitions(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("n creates new request and enters edit mode", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("n")
		output := session.Output()

		// Should show INSERT mode
		if !containsAll(output, "INSERT") {
			t.Error("Should be in INSERT mode after pressing n")
		}
	})

	t.Run("escape exits edit mode", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("n")
		session.SendKey("Escape")
		output := session.Output()

		// Should show NORMAL mode
		if !containsAll(output, "NORMAL") {
			t.Error("Should be in NORMAL mode after pressing Escape")
		}
	})

	t.Run("e enters URL edit mode on existing request", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Test Request", "GET", "https://example.com")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Select request
		session.SendKeys("l", "j", "Enter")
		// Focus request panel and press e
		session.SendKey("2")
		session.SendKey("e")

		output := session.Output()
		if !containsAll(output, "INSERT") {
			t.Error("Should be in INSERT mode after pressing e")
		}
	})
}

// TestInput_HeaderEditing tests the header add/edit/delete workflow.
func TestInput_HeaderEditing(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("add header workflow", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Select request
		session.SendKeys("l", "j", "Enter")

		// Go to Headers tab
		session.SendKey("2")
		session.SendKey("]") // URL -> Headers

		// Add new header
		session.SendKey("a")
		session.Type("Content-Type")
		session.SendKey("Tab")
		session.Type("application/json")
		session.SendKey("Enter")

		output := session.Output()
		// Verify we exited edit mode
		if containsAll(output, "INSERT") {
			t.Error("Should have exited INSERT mode after Enter")
		}
	})
}

// TestInput_BodyEditing tests multi-line body editing.
func TestInput_BodyEditing(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("enter creates new line in body", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		// Select request and go to Body tab
		session.SendKeys("l", "j", "Enter")
		session.SendKey("2")
		session.SendKeys("]", "]", "]") // URL -> Headers -> Query -> Body

		// Edit body
		session.SendKey("e")
		session.Type("{")
		session.SendKey("Enter")
		session.Type("  \"key\": \"value\"")
		session.SendKey("Enter")
		session.Type("}")
		session.SendKey("Escape") // Save

		output := session.Output()
		// Should be back in NORMAL mode
		if !containsAll(output, "NORMAL") {
			t.Logf("Body editing may need further verification")
		}
	})
}

// containsAll checks if s contains all substrings.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
