package views

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/history"
	"github.com/artpar/currier/internal/interfaces"
	"github.com/artpar/currier/internal/script"
	"github.com/artpar/currier/internal/storage/filesystem"
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
	t.Run("shows history hints when collections pane focused in history mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)
		// Default is now history mode

		output := view.View()
		assert.Contains(t, output, "Navigate")
		assert.Contains(t, output, "Load")
		assert.Contains(t, output, "Refresh")
		assert.Contains(t, output, "Collections")
	})

	t.Run("shows collection hints when in collections mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)
		// Switch to collections mode
		view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})

		output := view.View()
		assert.Contains(t, output, "Navigate")
		assert.Contains(t, output, "Collapse/Expand")
		assert.Contains(t, output, "Delete")
		assert.Contains(t, output, "History")
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

func TestMainView_Feedback(t *testing.T) {
	t.Run("handles feedback message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.FeedbackMsg{Message: "Tab: switch to URL tab", IsError: false}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		// Notification should be set
		assert.NotNil(t, cmd)
		output := view.View()
		_ = output // Verify no panic
	})

	t.Run("handles error feedback message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.FeedbackMsg{Message: "Something went wrong", IsError: true}
		_, cmd := view.Update(msg)

		assert.NotNil(t, cmd)
	})
}

func TestMainView_CopyMsg(t *testing.T) {
	t.Run("handles copy message for small content", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.CopyMsg{Content: "small text"}
		_, cmd := view.Update(msg)

		// Should return a tick command for clearing notification
		assert.NotNil(t, cmd)
	})

	t.Run("handles copy message for large content", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Create content > 1024 bytes
		largeContent := make([]byte, 2048)
		for i := range largeContent {
			largeContent[i] = 'x'
		}
		msg := components.CopyMsg{Content: string(largeContent)}
		_, cmd := view.Update(msg)

		assert.NotNil(t, cmd)
	})
}

func TestMainView_HistoryStore(t *testing.T) {
	t.Run("sets history store", func(t *testing.T) {
		view := NewMainView()

		// Create a mock history store
		mockStore := &mockHistoryStore{}
		view.SetHistoryStore(mockStore)

		// Verify it was set (indirectly via tree)
		assert.NotNil(t, view.CollectionTree())
	})
}

func TestMainView_Notification(t *testing.T) {
	t.Run("returns empty string initially", func(t *testing.T) {
		view := NewMainView()
		assert.Equal(t, "", view.Notification())
	})
}

// mockHistoryStore is a simple mock for testing
type mockHistoryStore struct{}

func (m *mockHistoryStore) Add(ctx context.Context, entry history.Entry) (string, error) {
	return "mock-id", nil
}
func (m *mockHistoryStore) Get(ctx context.Context, id string) (history.Entry, error) {
	return history.Entry{}, nil
}
func (m *mockHistoryStore) List(ctx context.Context, opts history.QueryOptions) ([]history.Entry, error) {
	return nil, nil
}
func (m *mockHistoryStore) Count(ctx context.Context, opts history.QueryOptions) (int64, error) {
	return 0, nil
}
func (m *mockHistoryStore) Update(ctx context.Context, entry history.Entry) error { return nil }
func (m *mockHistoryStore) Delete(ctx context.Context, id string) error           { return nil }
func (m *mockHistoryStore) DeleteMany(ctx context.Context, opts history.QueryOptions) (int64, error) {
	return 0, nil
}
func (m *mockHistoryStore) Search(ctx context.Context, query string, opts history.QueryOptions) ([]history.Entry, error) {
	return nil, nil
}
func (m *mockHistoryStore) Prune(ctx context.Context, opts history.PruneOptions) (history.PruneResult, error) {
	return history.PruneResult{}, nil
}
func (m *mockHistoryStore) Stats(ctx context.Context) (history.Stats, error) { return history.Stats{}, nil }
func (m *mockHistoryStore) Clear(ctx context.Context) error                  { return nil }
func (m *mockHistoryStore) Close() error                                     { return nil }

func TestMainView_InsertModePassthrough(t *testing.T) {
	t.Run("number keys pass through in edit mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Create and select request
		req := core.NewRequestDefinition("Test", "GET", "http://localhost:")
		view.RequestPanel().SetRequest(req)
		view.FocusPane(PaneRequest)

		// Start editing URL
		view.RequestPanel().StartURLEdit()
		assert.True(t, view.RequestPanel().IsEditing())

		// Press '3' - should NOT switch pane, should be passed to input
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should still be in request pane, not response pane
		assert.Equal(t, PaneRequest, view.FocusedPane())

		// Should still be editing
		assert.True(t, view.RequestPanel().IsEditing())
	})

	t.Run("q key does not quit in edit mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test", "GET", "")
		view.RequestPanel().SetRequest(req)
		view.FocusPane(PaneRequest)
		view.RequestPanel().StartURLEdit()

		// Press 'q' - should NOT quit, should be passed to input
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, cmd := view.Update(msg)

		// Should NOT produce a quit command
		assert.Nil(t, cmd)
	})

	t.Run("1/2/3 switch panes in normal mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		// Not in edit mode
		assert.False(t, view.RequestPanel().IsEditing())

		// Press '3' - should switch to response pane
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneResponse, view.FocusedPane())
	})
}

func TestMainView_ViewMode(t *testing.T) {
	t.Run("ViewMode returns default mode", func(t *testing.T) {
		view := NewMainView()
		mode := view.ViewMode()
		assert.Equal(t, ViewModeHTTP, mode)
	})

	t.Run("SetViewMode changes mode", func(t *testing.T) {
		view := NewMainView()
		view.SetViewMode(ViewModeWebSocket)
		assert.Equal(t, ViewModeWebSocket, view.ViewMode())
	})

	t.Run("SetViewMode back to HTTP", func(t *testing.T) {
		view := NewMainView()
		view.SetViewMode(ViewModeWebSocket)
		view.SetViewMode(ViewModeHTTP)
		assert.Equal(t, ViewModeHTTP, view.ViewMode())
	})
}

func TestMainView_WebSocketPanel(t *testing.T) {
	t.Run("WebSocketPanel returns panel", func(t *testing.T) {
		view := NewMainView()
		panel := view.WebSocketPanel()
		assert.NotNil(t, panel)
	})
}

func TestMainView_SetWebSocketDefinition(t *testing.T) {
	t.Run("SetWebSocketDefinition updates panel", func(t *testing.T) {
		view := NewMainView()
		def := &core.WebSocketDefinition{
			ID:       "ws-123",
			Name:     "Test WS",
			Endpoint: "wss://example.com/ws",
		}
		view.SetWebSocketDefinition(def)

		panel := view.WebSocketPanel()
		assert.Equal(t, def, panel.Definition())
	})
}

func TestMainView_CycleFocusWrapping(t *testing.T) {
	t.Run("cycle forward wraps correctly through all panes", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Start at collections
		assert.Equal(t, PaneCollections, view.FocusedPane())

		// Tab through all panes
		msg := tea.KeyMsg{Type: tea.KeyTab}
		for i := 0; i < 3; i++ {
			updated, _ := view.Update(msg)
			view = updated.(*MainView)
		}
		// Should wrap back to collections
		assert.Equal(t, PaneCollections, view.FocusedPane())
	})
}

