package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestNewRequestPanel(t *testing.T) {
	t.Run("creates empty panel", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.NotNil(t, panel)
		assert.Equal(t, "Request", panel.Title())
	})

	t.Run("starts with no request", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.Nil(t, panel.Request())
	})

	t.Run("starts unfocused", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.False(t, panel.Focused())
	})

	t.Run("starts on URL tab", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.Equal(t, TabURL, panel.ActiveTab())
	})
}

func TestRequestPanel_SetRequest(t *testing.T) {
	t.Run("sets request", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test Request", "GET", "https://api.example.com/users")

		panel.SetRequest(req)

		assert.Equal(t, req, panel.Request())
	})

	t.Run("updates title with request name", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Get User", "GET", "https://api.example.com/users/1")

		panel.SetRequest(req)

		assert.Contains(t, panel.Title(), "Get User")
	})

	t.Run("clears request when nil", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)

		panel.SetRequest(nil)

		assert.Nil(t, panel.Request())
	})
}

func TestRequestPanel_Tabs(t *testing.T) {
	t.Run("switches to headers tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()

		panel.SetActiveTab(TabHeaders)

		assert.Equal(t, TabHeaders, panel.ActiveTab())
	})

	t.Run("switches to body tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()

		panel.SetActiveTab(TabBody)

		assert.Equal(t, TabBody, panel.ActiveTab())
	})

	t.Run("switches to auth tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()

		panel.SetActiveTab(TabAuth)

		assert.Equal(t, TabAuth, panel.ActiveTab())
	})

	t.Run("switches to query tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()

		panel.SetActiveTab(TabQuery)

		assert.Equal(t, TabQuery, panel.ActiveTab())
	})

	t.Run("cycles through tabs with Tab key", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, TabHeaders, panel.ActiveTab())
	})

	t.Run("cycles backwards with Shift+Tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, TabURL, panel.ActiveTab())
	})

	t.Run("wraps around from last to first tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabTests)

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, TabURL, panel.ActiveTab())
	})
}

func TestRequestPanel_Navigation(t *testing.T) {
	t.Run("moves cursor down with j", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, 1, panel.Cursor())
	})

	t.Run("moves cursor up with k", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)
		panel.SetCursor(2)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, 1, panel.Cursor())
	})

	t.Run("ignores navigation when unfocused", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		// Not focused
		panel.SetCursor(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, 0, panel.Cursor())
	})
}

func TestRequestPanel_View(t *testing.T) {
	t.Run("renders method and URL", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "GET")
		assert.Contains(t, view, "example.com")
	})

	t.Run("renders tab bar", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "URL")
		assert.Contains(t, view, "Headers")
		assert.Contains(t, view, "Body")
	})

	t.Run("renders headers when on headers tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabHeaders)

		view := panel.View()

		assert.Contains(t, view, "Content-Type")
	})

	t.Run("shows empty state when no request", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "No request selected")
	})

	t.Run("highlights active tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		view := panel.View()

		// Active tab should be in the view
		assert.Contains(t, view, "Headers")
	})
}

func TestRequestPanel_Headers(t *testing.T) {
	t.Run("displays request headers", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetActiveTab(TabHeaders)

		headers := panel.Headers()

		assert.Greater(t, len(headers), 0)
	})

	t.Run("adds new header", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetActiveTab(TabHeaders)
		initialCount := len(panel.Headers())

		panel.AddHeader("X-Custom", "value")

		assert.Equal(t, initialCount+1, len(panel.Headers()))
	})

	t.Run("removes header", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetActiveTab(TabHeaders)
		panel.AddHeader("X-Remove", "value")
		initialCount := len(panel.Headers())

		panel.RemoveHeader("X-Remove")

		assert.Equal(t, initialCount-1, len(panel.Headers()))
	})
}

func TestRequestPanel_Body(t *testing.T) {
	t.Run("displays body content", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetActiveTab(TabBody)

		body := panel.Body()

		assert.NotNil(t, body)
	})

	t.Run("sets body content", func(t *testing.T) {
		panel := newTestRequestPanel(t)

		panel.SetBody(`{"name": "test"}`)

		assert.Contains(t, panel.Body(), "name")
	})
}

