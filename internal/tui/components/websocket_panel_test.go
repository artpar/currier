package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
	"github.com/artpar/currier/internal/tui"
	"github.com/stretchr/testify/assert"
)

func TestNewWebSocketPanel(t *testing.T) {
	t.Run("creates panel with defaults", func(t *testing.T) {
		panel := NewWebSocketPanel()
		assert.NotNil(t, panel)
		assert.Equal(t, "WebSocket", panel.Title())
		assert.False(t, panel.Focused())
		assert.Equal(t, WebSocketTabMessages, panel.ActiveTab())
		assert.True(t, panel.AutoScroll())
	})

	t.Run("starts disconnected", func(t *testing.T) {
		panel := NewWebSocketPanel()
		assert.Equal(t, interfaces.ConnectionStateDisconnected, panel.ConnectionState())
	})

	t.Run("starts with no messages", func(t *testing.T) {
		panel := NewWebSocketPanel()
		assert.Equal(t, 0, panel.MessageCount())
	})
}

func TestWebSocketPanel_Init(t *testing.T) {
	panel := NewWebSocketPanel()
	cmd := panel.Init()
	assert.Nil(t, cmd)
}

func TestWebSocketPanel_FocusBlur(t *testing.T) {
	t.Run("Focus sets focused state", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.Focus()
		assert.True(t, panel.Focused())
	})

	t.Run("Blur removes focused state", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.Focus()
		panel.Blur()
		assert.False(t, panel.Focused())
	})

	t.Run("Blur exits input mode", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		// Enter input mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		panel.Update(msg)

		panel.Blur()

		assert.False(t, panel.IsInputMode())
	})
}

func TestWebSocketPanel_SetSize(t *testing.T) {
	panel := NewWebSocketPanel()
	panel.SetSize(100, 50)
	assert.Equal(t, 100, panel.Width())
	assert.Equal(t, 50, panel.Height())
}

func TestWebSocketPanel_Definition(t *testing.T) {
	t.Run("SetDefinition updates definition", func(t *testing.T) {
		panel := NewWebSocketPanel()
		def := newTestWSDefinition()

		panel.SetDefinition(def)

		assert.Equal(t, def, panel.Definition())
	})

	t.Run("SetDefinition clears messages", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.AddMessage(newTestWSMessage("test", true))

		panel.SetDefinition(newTestWSDefinition())

		assert.Equal(t, 0, panel.MessageCount())
	})

	t.Run("Title includes definition name", func(t *testing.T) {
		panel := NewWebSocketPanel()
		def := newTestWSDefinition()
		def.Name = "Test WS"

		panel.SetDefinition(def)

		assert.Contains(t, panel.Title(), "Test WS")
	})
}

func TestWebSocketPanel_ConnectionState(t *testing.T) {
	t.Run("SetConnectionState updates state", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetConnectionState(interfaces.ConnectionStateConnected)
		assert.Equal(t, interfaces.ConnectionStateConnected, panel.ConnectionState())
	})

	t.Run("SetConnectionID updates ID", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetConnectionID("conn-123")
		assert.Equal(t, "conn-123", panel.ConnectionID())
	})
}

func TestWebSocketPanel_Messages(t *testing.T) {
	t.Run("AddMessage adds to list", func(t *testing.T) {
		panel := NewWebSocketPanel()
		msg := newTestWSMessage("hello", true)

		panel.AddMessage(msg)

		assert.Equal(t, 1, panel.MessageCount())
		assert.Equal(t, msg, panel.Messages()[0])
	})

	t.Run("ClearMessages empties list", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.AddMessage(newTestWSMessage("hello", true))
		panel.AddMessage(newTestWSMessage("world", false))

		panel.ClearMessages()

		assert.Equal(t, 0, panel.MessageCount())
	})
}

func TestWebSocketPanel_InputText(t *testing.T) {
	t.Run("SetInputText updates text and cursor", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetInputText("test message")

		assert.Equal(t, "test message", panel.InputText())
	})
}

func TestWebSocketPanel_Tabs(t *testing.T) {
	t.Run("SetActiveTab changes tab", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetActiveTab(WebSocketTabConnection)
		assert.Equal(t, WebSocketTabConnection, panel.ActiveTab())
		assert.Equal(t, "Connection", panel.ActiveTabName())
	})

	t.Run("cycles through tabs with ]", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, WebSocketTabConnection, panel.ActiveTab())
	})

	t.Run("cycles backwards with [", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabConnection)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, WebSocketTabMessages, panel.ActiveTab())
	})

	t.Run("wraps from last to first tab", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabAutoResponse)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, WebSocketTabMessages, panel.ActiveTab())
	})
}