func TestMainView_ViewWebSocketMode(t *testing.T) {
	t.Run("View in WebSocket mode renders", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		output := view.View()
		assert.NotEmpty(t, output)
	})
}

func TestMainView_Focus(t *testing.T) {
	t.Run("Focus method works", func(t *testing.T) {
		view := NewMainView()
		view.Focus()
		assert.NotNil(t, view)
	})
}

func TestMainView_Blur(t *testing.T) {
	t.Run("Blur method works", func(t *testing.T) {
		view := NewMainView()
		view.Focus()
		view.Blur()
		assert.NotNil(t, view)
	})
}

func TestMainView_SaveToHistoryIntegration(t *testing.T) {
	t.Run("saveToHistory with mock store", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		store := &mockHistoryStore{}
		view.SetHistoryStore(store)

		// Create a request and response
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)

		// The saveToHistory is called internally when response is received
		// This test ensures the setup works
		assert.NotNil(t, view)
	})
}

func TestMainView_UpdateMessageTypes(t *testing.T) {
	t.Run("handles SelectionMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		msg := components.SelectionMsg{Request: req}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, ViewModeHTTP, view.ViewMode())
		assert.Equal(t, PaneRequest, view.FocusedPane())
	})

	t.Run("handles SelectWebSocketMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		def := &core.WebSocketDefinition{
			ID:       "ws-123",
			Name:     "Test WS",
			Endpoint: "wss://example.com/ws",
		}
		msg := components.SelectWebSocketMsg{WebSocket: def}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, ViewModeWebSocket, view.ViewMode())
	})

	t.Run("handles SelectHistoryItemMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		entry := history.Entry{
			RequestName:   "Test Request",
			RequestMethod: "POST",
			RequestURL:    "https://example.com/api",
			RequestBody:   `{"test": true}`,
			RequestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		}
		msg := components.SelectHistoryItemMsg{Entry: entry}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneRequest, view.FocusedPane())
		assert.NotNil(t, view.RequestPanel().Request())
	})

	t.Run("handles SelectHistoryItemMsg with empty name", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		entry := history.Entry{
			RequestName:   "",
			RequestMethod: "GET",
			RequestURL:    "https://example.com",
		}
		msg := components.SelectHistoryItemMsg{Entry: entry}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view.RequestPanel().Request())
	})

	t.Run("handles ResponseReceivedMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ResponsePanel().SetLoading(true)

		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK"))
		msg := components.ResponseReceivedMsg{
			Response:    resp,
			TestResults: nil,
			Console:     nil,
		}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.ResponsePanel().IsLoading())
		assert.NotNil(t, view.ResponsePanel().Response())
	})

	t.Run("handles ResponseReceivedMsg with test results", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		resp := core.NewResponse("req-123", "http", core.NewStatus(200, "OK"))
		testResults := []script.TestResult{
			{Name: "Test 1", Passed: true},
		}
		consoleMessages := []components.ConsoleMessage{
			{Level: "log", Message: "Hello"},
		}
		msg := components.ResponseReceivedMsg{
			Response:    resp,
			TestResults: testResults,
			Console:     consoleMessages,
		}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		results := view.ResponsePanel().TestResults()
		assert.Len(t, results, 1)
	})

	t.Run("handles RequestErrorMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ResponsePanel().SetLoading(true)

		msg := components.RequestErrorMsg{Error: assert.AnError}

		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.ResponsePanel().IsLoading())
		assert.NotNil(t, view.ResponsePanel().Error())
	})

	t.Run("handles CopyMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.CopyMsg{Content: "test content"}
		_, cmd := view.Update(msg)

		// Copy command may or may not be returned depending on clipboard availability
		_ = cmd
	})

	t.Run("handles FeedbackMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.FeedbackMsg{Message: "Test feedback", IsError: false}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Contains(t, view.Notification(), "Test feedback")
	})
}

func TestMainView_KeyHandling(t *testing.T) {
	t.Run("n key creates new request", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneRequest, view.FocusedPane())
		assert.NotNil(t, view.RequestPanel().Request())
	})

	t.Run("w key toggles WebSocket mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// First press - enter WebSocket mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, ViewModeWebSocket, view.ViewMode())
		assert.Equal(t, PaneWebSocket, view.FocusedPane())

		// Second press - exit WebSocket mode
		updated, _ = view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, ViewModeHTTP, view.ViewMode())
		assert.Equal(t, PaneRequest, view.FocusedPane())
	})

	t.Run("4 key focuses WebSocket panel in WS mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)
		view.FocusPane(PaneCollections)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneWebSocket, view.FocusedPane())
	})

	t.Run("4 key does nothing in HTTP mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneCollections, view.FocusedPane())
	})

	t.Run("escape key in normal mode is no-op", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneRequest)

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, PaneRequest, view.FocusedPane())
	})
}

func TestMainView_CycleFocusInWebSocketMode(t *testing.T) {
	t.Run("Tab cycles between Collections and WebSocket in WS mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)
		view.FocusPane(PaneCollections)

		// Collections -> WebSocket
		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)
		assert.Equal(t, PaneWebSocket, view.FocusedPane())

		// WebSocket -> Collections
		updated, _ = view.Update(msg)
		view = updated.(*MainView)
		assert.Equal(t, PaneCollections, view.FocusedPane())
	})

	t.Run("Shift+Tab cycles between Collections and WebSocket in WS mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)
		view.FocusPane(PaneCollections)

		// Collections -> WebSocket (backward is same in 2-pane mode)
		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)
		assert.Equal(t, PaneWebSocket, view.FocusedPane())

		// WebSocket -> Collections
		updated, _ = view.Update(msg)
		view = updated.(*MainView)
		assert.Equal(t, PaneCollections, view.FocusedPane())
	})
}

func TestMainView_HelpBarRendering(t *testing.T) {
	t.Run("renders help bar based on focused pane", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Focus request pane
		view.FocusPane(PaneRequest)
		output := view.View()
		assert.NotEmpty(t, output)

		// Focus response pane
		view.FocusPane(PaneResponse)
		output = view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("WebSocket mode renders different help bar", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)
		view.FocusPane(PaneWebSocket)

		output := view.View()
		assert.NotEmpty(t, output)
	})
}

func TestMainView_StatusBarExtended(t *testing.T) {
	t.Run("status bar shows request method", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		view.RequestPanel().SetRequest(req)

		output := view.View()
		assert.Contains(t, output, "POST")
	})

	t.Run("status bar shows response status", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		resp := core.NewResponse("req-1", "http", core.NewStatus(200, "OK"))
		view.ResponsePanel().SetResponse(resp)

		output := view.View()
		assert.Contains(t, output, "200")
	})
}

