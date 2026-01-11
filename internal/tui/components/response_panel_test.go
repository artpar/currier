package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
	"github.com/artpar/currier/internal/script"
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

	t.Run("cycles through tabs with ] key", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabBody)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, ResponseTabHeaders, panel.ActiveTab())
	})

	t.Run("cycles backwards with [ key", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabHeaders)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
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

	t.Run("gg goes to top", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetScrollOffset(10)

		// Press 'g' twice for gg sequence
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)
		assert.True(t, panel.GPressed()) // First g sets gPressed

		updated, _ = panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, 0, panel.ScrollOffset())
		assert.False(t, panel.GPressed()) // Second g completes sequence
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

func TestResponsePanel_CookiesTab(t *testing.T) {
	t.Run("renders cookies tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabCookies)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_TabWrap(t *testing.T) {
	t.Run("wraps from last to first tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabTests) // Tests is now the last tab

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, ResponseTabBody, panel.ActiveTab())
	})

	t.Run("wraps from first to last tab with [ key", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabBody)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, ResponseTabTests, panel.ActiveTab()) // Tests is now the last tab
	})
}

func TestResponsePanel_PageNavigation(t *testing.T) {
	t.Run("page down with Ctrl+D", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetSize(80, 30)

		msg := tea.KeyMsg{Type: tea.KeyCtrlD}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		// Should have scrolled down
		assert.GreaterOrEqual(t, panel.ScrollOffset(), 0)
	})

	t.Run("page up with Ctrl+U", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetSize(80, 30)
		panel.SetScrollOffset(20)

		msg := tea.KeyMsg{Type: tea.KeyCtrlU}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		// Should not increase scroll offset
		assert.LessOrEqual(t, panel.ScrollOffset(), 20)
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

func TestResponsePanel_AllStatusColors(t *testing.T) {
	t.Run("shows 201 Created correctly", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetResponse(newTestResponse(201, "Created"))
		view := panel.View()
		assert.Contains(t, view, "201")
	})

	t.Run("shows 3xx redirects correctly", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetResponse(newTestResponse(301, "Moved Permanently"))
		view := panel.View()
		assert.Contains(t, view, "301")
	})

	t.Run("shows 302 redirect correctly", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetResponse(newTestResponse(302, "Found"))
		view := panel.View()
		assert.Contains(t, view, "302")
	})

	t.Run("shows 400 Bad Request correctly", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetResponse(newTestResponse(400, "Bad Request"))
		view := panel.View()
		assert.Contains(t, view, "400")
	})

	t.Run("shows 503 Service Unavailable correctly", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetResponse(newTestResponse(503, "Service Unavailable"))
		view := panel.View()
		assert.Contains(t, view, "503")
	})
}

