package tui_test

import (
	"testing"
	"time"

	"github.com/artpar/currier/e2e/harness"
)

func TestTUI_HelpOverlay(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
		Timeout:   5 * time.Second,
	})

	t.Run("? opens help overlay", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Press ?
		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.HelpVisible(output)
		assert.OutputContains(output, "Tab / Shift+Tab", "j / k")
	})

	t.Run("? again closes help overlay", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Open help
		session.SendKey("?")

		// Close with ? again
		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.HelpNotVisible(output)
	})

	t.Run("Escape closes help overlay", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Open help
		session.SendKey("?")

		// Close with Escape
		session.SendKey("Escape")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.HelpNotVisible(output)
	})

	t.Run("help overlay contains navigation instructions", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)

		// Check for key navigation instructions
		assert.OutputContains(output,
			"Navigation",
			"Tab",
			"1 / 2 / 3",
			"j / k",
		)
	})

	t.Run("help overlay contains action instructions", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)

		// Check for action instructions (Request/Response sections)
		assert.OutputContains(output,
			"Request",
			"Enter",
		)
	})

	t.Run("help overlay contains quit instructions", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)

		// Check for quit instructions
		assert.OutputContains(output,
			"q",
			"Ctrl+C",
		)
	})
}

func TestTUI_HelpOverlayFromDifferentPanes(t *testing.T) {
	h := harness.New(t, harness.Config{
		Timeout: 5 * time.Second,
	})

	t.Run("help works from collections pane", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("1") // Ensure we're on collections
		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.HelpVisible(output)
	})

	t.Run("help works from request pane", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("2") // Switch to request pane
		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.HelpVisible(output)
	})

	t.Run("help works from response pane", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("3") // Switch to response pane
		session.SendKey("?")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.HelpVisible(output)
	})
}
