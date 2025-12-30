package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// Session represents a tmux session for TUI testing.
type Session struct {
	t      *testing.T
	id     string
	binary string
	width  int
	height int
	mu     sync.Mutex
}

// NewSession creates a new tmux session for testing.
func NewSession(t *testing.T) *Session {
	t.Helper()
	return &Session{
		t:      t,
		id:     fmt.Sprintf("currier-test-%d", time.Now().UnixNano()),
		width:  120,
		height: 40,
	}
}

// WithSize sets the terminal size.
func (s *Session) WithSize(width, height int) *Session {
	s.width = width
	s.height = height
	return s
}

// Start starts the binary in a new tmux session.
// If CURRIER_BINARY env var is set, it overrides the binary parameter.
// If GOCOVERDIR env var is set, it's passed to the tmux session for coverage collection.
func (s *Session) Start(binary string, args ...string) error {
	s.t.Helper()

	// Allow override via environment variable (for coverage testing)
	if envBinary := os.Getenv("CURRIER_BINARY"); envBinary != "" {
		binary = envBinary
	}
	s.binary = binary

	// Build the command to run inside tmux
	cmd := binary
	if len(args) > 0 {
		cmd = binary + " " + shellQuoteArgs(args)
	}

	// If GOCOVERDIR is set, wrap the command to pass it through
	if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
		cmd = fmt.Sprintf("GOCOVERDIR=%s %s", coverDir, cmd)
	}

	// Create new detached session with specific size
	tmuxCmd := exec.Command("tmux", "new-session",
		"-d",                              // detached
		"-s", s.id,                        // session name
		"-x", fmt.Sprintf("%d", s.width),  // width
		"-y", fmt.Sprintf("%d", s.height), // height
		cmd,                               // command to run
	)

	var stderr bytes.Buffer
	tmuxCmd.Stderr = &stderr

	if err := tmuxCmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w: %s", err, stderr.String())
	}

	// Wait for app to initialize
	time.Sleep(500 * time.Millisecond)
	return nil
}

// SendKey sends a special key (Enter, Escape, Tab, etc.).
func (s *Session) SendKey(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmuxKey := mapKey(key)
	cmd := exec.Command("tmux", "send-keys", "-t", s.id, tmuxKey)
	return cmd.Run()
}

// Type sends literal text.
func (s *Session) Type(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Send as literal text (bypasses key interpretation)
	cmd := exec.Command("tmux", "send-keys", "-t", s.id, "-l", text)
	return cmd.Run()
}

// IsAlive checks if the session still exists.
func (s *Session) IsAlive() bool {
	cmd := exec.Command("tmux", "has-session", "-t", s.id)
	return cmd.Run() == nil
}

// Capture returns the current screen content.
func (s *Session) Capture() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAliveUnlocked() {
		return ""
	}

	cmd := exec.Command("tmux", "capture-pane", "-t", s.id, "-p")
	output, err := cmd.Output()
	if err != nil {
		// Only log if session should exist
		return ""
	}
	return string(output)
}

// isAliveUnlocked checks if session exists (caller must hold lock).
func (s *Session) isAliveUnlocked() bool {
	cmd := exec.Command("tmux", "has-session", "-t", s.id)
	return cmd.Run() == nil
}

// WaitFor waits for text to appear on screen.
func (s *Session) WaitFor(text string, timeout time.Duration) error {
	return s.WaitForCondition(func(screen string) bool {
		return strings.Contains(screen, text)
	}, timeout)
}

// WaitForCondition waits for a condition to be true.
func (s *Session) WaitForCondition(fn func(string) bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		screen := s.Capture()
		if fn(screen) {
			return nil
		}
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for condition after %v", timeout)
}

// Kill terminates the tmux session gracefully to allow coverage data flush.
func (s *Session) Kill() {
	// Send Ctrl+C first to allow graceful shutdown (needed for coverage data)
	if s.isAliveUnlocked() {
		_ = exec.Command("tmux", "send-keys", "-t", s.id, "C-c").Run()
		time.Sleep(200 * time.Millisecond) // Give app time to flush coverage
	}
	cmd := exec.Command("tmux", "kill-session", "-t", s.id)
	_ = cmd.Run() // Ignore errors - session may already be dead
}

// mapKey maps friendly key names to tmux key names.
func mapKey(key string) string {
	switch strings.ToLower(key) {
	case "enter", "return":
		return "Enter"
	case "escape", "esc":
		return "Escape"
	case "tab":
		return "Tab"
	case "backspace":
		return "BSpace"
	case "space":
		return "Space"
	case "up":
		return "Up"
	case "down":
		return "Down"
	case "left":
		return "Left"
	case "right":
		return "Right"
	case "ctrl+c":
		return "C-c"
	case "ctrl+u":
		return "C-u"
	case "ctrl+a":
		return "C-a"
	case "ctrl+e":
		return "C-e"
	default:
		return key
	}
}

// shellQuoteArgs quotes arguments for shell execution.
// Args containing spaces, quotes, or special characters are wrapped in single quotes.
func shellQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		// Use single quotes for shell safety, escape existing single quotes
		if strings.ContainsAny(arg, " \t\n\"'\\{}[]<>|&;$`()") {
			// Replace single quotes with '\'' (end quote, escaped quote, start quote)
			escaped := strings.ReplaceAll(arg, "'", `'\''`)
			quoted[i] = "'" + escaped + "'"
		} else {
			quoted[i] = arg
		}
	}
	return strings.Join(quoted, " ")
}
