// Package harness provides E2E testing utilities for Currier.
package harness

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// E2EHarness is the main test orchestrator.
type E2EHarness struct {
	t         *testing.T
	server    *httptest.Server
	tmpDir    string
	goldenDir string
	timeout   time.Duration
}

// Config configures the harness.
type Config struct {
	ServerHandlers map[string]http.HandlerFunc
	GoldenDir      string
	Timeout        time.Duration // Default: 5 seconds
}

// New creates a new E2E harness.
func New(t *testing.T, cfg Config) *E2EHarness {
	t.Helper()

	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	h := &E2EHarness{
		t:         t,
		goldenDir: cfg.GoldenDir,
		timeout:   cfg.Timeout,
	}

	// Create temporary directory for test data
	tmpDir, err := os.MkdirTemp("", "currier-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	h.tmpDir = tmpDir

	// Start test server if handlers provided
	if len(cfg.ServerHandlers) > 0 {
		mux := http.NewServeMux()
		for pattern, handler := range cfg.ServerHandlers {
			mux.HandleFunc(pattern, handler)
		}
		h.server = httptest.NewServer(mux)
	}

	t.Cleanup(h.cleanup)
	return h
}

func (h *E2EHarness) cleanup() {
	if h.server != nil {
		h.server.Close()
	}
	os.RemoveAll(h.tmpDir)
}

// ServerURL returns the test server URL.
func (h *E2EHarness) ServerURL() string {
	if h.server == nil {
		return ""
	}
	return h.server.URL
}

// TmpDir returns the temporary directory path.
func (h *E2EHarness) TmpDir() string {
	return h.tmpDir
}

// Timeout returns the configured timeout.
func (h *E2EHarness) Timeout() time.Duration {
	return h.timeout
}

// T returns the testing.T instance.
func (h *E2EHarness) T() *testing.T {
	return h.t
}

// CLI returns a CLI runner for this harness.
func (h *E2EHarness) CLI() *CLIRunner {
	return &CLIRunner{harness: h}
}

// TUI returns a TUI runner for this harness.
func (h *E2EHarness) TUI() *TUIRunner {
	return &TUIRunner{harness: h}
}

// Golden returns a golden file manager for this harness.
func (h *E2EHarness) Golden() *GoldenManager {
	return NewGoldenManager(h.goldenDir)
}