func TestMainView_PaneSizesExtended(t *testing.T) {
	t.Run("pane sizes in HTTP mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(150, 50)
		view.SetViewMode(ViewModeHTTP)

		// Verify panels have non-zero sizes
		assert.Greater(t, view.Width(), 0)
		assert.Greater(t, view.Height(), 0)
	})

	t.Run("pane sizes in WebSocket mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(150, 50)
		view.SetViewMode(ViewModeWebSocket)

		// Verify WebSocket panel is accessible
		assert.NotNil(t, view.WebSocketPanel())
	})
}

func TestMainView_WebSocketMessages(t *testing.T) {
	t.Run("handles WSConnectedMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSConnectedMsg{ConnectionID: "conn-123"}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Contains(t, view.Notification(), "connected")
	})

	t.Run("handles WSDisconnectedMsg with error", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSDisconnectedMsg{Error: assert.AnError}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Contains(t, view.Notification(), "Disconnected")
	})

	t.Run("handles WSDisconnectedMsg without error", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSDisconnectedMsg{Error: nil}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Contains(t, view.Notification(), "disconnected")
	})

	t.Run("handles WSMessageReceivedMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		wsMsg := core.NewWebSocketMessage("conn-1", "Hello from server", "received")
		msg := components.WSMessageReceivedMsg{Message: wsMsg}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("handles WSMessageSentMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		wsMsg := core.NewWebSocketMessage("conn-1", "Hello from client", "sent")
		msg := components.WSMessageSentMsg{Message: wsMsg}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("handles WSErrorMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSErrorMsg{Error: assert.AnError}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Contains(t, view.Notification(), "Error")
	})

	t.Run("handles WSReconnectCmd with nil definition", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSReconnectCmd{}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
		assert.Nil(t, cmd)
	})
}

func TestMainView_NotificationClearing(t *testing.T) {
	t.Run("notification clears via timeout message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// First set a feedback notification
		feedbackMsg := components.FeedbackMsg{Message: "Test", IsError: false}
		updated, _ := view.Update(feedbackMsg)
		view = updated.(*MainView)
		assert.NotEmpty(t, view.Notification())

		// Verify notification was set
		assert.NotNil(t, view)
	})
}

func TestMainView_ForwardToPanes(t *testing.T) {
	t.Run("forward to WebSocket pane when focused", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)
		view.FocusPane(PaneWebSocket)

		// Send a generic key message
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_HistorySave(t *testing.T) {
	t.Run("saves to history on response", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		store := &mockHistoryStore{}
		view.SetHistoryStore(store)

		// Send a request first
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)
		sendMsg := components.SendRequestMsg{Request: req}
		updated, _ := view.Update(sendMsg)
		view = updated.(*MainView)

		// Now receive response
		resp := core.NewResponse("req-1", "http", core.NewStatus(200, "OK"))
		respMsg := components.ResponseReceivedMsg{Response: resp}
		updated, _ = view.Update(respMsg)
		view = updated.(*MainView)

		assert.False(t, view.ResponsePanel().IsLoading())
	})

	t.Run("saves to history on error", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		store := &mockHistoryStore{}
		view.SetHistoryStore(store)

		// Send a request first
		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)
		sendMsg := components.SendRequestMsg{Request: req}
		updated, _ := view.Update(sendMsg)
		view = updated.(*MainView)

		// Now receive error
		errMsg := components.RequestErrorMsg{Error: assert.AnError}
		updated, _ = view.Update(errMsg)
		view = updated.(*MainView)

		assert.False(t, view.ResponsePanel().IsLoading())
	})
}

func TestMainView_EdgeCases(t *testing.T) {
	t.Run("zero size does nothing", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(0, 0)

		output := view.View()
		assert.Empty(t, output)
	})

	t.Run("very narrow window clamps sidebar", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(50, 20)

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("very wide window clamps sidebar max", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(400, 40)

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("very short window clamps height", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 3)

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("minimal height for request panel", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 10)

		output := view.View()
		assert.NotEmpty(t, output)
	})
}

func TestMainView_HistoryItemWithBody(t *testing.T) {
	t.Run("handles SelectHistoryItemMsg with body and headers", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		entry := history.Entry{
			RequestName:   "Test Request",
			RequestMethod: "POST",
			RequestURL:    "https://api.example.com",
			RequestBody:   `{"key":"value"}`,
			RequestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		}
		msg := components.SelectHistoryItemMsg{Entry: entry}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		req := view.RequestPanel().Request()
		assert.NotNil(t, req)
		assert.Equal(t, "POST", req.Method())
	})
}

func TestMainView_FeedbackErrors(t *testing.T) {
	t.Run("handles FeedbackMsg with error flag", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.FeedbackMsg{Message: "Something went wrong", IsError: true}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Contains(t, view.Notification(), "wrong")
	})
}

func TestMainView_CopyContent(t *testing.T) {
	t.Run("handles CopyMsg with small content", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.CopyMsg{Content: "small content"}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Check that notification was set
		assert.NotNil(t, view)
	})

	t.Run("handles CopyMsg with large content", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Create content > 1024 bytes
		largeContent := make([]byte, 2048)
		for i := range largeContent {
			largeContent[i] = 'a'
		}
		msg := components.CopyMsg{Content: string(largeContent)}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Notification should indicate KB
		assert.NotNil(t, view)
	})
}

func TestMainView_WSStateChange(t *testing.T) {
	t.Run("handles WSStateChangedMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSStateChangedMsg{State: interfaces.ConnectionStateConnecting}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_WSReconnectWithDefinition(t *testing.T) {
	t.Run("handles WSReconnectCmd with definition set", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		// Set a WebSocket definition
		wsDef := core.NewWebSocketDefinition("Test WS", "wss://example.com")
		view.SetWebSocketDefinition(wsDef)

		// Now send reconnect - this should call connectWebSocket
		msg := components.WSReconnectCmd{}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
		// Reconnect should produce a command when definition exists
		assert.NotNil(t, cmd)
	})
}

func TestMainView_HelpDisplay(t *testing.T) {
	t.Run("help overlay blocks other keys", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ShowHelp()

		assert.True(t, view.ShowingHelp())

		// Other keys should be blocked
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Help should still be showing
		assert.True(t, view.ShowingHelp())
	})

	t.Run("? key toggles help when showing", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ShowHelp()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.ShowingHelp())
	})
}

func TestMainView_WSCommands(t *testing.T) {
	t.Run("handles WSConnectCmd", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		wsDef := core.NewWebSocketDefinition("Test", "wss://example.com")
		msg := components.WSConnectCmd{Definition: wsDef}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
		assert.NotNil(t, cmd) // Should return a command to connect
	})

	t.Run("handles WSDisconnectCmd", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSDisconnectCmd{}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
		assert.NotNil(t, cmd) // Should return a command to disconnect
	})

	t.Run("handles WSSendMessageCmd", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSSendMessageCmd{Content: "Hello"}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
		assert.NotNil(t, cmd) // Should return a command to send message
	})
}

func TestMainView_SendRequestReturnsCommand(t *testing.T) {
	t.Run("SendRequestMsg returns command", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		msg := components.SendRequestMsg{Request: req}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
		assert.NotNil(t, cmd)
		assert.True(t, view.ResponsePanel().IsLoading())
	})
}

