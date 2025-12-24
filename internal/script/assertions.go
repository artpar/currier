package script

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// TestResult represents the result of a single test.
type TestResult struct {
	Name   string
	Passed bool
	Error  string
}

// TestSummary provides aggregate test statistics.
type TestSummary struct {
	Total  int
	Passed int
	Failed int
}

// ScopeWithAssertions extends Scope with test assertion capabilities.
type ScopeWithAssertions struct {
	*Scope
	testMu      sync.RWMutex
	testResults []TestResult
	initialized bool
}

// NewScopeWithAssertions creates a new scope with assertion capabilities.
func NewScopeWithAssertions() *ScopeWithAssertions {
	s := &ScopeWithAssertions{
		Scope:       NewScope(),
		testResults: make([]TestResult, 0),
		initialized: false,
	}
	return s
}

// initAssertions sets up the assertion API.
func (s *ScopeWithAssertions) initAssertions() {
	// Register the test function (only needs to be done once, but is idempotent)
	if !s.initialized {
		s.engine.RegisterFunction("__currier_test", s.testFunc())
		s.initialized = true
	}

	// Add the test and expect functions via JavaScript
	script := `
		currier.test = function(name, assertion) {
			if (typeof assertion === 'function') {
				try {
					var result = assertion();
					if (result === false) {
						__currier_test(name, false, "");
					} else {
						__currier_test(name, true, "");
					}
				} catch(e) {
					__currier_test(name, false, e.message || e.toString());
				}
			} else {
				__currier_test(name, !!assertion, "");
			}
		};

		currier.expect = function(actual) {
			return new CurrierExpect(actual, false);
		};

		function CurrierExpect(actual, negated) {
			this.actual = actual;
			this.negated = negated;

			Object.defineProperty(this, 'not', {
				get: function() {
					return new CurrierExpect(actual, !negated);
				}
			});
		}

		CurrierExpect.prototype._assert = function(passed, message) {
			var finalPassed = this.negated ? !passed : passed;
			if (!finalPassed) {
				throw new Error(message);
			}
		};

		CurrierExpect.prototype.toBe = function(expected) {
			var passed = this.actual === expected;
			this._assert(passed, "Expected " + JSON.stringify(this.actual) + " to be " + JSON.stringify(expected));
		};

		CurrierExpect.prototype.toEqual = function(expected) {
			var passed = JSON.stringify(this.actual) === JSON.stringify(expected);
			this._assert(passed, "Expected " + JSON.stringify(this.actual) + " to equal " + JSON.stringify(expected));
		};

		CurrierExpect.prototype.toContain = function(expected) {
			var passed = false;
			if (typeof this.actual === 'string') {
				passed = this.actual.indexOf(expected) !== -1;
			} else if (Array.isArray(this.actual)) {
				passed = this.actual.indexOf(expected) !== -1;
			}
			this._assert(passed, "Expected " + JSON.stringify(this.actual) + " to contain " + JSON.stringify(expected));
		};

		CurrierExpect.prototype.toMatch = function(pattern) {
			var regex = pattern instanceof RegExp ? pattern : new RegExp(pattern);
			var passed = regex.test(this.actual);
			this._assert(passed, "Expected " + JSON.stringify(this.actual) + " to match " + pattern);
		};

		CurrierExpect.prototype.toBeGreaterThan = function(expected) {
			var passed = this.actual > expected;
			this._assert(passed, "Expected " + this.actual + " to be greater than " + expected);
		};

		CurrierExpect.prototype.toBeLessThan = function(expected) {
			var passed = this.actual < expected;
			this._assert(passed, "Expected " + this.actual + " to be less than " + expected);
		};

		CurrierExpect.prototype.toBeGreaterThanOrEqual = function(expected) {
			var passed = this.actual >= expected;
			this._assert(passed, "Expected " + this.actual + " to be greater than or equal to " + expected);
		};

		CurrierExpect.prototype.toBeLessThanOrEqual = function(expected) {
			var passed = this.actual <= expected;
			this._assert(passed, "Expected " + this.actual + " to be less than or equal to " + expected);
		};

		CurrierExpect.prototype.toBeNull = function() {
			var passed = this.actual === null;
			this._assert(passed, "Expected " + JSON.stringify(this.actual) + " to be null");
		};

		CurrierExpect.prototype.toBeUndefined = function() {
			var passed = this.actual === undefined;
			this._assert(passed, "Expected value to be undefined");
		};

		CurrierExpect.prototype.toBeDefined = function() {
			var passed = this.actual !== undefined;
			this._assert(passed, "Expected value to be defined");
		};

		CurrierExpect.prototype.toBeTruthy = function() {
			var passed = !!this.actual;
			this._assert(passed, "Expected " + JSON.stringify(this.actual) + " to be truthy");
		};

		CurrierExpect.prototype.toBeFalsy = function() {
			var passed = !this.actual;
			this._assert(passed, "Expected " + JSON.stringify(this.actual) + " to be falsy");
		};

		CurrierExpect.prototype.toHaveProperty = function(property, value) {
			var passed = this.actual.hasOwnProperty(property);
			if (passed && arguments.length > 1) {
				passed = this.actual[property] === value;
			}
			this._assert(passed, "Expected object to have property " + property);
		};

		CurrierExpect.prototype.toHaveLength = function(length) {
			var actualLength = this.actual.length;
			var passed = actualLength === length;
			this._assert(passed, "Expected length " + actualLength + " to be " + length);
		};

		CurrierExpect.prototype.toBeInstanceOf = function(constructor) {
			var passed = this.actual instanceof constructor;
			this._assert(passed, "Expected value to be instance of " + (constructor.name || constructor));
		};

		CurrierExpect.prototype.toThrow = function(message) {
			var passed = false;
			var error = null;
			if (typeof this.actual === 'function') {
				try {
					this.actual();
				} catch(e) {
					passed = true;
					error = e;
					if (message && e.message.indexOf(message) === -1) {
						passed = false;
					}
				}
			}
			this._assert(passed, "Expected function to throw" + (message ? " with message containing " + message : ""));
		};
	`

	// Execute the setup script directly on the engine (bypassing the scope's Execute which would refresh)
	s.engine.Execute(context.Background(), script)
}