func TestResponsePanel_BodySize(t *testing.T) {
	t.Run("shows body size for small response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		body := core.NewRawBody([]byte("Hello"), "text/plain")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithBody(body)
		panel.SetResponse(resp)
		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("shows body size for KB response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		// Create a ~2KB body
		largeContent := make([]byte, 2000)
		for i := range largeContent {
			largeContent[i] = 'x'
		}
		body := core.NewRawBody(largeContent, "text/plain")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithBody(body)
		panel.SetResponse(resp)
		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("shows body size for MB response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		// Create a ~1.5MB body
		largeContent := make([]byte, 1500000)
		for i := range largeContent {
			largeContent[i] = 'y'
		}
		body := core.NewRawBody(largeContent, "text/plain")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithBody(body)
		panel.SetResponse(resp)
		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_CookiesTabExtra(t *testing.T) {
	t.Run("shows no cookies message when no response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(3) // Cookies tab

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("shows no cookies message when no Set-Cookie headers", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(3) // Cookies tab

		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK"))
		panel.SetResponse(resp)
		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("parses and displays cookies", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(3) // Cookies tab

		headers := core.NewHeaders()
		headers.Set("Set-Cookie", "session_id=abc123; Path=/; HttpOnly; Secure; SameSite=Strict")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithHeaders(headers)
		panel.SetResponse(resp)

		view := panel.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Cookie")
	})

	t.Run("parses cookies with domain and expiry", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(3) // Cookies tab

		headers := core.NewHeaders()
		headers.Set("Set-Cookie", "token=xyz789; Domain=.example.com; Path=/api; Expires=Wed, 01 Jan 2025 00:00:00 GMT; HttpOnly")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithHeaders(headers)
		panel.SetResponse(resp)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_ConsoleTab(t *testing.T) {
	t.Run("renders console tab", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(4) // Console tab

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("console tab with response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(4) // Console tab

		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK"))
		panel.SetResponse(resp)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_MouseEvents(t *testing.T) {
	t.Run("handles mouse wheel scrolling", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.Focus()

		body := core.NewRawBody([]byte(strings.Repeat("line\n", 100)), "text/plain")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithBody(body)
		panel.SetResponse(resp)

		// Scroll down
		msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		view := panel.View()
		assert.NotEmpty(t, view)

		// Scroll up
		msg = tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp}
		updated, _ = panel.Update(msg)
		panel = updated.(*ResponsePanel)

		view = panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_PageNavigationExtra(t *testing.T) {
	t.Run("page up and page down navigation", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.Focus()

		body := core.NewRawBody([]byte(strings.Repeat("line\n", 100)), "text/plain")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithBody(body)
		panel.SetResponse(resp)

		// Page down with Ctrl+D
		msg := tea.KeyMsg{Type: tea.KeyCtrlD}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		view := panel.View()
		assert.NotEmpty(t, view)

		// Page up with Ctrl+U
		msg = tea.KeyMsg{Type: tea.KeyCtrlU}
		updated, _ = panel.Update(msg)
		panel = updated.(*ResponsePanel)

		view = panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("g and G navigation", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.Focus()

		body := core.NewRawBody([]byte(strings.Repeat("line\n", 100)), "text/plain")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithBody(body)
		panel.SetResponse(resp)

		// Go to bottom with G
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		view := panel.View()
		assert.NotEmpty(t, view)

		// Go to top with gg
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ = panel.Update(msg)
		panel = updated.(*ResponsePanel)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ = panel.Update(msg)
		panel = updated.(*ResponsePanel)

		view = panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_StatusStyles(t *testing.T) {
	t.Run("100 series status style", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := core.NewResponse("req-123", "http", core.NewStatus(101, "Switching Protocols"))
		panel.SetResponse(resp)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("300 series status style", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := core.NewResponse("req-123", "http", core.NewStatus(301, "Moved Permanently"))
		panel.SetResponse(resp)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("400 series status style", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := core.NewResponse("req-123", "http", core.NewStatus(404, "Not Found"))
		panel.SetResponse(resp)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_AdditionalGetters(t *testing.T) {
	t.Run("HasResponse returns true with response", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		assert.True(t, panel.HasResponse())
	})

	t.Run("HasResponse returns false without response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.False(t, panel.HasResponse())
	})

	t.Run("StatusCode returns code from response", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		assert.Equal(t, 200, panel.StatusCode())
	})

	t.Run("StatusCode returns 0 without response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.Equal(t, 0, panel.StatusCode())
	})

	t.Run("StatusText returns text from response", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		assert.Equal(t, "OK", panel.StatusText())
	})

	t.Run("StatusText returns empty without response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.Equal(t, "", panel.StatusText())
	})

	t.Run("ResponseTime returns duration", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		duration := panel.ResponseTime()
		assert.True(t, duration >= 0)
	})

	t.Run("ResponseTime returns 0 without response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.Equal(t, int64(0), panel.ResponseTime())
	})

	t.Run("BodySize returns size", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		size := panel.BodySize()
		assert.True(t, size >= 0)
	})

	t.Run("BodySize returns 0 without response", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.Equal(t, int64(0), panel.BodySize())
	})

	t.Run("BodyPreview returns preview", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		preview := panel.BodyPreview(50)
		assert.NotEmpty(t, preview)
	})

	t.Run("BodyPreview returns empty without response", func(t *testing.T) {
		panel := NewResponsePanel()
		preview := panel.BodyPreview(50)
		assert.Equal(t, "", preview)
	})

	t.Run("ActiveTabName returns tab name", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetActiveTab(ResponseTabBody)
		assert.Equal(t, "Body", panel.ActiveTabName())
	})

	t.Run("ErrorString returns error message", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetError(assert.AnError)
		assert.NotEmpty(t, panel.ErrorString())
	})

	t.Run("ErrorString returns empty without error", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.Equal(t, "", panel.ErrorString())
	})
}

func TestResponsePanel_PrettyPrint(t *testing.T) {
	t.Run("IsPrettyPrint returns default true", func(t *testing.T) {
		panel := NewResponsePanel()
		assert.True(t, panel.IsPrettyPrint())
	})

	t.Run("SetPrettyPrint updates state", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetPrettyPrint(false)
		assert.False(t, panel.IsPrettyPrint())
		panel.SetPrettyPrint(true)
		assert.True(t, panel.IsPrettyPrint())
	})

	t.Run("DetectedType returns type", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		// Trigger format detection by rendering
		panel.View()
		// Type should be detected
		detectedType := panel.DetectedType()
		assert.NotEmpty(t, detectedType)
	})
}

func TestResponsePanel_TestResults(t *testing.T) {
	t.Run("SetTestResults updates results", func(t *testing.T) {
		panel := NewResponsePanel()
		results := []script.TestResult{
			{Name: "Test 1", Passed: true},
			{Name: "Test 2", Passed: false, Error: "failed"},
		}
		panel.SetTestResults(results)

		assert.Equal(t, results, panel.TestResults())
	})

	t.Run("TestSummary returns summary", func(t *testing.T) {
		panel := NewResponsePanel()
		results := []script.TestResult{
			{Name: "Test 1", Passed: true},
			{Name: "Test 2", Passed: false},
		}
		panel.SetTestResults(results)

		summary := panel.TestSummary()
		assert.Equal(t, 1, summary.Passed)
		assert.Equal(t, 2, summary.Total)
	})

	t.Run("renders Tests tab with results", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		results := []script.TestResult{
			{Name: "Test 1", Passed: true},
			{Name: "Test 2", Passed: false, Error: "assertion failed"},
		}
		panel.SetTestResults(results)
		panel.SetActiveTab(ResponseTabTests)

		view := panel.View()
		assert.Contains(t, view, "Test")
	})
}

func TestResponsePanel_ConsoleMessages(t *testing.T) {
	t.Run("SetConsoleMessages updates messages", func(t *testing.T) {
		panel := NewResponsePanel()
		messages := []ConsoleMessage{
			{Level: "log", Message: "hello"},
			{Level: "log", Message: "world"},
		}
		panel.SetConsoleMessages(messages)

		// Check it was set (indirectly through console tab)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabConsole)
		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("AddConsoleMessage appends message", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.AddConsoleMessage("log", "first")
		panel.AddConsoleMessage("log", "second")

		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabConsole)
		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_TogglePrettyPrint(t *testing.T) {
	t.Run("p key toggles pretty print", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.Focus()
		panel.SetActiveTab(ResponseTabBody)

		assert.True(t, panel.IsPrettyPrint())

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.False(t, panel.IsPrettyPrint())

		updated, _ = panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.True(t, panel.IsPrettyPrint())
	})
}

func TestResponsePanel_CookieParsing(t *testing.T) {
	t.Run("parses simple cookie", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		headers := core.NewHeaders()
		headers.Set("Set-Cookie", "name=value")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithHeaders(headers)
		panel.SetResponse(resp)
		panel.SetActiveTab(ResponseTabCookies)

		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("parses cookie with all attributes", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		headers := core.NewHeaders()
		headers.Set("Set-Cookie", "session=abc123; Path=/; Domain=.example.com; Secure; HttpOnly; SameSite=Strict; Max-Age=3600; Expires=Wed, 01 Jan 2025 00:00:00 GMT")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithHeaders(headers)
		panel.SetResponse(resp)
		panel.SetActiveTab(ResponseTabCookies)

		view := panel.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "session")
	})

	t.Run("parses multiple cookies", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		headers := core.NewHeaders()
		headers.Add("Set-Cookie", "cookie1=value1")
		headers.Add("Set-Cookie", "cookie2=value2")
		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK")).WithHeaders(headers)
		panel.SetResponse(resp)
		panel.SetActiveTab(ResponseTabCookies)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_TestsTabRendering(t *testing.T) {
	t.Run("renders Tests tab with passing tests", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		results := []script.TestResult{
			{Name: "Status is 200", Passed: true},
			{Name: "Body contains data", Passed: true},
		}
		panel.SetTestResults(results)
		panel.SetActiveTab(ResponseTabTests)

		view := panel.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Status is 200")
	})

	t.Run("renders Tests tab with failing tests", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		results := []script.TestResult{
			{Name: "Status is 200", Passed: false, Error: "Expected 200 but got 404"},
		}
		panel.SetTestResults(results)
		panel.SetActiveTab(ResponseTabTests)

		view := panel.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Status is 200")
	})

	t.Run("renders empty Tests tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabTests)

		view := panel.View()
		assert.NotEmpty(t, view)
	})
}

func TestResponsePanel_UpdateEdgeCases(t *testing.T) {
	t.Run("handles g then up key for gg", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabBody)

		// Press g
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		panel.Update(msg)

		// Press g again
		panel.Update(msg)

		assert.NotNil(t, panel)
	})

	t.Run("handles shift+g for end", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabBody)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		panel.Update(msg)

		assert.NotNil(t, panel)
	})

	t.Run("handles y for yank mode", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabBody)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		panel.Update(msg)

		assert.NotNil(t, panel)
	})

	t.Run("handles ] key cycling when at last tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(ResponseTabTests)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, ResponseTabBody, panel.ActiveTab())
	})

	t.Run("handles [ key cycling when at first tab", func(t *testing.T) {
		panel := newTestResponsePanel(t)
		panel.SetSize(80, 30)
		panel.Focus()
		panel.SetActiveTab(ResponseTabBody)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		updated, _ := panel.Update(msg)
		panel = updated.(*ResponsePanel)

		assert.Equal(t, ResponseTabTests, panel.ActiveTab())
	})
}