func TestWebSocketPanel_Navigation(t *testing.T) {
	t.Run("scrolls down with j", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 50)
		panel.Focus()
		panel.SetSize(80, 30)
		panel.SetAutoScroll(false)

		// Scroll down several times
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		panel.Update(msg)
		panel.Update(msg)
		panel.Update(msg)

		// View should render without error
		view := panel.View()
		assert.NotEmpty(t, view)
	})

	t.Run("scrolls up with k", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 20)
		panel.Focus()
		panel.SetSize(80, 30)

		// First scroll down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		panel.Update(msg)
		panel.Update(msg)

		// Then scroll up
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.False(t, panel.AutoScroll())
	})

	t.Run("G goes to bottom", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 20)
		panel.Focus()
		panel.SetSize(80, 30)
		panel.SetAutoScroll(false)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.True(t, panel.AutoScroll())
	})

	t.Run("gg goes to top", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 20)
		panel.Focus()
		panel.SetSize(80, 30)

		// Press g twice
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)
		assert.True(t, panel.GPressed())

		updated, _ = panel.Update(msg)
		panel = updated.(*WebSocketPanel)
		assert.False(t, panel.GPressed())
		assert.False(t, panel.AutoScroll())
	})

	t.Run("page down with Ctrl+D", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 50)
		panel.Focus()
		panel.SetSize(80, 30)

		msg := tea.KeyMsg{Type: tea.KeyCtrlD}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.False(t, panel.GPressed())
	})

	t.Run("page up with Ctrl+U", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 50)
		panel.Focus()
		panel.SetSize(80, 30)

		// First scroll down
		msg := tea.KeyMsg{Type: tea.KeyCtrlD}
		panel.Update(msg)

		// Then page up
		msg = tea.KeyMsg{Type: tea.KeyCtrlU}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.False(t, panel.AutoScroll())
	})
}

func TestWebSocketPanel_InputMode(t *testing.T) {
	t.Run("i enters input mode", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.True(t, panel.IsInputMode())
	})

	t.Run("Esc exits input mode", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)

		// Enter input mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		// Exit with Esc
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ = panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.False(t, panel.IsInputMode())
	})

	t.Run("typing in input mode adds text", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		// Enter input mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		// Type some text
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}}
		updated, _ = panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, "hello", panel.InputText())
	})

	t.Run("backspace deletes text", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		// Enter input mode and type
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		panel.Update(msg)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		// Backspace
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ = panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, "hell", panel.InputText())
	})

	t.Run("Ctrl+U clears input", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		// Enter input mode and type
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		panel.Update(msg)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		// Clear with Ctrl+U
		msg = tea.KeyMsg{Type: tea.KeyCtrlU}
		updated, _ = panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, "", panel.InputText())
	})

	t.Run("cursor movement with arrows", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		// Enter input mode and type
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		panel.Update(msg)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}}
		panel.Update(msg)

		// Move left
		msg = tea.KeyMsg{Type: tea.KeyLeft}
		panel.Update(msg)

		// Move right
		msg = tea.KeyMsg{Type: tea.KeyRight}
		panel.Update(msg)

		// Home
		msg = tea.KeyMsg{Type: tea.KeyHome}
		panel.Update(msg)

		// End
		msg = tea.KeyMsg{Type: tea.KeyEnd}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, "hello", panel.InputText())
	})

	t.Run("Delete key removes character", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		// Enter input mode and type
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		panel.Update(msg)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}}
		panel.Update(msg)

		// Move to beginning
		msg = tea.KeyMsg{Type: tea.KeyHome}
		panel.Update(msg)

		// Delete
		msg = tea.KeyMsg{Type: tea.KeyDelete}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, "ello", panel.InputText())
	})
}

func TestWebSocketPanel_SendMessage(t *testing.T) {
	t.Run("Enter sends message when connected", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)
		panel.SetConnectionState(interfaces.ConnectionStateConnected)
		panel.SetInputText("test message")

		// Enter input mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		panel.Update(msg)

		// Send with Enter
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, cmd := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.NotNil(t, cmd)
		assert.Equal(t, "", panel.InputText())
	})

	t.Run("Enter does nothing when disconnected", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetActiveTab(WebSocketTabMessages)
		panel.SetConnectionState(interfaces.ConnectionStateDisconnected)
		panel.SetInputText("test message")

		// Enter input mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		panel.Update(msg)

		// Try to send
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		// Message should still be in input
		assert.Equal(t, "test message", panel.InputText())
	})
}

