package harness

import (
	"fmt"
	"strings"
	"testing"
)

// Assertions provides E2E-specific assertions.
type Assertions struct {
	t *testing.T
}

// NewAssertions creates an assertions helper.
func NewAssertions(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

// OutputContains asserts the output contains all given strings.
func (a *Assertions) OutputContains(output string, expected ...string) {
	a.t.Helper()
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			a.t.Errorf("expected output to contain %q, got:\n%s", exp, truncate(output, 500))
		}
	}
}

// OutputNotContains asserts the output does not contain any of the given strings.
func (a *Assertions) OutputNotContains(output string, unexpected ...string) {
	a.t.Helper()
	for _, unexp := range unexpected {
		if strings.Contains(output, unexp) {
			a.t.Errorf("expected output NOT to contain %q, got:\n%s", unexp, truncate(output, 500))
		}
	}
}

// StatusCode asserts the response contains expected status code.
func (a *Assertions) StatusCode(output string, code int) {
	a.t.Helper()
	expected := fmt.Sprintf("%d", code)
	if !strings.Contains(output, expected) {
		a.t.Errorf("expected status code %d in output:\n%s", code, truncate(output, 500))
	}
}

// HelpVisible asserts the help overlay is visible.
func (a *Assertions) HelpVisible(output string) {
	a.t.Helper()
	indicators := []string{"Currier Help", "Navigation"}
	for _, ind := range indicators {
		if !strings.Contains(output, ind) {
			a.t.Errorf("help overlay not visible, missing %q in output:\n%s", ind, truncate(output, 500))
			return
		}
	}
}

// HelpNotVisible asserts the help overlay is not visible.
func (a *Assertions) HelpNotVisible(output string) {
	a.t.Helper()
	if strings.Contains(output, "Currier Help") {
		a.t.Errorf("help overlay should not be visible, but found 'Currier Help' in output")
	}
}

// PaneVisible asserts a pane label is visible in the output.
func (a *Assertions) PaneVisible(output string, paneName string) {
	a.t.Helper()
	if !strings.Contains(output, paneName) {
		a.t.Errorf("expected pane %q to be visible in output:\n%s", paneName, truncate(output, 500))
	}
}

// NoError asserts the output doesn't contain error indicators.
func (a *Assertions) NoError(output string) {
	a.t.Helper()
	errorIndicators := []string{"Error:", "error:", "panic:", "PANIC:"}
	for _, ind := range errorIndicators {
		if strings.Contains(output, ind) {
			a.t.Errorf("unexpected error in output: found %q in:\n%s", ind, truncate(output, 500))
			return
		}
	}
}

// ResponseReceived asserts a response was received (not loading/empty).
func (a *Assertions) ResponseReceived(output string) {
	a.t.Helper()
	// Check that we don't have the "No response yet" or loading indicator
	if strings.Contains(output, "No response yet") {
		a.t.Errorf("no response received yet")
	}
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
