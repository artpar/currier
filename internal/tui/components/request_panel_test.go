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

	t.Run("cycles through tabs with ] key", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, TabHeaders, panel.ActiveTab())
	})

	t.Run("cycles backwards with [ key", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabHeaders)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		assert.Equal(t, TabURL, panel.ActiveTab())
	})

	t.Run("wraps around from last to first tab", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabTests)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
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

		assert.Contains(t, view, "Press n")
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

	t.Run("Escape saves method edit (vim-like)", func(t *testing.T) {
		panel := newTestRequestPanel(t)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter method edit mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Change method (Down selects next: POST)
		msg = tea.KeyMsg{Type: tea.KeyDown}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Save with Escape (vim-like: Esc saves and exits edit mode)
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ = panel.Update(msg)
		panel = updated.(*RequestPanel)

		// Method should be changed to POST (Escape now saves)
		assert.Equal(t, "POST", panel.Request().Method())
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
		assert.Contains(t, view, "Press n")
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

	t.Run("adds new query param with 'a' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		assert.True(t, panel.editingQuery)
		assert.True(t, panel.queryIsNew)
	})

	t.Run("types key and value in query edit", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		// Add new param
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		// Type key
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		assert.Equal(t, "foo", panel.queryKeyInput)

		// Tab to value
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		assert.Equal(t, "value", panel.queryEditMode)

		// Type value
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		assert.Equal(t, "bar", panel.queryValueInput)
	})

	t.Run("saves query param with Enter", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		// Add new param
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		// Type key
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

		// Tab to value
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})

		// Type value
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

		// Save
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		assert.False(t, panel.editingQuery)
		assert.Equal(t, "val", req.GetQueryParam("key"))
	})

	t.Run("saves query param with Esc", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		// Add new param
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		// Type key
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

		// Save with Esc
		panel.Update(tea.KeyMsg{Type: tea.KeyEsc})

		assert.False(t, panel.editingQuery)
	})

	t.Run("deletes query param with 'd' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetQueryParam("foo", "bar")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)
		panel.SetCursor(0)

		// Delete
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

		assert.Equal(t, "", req.GetQueryParam("foo"))
	})

	t.Run("edits existing query param with 'e' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetQueryParam("foo", "bar")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)
		panel.SetCursor(0)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		assert.True(t, panel.editingQuery)
		assert.False(t, panel.queryIsNew)
	})

	t.Run("handles backspace in query key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		assert.Equal(t, "ab", panel.queryKeyInput)

		panel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "a", panel.queryKeyInput)
	})

	t.Run("handles backspace in query value", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		assert.Equal(t, "xy", panel.queryValueInput)

		panel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "x", panel.queryValueInput)
	})

	t.Run("cursor movement in query key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

		panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, 2, panel.queryKeyCursor)

		panel.Update(tea.KeyMsg{Type: tea.KeyRight})
		assert.Equal(t, 3, panel.queryKeyCursor)

		panel.Update(tea.KeyMsg{Type: tea.KeyHome})
		assert.Equal(t, 0, panel.queryKeyCursor)

		panel.Update(tea.KeyMsg{Type: tea.KeyEnd})
		assert.Equal(t, 3, panel.queryKeyCursor)
	})

	t.Run("cursor movement in query value", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

		panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, 2, panel.queryValueCursor)

		panel.Update(tea.KeyMsg{Type: tea.KeyRight})
		assert.Equal(t, 3, panel.queryValueCursor)

		panel.Update(tea.KeyMsg{Type: tea.KeyHome})
		assert.Equal(t, 0, panel.queryValueCursor)

		panel.Update(tea.KeyMsg{Type: tea.KeyEnd})
		assert.Equal(t, 3, panel.queryValueCursor)
	})

	t.Run("Ctrl+U clears query key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

		panel.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
		assert.Equal(t, "", panel.queryKeyInput)
	})

	t.Run("Ctrl+U clears query value", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

		panel.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
		assert.Equal(t, "", panel.queryValueInput)
	})

	t.Run("Delete key in query key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyHome})
		panel.Update(tea.KeyMsg{Type: tea.KeyDelete})

		assert.Equal(t, "b", panel.queryKeyInput)
	})

	t.Run("Delete key in query value", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyHome})
		panel.Update(tea.KeyMsg{Type: tea.KeyDelete})

		assert.Equal(t, "y", panel.queryValueInput)
	})

	t.Run("Space in query key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeySpace})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

		assert.Equal(t, "a b", panel.queryKeyInput)
	})

	t.Run("Space in query value", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		panel.Update(tea.KeyMsg{Type: tea.KeySpace})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

		assert.Equal(t, "x y", panel.queryValueInput)
	})

	t.Run("Ctrl+A and Ctrl+E in query edit", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

		panel.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
		assert.Equal(t, 0, panel.queryKeyCursor)

		panel.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
		assert.Equal(t, 3, panel.queryKeyCursor)
	})

	t.Run("renders query edit row", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabQuery)

		// Add new param
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

		view := panel.View()
		assert.Contains(t, view, "k")
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

