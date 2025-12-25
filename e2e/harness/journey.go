package harness

import (
	"fmt"
	"testing"
	"time"
)

// Journey represents a user journey test.
type Journey struct {
	t           *testing.T
	name        string
	harness     *E2EHarness
	session     *TUISession
	steps       []*Step
	currentStep int
}

// Step represents a single step in a journey.
type Step struct {
	name        string
	actions     []func(*TUISession)
	assertions  []func(*testing.T, *State)
	waitFor     func(*State) bool
	waitTimeout time.Duration
}

// NewJourney creates a new journey test.
func NewJourney(t *testing.T, name string) *Journey {
	h := New(t, Config{
		GoldenDir: "../golden/journeys",
	})

	return &Journey{
		t:       t,
		name:    name,
		harness: h,
		steps:   make([]*Step, 0),
	}
}

// Step adds a new step to the journey.
func (j *Journey) Step(name string) *StepBuilder {
	step := &Step{
		name:        name,
		actions:     make([]func(*TUISession), 0),
		assertions:  make([]func(*testing.T, *State), 0),
		waitTimeout: 5 * time.Second,
	}
	j.steps = append(j.steps, step)
	return &StepBuilder{journey: j, step: step}
}

// Run executes the journey.
func (j *Journey) Run() {
	j.t.Helper()
	j.t.Run(j.name, func(t *testing.T) {
		// Start TUI session
		j.session = j.harness.TUI().Start(t)
		defer j.session.Quit()

		// Execute each step
		for i, step := range j.steps {
			j.currentStep = i
			t.Logf("Step %d: %s", i+1, step.name)

			// Execute actions
			for _, action := range step.actions {
				action(j.session)
			}

			// Wait for condition if specified
			if step.waitFor != nil {
				if err := j.waitForCondition(step.waitFor, step.waitTimeout); err != nil {
					t.Fatalf("Step %d (%s): %v", i+1, step.name, err)
				}
			}

			// Run assertions
			state := j.session.CaptureState()
			for _, assertion := range step.assertions {
				assertion(t, state)
			}
		}
	})
}

func (j *Journey) waitForCondition(condition func(*State) bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 50 * time.Millisecond

	for time.Now().Before(deadline) {
		state := j.session.CaptureState()
		if condition(state) {
			return nil
		}
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for condition after %v", timeout)
}

// StepBuilder provides a fluent API for building steps.
type StepBuilder struct {
	journey *Journey
	step    *Step
}

// SendKey adds a key press action.
func (b *StepBuilder) SendKey(key string) *StepBuilder {
	b.step.actions = append(b.step.actions, func(s *TUISession) {
		s.SendKey(key)
	})
	return b
}

// SendKeys adds multiple key press actions.
func (b *StepBuilder) SendKeys(keys ...string) *StepBuilder {
	b.step.actions = append(b.step.actions, func(s *TUISession) {
		s.SendKeys(keys...)
	})
	return b
}

// Type adds a typing action.
func (b *StepBuilder) Type(text string) *StepBuilder {
	b.step.actions = append(b.step.actions, func(s *TUISession) {
		s.Type(text)
	})
	return b
}

// Wait adds a pause.
func (b *StepBuilder) Wait(d time.Duration) *StepBuilder {
	b.step.actions = append(b.step.actions, func(s *TUISession) {
		s.Wait(d)
	})
	return b
}

// WaitFor adds a condition to wait for before assertions.
func (b *StepBuilder) WaitFor(condition func(*State) bool, timeout time.Duration) *StepBuilder {
	b.step.waitFor = condition
	b.step.waitTimeout = timeout
	return b
}

// ExpectMode asserts the current mode.
func (b *StepBuilder) ExpectMode(mode string) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.MainView.Mode != mode {
			t.Errorf("Expected mode %q, got %q", mode, s.MainView.Mode)
		}
	})
	return b
}

// ExpectFocus asserts the focused pane.
func (b *StepBuilder) ExpectFocus(pane string) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.MainView.FocusedPane != pane {
			t.Errorf("Expected focus on %q, got %q", pane, s.MainView.FocusedPane)
		}
	})
	return b
}

// ExpectURL asserts the current URL.
func (b *StepBuilder) ExpectURL(url string) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Request.URL != url {
			t.Errorf("Expected URL %q, got %q", url, s.Request.URL)
		}
	})
	return b
}

// ExpectMethod asserts the current HTTP method.
func (b *StepBuilder) ExpectMethod(method string) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Request.Method != method {
			t.Errorf("Expected method %q, got %q", method, s.Request.Method)
		}
	})
	return b
}

// ExpectHasResponse asserts that a response exists.
func (b *StepBuilder) ExpectHasResponse() *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if !s.Response.HasResponse {
			t.Error("Expected response to exist")
		}
	})
	return b
}

// ExpectStatusCode asserts the response status code.
func (b *StepBuilder) ExpectStatusCode(code int) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Response.StatusCode != code {
			t.Errorf("Expected status code %d, got %d", code, s.Response.StatusCode)
		}
	})
	return b
}

// ExpectError asserts an error exists.
func (b *StepBuilder) ExpectError() *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Response.Error == "" {
			t.Error("Expected an error")
		}
	})
	return b
}

// ExpectNoError asserts no error exists.
func (b *StepBuilder) ExpectNoError() *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Response.Error != "" {
			t.Errorf("Expected no error, got: %s", s.Response.Error)
		}
	})
	return b
}

// ExpectViewMode asserts the collection tree view mode.
func (b *StepBuilder) ExpectViewMode(mode string) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Tree.ViewMode != mode {
			t.Errorf("Expected view mode %q, got %q", mode, s.Tree.ViewMode)
		}
	})
	return b
}

// ExpectHistoryCount asserts the number of history entries.
func (b *StepBuilder) ExpectHistoryCount(count int) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Tree.HistoryCount != count {
			t.Errorf("Expected history count %d, got %d", count, s.Tree.HistoryCount)
		}
	})
	return b
}

// ExpectState adds a custom state assertion.
func (b *StepBuilder) ExpectState(assertion func(*testing.T, *State)) *StepBuilder {
	b.step.assertions = append(b.step.assertions, assertion)
	return b
}

// ExpectIsEditing asserts editing state.
func (b *StepBuilder) ExpectIsEditing(editing bool) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Request.IsEditing != editing {
			t.Errorf("Expected IsEditing=%v, got %v", editing, s.Request.IsEditing)
		}
	})
	return b
}

// ExpectEditingField asserts which field is being edited.
func (b *StepBuilder) ExpectEditingField(field string) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Request.EditingField != field {
			t.Errorf("Expected EditingField=%q, got %q", field, s.Request.EditingField)
		}
	})
	return b
}

// ExpectActiveTab asserts the active request tab.
func (b *StepBuilder) ExpectActiveTab(tab string) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Request.ActiveTab != tab {
			t.Errorf("Expected ActiveTab=%q, got %q", tab, s.Request.ActiveTab)
		}
	})
	return b
}

// ExpectIsLoading asserts loading state.
func (b *StepBuilder) ExpectIsLoading(loading bool) *StepBuilder {
	b.step.assertions = append(b.step.assertions, func(t *testing.T, s *State) {
		t.Helper()
		if s.Response.IsLoading != loading {
			t.Errorf("Expected IsLoading=%v, got %v", loading, s.Response.IsLoading)
		}
	})
	return b
}

// Step starts a new step (returns to journey to continue chaining).
func (b *StepBuilder) Step(name string) *StepBuilder {
	return b.journey.Step(name)
}

// Run executes the journey (terminal operation).
func (b *StepBuilder) Run() {
	b.journey.Run()
}
