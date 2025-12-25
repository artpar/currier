package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestNewMainView(t *testing.T) {
	t.Run("creates main view", func(t *testing.T) {
		view := NewMainView()
		assert.NotNil(t, view)
		assert.Equal(t, "Main", view.Name())
	})

	t.Run("has three panes", func(t *testing.T) {
		view := NewMainView()
		assert.NotNil(t, view.CollectionTree())
		assert.NotNil(t, view.RequestPanel())
		assert.NotNil(t, view.ResponsePanel())
	})

	t.Run("starts with collection tree focused", func(t *testing.T) {
		view := NewMainView()
		assert.Equal(t, PaneCollections, view.FocusedPane())
	})
}

func TestMainView_Layout(t *testing.T) {
	t.Run("sets size on window resize", func(t *testing.T) {
		view := NewMainView()
		msg := tea.WindowSizeMsg{Width: 120, Height: 40}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 120, view.Width())
		assert.Equal(t, 40, view.Height())
	})

	t.Run("distributes width to panes", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Collection tree should get left portion
		tree := view.CollectionTree()
		assert.Greater(t, tree.Width(), 0)

		// Request panel should get middle portion
		request := view.RequestPanel()
		assert.Greater(t, request.Width(), 0)

		// Response panel should get right portion
		response := view.ResponsePanel()
		assert.Greater(t, response.Width(), 0)
	})
}

func TestMainView_PaneFocus(t *testing.T) {
	t.Run("cycles focus forward with Tab", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneRequest, view.FocusedPane())
	})

	t.Run("cycles focus backward with Shift+Tab", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneCollections, view.FocusedPane())
	})

	t.Run("wraps focus from last to first", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneResponse)

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneCollections, view.FocusedPane())
	})

	t.Run("focuses pane with number keys", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Press 2 to focus request pane
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneRequest, view.FocusedPane())
	})

	t.Run("focuses pane 3 with number key", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneResponse, view.FocusedPane())
	})
}

func TestMainView_RequestSelection(t *testing.T) {
	t.Run("loads request into request panel on selection", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test Request", "GET", "https://example.com")
		msg := components.SelectionMsg{Request: req}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, req, view.RequestPanel().Request())
	})

	t.Run("focuses request panel after selection", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test Request", "GET", "https://example.com")
		msg := components.SelectionMsg{Request: req}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneRequest, view.FocusedPane())
	})
}

func TestMainView_SendRequest(t *testing.T) {
	t.Run("shows loading state when request sent", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)

		msg := components.SendRequestMsg{Request: req}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.ResponsePanel().IsLoading())
	})
}

func TestMainView_View(t *testing.T) {
	t.Run("renders three panes", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		output := view.View()

		// Should contain elements from all panes
		assert.Contains(t, output, "Collections")
		assert.Contains(t, output, "Request")
		assert.Contains(t, output, "Response")
	})

	t.Run("highlights focused pane", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		output := view.View()

		// Output should contain the view
		assert.NotEmpty(t, output)
	})
}

func TestMainView_SetCollections(t *testing.T) {
	t.Run("sets collections on tree", func(t *testing.T) {
		view := NewMainView()

		c1 := core.NewCollection("API 1")
		c2 := core.NewCollection("API 2")
		view.SetCollections([]*core.Collection{c1, c2})

		tree := view.CollectionTree()
		assert.Equal(t, 2, tree.ItemCount())
	})
}

func TestMainView_Quit(t *testing.T) {
	t.Run("quits on Ctrl+C", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := view.Update(msg)

		// Should return quit command
		assert.NotNil(t, cmd)
	})

	t.Run("quits on q in normal mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, cmd := view.Update(msg)

		assert.NotNil(t, cmd)
	})
}

func TestMainView_Help(t *testing.T) {
	t.Run("toggles help overlay on ?", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.ShowingHelp())
	})

	t.Run("hides help on Escape", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ShowHelp()

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.ShowingHelp())
	})

	t.Run("renders help content", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ShowHelp()

		output := view.View()
		assert.Contains(t, output, "Help")
		assert.Contains(t, output, "Navigation")
	})

	t.Run("HideHelp method works", func(t *testing.T) {
		view := NewMainView()
		view.ShowHelp()
		view.HideHelp()
		assert.False(t, view.ShowingHelp())
	})
}

func TestMainView_Init(t *testing.T) {
	t.Run("Init returns nil", func(t *testing.T) {
		view := NewMainView()
		cmd := view.Init()
		assert.Nil(t, cmd)
	})
}

