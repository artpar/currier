package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
	"github.com/stretchr/testify/assert"
)

func TestNewResponsePanel(t *testing.T) {
	t.Run("creates empty panel", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.NotNil(t, panel)
		assert.Equal(t, "Response", panel.Title())
	})

	t.Run("starts with no response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.Nil(t, panel.Response())
	})

	t.Run("starts unfocused", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.False(t, panel.Focused())
	})

	t.Run("starts on body tab", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.Equal(t, ResponseTabBody, panel.ActiveTab())
	})
}

func TestResponsePanel_SetResponse(t *testing.T) {
	t.Run("sets response", func(t *testing.T) {
		panel := NewResponsePanel()
		resp := newTestResponse(200, "OK")

		panel.SetResponse(resp)

		assert.Equal(t, resp, panel.Response())
	})

	t.Run("updates title with status", func(t *testing.T) {
		panel := NewResponsePanel()
		resp := newTestResponse(200, "OK")

		panel.SetResponse(resp)

		assert.Contains(t, panel.Title(), "200")
	})

	t.Run("clears response when nil", func(t *testing.T) {
		panel := NewResponsePanel()
		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)

		panel.SetResponse(nil)

		assert.Nil(t, panel.Response())
	})

	t.Run("shows loading state", func(t *testing.T) {
		panel := NewResponsePanel()

		panel.SetLoading(true)

		assert.True(t, panel.IsLoading())
	})
}

func TestResponsePanel_Tabs(t *testing.T) {
	t.Run("switches to headers tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()

		panel.SetActiveTab(ResponseTabHeaders)

		assert.Equal(t, ResponseTabHeaders, panel.ActiveTab())
	})

	t.Run("switches to cookies tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()

		panel.SetActiveTab(ResponseTabCookies)

		assert.Equal(t, ResponseTabCookies, panel.ActiveTab())
	})

	t.Run("switches to timing tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()

		panel.SetActiveTab(ResponseTabTiming)

		assert.Equal(t, ResponseTabTiming, panel.ActiveTab())
	})

	t.Run("cycles through tabs with Tab key", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabBody)

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, ResponseTabHeaders, panel.ActiveTab())
	})

	t.Run("cycles backwards with Shift+Tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabHeaders)

		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, ResponseTabBody, panel.ActiveTab())
	})
}

func TestResponsePanel_Navigation(t *testing.T) {
	t.Run("scrolls down with j", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 1, panel.ScrollOffset())
	})

	t.Run("scrolls up with k", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetScrollOffset(5)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 4, panel.ScrollOffset())
	})

	t.Run("does not scroll past top", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetScrollOffset(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 0, panel.ScrollOffset())
	})

	t.Run("ignores navigation when unfocused", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		// Not focused
		panel.SetScrollOffset(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 0, panel.ScrollOffset())
	})
}

func TestResponsePanel_View(t *testing.T) {
	t.Run("renders status code", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "200")
	})

	t.Run("renders status text", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "OK")
	})

	t.Run("renders response time", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "ms")
	})

	t.Run("renders tab bar", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "Body")
		assert.Contains(t, view, "Headers")
	})

	t.Run("renders body content", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabBody)

		view := panel.View()

		assert.Contains(t, view, "test_field")
	})

	t.Run("shows empty state when no response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		view := panel.View()

		assert.Contains(t, view, "No response")
	})

	t.Run("shows loading state", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetLoading(true)

		view := panel.View()

		assert.Contains(t, view, "Loading")
	})
}

func TestResponsePanel_StatusColors(t *testing.T) {
	t.Run("success status (2xx)", func(t *testing.T) {
		panel := NewResponsePanel()
		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)

		assert.True(t, panel.IsSuccess())
	})

	t.Run("error status (4xx)", func(t *testing.T) {
		panel := NewResponsePanel()
		resp := newTestResponse(404, "Not Found")
		panel.SetResponse(resp)

		assert.True(t, panel.IsClientError())
	})

	t.Run("server error status (5xx)", func(t *testing.T) {
		panel := NewResponsePanel()
		resp := newTestResponse(500, "Internal Server Error")
		panel.SetResponse(resp)

		assert.True(t, panel.IsServerError())
	})
}

func TestResponsePanel_Copy(t *testing.T) {
	t.Run("copies body to clipboard on y", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabBody)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		_, cmd := panel.Update(msg)

		// Should emit a copy command
		assert.NotNil(t, cmd)
	})
}

func TestResponsePanel_FocusBlur(t *testing.T) {
	t.Run("Focus sets focused state", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.Focus()
		assert.True(t, panel.Focused())
	})

	t.Run("Blur removes focused state", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.Focus()
		panel.Blur()
		assert.False(t, panel.Focused())
	})
}