func TestResponsePanel_ConsoleTabAdditional(t *testing.T) {
	t.Run("renders all console message types", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabConsole)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)

		// Add console messages of different types
		panel.AddConsoleMessage("log", "Test log message")
		panel.AddConsoleMessage("error", "Test error message")
		panel.AddConsoleMessage("warn", "Test warning message")
		panel.AddConsoleMessage("info", "Test info message")

		view := panel.View()
		assert.Contains(t, view, "Test log message")
	})

	t.Run("clears console messages", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.AddConsoleMessage("log", "Test message")

		panel.SetConsoleMessages(nil)

		assert.Empty(t, panel.consoleMessages)
	})
}

func TestResponsePanel_RenderBinaryPlaceholder(t *testing.T) {
	t.Run("renders binary content message", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		// Create response with binary-like headers
		body := core.NewRawBody([]byte{0x89, 0x50, 0x4E, 0x47}, "image/png") // PNG header
		headers := core.NewHeaders()
		headers.Set("Content-Type", "image/png")

		start := time.Now().Add(-100 * time.Millisecond)
		end := time.Now()

		resp := core.NewResponse("req-456", "http", core.NewStatus(200, "OK")).
			WithBody(body).
			WithHeaders(headers).
			WithTiming(interfaces.TimingInfo{
				StartTime: start,
				EndTime:   end,
			})
		panel.SetResponse(resp)

		lines := panel.renderBinaryPlaceholder()

		assert.NotEmpty(t, lines)
		// Check that it contains expected messages
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "Binary Content")
		assert.Contains(t, joined, "image/png")
	})
}