func TestWebSocketPanel_ConnectionCommands(t *testing.T) {
	t.Run("c connects when disconnected", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetConnectionState(interfaces.ConnectionStateDisconnected)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		_, cmd := panel.Update(msg)

		assert.NotNil(t, cmd)
	})

	t.Run("d disconnects when connected", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd := panel.Update(msg)

		assert.NotNil(t, cmd)
	})

	t.Run("Ctrl+C disconnects when connected", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := panel.Update(msg)

		assert.NotNil(t, cmd)
	})

	t.Run("Ctrl+R reconnects", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.Focus()

		msg := tea.KeyMsg{Type: tea.KeyCtrlR}
		_, cmd := panel.Update(msg)

		assert.NotNil(t, cmd)
	})
}

func TestWebSocketPanel_CopyMessage(t *testing.T) {
	t.Run("y copies last message", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 5)
		panel.Focus()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		_, cmd := panel.Update(msg)

		assert.NotNil(t, cmd)
	})
}

func TestWebSocketPanel_View(t *testing.T) {
	t.Run("renders empty view with zero size", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetSize(0, 0)
		assert.Empty(t, panel.View())
	})

	t.Run("renders Messages tab", func(t *testing.T) {
		panel := newTestWebSocketPanelWithMessages(t, 3)
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabMessages)

		view := panel.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Messages")
	})

	t.Run("renders Connection tab", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabConnection)

		view := panel.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Connection")
		assert.Contains(t, view, "Endpoint")
	})

	t.Run("renders Scripts tab", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabScripts)

		view := panel.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Scripts")
	})

	t.Run("renders Auto-Response tab", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabAutoResponse)

		view := panel.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Auto-Response")
	})

	t.Run("shows no messages hint when empty", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabMessages)

		view := panel.View()

		assert.Contains(t, view, "No messages")
	})

	t.Run("shows connection status", func(t *testing.T) {
		panel := newTestWebSocketPanel(t)
		panel.SetSize(80, 30)
		panel.SetConnectionState(interfaces.ConnectionStateConnected)

		view := panel.View()

		assert.Contains(t, view, "Connected")
	})
}

func TestWebSocketPanel_UpdateMessages(t *testing.T) {
	t.Run("handles WSConnectedMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()

		msg := WSConnectedMsg{ConnectionID: "conn-123"}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, "conn-123", panel.ConnectionID())
		assert.Equal(t, interfaces.ConnectionStateConnected, panel.ConnectionState())
	})

	t.Run("handles WSDisconnectedMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetConnectionState(interfaces.ConnectionStateConnected)
		panel.SetConnectionID("conn-123")

		msg := WSDisconnectedMsg{ConnectionID: "conn-123"}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, "", panel.ConnectionID())
		assert.Equal(t, interfaces.ConnectionStateDisconnected, panel.ConnectionState())
	})

	t.Run("handles WSStateChangedMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()

		msg := WSStateChangedMsg{State: interfaces.ConnectionStateConnecting}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, interfaces.ConnectionStateConnecting, panel.ConnectionState())
	})

	t.Run("handles WSMessageReceivedMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()

		wsMsg := newTestWSMessage("hello", false)
		msg := WSMessageReceivedMsg{Message: wsMsg}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, 1, panel.MessageCount())
	})

	t.Run("handles WSMessageSentMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()

		wsMsg := newTestWSMessage("hello", true)
		msg := WSMessageSentMsg{Message: wsMsg}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, 1, panel.MessageCount())
	})

	t.Run("handles WindowSizeMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()

		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.Equal(t, 120, panel.Width())
		assert.Equal(t, 40, panel.Height())
	})

	t.Run("handles FocusMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()

		msg := tui.FocusMsg{}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.True(t, panel.Focused())
	})

	t.Run("handles BlurMsg", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.Focus()

		msg := tui.BlurMsg{}
		updated, _ := panel.Update(msg)
		panel = updated.(*WebSocketPanel)

		assert.False(t, panel.Focused())
	})
}

func TestWebSocketPanel_AutoScroll(t *testing.T) {
	t.Run("SetAutoScroll updates state", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetAutoScroll(false)
		assert.False(t, panel.AutoScroll())
		panel.SetAutoScroll(true)
		assert.True(t, panel.AutoScroll())
	})
}

