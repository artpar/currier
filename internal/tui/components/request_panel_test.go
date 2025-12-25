package components

import (
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