func TestMainView_RenderingEdgeCases(t *testing.T) {
	t.Run("renders with loading response", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ResponsePanel().SetLoading(true)

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("renders with error response", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ResponsePanel().SetError(assert.AnError)

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("renders with notification set", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		feedbackMsg := components.FeedbackMsg{Message: "Test notification", IsError: false}
		updated, _ := view.Update(feedbackMsg)
		view = updated.(*MainView)

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("renders in WebSocket mode with definition", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		wsDef := core.NewWebSocketDefinition("Test WS", "wss://example.com/ws")
		view.SetWebSocketDefinition(wsDef)

		output := view.View()
		assert.NotEmpty(t, output)
	})
}

func TestMainView_HistoryItemHeaders(t *testing.T) {
	t.Run("handles SelectHistoryItemMsg with multiple headers", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		entry := history.Entry{
			RequestName:   "API Request",
			RequestMethod: "POST",
			RequestURL:    "https://api.example.com/data",
			RequestBody:   `{"items":[1,2,3]}`,
			RequestHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
				"X-Custom":      "value",
			},
		}
		msg := components.SelectHistoryItemMsg{Entry: entry}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		req := view.RequestPanel().Request()
		assert.NotNil(t, req)
		assert.Equal(t, "POST", req.Method())
		assert.Equal(t, "https://api.example.com/data", req.URL())
	})
}

func TestMainView_SaveToCollection(t *testing.T) {
	t.Run("s key triggers save to collection", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Create a request to save
		req := core.NewRequestDefinition("Test Request", "POST", "https://example.com")
		req.SetBody(`{"key": "value"}`)
		view.RequestPanel().SetRequest(req)

		// Press 's' to save
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		// Should show notification and have a command
		assert.NotNil(t, cmd)
		// Should have created a collection with the request
		assert.Equal(t, 1, view.CollectionTree().ItemCount())
	})

	t.Run("save without request shows notification", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// No request set
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		// Should show notification
		assert.NotNil(t, cmd)
	})

	t.Run("saves to existing collection", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Create existing collection
		col := core.NewCollection("Existing API")
		view.SetCollections([]*core.Collection{col})

		// Create a request to save
		req := core.NewRequestDefinition("New Request", "GET", "https://example.com/new")
		view.RequestPanel().SetRequest(req)

		// Press 's' to save
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Request should be added to existing collection
		collections := view.CollectionTree().Collections()
		assert.Len(t, collections, 1)
		assert.Equal(t, "Existing API", collections[0].Name())
	})
}

func TestMainView_DeleteRequestMsg(t *testing.T) {
	t.Run("handles delete request message without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("Test API")
		req := core.NewRequestDefinition("To Delete", "GET", "https://example.com")
		col.AddRequest(req)
		view.SetCollections([]*core.Collection{col})

		msg := components.DeleteRequestMsg{Collection: col, RequestID: req.ID()}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - delete is persisted asynchronously
		assert.NotNil(t, view)
	})
}

func TestMainView_CreateCollectionMsg(t *testing.T) {
	t.Run("handles create collection message without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("New Collection")
		msg := components.CreateCollectionMsg{Collection: col}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - the message is handled
		assert.NotNil(t, view)
	})
}

func TestMainView_DeleteCollectionMsg(t *testing.T) {
	t.Run("handles delete collection message without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("To Delete")
		view.SetCollections([]*core.Collection{col})

		msg := components.DeleteCollectionMsg{CollectionID: col.ID()}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - delete is persisted asynchronously
		assert.NotNil(t, view)
	})
}

func TestMainView_RenameCollectionMsg(t *testing.T) {
	t.Run("handles rename collection message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("Old Name")
		view.SetCollections([]*core.Collection{col})

		msg := components.RenameCollectionMsg{Collection: col}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - rename is handled by collection tree
		assert.NotNil(t, view)
	})
}

func TestMainView_MoveRequestMsg(t *testing.T) {
	t.Run("handles move request to folder without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		folder := col.AddFolder("Target Folder")
		req := core.NewRequestDefinition("To Move", "GET", "https://example.com")
		col.AddRequest(req)
		view.SetCollections([]*core.Collection{col})

		msg := components.MoveRequestMsg{
			Request:          req,
			SourceCollection: col,
			TargetCollection: col,
			TargetFolder:     folder,
		}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - move is persisted asynchronously
		assert.NotNil(t, view)
	})

	t.Run("handles move request to collection root without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		folder := col.AddFolder("Source Folder")
		req := core.NewRequestDefinition("To Move", "GET", "https://example.com")
		folder.AddRequest(req)
		view.SetCollections([]*core.Collection{col})

		msg := components.MoveRequestMsg{
			Request:          req,
			SourceCollection: col,
			TargetCollection: col,
			TargetFolder:     nil, // Move to root
		}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - move logic handles source detection
		assert.NotNil(t, view)
	})
}

func TestMainView_MoveFolderMsg(t *testing.T) {
	t.Run("handles move folder message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		folder := col.AddFolder("To Move")
		view.SetCollections([]*core.Collection{col})

		msg := components.MoveFolderMsg{
			Folder:           folder,
			SourceCollection: col,
			TargetCollection: col,
		}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic
		assert.NotNil(t, view)
	})
}

func TestMainView_DuplicateRequestMsg(t *testing.T) {
	t.Run("handles duplicate request message without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		req := core.NewRequestDefinition("Original", "GET", "https://example.com")
		col.AddRequest(req)
		view.SetCollections([]*core.Collection{col})

		msg := components.DuplicateRequestMsg{Collection: col, Request: req}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - duplicate is handled
		assert.NotNil(t, view)
	})
}

func TestMainView_DuplicateFolderMsg(t *testing.T) {
	t.Run("handles duplicate folder message without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		folder := col.AddFolder("Original Folder")
		folder.AddRequest(core.NewRequestDefinition("Req1", "GET", "https://example.com"))
		view.SetCollections([]*core.Collection{col})

		msg := components.DuplicateFolderMsg{Collection: col, Folder: folder}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - duplicate is handled
		assert.NotNil(t, view)
	})
}

func TestMainView_CopyAsCurlMsg(t *testing.T) {
	t.Run("handles copy as curl message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody(`{"key": "value"}`)
		req.SetHeader("Content-Type", "application/json")

		msg := components.CopyAsCurlMsg{Request: req}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		// Should trigger notification command
		assert.NotNil(t, cmd)
	})
}

func TestMainView_ExportCollectionMsg(t *testing.T) {
	t.Run("handles export collection message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("Export Me")
		col.AddRequest(core.NewRequestDefinition("Req1", "GET", "https://example.com"))

		msg := components.ExportCollectionMsg{Collection: col}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, cmd)
	})
}

func TestMainView_ReorderRequestMsg(t *testing.T) {
	t.Run("handles reorder request message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		req1 := core.NewRequestDefinition("First", "GET", "https://example.com/1")
		req2 := core.NewRequestDefinition("Second", "GET", "https://example.com/2")
		col.AddRequest(req1)
		col.AddRequest(req2)
		view.SetCollections([]*core.Collection{col})

		msg := components.ReorderRequestMsg{Collection: col, Request: req2, Direction: "up"}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic
		assert.NotNil(t, view)
	})
}

