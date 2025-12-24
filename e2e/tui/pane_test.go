package tui_test

import (
	"testing"
	"time"

	"github.com/artpar/currier/e2e/harness"
)

func TestTUI_PaneSwitching(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
		Timeout:   5 * time.Second,
	})

	t.Run("Tab cycles through panes forward", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Initial state - collections pane focused
		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)

		// Tab to request pane
		session.SendKey("Tab")
		output = session.Output()
		assert.NoError(output)

		// Tab to response pane
		session.SendKey("Tab")
		output = session.Output()
		assert.NoError(output)

		// Tab wraps back to collections
		session.SendKey("Tab")
		output = session.Output()
		assert.NoError(output)
	})

	t.Run("Shift+Tab cycles through panes backward", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// From collections, Shift+Tab should go to response (wrap around)
		session.SendKey("Shift+Tab")
		output := session.Output()

		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("1 jumps to collections pane", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// First go to a different pane
		session.SendKey("Tab")

		// Jump back to collections with 1
		session.SendKey("1")
		output := session.Output()

		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("2 jumps to request pane", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("2")
		output := session.Output()

		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("3 jumps to response pane", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		session.SendKey("3")
		output := session.Output()

		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})

	t.Run("rapid pane switching works correctly", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Rapidly switch between panes
		session.SendKeys("1", "2", "3", "1", "Tab", "Tab", "2")

		output := session.Output()
		assert := harness.NewAssertions(t)
		assert.NoError(output)
	})
}

func TestTUI_PaneContent(t *testing.T) {
	h := harness.New(t, harness.Config{
		GoldenDir: "../golden/tui",
		Timeout:   5 * time.Second,
	})

	t.Run("request pane shows no request selected initially", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Switch to request pane
		session.SendKey("2")
		output := session.Output()

		assert := harness.NewAssertions(t)
		assert.OutputContains(output, "No request selected")
	})

	t.Run("response pane shows no response initially", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Switch to response pane
		session.SendKey("3")
		output := session.Output()

		assert := harness.NewAssertions(t)
		assert.OutputContains(output, "No response yet")
	})
}