// Execute runs a script in this scope with assertion support.
func (s *ScopeWithAssertions) Execute(ctx context.Context, script string) (interface{}, error) {
	// First, let the parent refresh the currier object
	s.refreshCurrierObject()

	// Then add our assertion API
	s.initAssertions()

	// Now execute the script
	return s.engine.Execute(ctx, script)
}

// testFunc returns the Go function that handles test results.
func (s *ScopeWithAssertions) testFunc() func(string, bool, string) {
	return func(name string, passed bool, errorMsg string) {
		s.testMu.Lock()
		defer s.testMu.Unlock()

		s.testResults = append(s.testResults, TestResult{
			Name:   name,
			Passed: passed,
			Error:  errorMsg,
		})
	}
}

// GetTestResults returns all test results.
func (s *ScopeWithAssertions) GetTestResults() []TestResult {
	s.testMu.RLock()
	defer s.testMu.RUnlock()

	results := make([]TestResult, len(s.testResults))
	copy(results, s.testResults)
	return results
}

// ClearTestResults clears all test results.
func (s *ScopeWithAssertions) ClearTestResults() {
	s.testMu.Lock()
	defer s.testMu.Unlock()
	s.testResults = make([]TestResult, 0)
}

// GetTestSummary returns aggregate test statistics.
func (s *ScopeWithAssertions) GetTestSummary() TestSummary {
	s.testMu.RLock()
	defer s.testMu.RUnlock()

	summary := TestSummary{
		Total: len(s.testResults),
	}

	for _, result := range s.testResults {
		if result.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}

	return summary
}

// AllTestsPassed returns true if all tests passed.
func (s *ScopeWithAssertions) AllTestsPassed() bool {
	summary := s.GetTestSummary()
	return summary.Failed == 0 && summary.Total > 0
}

// Helper functions for use in Go code

// AssertEqual checks if two values are equal.
func AssertEqual(actual, expected interface{}) bool {
	return reflect.DeepEqual(actual, expected)
}

// AssertContains checks if a string contains a substring or an array contains an element.
func AssertContains(container interface{}, element interface{}) bool {
	switch c := container.(type) {
	case string:
		if s, ok := element.(string); ok {
			return strings.Contains(c, s)
		}
	case []interface{}:
		for _, item := range c {
			if reflect.DeepEqual(item, element) {
				return true
			}
		}
	}
	return false
}

// AssertMatch checks if a string matches a regex pattern.
func AssertMatch(value string, pattern string) bool {
	matched, err := regexp.MatchString(pattern, value)
	return err == nil && matched
}

// AssertJSONEqual checks if two JSON strings represent equal values.
func AssertJSONEqual(actual, expected string) bool {
	var actualObj, expectedObj interface{}
	if err := json.Unmarshal([]byte(actual), &actualObj); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		return false
	}
	return reflect.DeepEqual(actualObj, expectedObj)
}

// FormatTestResults formats test results for display.
func FormatTestResults(results []TestResult) string {
	var sb strings.Builder

	for _, r := range results {
		if r.Passed {
			sb.WriteString(fmt.Sprintf("  ✓ %s\n", r.Name))
		} else {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", r.Name))
			if r.Error != "" {
				sb.WriteString(fmt.Sprintf("    Error: %s\n", r.Error))
			}
		}
	}

	return sb.String()
}