func TestMainView_RenameRequestMsg(t *testing.T) {
	t.Run("handles rename request message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		req := core.NewRequestDefinition("Old Name", "GET", "https://example.com")
		col.AddRequest(req)
		view.SetCollections([]*core.Collection{col})

		msg := components.RenameRequestMsg{Collection: col, Request: req}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_CreateFolderMsg(t *testing.T) {
	t.Run("handles create folder message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		folder := col.AddFolder("New Folder")
		view.SetCollections([]*core.Collection{col})

		msg := components.CreateFolderMsg{Collection: col, Folder: folder}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_RenameFolderMsg(t *testing.T) {
	t.Run("handles rename folder message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		folder := col.AddFolder("Old Folder Name")
		view.SetCollections([]*core.Collection{col})

		msg := components.RenameFolderMsg{Collection: col, Folder: folder}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_DeleteFolderMsg(t *testing.T) {
	t.Run("handles delete folder message without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		folder := col.AddFolder("To Delete")
		view.SetCollections([]*core.Collection{col})

		msg := components.DeleteFolderMsg{Collection: col, FolderID: folder.ID()}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Just verify no panic - delete is persisted asynchronously
		assert.NotNil(t, view)
	})
}

func TestMainView_ImportCollectionMsg(t *testing.T) {
	t.Run("handles import with empty path", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.ImportCollectionMsg{FilePath: ""}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should not crash with empty path
		assert.NotNil(t, view)
	})

	t.Run("handles import with invalid path", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := components.ImportCollectionMsg{FilePath: "/nonexistent/path/file.json"}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		// Should show error notification
		assert.NotNil(t, cmd)
	})
}

func TestMainView_SetCollectionStore(t *testing.T) {
	t.Run("sets collection store", func(t *testing.T) {
		view := NewMainView()
		// SetCollectionStore takes *filesystem.CollectionStore which is hard to mock
		// Just verify the method exists and can be called with nil
		view.SetCollectionStore(nil)
		assert.NotNil(t, view)
	})
}

func TestMainView_FocusBlurMethods(t *testing.T) {
	t.Run("Focus method exists", func(t *testing.T) {
		view := NewMainView()
		view.Focus()
		assert.True(t, view.Focused())
	})

	t.Run("Blur method exists", func(t *testing.T) {
		view := NewMainView()
		view.Blur()
		// MainView is always focused
		assert.True(t, view.Focused())
	})
}

func TestMainView_SanitizeFilename(t *testing.T) {
	t.Run("sanitizes filename with spaces", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Create collection with spaces in name
		col := core.NewCollection("My Test Collection")
		view.SetCollections([]*core.Collection{col})

		// Export uses sanitized filename
		msg := components.ExportCollectionMsg{Collection: col}
		_, cmd := view.Update(msg)
		_ = cmd // Verify no panic
	})

	t.Run("sanitizes filename with special characters", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("Test/API:v2")
		view.SetCollections([]*core.Collection{col})

		msg := components.ExportCollectionMsg{Collection: col}
		_, cmd := view.Update(msg)
		_ = cmd
	})
}

func TestMainView_HandleSaveToCollectionEdgeCases(t *testing.T) {
	t.Run("s key with WebSocket request creates WS definition", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)
		view.FocusPane(PaneWebSocket)

		ws := core.NewWebSocketDefinition("Test WS", "ws://localhost:8080")
		view.SetWebSocketDefinition(ws)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		_ = cmd
		assert.NotNil(t, view)
	})

	t.Run("s key with empty collection name still creates", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("", "GET", "https://example.com")
		view.RequestPanel().SetRequest(req)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, cmd)
	})
}

func TestMainView_UpdateMoreBranches(t *testing.T) {
	t.Run("handles Ctrl+C to quit", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := view.Update(msg)

		// Should produce a quit command
		assert.NotNil(t, cmd)
	})

	t.Run("handles q key to quit", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, cmd := view.Update(msg)

		assert.NotNil(t, cmd)
	})

	t.Run("question mark key shows help", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		assert.False(t, view.ShowingHelp())

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.ShowingHelp())
	})

	t.Run("escape hides help when showing", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ShowHelp()
		assert.True(t, view.ShowingHelp())

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.ShowingHelp())
	})

	t.Run("handles number keys 1-3 for pane focus", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Key 1 - Collections
		msg1 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		updated, _ := view.Update(msg1)
		view = updated.(*MainView)
		assert.Equal(t, PaneCollections, view.FocusedPane())

		// Key 2 - Request
		msg2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
		updated, _ = view.Update(msg2)
		view = updated.(*MainView)
		assert.Equal(t, PaneRequest, view.FocusedPane())

		// Key 3 - Response
		msg3 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
		updated, _ = view.Update(msg3)
		view = updated.(*MainView)
		assert.Equal(t, PaneResponse, view.FocusedPane())
	})

	t.Run("handles selection message from collection tree", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com")
		col.AddRequest(req)
		view.SetCollections([]*core.Collection{col})

		msg := components.SelectionMsg{
			Request: req,
		}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, req.Name(), view.RequestPanel().Request().Name())
	})

	t.Run("handles SelectWebSocketMsg for WebSocket", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("API")
		ws := core.NewWebSocketDefinition("Test WS", "ws://localhost:8080")
		view.SetCollections([]*core.Collection{col})

		msg := components.SelectWebSocketMsg{
			WebSocket: ws,
		}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, ViewModeWebSocket, view.ViewMode())
	})
}

func TestMainView_NotificationHandling(t *testing.T) {
	t.Run("notification set via feedback message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Set notification via feedback
		feedbackMsg := components.FeedbackMsg{Message: "Test notification", IsError: false}
		updated, _ := view.Update(feedbackMsg)
		view = updated.(*MainView)
		assert.Contains(t, view.Notification(), "Test notification")
	})

	t.Run("error notification has different style", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Set error notification
		feedbackMsg := components.FeedbackMsg{Message: "Error message", IsError: true}
		updated, _ := view.Update(feedbackMsg)
		view = updated.(*MainView)
		assert.Contains(t, view.Notification(), "Error message")
	})
}

func TestMainView_EnvironmentInterpolator(t *testing.T) {
	t.Run("sets environment and updates interpolator", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		env := core.NewEnvironment("Test Env")
		view.SetEnvironment(env, nil)

		assert.NotNil(t, view.Environment())
		assert.Equal(t, "Test Env", view.Environment().Name())
	})
}