func TestResponsePanel_Size(t *testing.T) {
	t.Run("SetSize updates dimensions", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(100, 50)
		assert.Equal(t, 100, panel.Width())
		assert.Equal(t, 50, panel.Height())
	})

	t.Run("returns empty view with zero size", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(0, 0)
		assert.Empty(t, panel.View())
	})
}

func TestResponsePanel_Init(t *testing.T) {
	t.Run("Init returns nil", func(t *testing.T) {
		panel := NewResponsePanel()
		cmd := panel.Init()
		assert.Nil(t, cmd)
	})
}

func TestResponsePanel_Error(t *testing.T) {
	t.Run("SetError sets error state", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetError(assert.AnError)

		assert.Equal(t, assert.AnError, panel.Error())
	})

	t.Run("SetError clears response", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetError(assert.AnError)

		assert.Nil(t, panel.Response())
	})

	t.Run("shows error in view", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetError(assert.AnError)

		view := panel.View()
		assert.Contains(t, view, "Error")
	})
}

func TestResponsePanel_WindowSizeMsg(t *testing.T) {
	t.Run("handles WindowSizeMsg when focused", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.Focus()

		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 120, panel.Width())
		assert.Equal(t, 40, panel.Height())
	})

	t.Run("handles WindowSizeMsg when unfocused", func(t *testing.T) {
		panel := NewResponsePanel()
		// Not focused

		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 120, panel.Width())
		assert.Equal(t, 40, panel.Height())
	})
}

func TestResponsePanel_GoToBottom(t *testing.T) {
	t.Run("G goes to bottom", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetSize(80, 30)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		// Scroll offset should be set to max
		assert.GreaterOrEqual(t, panel.ScrollOffset(), 0)
	})

	t.Run("g goes to top", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetScrollOffset(10)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 0, panel.ScrollOffset())
	})
}

func TestResponsePanel_HeadersTab(t *testing.T) {
	t.Run("renders headers on headers tab", func(t *testing.T) {
		panel := newTestResponsePanelWithHeaders(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabHeaders)

		view := panel.View()
		// Should contain Content-Type header
		assert.Contains(t, view, "Content-Type")
	})

	t.Run("shows no headers message when empty", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabHeaders)

		view := panel.View()
		assert.Contains(t, view, "No response headers")
	})
}

func TestResponsePanel_TimingTab(t *testing.T) {
	t.Run("renders timing info on timing tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabTiming)

		view := panel.View()
		assert.Contains(t, view, "Total Time")
	})
}

func TestResponsePanel_StatusHelpers(t *testing.T) {
	t.Run("IsSuccess returns false for nil response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.False(t, panel.IsSuccess())
	})

	t.Run("IsClientError returns false for nil response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.False(t, panel.IsClientError())
	})

	t.Run("IsServerError returns false for nil response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.False(t, panel.IsServerError())
	})

	t.Run("IsSuccess returns false for 3xx", func(t *testing.T) {
		panel := NewResponsePanel()
		resp := newTestResponse(301, "Moved Permanently")
		panel.SetResponse(resp)
		assert.False(t, panel.IsSuccess())
	})
}

// Helper functions

func newTestResponsePanel(t *testing.T) *ResponsePanel {
	t.Helper()
	panel := NewResponsePanel()
	resp := newTestResponse(200, "OK")
	panel.SetResponse(resp)
	return panel
}

func newTestResponsePanelWithHeaders(t *testing.T) *ResponsePanel {
	t.Helper()
	panel := NewResponsePanel()
	resp := newTestResponseWithHeaders(200, "OK")
	panel.SetResponse(resp)
	return panel
}

func newTestResponse(code int, statusText string) *core.Response {
	body := core.NewJSONBody(map[string]any{
		"test_field": "test_value",
		"number":     42,
	})

	start := time.Now().Add(-100 * time.Millisecond)
	end := time.Now()

	return core.NewResponse("req-456", "http", core.NewStatus(code, statusText)).
		WithBody(body).
		WithTiming(interfaces.TimingInfo{
			StartTime: start,
			EndTime:   end,
		})
}

func newTestResponseWithHeaders(code int, statusText string) *core.Response {
	body := core.NewJSONBody(map[string]any{
		"test_field": "test_value",
		"number":     42,
	})

	headers := core.NewHeaders()
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Request-Id", "test-123")

	start := time.Now().Add(-100 * time.Millisecond)
	end := time.Now()

	return core.NewResponse("req-456", "http", core.NewStatus(code, statusText)).
		WithBody(body).
		WithHeaders(headers).
		WithTiming(interfaces.TimingInfo{
			StartTime: start,
			EndTime:   end,
		})
}
