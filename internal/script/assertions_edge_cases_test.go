package script

import (
	"context"
	"strings"
	"testing"
)

func TestAssertionEdgeCases(t *testing.T) {
	t.Run("toThrow catches errors", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("function throws", function() {
				pm.expect(function() { throw new Error("boom"); }).toThrow();
			});
			pm.test("function throws with message", function() {
				pm.expect(function() { throw new Error("boom"); }).toThrow("boom");
			});
			pm.test("function does not throw", function() {
				pm.expect(function() { return 1; }).not.toThrow();
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 3 {
			t.Fatalf("Expected 3 test results, got %d", len(results))
		}
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("toBeInstanceOf works", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("array is Array", function() {
				pm.expect([1,2,3]).toBeInstanceOf(Array);
			});
			pm.test("object is Object", function() {
				pm.expect({}).toBeInstanceOf(Object);
			});
			pm.test("error is Error", function() {
				pm.expect(new Error("test")).toBeInstanceOf(Error);
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("null and undefined handling", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("null toBeNull", function() {
				pm.expect(null).toBeNull();
			});
			pm.test("undefined toBeUndefined", function() {
				pm.expect(undefined).toBeUndefined();
			});
			pm.test("null not toBeUndefined", function() {
				pm.expect(null).not.toBeUndefined();
			});
			pm.test("undefined not toBeNull", function() {
				pm.expect(undefined).not.toBeNull();
			});
			pm.test("null toBeFalsy", function() {
				pm.expect(null).toBeFalsy();
			});
			pm.test("undefined toBeFalsy", function() {
				pm.expect(undefined).toBeFalsy();
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("empty values", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("empty string toBeFalsy", function() {
				pm.expect("").toBeFalsy();
			});
			pm.test("empty array toHaveLength 0", function() {
				pm.expect([]).toHaveLength(0);
			});
			pm.test("zero toBeFalsy", function() {
				pm.expect(0).toBeFalsy();
			});
			pm.test("empty object toBeDefined", function() {
				pm.expect({}).toBeDefined();
			});
			pm.test("empty string toBeDefined", function() {
				pm.expect("").toBeDefined();
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("type coercion with toBe vs toEqual", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("1 toBe 1", function() {
				pm.expect(1).toBe(1);
			});
			pm.test("'1' not toBe 1", function() {
				pm.expect("1").not.toBe(1);
			});
			pm.test("objects with same values toEqual", function() {
				pm.expect({a:1,b:2}).toEqual({a:1,b:2});
			});
			pm.test("nested objects toEqual", function() {
				pm.expect({a:{b:1}}).toEqual({a:{b:1}});
			});
			pm.test("arrays toEqual", function() {
				pm.expect([1,[2,3]]).toEqual([1,[2,3]]);
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("chained .not assertions", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("not toBe", function() { pm.expect(1).not.toBe(2); });
			pm.test("not toEqual", function() { pm.expect({a:1}).not.toEqual({a:2}); });
			pm.test("not toContain", function() { pm.expect("abc").not.toContain("z"); });
			pm.test("not toMatch", function() { pm.expect("abc").not.toMatch("\\d+"); });
			pm.test("not toBeGreaterThan", function() { pm.expect(1).not.toBeGreaterThan(5); });
			pm.test("not toBeLessThan", function() { pm.expect(5).not.toBeLessThan(1); });
			pm.test("not toBeNull", function() { pm.expect(1).not.toBeNull(); });
			pm.test("not toBeUndefined", function() { pm.expect(1).not.toBeUndefined(); });
			pm.test("not toBeTruthy", function() { pm.expect(0).not.toBeTruthy(); });
			pm.test("not toBeFalsy", function() { pm.expect(1).not.toBeFalsy(); });
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("error messages are descriptive", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, _ = scope.Execute(context.Background(), `
			pm.test("toBe failure", function() { pm.expect(1).toBe(2); });
			pm.test("toContain failure", function() { pm.expect("abc").toContain("xyz"); });
			pm.test("toBeGreaterThan failure", function() { pm.expect(1).toBeGreaterThan(5); });
		`)

		results := scope.GetTestResults()
		if len(results) != 3 {
			t.Fatalf("Expected 3 test results, got %d", len(results))
		}

		// All should fail
		for _, r := range results {
			if r.Passed {
				t.Errorf("Test '%s' should have failed", r.Name)
			}
			if r.Error == "" {
				t.Errorf("Test '%s' should have error message", r.Name)
			}
		}

		// Check specific error messages
		if !strings.Contains(results[0].Error, "Expected") {
			t.Errorf("Error message should contain 'Expected': %s", results[0].Error)
		}
	})

	t.Run("test with boolean assertion (not function)", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("true boolean", true);
			pm.test("false boolean", false);
			pm.test("truthy value", 1);
			pm.test("falsy value", 0);
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 4 {
			t.Fatalf("Expected 4 test results, got %d", len(results))
		}

		if !results[0].Passed {
			t.Error("true boolean should pass")
		}
		if results[1].Passed {
			t.Error("false boolean should fail")
		}
		if !results[2].Passed {
			t.Error("truthy value should pass")
		}
		if results[3].Passed {
			t.Error("falsy value should fail")
		}
	})

	t.Run("test function returning false fails", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("return false", function() { return false; });
			pm.test("return true", function() { return true; });
			pm.test("return nothing", function() { });
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 3 {
			t.Fatalf("Expected 3 test results, got %d", len(results))
		}

		if results[0].Passed {
			t.Error("return false should fail")
		}
		if !results[1].Passed {
			t.Error("return true should pass")
		}
		if !results[2].Passed {
			t.Error("return nothing should pass (undefined is not false)")
		}
	})

	t.Run("response.json() parsing", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseBody(`{"user": {"name": "test", "id": 123}, "items": [1,2,3]}`)

		_, err := scope.Execute(context.Background(), `
			pm.test("json parsing works", function() {
				var json = pm.response.json();
				pm.expect(json.user.name).toBe("test");
			});
			pm.test("nested access", function() {
				var json = pm.response.json();
				pm.expect(json.user.id).toBe(123);
			});
			pm.test("array access", function() {
				var json = pm.response.json();
				pm.expect(json.items).toHaveLength(3);
				pm.expect(json.items[0]).toBe(1);
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("response.json() with invalid JSON", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseBody(`not valid json`)

		_, err := scope.Execute(context.Background(), `
			pm.test("invalid json returns null", function() {
				var json = pm.response.json();
				pm.expect(json).toBeNull();
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if !results[0].Passed {
			t.Errorf("Test should have passed: %s", results[0].Error)
		}
	})

	t.Run("multiple tests in sequence", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)

		// First execution
		_, err := scope.Execute(context.Background(), `
			pm.test("first test", function() {
				pm.expect(pm.response.status).toBe(200);
			});
		`)
		if err != nil {
			t.Fatalf("First execution failed: %v", err)
		}

		// Second execution (same scope)
		_, err = scope.Execute(context.Background(), `
			pm.test("second test", function() {
				pm.expect(pm.response.status).toBe(200);
			});
		`)
		if err != nil {
			t.Fatalf("Second execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 2 {
			t.Fatalf("Expected 2 test results, got %d", len(results))
		}
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("ClearTestResults works", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, _ = scope.Execute(context.Background(), `
			pm.test("test1", function() { pm.expect(1).toBe(1); });
		`)

		if len(scope.GetTestResults()) != 1 {
			t.Fatal("Should have 1 result")
		}

		scope.ClearTestResults()

		if len(scope.GetTestResults()) != 0 {
			t.Fatal("Should have 0 results after clear")
		}

		_, _ = scope.Execute(context.Background(), `
			pm.test("test2", function() { pm.expect(2).toBe(2); });
		`)

		results := scope.GetTestResults()
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results[0].Name != "test2" {
			t.Errorf("Expected test name 'test2', got '%s'", results[0].Name)
		}
	})

	t.Run("AllTestsPassed helper", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		// No tests = false (total must be > 0)
		if scope.AllTestsPassed() {
			t.Error("Should be false with no tests")
		}

		_, _ = scope.Execute(context.Background(), `
			pm.test("pass", function() { pm.expect(1).toBe(1); });
		`)

		if !scope.AllTestsPassed() {
			t.Error("Should be true with all passing")
		}

		_, _ = scope.Execute(context.Background(), `
			pm.test("fail", function() { pm.expect(1).toBe(2); });
		`)

		if scope.AllTestsPassed() {
			t.Error("Should be false with one failing")
		}
	})
}

func TestCurrierAPIStillWorks(t *testing.T) {
	t.Run("currier.test works", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)

		_, err := scope.Execute(context.Background(), `
			currier.test("status check", function() {
				currier.expect(currier.response.status).toBe(200);
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 1 || !results[0].Passed {
			t.Error("currier.test should work")
		}
	})

	t.Run("currier and pm can be mixed", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)

		_, err := scope.Execute(context.Background(), `
			pm.test("mixed test 1", function() {
				currier.expect(pm.response.status).toBe(200);
			});
			currier.test("mixed test 2", function() {
				pm.expect(currier.response.status).toBe(200);
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})
}

func TestUtilities(t *testing.T) {
	t.Run("base64 encode/decode", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("base64 encode", function() {
				pm.expect(pm.base64.encode("hello")).toBe("aGVsbG8=");
			});
			pm.test("base64 decode", function() {
				pm.expect(pm.base64.decode("aGVsbG8=")).toBe("hello");
			});
			pm.test("roundtrip", function() {
				var original = "test string";
				var encoded = pm.base64.encode(original);
				var decoded = pm.base64.decode(encoded);
				pm.expect(decoded).toBe(original);
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("crypto md5", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("md5 hash", function() {
				pm.expect(pm.crypto.md5("hello")).toBe("5d41402abc4b2a76b9719d911017c592");
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if !results[0].Passed {
			t.Errorf("Test failed: %s", results[0].Error)
		}
	})

	t.Run("crypto sha256", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("sha256 hash", function() {
				pm.expect(pm.crypto.sha256("hello")).toBe("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824");
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if !results[0].Passed {
			t.Errorf("Test failed: %s", results[0].Error)
		}
	})
}

func TestFormatTestResultsOutput(t *testing.T) {
	results := []TestResult{
		{Name: "passing test", Passed: true, Error: ""},
		{Name: "failing test", Passed: false, Error: "Expected 1 to be 2"},
	}

	output := FormatTestResults(results)

	if !strings.Contains(output, "✓ passing test") {
		t.Error("Should contain checkmark for passing test")
	}
	if !strings.Contains(output, "✗ failing test") {
		t.Error("Should contain X for failing test")
	}
	if !strings.Contains(output, "Expected 1 to be 2") {
		t.Error("Should contain error message")
	}
}

func TestGoHelperFunctions(t *testing.T) {
	t.Run("AssertEqual", func(t *testing.T) {
		if !AssertEqual(1, 1) {
			t.Error("1 should equal 1")
		}
		if AssertEqual(1, 2) {
			t.Error("1 should not equal 2")
		}
		if !AssertEqual(map[string]int{"a": 1}, map[string]int{"a": 1}) {
			t.Error("equal maps should be equal")
		}
	})

	t.Run("AssertContains", func(t *testing.T) {
		if !AssertContains("hello world", "world") {
			t.Error("string should contain substring")
		}
		if AssertContains("hello", "xyz") {
			t.Error("string should not contain missing substring")
		}
		if !AssertContains([]interface{}{1, 2, 3}, 2) {
			t.Error("array should contain element")
		}
	})

	t.Run("AssertMatch", func(t *testing.T) {
		if !AssertMatch("hello123", "\\d+") {
			t.Error("should match digits")
		}
		if AssertMatch("hello", "\\d+") {
			t.Error("should not match digits")
		}
	})

	t.Run("AssertJSONEqual", func(t *testing.T) {
		if !AssertJSONEqual(`{"a":1}`, `{"a":1}`) {
			t.Error("equal JSON should be equal")
		}
		if !AssertJSONEqual(`{"a":1,"b":2}`, `{"b":2,"a":1}`) {
			t.Error("JSON with different key order should be equal")
		}
		if AssertJSONEqual(`{"a":1}`, `{"a":2}`) {
			t.Error("different JSON should not be equal")
		}
	})
}