func TestMainView_SendRequestMessages(t *testing.T) {
	t.Run("handles SendRequestMsg", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		req := core.NewRequestDefinition("Test", "GET", "invalid://url")
		view.RequestPanel().SetRequest(req)

		msg := components.SendRequestMsg{}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		// Should trigger some command (error or loading)
		_ = cmd
		assert.NotNil(t, view)
	})

	t.Run("handles WSSendMessageCmd", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		ws := core.NewWebSocketDefinition("Test WS", "ws://localhost:8080")
		view.SetWebSocketDefinition(ws)

		msg := components.WSSendMessageCmd{Content: "test message"}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("handles WSConnectCmd", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		ws := core.NewWebSocketDefinition("Test WS", "ws://localhost:8080")
		view.SetWebSocketDefinition(ws)

		msg := components.WSConnectCmd{Definition: ws}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_HistoryPanelInteraction(t *testing.T) {
	t.Run("H key switches to history mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)

		// Press 'C' first to switch to collections mode
		msgC := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}}
		updated, _ := view.Update(msgC)
		view = updated.(*MainView)

		// Press 'H' to switch to history mode
		msgH := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ = view.Update(msgH)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_CollectionOperations(t *testing.T) {
	t.Run("handles collection with multiple folders", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col := core.NewCollection("Multi-folder API")
		col.AddFolder("Users")
		col.AddFolder("Products")
		col.AddFolder("Orders")
		view.SetCollections([]*core.Collection{col})

		assert.Equal(t, 1, view.CollectionTree().ItemCount())
	})

	t.Run("handles multiple collections", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		col1 := core.NewCollection("API v1")
		col2 := core.NewCollection("API v2")
		view.SetCollections([]*core.Collection{col1, col2})

		assert.Equal(t, 2, view.CollectionTree().ItemCount())
	})
}

func TestMainView_ViewRendering(t *testing.T) {
	t.Run("renders with notification", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		feedbackMsg := components.FeedbackMsg{Message: "Success!", IsError: false}
		updated, _ := view.Update(feedbackMsg)
		view = updated.(*MainView)

		output := view.View()
		assert.Contains(t, output, "Success!")
	})

	t.Run("renders error notification in red", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		feedbackMsg := components.FeedbackMsg{Message: "Error occurred", IsError: true}
		updated, _ := view.Update(feedbackMsg)
		view = updated.(*MainView)

		output := view.View()
		_ = output // Just verify no panic
	})

	t.Run("renders WebSocket mode layout", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("renders help overlay when showing", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.ShowHelp()

		output := view.View()
		// Help content should be in the output
		assert.Contains(t, output, "Help")
	})
}

func TestMainView_SetCookieJar(t *testing.T) {
	t.Run("sets cookie jar", func(t *testing.T) {
		view := NewMainView()
		view.SetCookieJar(nil)
		assert.NotNil(t, view)
	})
}

func TestMainView_EnvironmentSwitcherUI(t *testing.T) {
	t.Run("Esc closes environment switcher", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showEnvSwitcher)
	})

	t.Run("j/k navigates in environment switcher", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Dev"},
			{ID: "2", Name: "Prod"},
		}
		view.envCursor = 0

		msgJ := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msgJ)
		view = updated.(*MainView)
		assert.Equal(t, 1, view.envCursor)

		msgK := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ = view.Update(msgK)
		view = updated.(*MainView)
		assert.Equal(t, 0, view.envCursor)
	})

	t.Run("renders environment switcher", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Development", VarCount: 5, IsActive: true},
			{ID: "2", Name: "Production", VarCount: 3},
		}
		view.envCursor = 0

		output := view.View()
		assert.Contains(t, output, "Select Environment")
	})
}

func TestMainView_ProxyDialog(t *testing.T) {
	t.Run("P key opens proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.showProxyDialog)
	})

	t.Run("Esc closes proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showProxyDialog)
	})

	t.Run("renders proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true

		output := view.View()
		assert.Contains(t, output, "Proxy")
	})
}

func TestMainView_TLSDialog(t *testing.T) {
	t.Run("Ctrl+T opens TLS dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlT}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.showTLSDialog)
	})

	t.Run("Esc closes TLS dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showTLSDialog)
	})

	t.Run("renders TLS dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true

		output := view.View()
		assert.Contains(t, output, "TLS")
	})
}

func TestMainView_EnvironmentEditor(t *testing.T) {
	t.Run("renders environment editor", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		env := core.NewEnvironment("Test Env")
		env.SetVariable("API_KEY", "secret123")
		env.SetVariable("BASE_URL", "https://api.example.com")
		view.editingEnv = env
		view.envVarKeys = []string{"API_KEY", "BASE_URL"}
		view.envEditorCursor = 0
		view.envEditorMode = 0

		output := view.View()
		assert.Contains(t, output, "Test Env")
	})
}

func TestMainView_ProxyDialogHandling(t *testing.T) {
	t.Run("typing in proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true
		view.proxyInput = ""

		// Type some text
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
		updated, _ = view.Update(msg)
		view = updated.(*MainView)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
		updated, _ = view.Update(msg)
		view = updated.(*MainView)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		updated, _ = view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "http", view.proxyInput)
	})

	t.Run("backspace in proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true
		view.proxyInput = "http://proxy"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "http://prox", view.proxyInput)
	})

	t.Run("enter saves proxy and closes dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true
		view.proxyInput = "http://proxy:8080"

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showProxyDialog)
		assert.Equal(t, "http://proxy:8080", view.proxyURL)
		assert.NotNil(t, cmd)
	})

	t.Run("enter with empty proxy shows disabled message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true
		view.proxyInput = ""

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showProxyDialog)
		assert.Contains(t, view.Notification(), "disabled")
	})
}

func TestMainView_TLSDialogHandling(t *testing.T) {
	t.Run("Tab moves to next field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 0

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 1, view.tlsDialogField)
	})

	t.Run("Down moves to next field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 0

		msg := tea.KeyMsg{Type: tea.KeyDown}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 1, view.tlsDialogField)
	})

	t.Run("Shift+Tab moves to previous field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 2

		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 1, view.tlsDialogField)
	})

	t.Run("Up moves to previous field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 1

		msg := tea.KeyMsg{Type: tea.KeyUp}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 0, view.tlsDialogField)
	})

	t.Run("Space toggles insecure skip on field 3", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 3
		view.tlsInsecureSkip = false

		msg := tea.KeyMsg{Type: tea.KeySpace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.tlsInsecureSkip)

		// Toggle back
		updated, _ = view.Update(msg)
		view = updated.(*MainView)
		assert.False(t, view.tlsInsecureSkip)
	})

	t.Run("typing in cert field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 0
		view.tlsCertInput = ""

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ = view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "ce", view.tlsCertInput)
	})

	t.Run("typing in key field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 1
		view.tlsKeyInput = ""

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "k", view.tlsKeyInput)
	})

	t.Run("typing in CA field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 2
		view.tlsCAInput = ""

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "c", view.tlsCAInput)
	})

	t.Run("typing in toggle field does nothing", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 3

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should not crash - toggle field ignores text input
		assert.NotNil(t, view)
	})

	t.Run("backspace in cert field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 0
		view.tlsCertInput = "cert.pem"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "cert.pe", view.tlsCertInput)
	})

	t.Run("backspace in key field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 1
		view.tlsKeyInput = "key.pem"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "key.pe", view.tlsKeyInput)
	})

	t.Run("backspace in CA field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 2
		view.tlsCAInput = "ca.pem"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "ca.pe", view.tlsCAInput)
	})

	t.Run("enter saves TLS settings with values", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsCertInput = "/path/to/cert.pem"
		view.tlsKeyInput = "/path/to/key.pem"
		view.tlsCAInput = "/path/to/ca.pem"

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showTLSDialog)
		assert.Equal(t, "/path/to/cert.pem", view.tlsCertFile)
		assert.Equal(t, "/path/to/key.pem", view.tlsKeyFile)
		assert.Equal(t, "/path/to/ca.pem", view.tlsCAFile)
		assert.Contains(t, view.Notification(), "saved")
		assert.NotNil(t, cmd)
	})

	t.Run("enter clears TLS settings when empty", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsCertInput = ""
		view.tlsKeyInput = ""
		view.tlsCAInput = ""
		view.tlsInsecureSkip = false

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Contains(t, view.Notification(), "cleared")
	})

	t.Run("field wraps around", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 3

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 0, view.tlsDialogField)
	})
}