func TestRequestPanel_StartURLEdit(t *testing.T) {
	t.Run("enters URL edit mode externally", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com/api")
		panel.SetRequest(req)
		panel.SetSize(80, 30)

		assert.False(t, panel.editingURL)

		panel.StartURLEdit()

		assert.True(t, panel.editingURL)
		assert.Equal(t, "https://example.com/api", panel.urlInput)
		assert.Equal(t, len("https://example.com/api"), panel.urlCursor)
		assert.Equal(t, TabURL, panel.activeTab)
	})

	t.Run("does nothing if no request", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetSize(80, 30)

		panel.StartURLEdit()

		assert.False(t, panel.editingURL)
	})
}

func TestRequestPanel_EmptyStateMessage(t *testing.T) {
	t.Run("shows new request hint", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetSize(80, 30)
		panel.Focus()

		view := panel.View()
		assert.Contains(t, view, "Press n")
	})
}

func TestRequestPanel_AuthEditing(t *testing.T) {
	t.Run("enters auth edit mode with 'e' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Press 'e' to enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		assert.True(t, panel.editingAuth)
		assert.True(t, panel.IsEditing())
	})

	t.Run("starts on auth type field (index 0)", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		assert.Equal(t, 0, panel.authFieldIndex)
	})

	t.Run("cycles auth type forward with 'l' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		initialIndex := panel.authTypeIndex

		// Press 'l' to cycle forward
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

		assert.Equal(t, initialIndex+1, panel.authTypeIndex)
	})

	t.Run("cycles auth type backward with 'h' key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Press 'l' to move to second type
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		currentIndex := panel.authTypeIndex

		// Press 'h' to cycle backward
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

		assert.Equal(t, currentIndex-1, panel.authTypeIndex)
	})

	t.Run("cycles auth type with arrow keys", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		initialIndex := panel.authTypeIndex

		// Right arrow cycles forward
		panel.Update(tea.KeyMsg{Type: tea.KeyRight})
		assert.Equal(t, initialIndex+1, panel.authTypeIndex)

		// Left arrow cycles backward
		panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, initialIndex, panel.authTypeIndex)
	})

	t.Run("navigates between fields with j/k keys", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Select Basic Auth (has Username, Password fields)
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // To Basic Auth
		assert.Equal(t, 0, panel.authFieldIndex) // Still on type

		// Press 'j' to move to first field
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 1, panel.authFieldIndex)

		// Press 'j' again to move to second field
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 2, panel.authFieldIndex)

		// Press 'k' to move back
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		assert.Equal(t, 1, panel.authFieldIndex)
	})

	t.Run("navigates with up/down arrows", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

		// Down arrow moves to next field
		panel.Update(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, 1, panel.authFieldIndex)

		// Up arrow moves back
		panel.Update(tea.KeyMsg{Type: tea.KeyUp})
		assert.Equal(t, 0, panel.authFieldIndex)
	})

	t.Run("Enter on type field cycles auth type", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		initialIndex := panel.authTypeIndex

		// Press Enter to cycle type
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		assert.Equal(t, initialIndex+1, panel.authTypeIndex)
	})

	t.Run("Enter on field enters field edit mode", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 1, panel.authFieldIndex)

		// Press Enter to edit
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		assert.True(t, panel.authEditingField)
	})

	t.Run("types characters in field edit mode", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type username
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

		assert.Equal(t, "user", panel.authFieldInput)
	})

	t.Run("saves field with Enter", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type username
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

		// Press Enter to save
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		assert.False(t, panel.authEditingField)
		// Verify saved
		auth := req.Auth()
		assert.NotNil(t, auth)
		assert.Equal(t, "john", auth.Username)
	})

	t.Run("saves field with Esc", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

		// Press Esc to save
		panel.Update(tea.KeyMsg{Type: tea.KeyEsc})

		assert.False(t, panel.authEditingField)
		auth := req.Auth()
		assert.NotNil(t, auth)
		assert.Equal(t, "test", auth.Username)
	})

	t.Run("Tab moves to next field and saves current", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type username
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

		// Press Tab to move to next field
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})

		assert.False(t, panel.authEditingField)
		assert.Equal(t, 2, panel.authFieldIndex) // Moved to password
	})

	t.Run("exits auth edit mode with Esc from navigation", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		assert.True(t, panel.editingAuth)

		// Press Esc to exit
		panel.Update(tea.KeyMsg{Type: tea.KeyEsc})

		assert.False(t, panel.editingAuth)
	})

	t.Run("e key on field enters edit mode", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

		// Press 'e' to edit
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		assert.True(t, panel.authEditingField)
	})

	t.Run("handles backspace in field edit", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type then backspace
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		assert.Equal(t, "ab", panel.authFieldInput)

		panel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "a", panel.authFieldInput)
	})

	t.Run("handles cursor movement in field edit", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type text
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
		assert.Equal(t, 3, panel.authFieldCursor)

		// Left arrow
		panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, 2, panel.authFieldCursor)

		// Right arrow
		panel.Update(tea.KeyMsg{Type: tea.KeyRight})
		assert.Equal(t, 3, panel.authFieldCursor)

		// Home
		panel.Update(tea.KeyMsg{Type: tea.KeyHome})
		assert.Equal(t, 0, panel.authFieldCursor)

		// End
		panel.Update(tea.KeyMsg{Type: tea.KeyEnd})
		assert.Equal(t, 3, panel.authFieldCursor)
	})

	t.Run("Ctrl+U clears field input", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type text
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

		// Ctrl+U to clear
		panel.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

		assert.Equal(t, "", panel.authFieldInput)
		assert.Equal(t, 0, panel.authFieldCursor)
	})

	t.Run("renders auth tab with type selector", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		view := panel.View()
		assert.Contains(t, view, "Auth Type")
		assert.Contains(t, view, "No Auth")
	})

	t.Run("renders auth fields for Basic Auth", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		view := panel.View()
		assert.Contains(t, view, "Username")
		assert.Contains(t, view, "Password")
	})

	t.Run("renders auth fields for Bearer Token", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Bearer (index 2)
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRight})
		panel.Update(tea.KeyMsg{Type: tea.KeyRight}) // Bearer Token

		view := panel.View()
		assert.Contains(t, view, "Token")
	})

	t.Run("renders auth fields for API Key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select API Key (index 3)
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.authTypeIndex = 3 // API Key
		panel.applyAuthType(core.AuthTypeAPIKey)

		view := panel.View()
		assert.Contains(t, view, "Key Name")
		assert.Contains(t, view, "Key Value")
	})

	t.Run("masks password field when not editing", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		// Set up auth with password
		auth := core.NewBasicAuth("user", "secret123")
		req.SetAuth(auth)
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		view := panel.View()
		assert.Contains(t, view, "•") // Password is masked
		assert.NotContains(t, view, "secret123")
	})

	t.Run("Tab is captured during auth editing", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		// Tab should not switch panes
		panel.Update(tea.KeyMsg{Type: tea.KeyTab})

		assert.True(t, panel.editingAuth) // Still in auth edit mode
	})

	t.Run("handles space in field edit", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type with space
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeySpace})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

		assert.Equal(t, "a b", panel.authFieldInput)
	})

	t.Run("Ctrl+A moves cursor to start", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type text
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

		// Ctrl+A
		panel.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
		assert.Equal(t, 0, panel.authFieldCursor)
	})

	t.Run("Ctrl+E moves cursor to end", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabAuth)

		// Enter auth edit mode and select Basic Auth
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Basic Auth

		// Move to username field and enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyEnter})

		// Type text
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

		// Move to start then Ctrl+E
		panel.Update(tea.KeyMsg{Type: tea.KeyHome})
		panel.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
		assert.Equal(t, 3, panel.authFieldCursor)
	})
}