func TestMainView_Title(t *testing.T) {
	t.Run("returns Currier title", func(t *testing.T) {
		view := NewMainView()
		assert.Equal(t, "Currier", view.Title())
	})
}

func TestMainView_FocusBlur(t *testing.T) {
	t.Run("Focused returns true", func(t *testing.T) {
		view := NewMainView()
		assert.True(t, view.Focused())
	})

	t.Run("Focus is no-op", func(t *testing.T) {
		view := NewMainView()
		view.Focus()
		assert.True(t, view.Focused())
	})

	t.Run("Blur is no-op", func(t *testing.T) {
		view := NewMainView()
		view.Blur()
		assert.True(t, view.Focused())
	})
}

func TestMainView_Environment(t *testing.T) {
	t.Run("returns nil environment initially", func(t *testing.T) {
		view := NewMainView()
		assert.Nil(t, view.Environment())
	})

	t.Run("returns interpolator", func(t *testing.T) {
		view := NewMainView()
		assert.NotNil(t, view.Interpolator())
	})

	t.Run("SetEnvironment sets environment and interpolator", func(t *testing.T) {
		view := NewMainView()
		env := core.NewEnvironment("Test Env")
		env.SetVariable("base_url", "https://api.example.com")

		view.SetEnvironment(env, nil)
		assert.Equal(t, env, view.Environment())
	})
}

func TestMainView_ForwardMessages(t *testing.T) {
	t.Run("forwards messages to collection tree when focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)

		// Add a collection so we have items to navigate
		c := core.NewCollection("Test")
		c.AddFolder("Folder")
		view.SetCollections([]*core.Collection{c})

		// Send j key to move cursor down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		view.Update(msg)

		// No error means message was forwarded
		assert.Equal(t, PaneCollections, view.FocusedPane())
	})

	t.Run("forwards messages to request panel when focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)

		// Send j key (not Tab, which cycles panes)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		view.Update(msg)

		// No error means message was forwarded
		assert.Equal(t, PaneRequest, view.FocusedPane())
	})

	t.Run("forwards messages to response panel when focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneResponse)

		// Send j key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		view.Update(msg)

		// No error means message was forwarded
		assert.Equal(t, PaneResponse, view.FocusedPane())
	})
}

func TestMainView_ResponseReceived(t *testing.T) {
	t.Run("clears loading state on response", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ResponsePanel().SetLoading(true)

		msg := components.ResponseReceivedMsg{Response: nil}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.ResponsePanel().IsLoading())
	})
}

func TestMainView_RequestError(t *testing.T) {
	t.Run("clears loading state on error", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ResponsePanel().SetLoading(true)

		msg := components.RequestErrorMsg{Error: assert.AnError}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.ResponsePanel().IsLoading())
	})
}

func TestMainView_EscapeKey(t *testing.T) {
	t.Run("Escape does nothing in normal mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneCollections, view.FocusedPane())
	})
}

func TestMainView_EmptyView(t *testing.T) {
	t.Run("returns empty string with zero size", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(0, 0)

		output := view.View()
		assert.Empty(t, output)
	})
}

func TestMainView_StatusBar(t *testing.T) {
	t.Run("shows environment in status bar", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		env := core.NewEnvironment("Production")
		env.SetVariable("base_url", "https://api.example.com")
		env.SetSecret("api_key", "secret")
		view.SetEnvironment(env, nil)

		output := view.View()
		assert.Contains(t, output, "Production")
	})

	t.Run("shows no environment message when nil", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		output := view.View()
		assert.Contains(t, output, "No Environment")
	})
}

func TestMainView_CopyKey(t *testing.T) {
	t.Run("y key triggers copy", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneResponse)

		// Set up a response to copy
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify it doesn't panic - actual clipboard testing is difficult
		assert.Equal(t, PaneResponse, view.FocusedPane())
	})
}

func TestMainView_SendRequestKey(t *testing.T) {
	t.Run("Enter key in request pane", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := view.Update(msg)

		// Should trigger some command for send (or nil if no request)
		_ = cmd // Just verify it doesn't panic
	})
}

func TestMainView_StatusBarVariableCount(t *testing.T) {
	t.Run("shows variable count in status bar", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		env := core.NewEnvironment("Dev")
		env.SetVariable("var1", "value1")
		env.SetVariable("var2", "value2")
		view.SetEnvironment(env, nil)

		output := view.View()
		assert.Contains(t, output, "2 vars")
	})

	t.Run("shows secret count in status bar", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		env := core.NewEnvironment("Dev")
		env.SetSecret("api_key", "secret")
		view.SetEnvironment(env, nil)

		output := view.View()
		assert.Contains(t, output, "1 secrets")
	})
}

