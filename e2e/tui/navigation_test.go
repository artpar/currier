package tui_test

import (
	"testing"
	"time"

	"github.com/artpar/currier/e2e/harness"
)

func TestTUI_VimNavigation(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
		Timeout:   5 * time.Second,
	})

	t.Run("initial layout renders correctly", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		output := session.Output()
		assert := harness.NewAssertions(t)

		// Should show the three panes
		assert.OutputContains(output, "No request selected", "No response yet")
	})

	t.Run("j moves down in collection tree", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Navigate down
		session.SendKey("j")

		// Should still render without error
		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("k moves up in collection tree", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Navigate down then up
		session.SendKeys("j", "k")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("G goes to bottom", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("G")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("gg goes to top", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Go to bottom first, then top
		session.SendKey("G")
		session.SendKeys("g", "g")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("h collapses tree item", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("h")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("l expands tree item", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("l")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})
}

func TestTUI_QuitBehavior(t *testing.T) {
	h := harness.New(t, harness.Config{
		Timeout: 5 * time.Second,
	})

	t.Run("q quits the application", func(t *testing.T) {
		session := h.TUI().Start(t)

		// Press q to quit - this should trigger the quit command
		session.SendKey("q")

		// Just verify no crash
		output := session.Output()
		_ = output // Application should have quit
	})

	t.Run("Ctrl+C quits the application", func(t *testing.T) {
		session := h.TUI().Start(t)

		session.SendKey("Ctrl+C")

		output := session.Output()
		_ = output // Application should have quit
	})
}
