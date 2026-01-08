package tmux_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/artpar/currier/e2e/tmux"
)

// =============================================================================
// CAPTURE TAB TESTS - Test the Capture tab in TUI mode
// =============================================================================

// switchToCaptureMode sends keys to switch from the default History mode to Capture mode
// The app starts in History mode, so we need: C (History->Collections), C (Collections->Capture)
func switchToCaptureMode(sess *tmux.Session) {
	sess.SendKey("C") // History -> Collections
	time.Sleep(200 * time.Millisecond)
	sess.SendKey("C") // Collections -> Capture
	time.Sleep(300 * time.Millisecond)
}

func TestCapture_SwitchToCaptureTab(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	// Wait for initial UI
	if err := sess.WaitFor("Collections", 5*time.Second); err != nil {
		t.Fatalf("App did not start: %v", err)
	}

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// In capture mode, header shows "Capture [OFF]" or "Capture [ON]" without "(C)" hint
	screen := sess.Capture()
	t.Logf("Screen after pressing C twice:\n%s", screen)

	// Verify we're in Capture mode - header should NOT have "(C)" hint
	if strings.Contains(screen, "Capture (C)") {
		t.Error("Should be in Capture mode but header still shows hint")
	}

	// Should see proxy status indicator
	if !strings.Contains(screen, "[OFF]") && !strings.Contains(screen, "[ON]") {
		t.Error("Should see proxy status indicator")
	}
}

func TestCapture_SwitchBackToCollections(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Switch back to Collections by pressing 'C' again (from Capture mode)
	sess.SendKey("C")
	time.Sleep(300 * time.Millisecond)

	// Should see Collections header without hint (active mode)
	screen := sess.Capture()
	// When in Collections mode, Collections header has no hint
	if strings.Contains(screen, "Collections (C)") {
		t.Error("Should be back in Collections mode")
		t.Logf("Screen:\n%s", screen)
	}
}

func TestCapture_ToggleProxyOn(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Verify initial state is OFF
	screen := sess.Capture()
	if !strings.Contains(screen, "[OFF]") {
		t.Error("Expected proxy to be OFF initially")
	}

	// Press 'p' to toggle proxy on
	sess.SendKey("p")
	time.Sleep(500 * time.Millisecond)

	// Should show proxy is on
	screen = sess.Capture()
	t.Logf("Screen after proxy toggle:\n%s", screen)

	if !strings.Contains(screen, "[ON]") {
		t.Error("Expected proxy to be ON after toggle")
	}

	// Toggle off before exiting
	sess.SendKey("p")
	time.Sleep(300 * time.Millisecond)
}

func TestCapture_ProxyToggleOffOn(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Initially proxy should be OFF
	screen := sess.Capture()
	if !strings.Contains(screen, "[OFF]") {
		t.Error("Expected proxy to be OFF initially")
		t.Logf("Screen:\n%s", screen)
	}

	// Toggle ON
	sess.SendKey("p")
	time.Sleep(500 * time.Millisecond)
	screen = sess.Capture()
	t.Logf("After toggle ON:\n%s", screen)

	if !strings.Contains(screen, "[ON]") {
		t.Error("Expected proxy to be ON after toggle")
	}

	// Toggle OFF
	sess.SendKey("p")
	time.Sleep(500 * time.Millisecond)
	screen = sess.Capture()
	t.Logf("After toggle OFF:\n%s", screen)

	if !strings.Contains(screen, "[OFF]") {
		t.Error("Expected proxy to be OFF after second toggle")
	}
}

func TestCapture_FilterByMethod(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Press 'm' to cycle method filter
	sess.SendKey("m")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	t.Logf("Screen with method filter:\n%s", screen)

	// Pressing 'm' cycles through method filters
	// The header should show the filter (GET, POST, etc.)
	// This is a basic smoke test - the filter UI may not be visible without captures
}

func TestCapture_FilterByStatus(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Press 's' to cycle status filter
	sess.SendKey("s")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	t.Logf("Screen with status filter:\n%s", screen)

	// Pressing 's' cycles through status filters
	// The header should show the filter (2xx, 3xx, etc.)
	// This is a basic smoke test
}

func TestCapture_ClearCaptures(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Press 'X' to clear all captures (should work even if empty)
	sess.SendKey("X")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	t.Logf("Screen after clear:\n%s", screen)

	// Should still be in capture mode
	if strings.Contains(screen, "Capture (C)") {
		t.Error("Should still be in Capture mode after clear")
	}
}

func TestCapture_NavigationKeys(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Test navigation keys (j/k) - should work even without captures
	sess.SendKey("j")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("j")
	time.Sleep(100 * time.Millisecond)
	sess.SendKey("k")
	time.Sleep(100 * time.Millisecond)

	screen := sess.Capture()
	t.Logf("Screen after navigation:\n%s", screen)

	// Should still be in capture mode
	if strings.Contains(screen, "Capture (C)") {
		t.Error("Should still be in Capture mode after navigation")
	}
}

