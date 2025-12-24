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