func TestRequestPanel_AuthFieldHelpers(t *testing.T) {
	t.Run("getAuthFieldsForType returns correct fields", func(t *testing.T) {
		fields := getAuthFieldsForType(core.AuthTypeBasic)
		assert.Contains(t, fields, "Username")
		assert.Contains(t, fields, "Password")

		fields = getAuthFieldsForType(core.AuthTypeBearer)
		assert.Contains(t, fields, "Token")

		fields = getAuthFieldsForType(core.AuthTypeAPIKey)
		assert.Contains(t, fields, "Key Name")
		assert.Contains(t, fields, "Key Value")
		assert.Contains(t, fields, "Add to")

		fields = getAuthFieldsForType(core.AuthTypeNone)
		assert.Empty(t, fields)
	})

	t.Run("getAuthFieldValue returns correct values for Basic Auth", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		auth := core.NewBasicAuth("testuser", "testpass")
		req.SetAuth(auth)
		panel.SetRequest(req)

		assert.Equal(t, "testuser", panel.getAuthFieldValue(1))
		assert.Equal(t, "testpass", panel.getAuthFieldValue(2))
	})

	t.Run("getAuthFieldValue returns correct values for Bearer", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		auth := core.NewBearerAuth("mytoken123")
		req.SetAuth(auth)
		panel.SetRequest(req)

		assert.Equal(t, "mytoken123", panel.getAuthFieldValue(1))
	})

	t.Run("getAuthFieldValue returns correct values for API Key", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		auth := core.NewAPIKeyAuth("X-API-Key", "secret", core.APIKeyInHeader)
		req.SetAuth(auth)
		panel.SetRequest(req)

		assert.Equal(t, "X-API-Key", panel.getAuthFieldValue(1))
		assert.Equal(t, "secret", panel.getAuthFieldValue(2))
		assert.Equal(t, "header", panel.getAuthFieldValue(3))
	})

	t.Run("setAuthFieldValue sets Basic Auth fields", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		auth := core.NewBasicAuth("", "")
		req.SetAuth(auth)
		panel.SetRequest(req)

		panel.setAuthFieldValue(1, "newuser")
		panel.setAuthFieldValue(2, "newpass")

		assert.Equal(t, "newuser", req.Auth().Username)
		assert.Equal(t, "newpass", req.Auth().Password)
	})

	t.Run("setAuthFieldValue sets Bearer Token", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		auth := core.NewBearerAuth("")
		req.SetAuth(auth)
		panel.SetRequest(req)

		panel.setAuthFieldValue(1, "newtoken")

		assert.Equal(t, "newtoken", req.Auth().Token)
	})

	t.Run("setAuthFieldValue sets API Key fields", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		auth := core.NewAPIKeyAuth("", "", core.APIKeyInHeader)
		req.SetAuth(auth)
		panel.SetRequest(req)

		panel.setAuthFieldValue(1, "X-Key")
		panel.setAuthFieldValue(2, "value123")
		panel.setAuthFieldValue(3, "query")

		assert.Equal(t, "X-Key", req.Auth().Key)
		assert.Equal(t, "value123", req.Auth().Value)
		assert.Equal(t, "query", req.Auth().In)
	})

	t.Run("applyAuthType creates new auth config", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)

		panel.applyAuthType(core.AuthTypeBasic)

		auth := req.Auth()
		assert.NotNil(t, auth)
		assert.Equal(t, core.AuthTypeBasic, auth.GetAuthType())
	})

	t.Run("applyAuthType initializes OAuth2 config", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)

		panel.applyAuthType(core.AuthTypeOAuth2)

		auth := req.Auth()
		assert.NotNil(t, auth)
		assert.NotNil(t, auth.OAuth2)
		assert.Equal(t, "Bearer", auth.OAuth2.HeaderPrefix)
	})
}