func TestWebSocketPanel_StatusColors(t *testing.T) {
	testCases := []struct {
		state interfaces.ConnectionState
		text  string
	}{
		{interfaces.ConnectionStateConnected, "Connected"},
		{interfaces.ConnectionStateConnecting, "Connecting"},
		{interfaces.ConnectionStateDisconnecting, "Disconnecting"},
		{interfaces.ConnectionStateError, "Error"},
		{interfaces.ConnectionStateDisconnected, "Disconnected"},
	}

	for _, tc := range testCases {
		t.Run(tc.text, func(t *testing.T) {
			panel := newTestWebSocketPanel(t)
			panel.SetSize(80, 30)
			panel.SetConnectionState(tc.state)

			view := panel.View()

			assert.Contains(t, view, tc.text)
		})
	}
}

func TestWebSocketPanel_DefinitionWithScripts(t *testing.T) {
	t.Run("shows scripts in Scripts tab", func(t *testing.T) {
		panel := NewWebSocketPanel()
		def := newTestWSDefinition()
		def.PreConnectScript = "console.log('pre-connect')"
		def.PreMessageScript = "console.log('pre-message')"
		def.PostMessageScript = "console.log('post-message')"
		def.FilterScript = "return true"

		panel.SetDefinition(def)
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabScripts)

		view := panel.View()

		assert.Contains(t, view, "Pre-Connect")
		assert.Contains(t, view, "Pre-Message")
		assert.Contains(t, view, "Post-Message")
		assert.Contains(t, view, "Filter")
	})
}

func TestWebSocketPanel_DefinitionWithAutoResponse(t *testing.T) {
	t.Run("shows auto-response rules", func(t *testing.T) {
		panel := NewWebSocketPanel()
		def := newTestWSDefinition()
		def.AutoResponseRules = []core.AutoResponseRule{
			{
				Name:        "Ping Pong",
				Enabled:     true,
				MatchScript: "msg.includes('ping')",
				Response:    "pong",
			},
		}

		panel.SetDefinition(def)
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabAutoResponse)

		view := panel.View()

		assert.Contains(t, view, "Ping Pong")
	})
}

func TestWebSocketPanel_NoDefinition(t *testing.T) {
	t.Run("Connection tab shows no definition message", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabConnection)

		view := panel.View()

		assert.Contains(t, view, "No WebSocket definition")
	})

	t.Run("Scripts tab shows no definition message", func(t *testing.T) {
		panel := NewWebSocketPanel()
		panel.SetSize(80, 30)
		panel.SetActiveTab(WebSocketTabScripts)

		view := panel.View()

		assert.Contains(t, view, "No WebSocket definition")
	})
}

// Helper functions

func newTestWebSocketPanel(t *testing.T) *WebSocketPanel {
	t.Helper()
	panel := NewWebSocketPanel()
	panel.SetDefinition(newTestWSDefinition())
	return panel
}

func newTestWebSocketPanelWithMessages(t *testing.T, count int) *WebSocketPanel {
	t.Helper()
	panel := newTestWebSocketPanel(t)
	for i := 0; i < count; i++ {
		sent := i%2 == 0
		msg := newTestWSMessage("message "+string(rune('0'+i%10)), sent)
		panel.AddMessage(msg)
	}
	return panel
}

func newTestWSDefinition() *core.WebSocketDefinition {
	return &core.WebSocketDefinition{
		ID:           "ws-test-123",
		Name:         "Test WebSocket",
		Endpoint:     "wss://example.com/ws",
		Headers:      map[string]string{"Authorization": "Bearer token"},
		Subprotocols: []string{"v1"},
		PingInterval: 30,
	}
}

func newTestWSMessage(content string, sent bool) *core.WebSocketMessage {
	direction := "received"
	if sent {
		direction = "sent"
	}
	return &core.WebSocketMessage{
		ID:        "msg-123",
		Content:   content,
		Direction: direction,
		Timestamp: time.Now(),
	}
}

func TestTruncateScript(t *testing.T) {
	t.Run("truncates long script", func(t *testing.T) {
		longScript := "console.log('this is a very long script that should be truncated')"
		result := truncateScript(longScript, 20)
		assert.Len(t, result, 20)
		assert.True(t, len(result) <= 20)
	})

	t.Run("removes newlines", func(t *testing.T) {
		script := "line1\nline2\nline3"
		result := truncateScript(script, 50)
		assert.NotContains(t, result, "\n")
	})
}