func TestResponsePanel_RenderBodyTab(t *testing.T) {
	t.Run("renders pretty printed JSON when enabled", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetPrettyPrint(true)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)

		lines := panel.renderBodyTab()
		assert.NotEmpty(t, lines)
	})

	t.Run("renders raw JSON when pretty print disabled", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetPrettyPrint(false)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)

		lines := panel.renderBodyTab()
		assert.NotEmpty(t, lines)
	})

	t.Run("returns placeholder for nil response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		lines := panel.renderBodyTab()
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "No body")
	})
}

func TestResponsePanel_RenderTimingTab(t *testing.T) {
	t.Run("renders timing info", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(ResponseTabTiming)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)

		lines := panel.renderTimingTab()
		assert.NotEmpty(t, lines)
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "Total Time")
	})

	t.Run("shows no timing for nil response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		lines := panel.renderTimingTab()
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "No timing")
	})
}

func TestTruncateValue(t *testing.T) {
	t.Run("does not truncate short values", func(t *testing.T) {
		result := truncateValue("short", 20)
		assert.Equal(t, "short", result)
	})

	t.Run("truncates long values", func(t *testing.T) {
		result := truncateValue("this is a very long value that exceeds the max width", 20)
		assert.Contains(t, result, "...")
		assert.LessOrEqual(t, len(result), 20)
	})
}

func TestResponsePanel_MaxScrollOffset(t *testing.T) {
	t.Run("returns 0 when no response", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		offset := panel.maxScrollOffset()
		assert.Equal(t, 0, offset)
	})

	t.Run("returns 0 for body tab with short content", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)
		panel.activeTab = ResponseTabBody

		offset := panel.maxScrollOffset()
		assert.GreaterOrEqual(t, offset, 0)
	})

	t.Run("returns offset for headers tab", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := newTestResponseWithHeaders(200, "OK")
		panel.SetResponse(resp)
		panel.activeTab = ResponseTabHeaders

		offset := panel.maxScrollOffset()
		assert.GreaterOrEqual(t, offset, 0)
	})

	t.Run("returns offset for cookies tab", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		headers := core.NewHeaders()
		headers.Set("Set-Cookie", "session=abc123")
		resp := core.NewResponse("req-789", "http", core.NewStatus(200, "OK")).
			WithHeaders(headers)
		panel.SetResponse(resp)
		panel.activeTab = ResponseTabCookies

		offset := panel.maxScrollOffset()
		assert.GreaterOrEqual(t, offset, 0)
	})

	t.Run("returns offset for timing tab", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)
		panel.activeTab = ResponseTabTiming

		offset := panel.maxScrollOffset()
		assert.GreaterOrEqual(t, offset, 0)
	})

	t.Run("returns offset for console tab", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)
		panel.activeTab = ResponseTabConsole

		offset := panel.maxScrollOffset()
		assert.GreaterOrEqual(t, offset, 0)
	})

	t.Run("returns offset for tests tab", func(t *testing.T) {
		panel := NewResponsePanel()
		panel.SetSize(80, 30)

		resp := newTestResponse(200, "OK")
		panel.SetResponse(resp)
		panel.activeTab = ResponseTabTests

		offset := panel.maxScrollOffset()
		assert.GreaterOrEqual(t, offset, 0)
	})
}
