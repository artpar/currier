package script

import (
	"context"
	"testing"
)

func TestPmAlias(t *testing.T) {
	t.Run("pm.test works like currier.test", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)

		_, err := scope.Execute(context.Background(), `
			pm.test("Status is 200", function() {
				pm.expect(pm.response.status).toBe(200);
			});
		`)

		if err != nil {
			t.Fatalf("pm.test failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 1 {
			t.Fatalf("Expected 1 test result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("Test should have passed: %s", results[0].Error)
		}
	})

	t.Run("pm.expect works like currier.expect", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("expect toBe", function() {
				pm.expect(42).toBe(42);
			});
			pm.test("expect toEqual", function() {
				pm.expect({a: 1}).toEqual({a: 1});
			});
			pm.test("expect toContain", function() {
				pm.expect("hello world").toContain("world");
			});
		`)

		if err != nil {
			t.Fatalf("pm.expect tests failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 3 {
			t.Fatalf("Expected 3 test results, got %d", len(results))
		}
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' should have passed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("pm.response has all properties", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(201)
		scope.SetResponseStatusText("Created")
		scope.SetResponseBody(`{"id": 123}`)
		scope.SetResponseTime(150)
		scope.SetResponseSize(50)
		scope.SetResponseHeaders(map[string]string{"Content-Type": "application/json"})

		_, err := scope.Execute(context.Background(), `
			pm.test("response status", function() {
				pm.expect(pm.response.status).toBe(201);
			});
			pm.test("response statusText", function() {
				pm.expect(pm.response.statusText).toBe("Created");
			});
			pm.test("response body", function() {
				pm.expect(pm.response.body).toContain("123");
			});
			pm.test("response time", function() {
				pm.expect(pm.response.time).toBe(150);
			});
			pm.test("response size", function() {
				pm.expect(pm.response.size).toBe(50);
			});
			pm.test("response headers", function() {
				pm.expect(pm.response.headers["Content-Type"]).toBe("application/json");
			});
		`)

		if err != nil {
			t.Fatalf("pm.response tests failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("pm.environment get/set works", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetEnvironmentVariable("base_url", "https://api.example.com")

		_, err := scope.Execute(context.Background(), `
			pm.test("environment get", function() {
				pm.expect(pm.environment.get("base_url")).toBe("https://api.example.com");
			});
			pm.environment.set("new_var", "test_value");
			pm.test("environment set", function() {
				pm.expect(pm.environment.get("new_var")).toBe("test_value");
			});
		`)

		if err != nil {
			t.Fatalf("pm.environment tests failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("pm.setVariable and pm.getVariable work", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.setVariable("my_var", "my_value");
			pm.test("getVariable", function() {
				pm.expect(pm.getVariable("my_var")).toBe("my_value");
			});
		`)

		if err != nil {
			t.Fatalf("pm.setVariable tests failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("pm.request has all properties", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetRequestMethod("POST")
		scope.SetRequestURL("https://api.example.com/users")
		scope.SetRequestBody(`{"name": "test"}`)
		scope.SetRequestHeaders(map[string]string{"Authorization": "Bearer token123"})

		_, err := scope.Execute(context.Background(), `
			pm.test("request method", function() {
				pm.expect(pm.request.method).toBe("POST");
			});
			pm.test("request url", function() {
				pm.expect(pm.request.url).toBe("https://api.example.com/users");
			});
			pm.test("request body", function() {
				pm.expect(pm.request.body).toContain("test");
			});
			pm.test("request headers", function() {
				pm.expect(pm.request.headers["Authorization"]).toBe("Bearer token123");
			});
		`)

		if err != nil {
			t.Fatalf("pm.request tests failed: %v", err)
		}

		results := scope.GetTestResults()
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("all expect assertions work with pm", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("toBe", function() { pm.expect(1).toBe(1); });
			pm.test("toEqual", function() { pm.expect([1,2]).toEqual([1,2]); });
			pm.test("toContain string", function() { pm.expect("abc").toContain("b"); });
			pm.test("toContain array", function() { pm.expect([1,2,3]).toContain(2); });
			pm.test("toMatch", function() { pm.expect("hello123").toMatch("\\d+"); });
			pm.test("toBeGreaterThan", function() { pm.expect(5).toBeGreaterThan(3); });
			pm.test("toBeLessThan", function() { pm.expect(3).toBeLessThan(5); });
			pm.test("toBeGreaterThanOrEqual", function() { pm.expect(5).toBeGreaterThanOrEqual(5); });
			pm.test("toBeLessThanOrEqual", function() { pm.expect(5).toBeLessThanOrEqual(5); });
			pm.test("toBeNull", function() { pm.expect(null).toBeNull(); });
			pm.test("toBeUndefined", function() { pm.expect(undefined).toBeUndefined(); });
			pm.test("toBeDefined", function() { pm.expect(123).toBeDefined(); });
			pm.test("toBeTruthy", function() { pm.expect(1).toBeTruthy(); });
			pm.test("toBeFalsy", function() { pm.expect(0).toBeFalsy(); });
			pm.test("toHaveProperty", function() { pm.expect({a:1}).toHaveProperty("a"); });
			pm.test("toHaveProperty with value", function() { pm.expect({a:1}).toHaveProperty("a", 1); });
			pm.test("toHaveLength", function() { pm.expect([1,2,3]).toHaveLength(3); });
			pm.test("not.toBe", function() { pm.expect(1).not.toBe(2); });
			pm.test("not.toEqual", function() { pm.expect({a:1}).not.toEqual({a:2}); });
		`)

		if err != nil {
			t.Fatalf("All assertions test failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 19 {
			t.Errorf("Expected 19 test results, got %d", len(results))
		}
		for _, r := range results {
			if !r.Passed {
				t.Errorf("Test '%s' failed: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("failing tests are recorded correctly", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			pm.test("should fail", function() {
				pm.expect(1).toBe(2);
			});
		`)

		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		results := scope.GetTestResults()
		if len(results) != 1 {
			t.Fatalf("Expected 1 test result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("Test should have failed")
		}
		if results[0].Error == "" {
			t.Error("Failed test should have error message")
		}
	})

	t.Run("test summary works", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, _ = scope.Execute(context.Background(), `
			pm.test("pass1", function() { pm.expect(1).toBe(1); });
			pm.test("pass2", function() { pm.expect(2).toBe(2); });
			pm.test("fail1", function() { pm.expect(1).toBe(2); });
		`)

		summary := scope.GetTestSummary()
		if summary.Total != 3 {
			t.Errorf("Expected total 3, got %d", summary.Total)
		}
		if summary.Passed != 2 {
			t.Errorf("Expected passed 2, got %d", summary.Passed)
		}
		if summary.Failed != 1 {
			t.Errorf("Expected failed 1, got %d", summary.Failed)
		}
	})
}
