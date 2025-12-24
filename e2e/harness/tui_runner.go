package harness

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/tui/views"
)

// TUIRunner provides TUI testing capabilities.
type TUIRunner struct {
	harness *E2EHarness
}

// TUISession represents an active TUI test session.
type TUISession struct {
	runner *TUIRunner
	model  *views.MainView
	t      *testing.T
}

// Start starts a new TUI session.
func (r *TUIRunner) Start(t *testing.T) *TUISession {
	t.Helper()

	model := views.NewMainView()
	// Initialize with a reasonable terminal size
	model.SetSize(120, 40)

	return &TUISession{
		runner: r,
		model:  model,
		t:      t,
	}
}

// StartWithSize starts a TUI session with custom dimensions.
func (r *TUIRunner) StartWithSize(t *testing.T, width, height int) *TUISession {
	t.Helper()

	model := views.NewMainView()
	model.SetSize(width, height)

	return &TUISession{
		runner: r,
		model:  model,
		t:      t,
	}
}

// SendKey sends a key press.
func (s *TUISession) SendKey(key string) *TUISession {
	msg := parseKeyMsg(key)
	updated, _ := s.model.Update(msg)
	s.model = updated.(*views.MainView)
	return s
}

// SendKeys sends multiple key presses.
func (s *TUISession) SendKeys(keys ...string) *TUISession {
	for _, key := range keys {
		s.SendKey(key)
	}
	return s
}

// Type sends a sequence of rune keys.
func (s *TUISession) Type(text string) *TUISession {
	for _, r := range text {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		updated, _ := s.model.Update(msg)
		s.model = updated.(*views.MainView)
	}
	return s
}

// Wait pauses for the specified duration.
func (s *TUISession) Wait(d time.Duration) *TUISession {
	time.Sleep(d)
	return s
}

// WaitForOutput waits for specific text in output.
func (s *TUISession) WaitForOutput(text string) error {
	timeout := s.runner.harness.timeout
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		output := s.Output()
		if strings.Contains(output, text) {
			return nil
		}
		time.Sleep(pollInterval)
	}

	return &TimeoutError{text: text, timeout: timeout}
}

// Output returns the current TUI output.
func (s *TUISession) Output() string {
	return s.model.View()
}

// Quit is a no-op for direct model testing (kept for API compatibility).
func (s *TUISession) Quit() {
	// No-op - we're testing the model directly, not a running program
}

// Model returns the underlying MainView for direct assertions.
func (s *TUISession) Model() *views.MainView {
	return s.model
}

// FocusedPane returns the currently focused pane.
func (s *TUISession) FocusedPane() views.Pane {
	return s.model.FocusedPane()
}

// ShowingHelp returns true if help overlay is visible.
func (s *TUISession) ShowingHelp() bool {
	return s.model.ShowingHelp()
}

// TimeoutError represents a timeout waiting for output.
type TimeoutError struct {
	text    string
	timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return "timeout after " + e.timeout.String() + " waiting for: " + e.text
}

// parseKeyMsg converts key string to tea.KeyMsg.
func parseKeyMsg(key string) tea.KeyMsg {
	switch strings.ToLower(key) {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc", "escape":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		// Handle single character keys
		if len(key) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		// Handle multi-character sequences (like "gg")
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