func TestCapture_SwitchToHistoryFromCapture(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Press 'H' to switch to History mode
	sess.SendKey("H")
	time.Sleep(300 * time.Millisecond)

	screen := sess.Capture()
	t.Logf("Screen after H:\n%s", screen)

	// When in History mode, History header has no hint
	// and Collections should have (C) hint
	if strings.Contains(screen, "History (H)") {
		t.Error("Should be in History mode but History still shows hint")
	}
}

func TestCapture_RefreshKey(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Press 'r' to refresh captures
	sess.SendKey("r")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	t.Logf("Screen after refresh:\n%s", screen)

	// Should still be in capture mode
	if strings.Contains(screen, "Capture (C)") {
		t.Error("Should still be in Capture mode after refresh")
	}
}

func TestCapture_ClearFiltersKey(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Set a filter first
	sess.SendKey("m") // Set method filter
	time.Sleep(100 * time.Millisecond)

	// Press 'x' to clear filters
	sess.SendKey("x")
	time.Sleep(200 * time.Millisecond)

	screen := sess.Capture()
	t.Logf("Screen after clear filters:\n%s", screen)

	// Should still be in capture mode
	if strings.Contains(screen, "Capture (C)") {
		t.Error("Should still be in Capture mode after clear filters")
	}
}

func TestCapture_EscapeStaysInCaptureMode(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Verify we're in Capture mode
	screen := sess.Capture()
	if strings.Contains(screen, "Capture (C)") {
		t.Error("Should be in Capture mode")
	}

	// Press Escape - this should stay in Capture mode (not switch to Collections)
	sess.SendKey("Escape")
	time.Sleep(300 * time.Millisecond)

	screen = sess.Capture()
	t.Logf("Screen after Escape:\n%s", screen)

	// Should still be in Capture mode (Capture header shows [OFF] or [ON], no hint)
	if strings.Contains(screen, "Capture (C)") {
		t.Error("Should still be in Capture mode after Escape")
	}

	// Verify proxy status is still shown
	if !strings.Contains(screen, "[OFF]") && !strings.Contains(screen, "[ON]") {
		t.Error("Should see proxy status indicator")
	}
}

func TestCapture_ActuallyCapturesHTTPTraffic(t *testing.T) {
	sess := tmux.NewSession(t)
	defer sess.Kill()

	if err := sess.Start(binaryPath); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	sess.WaitFor("Collections", 5*time.Second)

	// Switch to Capture mode
	switchToCaptureMode(sess)

	// Toggle proxy ON
	sess.SendKey("p")
	time.Sleep(500 * time.Millisecond)

	// Capture screen to find the proxy port
	screen := sess.Capture()
	t.Logf("Screen after proxy start:\n%s", screen)

	// Verify proxy is ON
	if !strings.Contains(screen, "[ON]") {
		t.Fatal("Proxy should be ON")
	}

	// Find the port from "Proxy started on [::]:XXXXX"
	var proxyPort string
	lines := strings.Split(screen, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Proxy started on") {
			// Extract port number
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				// Get last part which should be the port
				proxyPort = strings.TrimSpace(parts[len(parts)-1])
				// Remove any trailing non-digit characters
				for i, c := range proxyPort {
					if c < '0' || c > '9' {
						proxyPort = proxyPort[:i]
						break
					}
				}
			}
		}
	}

	if proxyPort == "" {
		t.Log("Could not find proxy port, using default 8080")
		proxyPort = "8080"
	}
	t.Logf("Using proxy port: %s", proxyPort)

	// Make an HTTP request through the proxy
	proxyURL, _ := url.Parse("http://localhost:" + proxyPort)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	// Make request to a simple HTTP endpoint
	resp, err := client.Get("http://httpbin.org/get")
	if err != nil {
		t.Logf("HTTP request failed (may be network issue): %v", err)
		// Try a local request that might work
		resp, err = client.Get("http://example.com/")
		if err != nil {
			t.Logf("Fallback request also failed: %v", err)
		}
	}
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Logf("Request completed with status: %d", resp.StatusCode)
	}

	// Wait for capture to be recorded
	time.Sleep(500 * time.Millisecond)

	// Capture screen to see if request was captured
	screen = sess.Capture()
	t.Logf("Screen after HTTP request:\n%s", screen)

	// Check if the capture count is shown in header (e.g., "Capture [ON] (1)")
	hasCaptureCount := strings.Contains(screen, "(1)") || strings.Contains(screen, "(2)")

	// Also check that we no longer see "No captures"
	noCaptures := strings.Contains(screen, "No captures")

	if noCaptures {
		t.Error("Capture should have been recorded but still shows 'No captures'")
	} else if hasCaptureCount {
		t.Log("SUCCESS: HTTP request was captured and displayed in the TUI!")
	} else {
		t.Log("Note: Capture may have been recorded but count not visible in header")
	}

	// Toggle proxy off
	sess.SendKey("p")
	time.Sleep(300 * time.Millisecond)
}
