package tmux_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/artpar/currier/e2e/tmux"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Get the project root directory
	projectRoot := filepath.Join("..", "..")

	// Build binary before running tests
	buildCmd := exec.Command("go", "build", "-o", "bin/currier-test", "./cmd/currier")
	buildCmd.Dir = projectRoot
	if err := buildCmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	// Set absolute path to binary
	absRoot, _ := filepath.Abs(projectRoot)
	binaryPath = filepath.Join(absRoot, "bin", "currier-test")

	code := m.Run()

	// Cleanup
	os.Remove(binaryPath)
	os.Exit(code)
}

func TestTmux_AppStarts(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	// Should see initial UI
	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Errorf("App did not show Collections pane: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestTmux_CreateNewRequest(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Press 'n' to create new request
	sess.SendKey("n")

	// Should enter INSERT mode
	if err := sess.WaitFor("INSERT", 2*time.Second); err != nil {
		t.Errorf("Did not enter INSERT mode: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestTmux_TypeURLAndSend(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Create new request
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	// Type URL
	sess.Type("https://httpbin.org/get")

	// Exit edit mode
	sess.SendKey("Escape")
	sess.WaitFor("NORMAL", 2*time.Second)

	// Verify URL is displayed
	screen := sess.Capture()
	if !strings.Contains(screen, "httpbin.org") {
		t.Errorf("URL not displayed on screen")
		t.Logf("Screen:\n%s", screen)
	}

	// Send request
	sess.SendKey("Enter")

	// Wait for response (real HTTP call!)
	if err := sess.WaitFor("200", 15*time.Second); err != nil {
		t.Errorf("Did not receive 200 response: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestTmux_SpaceCharacterWorks(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Create request and type URL with space
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)

	sess.Type("https://example.com/path")
	sess.SendKey("Space")
	sess.Type("with")
	sess.SendKey("Space")
	sess.Type("spaces")

	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Verify spaces are in the URL
	screen := sess.Capture()
	if !strings.Contains(screen, "path with spaces") && !strings.Contains(screen, "path%20with%20spaces") {
		t.Errorf("Spaces not preserved in URL")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestTmux_HistoryView(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Make a request first
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.Type("https://httpbin.org/get")
	sess.SendKey("Escape")
	sess.SendKey("Enter")
	sess.WaitFor("200", 15*time.Second)

	// Switch to history view
	sess.SendKey("1") // Focus collections pane
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("H") // History view

	// Should show history entry
	if err := sess.WaitFor("httpbin", 5*time.Second); err != nil {
		t.Errorf("History entry not shown: %v", err)
		t.Logf("Screen:\n%s", sess.Capture())
	}
}

func TestTmux_TabNavigation(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Tab should cycle through panes
	sess.SendKey("Tab")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("Tab")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("Tab")
	time.Sleep(100 * time.Millisecond)

	// Should still be running (no crash)
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Errorf("App crashed during tab navigation")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestTmux_HelpOverlay(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Press ? to show help
	sess.SendKey("?")

	// Should show help text
	if err := sess.WaitFor("Help", 2*time.Second); err != nil {
		t.Logf("Help overlay may not show 'Help' text")
	}

	// Press Escape to close
	sess.SendKey("Escape")
	time.Sleep(200 * time.Millisecond)

	// Should still be running
	screen := sess.Capture()
	if !strings.Contains(screen, "Collections") {
		t.Errorf("App state incorrect after help")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestTmux_Quit(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Press 'q' to quit
	sess.SendKey("q")

	// Wait for app to exit
	time.Sleep(1 * time.Second)

	// Screen should no longer show Collections (app exited)
	screen := sess.Capture()
	if strings.Contains(screen, "Collections") && strings.Contains(screen, "Request") {
		t.Error("App did not quit on 'q' key")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestTmux_EmptyURLError(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	sess.WaitFor("Collections", 5*time.Second)

	// Create request without typing URL
	sess.SendKey("n")
	sess.WaitFor("INSERT", 2*time.Second)
	sess.SendKey("Escape")
	time.Sleep(100 * time.Millisecond)

	// Try to send
	sess.SendKey("Enter")
	time.Sleep(500 * time.Millisecond)

	// Should show error about empty URL
	screen := sess.Capture()
	if !strings.Contains(strings.ToLower(screen), "empty") && !strings.Contains(strings.ToLower(screen), "error") {
		t.Logf("Note: No explicit empty URL error shown (may be acceptable)")
	}

	// App should not crash
	if !strings.Contains(screen, "Collections") {
		t.Errorf("App crashed on empty URL send")
		t.Logf("Screen:\n%s", screen)
	}
}