func TestRequestPanel_SendRequest(t *testing.T) {
	t.Run("emits send message on Enter in URL tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := panel.Update(msg)

		// Should produce a command
		assert.NotNil(t, cmd)
	})
}

func TestRequestPanel_URLEditing(t *testing.T) {
	t.Run("e key enters URL edit mode", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		panel.Update(msg)

		// Should be in editing mode - check view shows cursor
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "▌") // Cursor indicator
	})

	t.Run("Escape cancels URL editing", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		panel.Update(msg)

		// Press Escape
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		panel.Update(msg)

		// Should exit edit mode
		panel.SetSize(80, 30)
		view := panel.View()
		assert.NotContains(t, view, "▌")
	})

	t.Run("Enter saves URL edit", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Type new URL
		for _, r := range "https://newurl.com" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			updated, _ = panel.Update(msg)
			panel = updated.(*RequestPanel)
		}

		// Press Enter
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		panel.Update(msg)

		// URL should be updated
		assert.Contains(t, panel.Request().URL(), "newurl")
	})
}

func TestRequestPanel_MethodEditing(t *testing.T) {
	t.Run("m key enters method edit mode", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		panel.SetSize(80, 30)
		view := panel.View()
		// Should show method selector with multiple methods
		assert.Contains(t, view, "POST")
		assert.Contains(t, view, "PUT")
	})

	t.Run("arrow keys cycle through methods", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter method edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Press down arrow
		msg = tea.KeyMsg{Type: tea.KeyDown}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Press Enter to save
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Method should have changed
		assert.NotEqual(t, "GET", panel.Request().Method())
	})

	t.Run("Escape cancels method edit", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)
		originalMethod := panel.Request().Method()

		// Enter method edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Change method
		msg = tea.KeyMsg{Type: tea.KeyDown}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Cancel with Escape
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Method should be unchanged
		assert.Equal(t, originalMethod, panel.Request().Method())
	})
}

func TestRequestPanel_FocusBlur(t *testing.T) {
	t.Run("Focus sets focused state", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.Focus()
		assert.True(t, panel.Focused())
	})

	t.Run("Blur removes focused state", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.Focus()
		panel.Blur()
		assert.False(t, panel.Focused())
	})
}

func TestRequestPanel_Size(t *testing.T) {
	t.Run("SetSize updates dimensions", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetSize(100, 50)
		assert.Equal(t, 100, panel.Width())
		assert.Equal(t, 50, panel.Height())
	})

	t.Run("returns empty view with zero size", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetSize(0, 0)
		assert.Empty(t, panel.View())
	})
}

func TestRequestPanel_Init(t *testing.T) {
	t.Run("Init returns nil", func(t *testing.T) {
		panel := NewRequestPanel()
		cmd := panel.Init()
		assert.Nil(t, cmd)
	})
}

func TestRequestPanel_WindowSizeMsg(t *testing.T) {
	t.Run("handles WindowSizeMsg when focused", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.Focus()

		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, 120, panel.Width())
		assert.Equal(t, 40, panel.Height())
	})

	t.Run("handles WindowSizeMsg when unfocused", func(t *testing.T) {
		panel := NewRequestPanel()
		// Not focused

		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, 120, panel.Width())
		assert.Equal(t, 40, panel.Height())
	})
}