func TestMainView_UpdatePaneSizes(t *testing.T) {
	t.Run("updates pane sizes on resize", func(t *testing.T) {
		view := NewMainView()

		// Set initial size
		view.SetSize(120, 40)

		// Resize
		view.SetSize(200, 60)

		// Panes should have updated sizes
		assert.Equal(t, 200, view.Width())
		assert.Equal(t, 60, view.Height())
	})
}

func TestMainView_ClearNotification(t *testing.T) {
	t.Run("handles clear notification message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// The clearNotificationMsg type is private, so we test via the public API
		// Just verify the view renders without notification
		output := view.View()
		assert.NotEmpty(t, output)
	})
}

func TestMainView_NumberKeyFocus(t *testing.T) {
	t.Run("1 focuses collection pane", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneCollections, view.FocusedPane())
	})

	t.Run("2 focuses request pane", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneRequest, view.FocusedPane())
	})

	t.Run("3 focuses response pane", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneResponse, view.FocusedPane())
	})
}


func TestMainView_StatusBarEdgeCases(t *testing.T) {
	t.Run("handles narrow width", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(40, 20) // Very narrow

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("shows help hint", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		output := view.View()
		assert.Contains(t, output, "help")
		assert.Contains(t, output, "quit")
	})
}

func TestMainView_HelpBar(t *testing.T) {
	t.Run("shows collection hints when collections pane focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)

		output := view.View()
		assert.Contains(t, output, "Navigate")
		assert.Contains(t, output, "Search")
	})

	t.Run("shows request hints when request pane focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		output := view.View()
		assert.Contains(t, output, "Edit URL")
		assert.Contains(t, output, "Method")
		assert.Contains(t, output, "Send")
	})

	t.Run("shows response hints when response pane focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneResponse)

		output := view.View()
		assert.Contains(t, output, "Scroll")
		assert.Contains(t, output, "Copy")
	})

	t.Run("shows global hints", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		output := view.View()
		assert.Contains(t, output, "Pane")
		assert.Contains(t, output, "Help")
		assert.Contains(t, output, "Quit")
	})
}

func TestMainView_ModeIndicator(t *testing.T) {
	t.Run("shows NORMAL mode by default", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		output := view.View()
		assert.Contains(t, output, "NORMAL")
	})

	t.Run("shows pane name in status bar", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		output := view.View()
		assert.Contains(t, output, "Request")
	})

	t.Run("shows Collections when collections focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)

		output := view.View()
		assert.Contains(t, output, "Collections")
	})

	t.Run("shows Response when response focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneResponse)

		output := view.View()
		assert.Contains(t, output, "Response")
	})
}

func TestMainView_NewRequest(t *testing.T) {
	t.Run("n key creates new request and enters URL edit mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Initially no request
		assert.Nil(t, view.RequestPanel().Request())

		// Press 'n' to create new request
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should have a new request
		assert.NotNil(t, view.RequestPanel().Request())

		// Should be in edit mode
		assert.True(t, view.RequestPanel().IsEditing())

		// Should focus request pane
		assert.Equal(t, PaneRequest, view.FocusedPane())
	})
}

func TestMainView_AutoSelectFirstRequest(t *testing.T) {
	t.Run("auto-selects first request when collections are loaded", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Initially no request
		assert.Nil(t, view.RequestPanel().Request())

		// Create collection with a request
		col := core.NewCollection("Test API")
		req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		col.AddRequest(req)

		// Set collections
		view.SetCollections([]*core.Collection{col})

		// Should auto-select the first request
		assert.NotNil(t, view.RequestPanel().Request())
		assert.Equal(t, "Get Users", view.RequestPanel().Request().Name())
	})

	t.Run("auto-selects first request in folder if no root requests", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Create collection with request in folder
		col := core.NewCollection("Test API")
		folder := col.AddFolder("Users")
		req := core.NewRequestDefinition("List Users", "GET", "https://api.example.com/users")
		folder.AddRequest(req)

		// Set collections
		view.SetCollections([]*core.Collection{col})

		// Should auto-select the first request from folder
		assert.NotNil(t, view.RequestPanel().Request())
		assert.Equal(t, "List Users", view.RequestPanel().Request().Name())
	})
}

func TestMainView_HelpBarShowsNewHint(t *testing.T) {
	t.Run("help bar shows n for New", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		output := view.View()
		assert.Contains(t, output, "n")
		assert.Contains(t, output, "New")
	})
}
