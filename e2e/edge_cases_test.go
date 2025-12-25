package e2e

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/tui/views"
)

// TestEdgeCases tests potential crash scenarios
func TestEdgeCases(t *testing.T) {
	t.Run("zero_size_terminal", func(t *testing.T) {
		view := views.NewMainView()
		// No size set - should not crash
		output := view.View()
		t.Logf("Zero size output: %d bytes", len(output))
	})
	
	t.Run("very_small_terminal", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(10, 5) // Very small
		output := view.View()
		t.Logf("Small terminal output: %d bytes", len(output))
	})
	
	t.Run("navigation_on_empty_tree", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(120, 40)
		
		// Try navigating on empty tree - should not crash
		keys := []string{"j", "k", "G", "g", "g", "l", "h", "Enter"}
		for _, key := range keys {
			var msg tea.KeyMsg
			if key == "Enter" {
				msg = tea.KeyMsg{Type: tea.KeyEnter}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			}
			updated, _ := view.Update(msg)
			view = updated.(*views.MainView)
		}
		
		output := view.View()
		t.Logf("After navigation on empty tree: %d bytes", len(output))
	})
	
	t.Run("switch_to_history_without_store", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(120, 40)
		
		// Try switching to history view without history store
		updated, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
		view = updated.(*views.MainView)
		
		output := view.View()
		t.Logf("History view without store: %d bytes", len(output))
	})
	
	t.Run("send_request_without_request", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(120, 40)
		
		// Focus request panel
		updated, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		view = updated.(*views.MainView)
		
		// Try to send without request - should not crash
		updated, cmd := view.Update(tea.KeyMsg{Type: tea.KeyEnter})
		view = updated.(*views.MainView)
		
		t.Logf("Send without request - cmd: %v", cmd != nil)
	})
	
	t.Run("edit_url_without_request", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(120, 40)
		
		// Focus request panel
		updated, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		view = updated.(*views.MainView)
		
		// Try to edit URL without request - should not crash
		updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		view = updated.(*views.MainView)
		
		output := view.View()
		t.Logf("Edit without request: %d bytes", len(output))
	})
	
	t.Run("tab_switching_without_request", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(120, 40)
		
		// Focus request panel
		updated, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		view = updated.(*views.MainView)
		
		// Switch tabs without request - should not crash
		for i := 0; i < 10; i++ {
			updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
			view = updated.(*views.MainView)
		}
		
		output := view.View()
		t.Logf("Tab switching without request: %d bytes", len(output))
	})
	
	t.Run("delete_header_without_headers", func(t *testing.T) {
		view := views.NewMainView()
		view.SetSize(120, 40)
		
		// Create request
		updated, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		view = updated.(*views.MainView)
		updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyEsc})
		view = updated.(*views.MainView)
		
		// Switch to headers tab
		updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
		view = updated.(*views.MainView)
		
		// Try to delete when no headers exist
		updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		view = updated.(*views.MainView)
		
		output := view.View()
		t.Logf("Delete header without headers: %d bytes", len(output))
	})
}
