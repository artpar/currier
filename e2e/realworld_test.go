package e2e

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/tui/views"
)

// tuiModel wraps MainView exactly like root.go does
type tuiModel struct {
	view *views.MainView
}

func (m tuiModel) Init() tea.Cmd {
	return m.view.Init()
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.view.Update(msg)
	m.view = updated.(*views.MainView)
	return m, cmd
}

func (m tuiModel) View() string {
	return m.view.View()
}

// helper to update and execute cmd
func updateAndExec(m tuiModel, msg tea.Msg) tuiModel {
	newModel, cmd := m.Update(msg)
	m = newModel.(tuiModel)
	if cmd != nil {
		result := cmd()
		if result != nil {
			m = updateAndExec(m, result)
		}
	}
	return m
}

// TestRealWorldUsage simulates what tea.Program does
func TestRealWorldUsage(t *testing.T) {
	// Create like root.go does
	view := views.NewMainView()
	model := tuiModel{view: view}

	// Init
	initCmd := model.Init()
	if initCmd != nil {
		msg := initCmd()
		if msg != nil {
			model = updateAndExec(model, msg)
		}
	}

	// First thing tea.Program sends is WindowSizeMsg
	model = updateAndExec(model, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Render initial
	output := model.View()
	t.Logf("Initial render: %d bytes", len(output))
	if len(output) == 0 {
		t.Error("Initial render returned empty output")
	}

	// Press 'n' to create new request
	model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	output = model.View()
	t.Logf("After 'n': %d bytes", len(output))

	// Type URL character by character
	for _, c := range "https://httpbin.org/get" {
		model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{c}})
	}

	output = model.View()
	t.Logf("After typing URL: %d bytes", len(output))

	// Press Escape to exit edit mode
	model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyEsc})

	output = model.View()
	t.Logf("After Escape: %d bytes", len(output))

	// Press Enter to send request
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newModel.(tuiModel)
	t.Logf("After Enter - got cmd: %v", cmd != nil)

	// Execute the command (this is what would send the HTTP request)
	if cmd != nil {
		msg := cmd()
		t.Logf("Cmd returned msg type: %T", msg)
		if msg != nil {
			model = updateAndExec(model, msg)
		}
	}

	output = model.View()
	t.Logf("Final output: %d bytes", len(output))
}

// TestWithCollection simulates loading a collection like the real app
func TestWithCollection(t *testing.T) {
	view := views.NewMainView()

	// Create a sample collection like real app would load
	collection := core.NewCollection("Test Collection")
	req := core.NewRequestDefinition("Test Request", "GET", "https://httpbin.org/get")
	collection.AddRequest(req)

	view.SetCollections([]*core.Collection{collection})

	model := tuiModel{view: view}

	// WindowSizeMsg first
	model = updateAndExec(model, tea.WindowSizeMsg{Width: 120, Height: 40})

	output := model.View()
	t.Logf("Initial with collection: %d bytes", len(output))

	// Navigate down to the request
	model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // expand
	model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // down to request

	// Press Enter to select request
	model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyEnter})

	output = model.View()
	t.Logf("After selecting request: %d bytes", len(output))

	// Press Tab to go to request panel
	model = updateAndExec(model, tea.KeyMsg{Type: tea.KeyTab})

	// Now send the request
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newModel.(tuiModel)
	if cmd != nil {
		msg := cmd()
		if msg != nil {
			model = updateAndExec(model, msg)
		}
	}

	output = model.View()
	t.Logf("After sending: %d bytes", len(output))
}