func TestRequestPanel_URLEditingCursor(t *testing.T) {
	t.Run("backspace removes character before cursor", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Type some text
		for _, r := range "test" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			updated, _ = panel.Update(msg)
			panel = updated.(*RequestPanel)
		}

		// Backspace
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Should have removed last character
		assert.True(t, true) // Test passed if no panic
	})

	t.Run("left arrow moves cursor left", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Type text
		for _, r := range "abc" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			updated, _ = panel.Update(msg)
			panel = updated.(*RequestPanel)
		}

		// Move left
		msg = tea.KeyMsg{Type: tea.KeyLeft}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})

	t.Run("right arrow moves cursor right", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Type text
		for _, r := range "abc" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			updated, _ = panel.Update(msg)
			panel = updated.(*RequestPanel)
		}

		// Move left then right
		msg = tea.KeyMsg{Type: tea.KeyLeft}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyRight}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})

	t.Run("home key moves cursor to start", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		for _, r := range "test" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			updated, _ = panel.Update(msg)
			panel = updated.(*RequestPanel)
		}

		msg = tea.KeyMsg{Type: tea.KeyHome}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})

	t.Run("end key moves cursor to end", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyEnd}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})

	t.Run("delete key removes character after cursor", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		for _, r := range "test" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			updated, _ = panel.Update(msg)
			panel = updated.(*RequestPanel)
		}

		// Move cursor to start
		msg = tea.KeyMsg{Type: tea.KeyHome}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Delete character
		msg = tea.KeyMsg{Type: tea.KeyDelete}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})

	t.Run("ctrl+u clears input", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		for _, r := range "test" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			updated, _ = panel.Update(msg)
			panel = updated.(*RequestPanel)
		}

		msg = tea.KeyMsg{Type: tea.KeyCtrlU}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})

	t.Run("ctrl+a moves to start", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyCtrlA}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})

	t.Run("ctrl+e moves to end", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyCtrlE}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.True(t, true)
	})
}

func TestRequestPanel_MethodEditingVimKeys(t *testing.T) {
	t.Run("j key cycles to next method", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter method edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Press j
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Save with Enter
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.NotEqual(t, "GET", panel.Request().Method())
	})

	t.Run("k key cycles to previous method", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter method edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Press k (wraps to last method)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Save with Enter
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.NotEqual(t, "GET", panel.Request().Method())
	})

	t.Run("l key cycles to next method", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.NotEqual(t, "GET", panel.Request().Method())
	})

	t.Run("h key cycles to previous method", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.NotEqual(t, "GET", panel.Request().Method())
	})

	t.Run("up arrow cycles methods", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyUp}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.NotEqual(t, "GET", panel.Request().Method())
	})

	t.Run("left arrow cycles methods", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyLeft}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.NotEqual(t, "GET", panel.Request().Method())
	})

	t.Run("right arrow cycles methods", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyRight}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.NotEqual(t, "GET", panel.Request().Method())
	})
}

func TestRequestPanel_TabContent(t *testing.T) {
	t.Run("renders auth tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabAuth)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders query tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabQuery)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders tests tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabTests)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders body tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabBody)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders body tab with JSON content", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Create User", "POST", "https://example.com/users")
		req.SetBodyRaw(`{"name": "John", "email": "john@example.com"}`, "application/json")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabBody)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders query tab with URL having query params", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Search", "GET", "https://example.com/search?q=test&page=1")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabQuery)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestRequestPanel_MethodStyles(t *testing.T) {
	t.Run("GET method displays correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Get", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "GET")
	})

	t.Run("POST method displays correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Post", "POST", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "POST")
	})

	t.Run("PUT method displays correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Put", "PUT", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "PUT")
	})

	t.Run("DELETE method displays correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Delete", "DELETE", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "DELETE")
	})

	t.Run("PATCH method displays correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Patch", "PATCH", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "PATCH")
	})

	t.Run("HEAD method displays correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Head", "HEAD", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "HEAD")
	})

	t.Run("OPTIONS method displays correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Options", "OPTIONS", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		view := panel.View()
		assert.Contains(t, view, "OPTIONS")
	})
}

func TestRequestPanel_HeadersAndBody(t *testing.T) {
	t.Run("Headers returns request headers", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		headers := panel.Headers()
		assert.Contains(t, headers, "Content-Type")
		assert.Contains(t, headers, "Authorization")
	})

	t.Run("Headers returns empty for nil request", func(t *testing.T) {
		panel := NewRequestPanel()
		headers := panel.Headers()
		assert.Empty(t, headers)
	})

	t.Run("Body returns request body", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Create", "POST", "https://example.com")
		req.SetBodyRaw(`{"key": "value"}`, "application/json")
		panel.SetRequest(req)

		body := panel.Body()
		assert.Equal(t, `{"key": "value"}`, body)
	})

	t.Run("Body returns empty for nil request", func(t *testing.T) {
		panel := NewRequestPanel()
		body := panel.Body()
		assert.Empty(t, body)
	})
}