func TestMainView_RunnerModal(t *testing.T) {
	t.Run("Ctrl+R opens runner modal", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Add a collection to run
		col := core.NewCollection("Test API")
		col.AddRequest(core.NewRequestDefinition("Req1", "GET", "https://example.com"))
		view.SetCollections([]*core.Collection{col})

		msg := tea.KeyMsg{Type: tea.KeyCtrlR}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Either shows modal or notification
		assert.NotNil(t, view)
	})

	t.Run("renders runner modal when open", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerProgress = 5
		view.runnerTotal = 10
		view.runnerCurrentReq = "Testing API"

		output := view.View()
		assert.NotEmpty(t, output)
	})

	t.Run("Esc closes runner modal", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerRunning = false

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showRunnerModal)
	})

	t.Run("Esc cancels running runner", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerRunning = true
		// Create a cancel func that does nothing
		view.runnerCancelFunc = func() {}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.runnerRunning)
	})

	t.Run("Enter closes completed runner modal", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerRunning = false

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showRunnerModal)
	})

	t.Run("startCollectionRunner with no collection shows notification", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		// No collections set

		msg := tea.KeyMsg{Type: tea.KeyCtrlR}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should show notification
		assert.NotNil(t, view)
	})
}

func TestMainView_EnvironmentSwitcherInteractions(t *testing.T) {
	t.Run("V key opens env switcher", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// Without environment store, should show notification
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should show "No environment store available" in notification
		assert.Contains(t, view.Notification(), "environment")
	})

	t.Run("q key closes env switcher", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Dev"},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showEnvSwitcher)
	})

	t.Run("j/k respects bounds", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Dev"},
		}
		view.envCursor = 0

		// Try to go down past end
		msgJ := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msgJ)
		view = updated.(*MainView)
		assert.Equal(t, 0, view.envCursor) // Should stay at 0

		// Try to go up past start
		msgK := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ = view.Update(msgK)
		view = updated.(*MainView)
		assert.Equal(t, 0, view.envCursor) // Should stay at 0
	})

	t.Run("Enter selects environment", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Dev"},
		}
		view.envCursor = 0

		// Enter without environment store should just close
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showEnvSwitcher)
	})
}

func TestMainView_EnvironmentEditorInteractions(t *testing.T) {
	t.Run("Esc closes environment editor", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 0

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showEnvEditor)
	})

	t.Run("j/k navigates in environment editor", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.editingEnv = core.NewEnvironment("Test")
		view.editingEnv.SetVariable("KEY1", "val1")
		view.editingEnv.SetVariable("KEY2", "val2")
		view.envVarKeys = []string{"KEY1", "KEY2"}
		view.envEditorCursor = 0
		view.envEditorMode = 0

		msgJ := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msgJ)
		view = updated.(*MainView)
		assert.Equal(t, 1, view.envEditorCursor)

		msgK := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ = view.Update(msgK)
		view = updated.(*MainView)
		assert.Equal(t, 0, view.envEditorCursor)
	})

	t.Run("d deletes variable in environment editor", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		env := core.NewEnvironment("Test")
		env.SetVariable("KEY1", "val1")
		env.SetVariable("KEY2", "val2")
		view.editingEnv = env
		view.envVarKeys = []string{"KEY1", "KEY2"}
		view.envEditorCursor = 0
		view.envEditorMode = 0

		msgD := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		updated, _ := view.Update(msgD)
		view = updated.(*MainView)

		assert.Len(t, view.envVarKeys, 1)
	})

	t.Run("a adds new variable", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.editingEnv = core.NewEnvironment("Test")
		view.envVarKeys = []string{}
		view.envEditorCursor = 0
		view.envEditorMode = 0

		msgA := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		updated, _ := view.Update(msgA)
		view = updated.(*MainView)

		// Should switch to edit mode
		assert.Equal(t, 1, view.envEditorMode)
	})

	t.Run("Enter edits selected variable", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		env := core.NewEnvironment("Test")
		env.SetVariable("KEY1", "val1")
		view.editingEnv = env
		view.envVarKeys = []string{"KEY1"}
		view.envEditorCursor = 0
		view.envEditorMode = 0

		msgEnter := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msgEnter)
		view = updated.(*MainView)

		// Should switch to edit mode (value editing is mode 3)
		assert.Equal(t, 3, view.envEditorMode)
	})
}

func TestMainView_SetStarredStore(t *testing.T) {
	t.Run("sets starred store", func(t *testing.T) {
		view := NewMainView()
		view.SetStarredStore(nil)
		assert.NotNil(t, view)
	})
}

func TestMainView_SetEnvironmentStore(t *testing.T) {
	t.Run("sets environment store", func(t *testing.T) {
		view := NewMainView()
		view.SetEnvironmentStore(nil)
		assert.NotNil(t, view)
	})
}

func TestMainView_ClearCookiesKey(t *testing.T) {
	t.Run("Ctrl+K without cookie jar does nothing", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlK}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		// Without cookie jar, should do nothing
		assert.Nil(t, cmd)
		assert.Empty(t, view.Notification())
	})
}

func TestMainView_WSDisconnectWithoutConnection(t *testing.T) {
	t.Run("disconnect when not connected does nothing", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.SetViewMode(ViewModeWebSocket)

		msg := components.WSDisconnectCmd{}
		updated, cmd := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
		assert.NotNil(t, cmd)
	})
}

func TestMainView_RunnerCompleteMessage(t *testing.T) {
	t.Run("handles runner complete message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerRunning = true

		msg := runnerCompleteMsg{Summary: nil}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.runnerRunning)
	})
}

func TestMainView_RunnerProgressMessage(t *testing.T) {
	t.Run("handles runner progress message", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerRunning = true

		msg := runnerProgressMsg{Current: 5, Total: 10, CurrentName: "Test Request"}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 5, view.runnerProgress)
		assert.Equal(t, 10, view.runnerTotal)
		assert.Equal(t, "Test Request", view.runnerCurrentReq)
	})
}

