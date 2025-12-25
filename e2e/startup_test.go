package e2e

import (
    "testing"
    
    "github.com/artpar/currier/internal/tui/views"
    tea "github.com/charmbracelet/bubbletea"
)

func TestActualStartup(t *testing.T) {
    // Exactly like root.go does it
    view := views.NewMainView()
    
    // Simulate WindowSizeMsg that tea.Program sends FIRST
    updated, _ := view.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
    view = updated.(*views.MainView)
    
    // Render - this shouldn't crash
    output := view.View()
    t.Logf("Rendered %d bytes", len(output))
    
    // Press 'n' - create new request
    updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
    view = updated.(*views.MainView)
    
    output = view.View()
    t.Logf("After 'n': %d bytes", len(output))
    
    // Type URL
    for _, c := range "https://example.com" {
        updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{c}})
        view = updated.(*views.MainView)
    }
    
    output = view.View()
    t.Logf("After typing URL: %d bytes", len(output))
    
    // Press Escape
    updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyEsc})
    view = updated.(*views.MainView)
    
    output = view.View()
    t.Logf("After Escape: %d bytes", len(output))
    
    // Press Enter to send
    updated, cmd := view.Update(tea.KeyMsg{Type: tea.KeyEnter})
    view = updated.(*views.MainView)
    
    t.Logf("After Enter - got cmd: %v", cmd != nil)
    
    output = view.View()
    t.Logf("Final output: %d bytes", len(output))
}