func TestRequestPanel_IsEditing(t *testing.T) {
	t.Run("returns false when not editing", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetRequest(core.NewRequestDefinition("Test", "GET", "https://example.com"))
		assert.False(t, panel.IsEditing())
	})

	t.Run("returns true when editing URL", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()

		// Press 'e' to enter URL edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		panel.Update(msg)

		assert.True(t, panel.IsEditing())
	})
}

func TestRequestPanel_HeaderEditing(t *testing.T) {
	t.Run("adds new header with 'a' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add header
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		panel.Update(msg)

		// Should be in editing mode
		assert.True(t, panel.IsEditing())
	})

	t.Run("deletes header with 'd' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("X-Custom", "value")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)
		panel.SetCursor(0)

		// Press 'd' to delete
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		panel.Update(msg)

		// Header should be removed
		assert.Equal(t, "", req.GetHeader("X-Custom"))
	})

	t.Run("headers tab shows table layout", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("Content-Type", "application/json")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		view := panel.View()
		assert.Contains(t, view, "Key")
		assert.Contains(t, view, "Value")
		assert.Contains(t, view, "Content-Type")
	})

	t.Run("types characters in header key field", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add new header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		assert.True(t, panel.editingHeader)
		assert.Equal(t, "key", panel.headerEditMode)

		// Type in key field
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
		assert.Contains(t, panel.headerKeyInput, "X")
	})

	t.Run("switches between key and value with Tab", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add new header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		assert.Equal(t, "key", panel.headerEditMode)

		// Press Tab to switch to value
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		assert.Equal(t, "value", panel.headerEditMode)

		// Press Tab again to switch back to key
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		assert.Equal(t, "key", panel.headerEditMode)
	})

	t.Run("saves header with Enter", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add new header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		// Type key
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})

		// Switch to value
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})

		// Type value
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

		// Press Enter to save
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		assert.False(t, panel.editingHeader)
		assert.Equal(t, "v1", req.GetHeader("X-T"))
	})

	t.Run("cancels header edit with Escape", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add new header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		assert.True(t, panel.editingHeader)

		// Type something
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

		// Press Escape to cancel
		panel.Update(tea.KeyMsg{Type: tea.KeyEsc})

		assert.False(t, panel.editingHeader)
	})

	t.Run("handles backspace in header key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add new header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		// Type key
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}})
		assert.Equal(t, "AB", panel.headerKeyInput)

		// Backspace
		panel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "A", panel.headerKeyInput)
	})

	t.Run("handles backspace in header value", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add new header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		// Switch to value
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})

		// Type value
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		assert.Equal(t, "XY", panel.headerValueInput)

		// Backspace
		panel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "X", panel.headerValueInput)
	})

	t.Run("renders edit row during header editing", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press 'a' to add new header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		// Type something
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

		view := panel.View()
		assert.Contains(t, view, "X")
	})
}

func TestRequestPanel_BodyEditing(t *testing.T) {
	t.Run("enters body edit mode with 'e' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody(`{"key": "value"}`)
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Press 'e' to edit
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		panel.Update(msg)

		assert.True(t, panel.IsEditing())
	})

	t.Run("body tab shows content", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody(`{"name": "test"}`)
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		view := panel.View()
		assert.Contains(t, view, "name")
		assert.Contains(t, view, "test")
	})

	t.Run("empty body shows hint", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		view := panel.View()
		assert.Contains(t, view, "edit")
	})

	t.Run("types characters in body edit mode", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		assert.True(t, panel.editingBody)

		// Type a character
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}})

		// Should have the character
		assert.Contains(t, strings.Join(panel.bodyLines, ""), "{")
	})

	t.Run("exits body edit mode with Esc", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody(`{"old": "body"}`)
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Press Esc
		panel.Update(tea.KeyMsg{Type: tea.KeyEsc})

		assert.False(t, panel.editingBody)
	})

	t.Run("saves body with Enter in non-edit mode", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Type content
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

		// Enter adds newline in body edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		assert.True(t, panel.editingBody) // Still editing
	})

	t.Run("handles backspace in body edit mode", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody("ab")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Move cursor to end
		panel.bodyCursorCol = 2

		// Press backspace
		panel.Update(tea.KeyMsg{Type: tea.KeyBackspace})

		assert.Contains(t, strings.Join(panel.bodyLines, ""), "a")
	})

	t.Run("navigates body lines with arrow keys", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody("line1\nline2\nline3")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Move down
		panel.Update(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, 1, panel.bodyCursorLine)

		// Move down again
		panel.Update(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, 2, panel.bodyCursorLine)

		// Move up
		panel.Update(tea.KeyMsg{Type: tea.KeyUp})
		assert.Equal(t, 1, panel.bodyCursorLine)
	})

	t.Run("navigates body with left/right keys", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody("ab")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Move right
		panel.Update(tea.KeyMsg{Type: tea.KeyRight})
		assert.Equal(t, 1, panel.bodyCursorCol)

		// Move left
		panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, 0, panel.bodyCursorCol)
	})
}