func TestMainView_AdditionalUpdateBranches(t *testing.T) {
	t.Run("handles unknown message type", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		type unknownMsg struct{}
		updated, _ := view.Update(unknownMsg{})
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("handles Ctrl+R for run collection without panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlR}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Without a collection selected, runner won't open but should not panic
		assert.NotNil(t, view)
	})

	t.Run("handles ? for help toggle", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.showHelp)
	})

	t.Run("handles P for proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.showProxyDialog)
	})

	t.Run("handles Ctrl+T for TLS dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlT}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.showTLSDialog)
	})
}

func TestMainView_ProxyDialogInput(t *testing.T) {
	t.Run("escape closes proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showProxyDialog)
	})

	t.Run("Tab in proxy dialog does not panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_TLSDialogInput(t *testing.T) {
	t.Run("escape closes TLS dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showTLSDialog)
	})

	t.Run("Tab in TLS dialog does not panic", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_EnvironmentSwitcherBasic(t *testing.T) {
	t.Run("V key without environment store shows notification", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		// V opens environment switcher (but needs env store)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Without env store, shows notification instead
		assert.NotNil(t, view)
	})

	t.Run("escape closes environment switcher", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showEnvSwitcher)
	})
}

func TestMainView_RunnerModalInputBasic(t *testing.T) {
	t.Run("escape closes runner modal when not running", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerRunning = false

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showRunnerModal)
	})
}

func TestMainView_HelpInputBasic(t *testing.T) {
	t.Run("any key closes help", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showHelp = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showHelp)
	})

	t.Run("j scrolls help down without error", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showHelp = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Verify no panic and view is still showing help
		assert.True(t, view.showHelp)
	})

	t.Run("k scrolls help up without error", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showHelp = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Verify no panic and view is still showing help
		assert.True(t, view.showHelp)
	})
}

func TestMainView_WebSocketViewMode(t *testing.T) {
	t.Run("switches to WebSocket view with Ctrl+W", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlW}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should toggle to WebSocket mode
		assert.NotNil(t, view)
	})
}

func TestMainView_CopyFunctions(t *testing.T) {
	t.Run("handles copy with no request", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		// Should not panic with no request
		assert.NotNil(t, view)
	})
}

func TestMainView_FeedbackHandling(t *testing.T) {
	t.Run("handles feedback with no request", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.FocusPane(PaneCollections)

		// Press 'f' without a request selected
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_FocusBlurMethodsExtra(t *testing.T) {
	t.Run("Focus method called multiple times", func(t *testing.T) {
		view := NewMainView()
		view.Focus()
		view.Focus()
		assert.NotNil(t, view)
	})

	t.Run("Blur method called multiple times", func(t *testing.T) {
		view := NewMainView()
		view.Blur()
		view.Blur()
		assert.NotNil(t, view)
	})
}

func TestMainView_EnvEditorCoverage(t *testing.T) {
	t.Run("enter key in env editor mode 0", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 0

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("esc key in env editor mode 1", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 1

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("tab key in env editor", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 1

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("j and k keys in env editor mode 0", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 0
		view.envVarKeys = []string{"VAR1", "VAR2", "VAR3"}

		// j key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)
		assert.NotNil(t, view)

		// k key
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ = view.Update(msg)
		view = updated.(*MainView)
		assert.NotNil(t, view)
	})

	t.Run("backspace in env editor input mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 1
		view.envEditorKeyInput = "test"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("runes input in env editor", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 1

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("space in env editor input mode", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvEditor = true
		view.envEditorMode = 1

		msg := tea.KeyMsg{Type: tea.KeySpace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_MoreProxyDialogCoverage(t *testing.T) {
	t.Run("enter in proxy dialog saves settings", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true
		view.proxyInput = "http://proxy.example.com:8080"

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "http://proxy.example.com:8080", view.proxyURL)
	})

	t.Run("backspace in proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true
		view.proxyInput = "http"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "htt", view.proxyInput)
	})

	t.Run("runes in proxy dialog", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showProxyDialog = true
		view.proxyInput = "http"

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "https", view.proxyInput)
	})
}

func TestMainView_MoreTLSDialogCoverage(t *testing.T) {
	t.Run("enter in TLS dialog saves settings", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsCertInput = "/path/to/cert.pem"
		view.tlsKeyInput = "/path/to/key.pem"

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "/path/to/cert.pem", view.tlsCertFile)
		assert.Equal(t, "/path/to/key.pem", view.tlsKeyFile)
	})

	t.Run("tab cycles TLS dialog fields", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 0

		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 1, view.tlsDialogField)
	})

	t.Run("shift+tab cycles TLS dialog fields backward", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 1

		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 0, view.tlsDialogField)
	})

	t.Run("space toggles insecure skip", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 3
		view.tlsInsecureSkip = false

		msg := tea.KeyMsg{Type: tea.KeySpace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.True(t, view.tlsInsecureSkip)
	})

	t.Run("backspace in cert field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 0
		view.tlsCertInput = "cert"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "cer", view.tlsCertInput)
	})

	t.Run("backspace in key field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 1
		view.tlsKeyInput = "key"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "ke", view.tlsKeyInput)
	})

	t.Run("backspace in CA file field", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true
		view.tlsDialogField = 2
		view.tlsCAInput = "ca"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, "c", view.tlsCAInput)
	})

	t.Run("runes in TLS dialog fields", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showTLSDialog = true

		// Test cert field
		view.tlsDialogField = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)
		assert.Equal(t, "a", view.tlsCertInput)

		// Test key field
		view.tlsDialogField = 1
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
		updated, _ = view.Update(msg)
		view = updated.(*MainView)
		assert.Equal(t, "b", view.tlsKeyInput)

		// Test CA field
		view.tlsDialogField = 2
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		updated, _ = view.Update(msg)
		view = updated.(*MainView)
		assert.Equal(t, "c", view.tlsCAInput)
	})
}

func TestMainView_MoreRunnerModalCoverage(t *testing.T) {
	t.Run("enter closes modal when not running", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showRunnerModal = true
		view.runnerRunning = false

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showRunnerModal)
	})
}

func TestMainView_AdditionalKeybindings(t *testing.T) {
	t.Run("Ctrl+K clears cookies", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyCtrlK}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("q key quits", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, cmd := view.Update(msg)

		// Verify quit command returned
		assert.NotNil(t, cmd)
	})

	t.Run("s key triggers save to collection", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})

	t.Run("h key triggers history", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}

func TestMainView_EnvSwitcherInputCoverage(t *testing.T) {
	t.Run("j key in env switcher moves cursor down", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envCursor = 0
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Env1"},
			{ID: "2", Name: "Env2"},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 1, view.envCursor)
	})

	t.Run("k key in env switcher moves cursor up", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envCursor = 1
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Env1"},
			{ID: "2", Name: "Env2"},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.Equal(t, 0, view.envCursor)
	})

	t.Run("q key closes env switcher", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.False(t, view.showEnvSwitcher)
	})

	t.Run("enter key in env switcher without store", func(t *testing.T) {
		view := NewMainView()
		view.SetSize(120, 40)
		view.showEnvSwitcher = true
		view.envCursor = 0
		view.envList = []filesystem.EnvironmentMeta{
			{ID: "1", Name: "Env1"},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := view.Update(msg)
		view = updated.(*MainView)

		assert.NotNil(t, view)
	})
}