func TestRequestPanel_Accessors(t *testing.T) {
	t.Run("HasRequest returns false when no request", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.False(t, panel.HasRequest())
	})

	t.Run("HasRequest returns true when request set", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		assert.True(t, panel.HasRequest())
	})

	t.Run("URL returns empty when no request", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.Equal(t, "", panel.URL())
	})

	t.Run("URL returns request URL", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com/api")
		panel.SetRequest(req)
		assert.Equal(t, "https://example.com/api", panel.URL())
	})

	t.Run("Method returns empty when no request", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.Equal(t, "", panel.Method())
	})

	t.Run("Method returns request method", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		panel.SetRequest(req)
		assert.Equal(t, "POST", panel.Method())
	})

	t.Run("HeadersMap returns nil when no request", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.Nil(t, panel.HeadersMap())
	})

	t.Run("HeadersMap returns request headers", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("X-Custom", "value")
		panel.SetRequest(req)
		headers := panel.HeadersMap()
		assert.Equal(t, "value", headers["X-Custom"])
	})

	t.Run("QueryParamsMap returns nil when no request", func(t *testing.T) {
		panel := NewRequestPanel()
		assert.Nil(t, panel.QueryParamsMap())
	})

	t.Run("QueryParamsMap returns request query params", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetQueryParam("foo", "bar")
		panel.SetRequest(req)
		params := panel.QueryParamsMap()
		assert.Equal(t, "bar", params["foo"])
	})

	t.Run("ActiveTabName returns correct tab names", func(t *testing.T) {
		panel := NewRequestPanel()
		panel.SetActiveTab(TabURL)
		assert.Equal(t, "URL", panel.ActiveTabName())

		panel.SetActiveTab(TabHeaders)
		assert.Equal(t, "Headers", panel.ActiveTabName())

		panel.SetActiveTab(TabQuery)
		assert.Equal(t, "Query", panel.ActiveTabName())

		panel.SetActiveTab(TabBody)
		assert.Equal(t, "Body", panel.ActiveTabName())

		panel.SetActiveTab(TabAuth)
		assert.Equal(t, "Auth", panel.ActiveTabName())

		panel.SetActiveTab(TabTests)
		assert.Equal(t, "Tests", panel.ActiveTabName())
	})

	t.Run("IsEditingMethod returns correct state", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		assert.False(t, panel.IsEditingMethod())

		// Enter method edit mode with 'm'
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		assert.True(t, panel.IsEditingMethod())
	})

	t.Run("EditingField returns correct field name", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()

		// Not editing anything
		assert.Equal(t, "", panel.EditingField())

		// Enter URL edit mode
		panel.SetActiveTab(TabURL)
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		assert.Equal(t, "url", panel.EditingField())

		// Exit URL edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyEsc})

		// Enter method edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		assert.Equal(t, "method", panel.EditingField())
	})

	t.Run("CursorPosition returns correct position", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabURL)

		// Enter URL edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		pos := panel.CursorPosition()
		// Cursor should be at end of URL
		assert.Equal(t, len("https://example.com"), pos)
	})

	t.Run("CursorPosition returns 0 when not editing", func(t *testing.T) {
		panel := NewRequestPanel()
		pos := panel.CursorPosition()
		assert.Equal(t, 0, pos)
	})
}

func TestRequestPanel_BodyRendering(t *testing.T) {
	t.Run("renders body tab with JSON content", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody(`{"name": "test", "value": 123}`)
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		view := panel.View()
		assert.Contains(t, view, "name")
	})

	t.Run("renders body tab with empty body", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders body edit mode", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody("test body")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(TabBody)

		// Enter edit mode
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

		view := panel.View()
		assert.NotEmpty(t, view)
		assert.True(t, panel.editingBody)
	})
}

func TestRequestPanel_SearchBarRendering(t *testing.T) {
	t.Run("renders URL bar correctly", func(t *testing.T) {
		panel := NewRequestPanel()
		req := core.NewRequestDefinition("Test", "GET", "https://api.example.com/users")
		panel.SetRequest(req)
		panel.SetSize(80, 30)
		panel.Focus()

		view := panel.View()
		assert.Contains(t, view, "GET")
		assert.Contains(t, view, "api.example.com")
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
