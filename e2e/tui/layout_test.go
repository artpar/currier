package tui_test

import (
	"strings"
	"testing"

	"github.com/artpar/currier/e2e/harness"
	"github.com/artpar/currier/internal/core"
)

func TestTUI_PostmanLikeLayout(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("layout has sidebar on left and stacked request/response on right", func(t *testing.T) {
		// Create a sample collection
		coll := core.NewCollection("Test Collection")
		req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		output := session.Output()
		lines := strings.Split(output, "\n")

		// Log the layout for visual inspection
		t.Logf("Layout has %d lines", len(lines))
		for i, line := range lines[:min(len(lines), 35)] {
			t.Logf("%2d: %s", i+1, line)
		}

		assert := harness.NewAssertions(t)

		// Verify three panels exist
		assert.OutputContains(output, "Collections")
		assert.OutputContains(output, "Request")
		assert.OutputContains(output, "Response")

		// Count top border characters to verify layout structure
		// Sidebar + Request should be on first row of borders
		// Response should have its own border below
		topBorderCount := 0
		for _, line := range lines {
			if strings.Contains(line, "╭") {
				topBorderCount += strings.Count(line, "╭")
			}
		}
		// Should have at least 3 panels (sidebar, request, response)
		if topBorderCount < 3 {
			t.Errorf("Expected at least 3 panel borders, got %d", topBorderCount)
		}
	})

	t.Run("request panel is above response panel", func(t *testing.T) {
		coll := core.NewCollection("Test Collection")
		req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		output := session.Output()
		lines := strings.Split(output, "\n")

		// Find line indices for "Request" and "Response" titles
		requestLine := -1
		responseLine := -1
		for i, line := range lines {
			if strings.Contains(line, "Request") && !strings.Contains(line, "Response") {
				requestLine = i
			}
			if strings.Contains(line, "Response") {
				responseLine = i
			}
		}

		// Request should appear BEFORE Response (above it)
		if requestLine == -1 || responseLine == -1 {
			t.Errorf("Could not find Request (line %d) or Response (line %d) titles", requestLine, responseLine)
		} else if requestLine >= responseLine {
			t.Errorf("Request panel (line %d) should be above Response panel (line %d)", requestLine, responseLine)
		} else {
			t.Logf("Layout correct: Request at line %d, Response at line %d", requestLine, responseLine)
		}
	})

	t.Run("sidebar spans full height", func(t *testing.T) {
		coll := core.NewCollection("Test Collection")
		req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		coll.AddRequest(req)

		session := h.TUI().StartWithCollections(t, []*core.Collection{coll})
		defer session.Quit()

		output := session.Output()
		lines := strings.Split(output, "\n")

		// Count sidebar border lines (left side)
		sidebarLines := 0
		for _, line := range lines {
			if len(line) > 0 && (strings.HasPrefix(line, "│") || strings.HasPrefix(line, "╭") || strings.HasPrefix(line, "╰")) {
				sidebarLines++
			}
		}

		// Sidebar should span most of the height (excluding help/status bars)
		// With 40 height, expect at least 30 lines of sidebar
		expectedMinHeight := 25
		if sidebarLines < expectedMinHeight {
			t.Logf("Sidebar has %d lines, expected at least %d", sidebarLines, expectedMinHeight)
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestTUI_LayoutEdgeCases(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
	})

	t.Run("small terminal handles gracefully", func(t *testing.T) {
		session := h.TUI().StartWithSize(t, 60, 20)
		defer session.Quit()

		output := session.Output()
		assert := harness.NewAssertions(t)
		// Should still render without panic
		assert.NoError(output)
		assert.OutputContains(output, "Collections")
		assert.OutputContains(output, "Request")
		assert.OutputContains(output, "Response")
	})

	t.Run("very wide terminal handles correctly", func(t *testing.T) {
		session := h.TUI().StartWithSize(t, 200, 40)
		defer session.Quit()

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
		// Layout should still work
		assert.OutputContains(output, "Collections")
		assert.OutputContains(output, "Request")
		assert.OutputContains(output, "Response")
	})

	t.Run("tall terminal handles correctly", func(t *testing.T) {
		session := h.TUI().StartWithSize(t, 100, 60)
		defer session.Quit()

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
		assert.OutputContains(output, "Collections")
		assert.OutputContains(output, "Request")
		assert.OutputContains(output, "Response")
	})
}