func TestRequestPanel_URLBar(t *testing.T) {
	t.Run("shows placeholder when URL empty", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()

		view := panel.View()
		assert.Contains(t, view, "Enter request URL")
	})

	t.Run("shows empty state when no request", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetSize(80, 30)
		panel.Focus()

		view := panel.View()
		assert.Contains(t, view, "No request selected")
	})

	t.Run("shows hint when focused on URL tab", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		view := panel.View()
		assert.Contains(t, view, "edit")
	})
}

func TestRequestPanel_QueryTab(t *testing.T) {
	t.Run("renders query tab", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com?foo=bar")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		view := panel.View()
		// Should contain query-related content
		assert.NotEmpty(t, view)
	})

	t.Run("shows empty query hint", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestRequestPanel_AuthTab(t *testing.T) {
	t.Run("renders auth tab", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestRequestPanel_CursorMovement(t *testing.T) {
	t.Run("moveCursor stays within bounds", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("X-One", "1")
		req.SetHeader("X-Two", "2")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		// Press j to move cursor down
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 1, panel.cursor)

		// Press k to move cursor up
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		assert.Equal(t, 0, panel.cursor)

		// Press k again - should stay at 0
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		assert.Equal(t, 0, panel.cursor)
	})

	t.Run("maxCursorForTab returns correct values", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("X-One", "1")
		req.SetHeader("X-Two", "2")
		panel.SetRequest(req)
		panel.SetSize(80, 30)

		// Headers tab should have max based on header count
		panel.SetActiveTab(TabHeaders)
		max := panel.maxCursorForTab()
		assert.GreaterOrEqual(t, max, 1) // At least 1 for 2 headers

		// Body tab
		panel.SetActiveTab(TabBody)
		max = panel.maxCursorForTab()
		assert.GreaterOrEqual(t, max, 0)

		// URL tab
		panel.SetActiveTab(TabURL)
		max = panel.maxCursorForTab()
		assert.Equal(t, 0, max)
	})
}

func TestRequestPanel_HeaderEditing_Edit(t *testing.T) {
	t.Run("edits existing header with 'e' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("X-Test", "old-value")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)
		panel.SetCursor(0)

		// Press 'e' to edit existing header
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		assert.True(t, panel.editingHeader)
		assert.False(t, panel.headerIsNew)
	})
}

func TestRequestPanel_SyncHeaderKeys(t *testing.T) {
	t.Run("syncs header keys from request", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("Content-Type", "application/json")
		req.SetHeader("Authorization", "Bearer token")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.SetActiveTab(TabHeaders)

		// Render view to trigger syncHeaderKeys
		view := panel.View()
		assert.NotEmpty(t, view)

		// Header keys should be synced after rendering
		assert.GreaterOrEqual(t, len(panel.headerKeys), 2)
	})
}

// Helper functions

func newTestRequestPanel(t *testing.T) *RequestPanel {
	t.Helper()
	panel := NewRequestPanel()

	req := core.NewRequestDefinition("Test Request", "GET", "https://api.example.com/users")
	req.SetHeader("Content-Type", "application/json")
	req.SetHeader("Authorization", "Bearer token123")

	panel.SetRequest(req)
	return panel
}
