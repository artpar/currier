package script

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertions_Test(t *testing.T) {
	t.Run("currier.test passes for true assertion", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `currier.test("Status is 200", 200 === 200)`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.Equal(t, "Status is 200", results[0].Name)
		assert.True(t, results[0].Passed)
	})

	t.Run("currier.test fails for false assertion", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `currier.test("Status is 200", 404 === 200)`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.Equal(t, "Status is 200", results[0].Name)
		assert.False(t, results[0].Passed)
	})

	t.Run("currier.test with callback function", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Complex test", function() {
				var x = 5;
				var y = 10;
				return x + y === 15;
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("currier.test callback throwing error fails test", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Should fail", function() {
				throw new Error("Test failed");
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.False(t, results[0].Passed)
		assert.Contains(t, results[0].Error, "Test failed")
	})

	t.Run("multiple tests tracked", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Test 1", true);
			currier.test("Test 2", false);
			currier.test("Test 3", true);
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 3)
		assert.True(t, results[0].Passed)
		assert.False(t, results[1].Passed)
		assert.True(t, results[2].Passed)
	})
}

func TestAssertions_Expect(t *testing.T) {
	t.Run("expect.toBe passes for equal values", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Value equals 5", function() {
				currier.expect(5).toBe(5);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toBe fails for unequal values", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Value equals 5", function() {
				currier.expect(10).toBe(5);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.False(t, results[0].Passed)
	})

	t.Run("expect.toEqual for deep equality", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Objects equal", function() {
				currier.expect({a: 1, b: 2}).toEqual({a: 1, b: 2});
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toContain for strings", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Contains substring", function() {
				currier.expect("hello world").toContain("world");
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toContain fails for missing substring", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Contains substring", function() {
				currier.expect("hello").toContain("world");
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.False(t, results[0].Passed)
	})

	t.Run("expect.toContain for arrays", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Array contains value", function() {
				currier.expect([1, 2, 3]).toContain(2);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toMatch for regex", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Matches pattern", function() {
				currier.expect("hello123").toMatch(/hello\d+/);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toBeGreaterThan", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Greater than", function() {
				currier.expect(10).toBeGreaterThan(5);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toBeLessThan", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Less than", function() {
				currier.expect(5).toBeLessThan(10);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toBeNull", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Is null", function() {
				currier.expect(null).toBeNull();
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toBeUndefined", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Is undefined", function() {
				currier.expect(undefined).toBeUndefined();
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toBeTruthy", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Is truthy", function() {
				currier.expect("hello").toBeTruthy();
				currier.expect(1).toBeTruthy();
				currier.expect(true).toBeTruthy();
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toBeFalsy", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Is falsy", function() {
				currier.expect("").toBeFalsy();
				currier.expect(0).toBeFalsy();
				currier.expect(false).toBeFalsy();
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toHaveProperty", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Has property", function() {
				currier.expect({name: "John"}).toHaveProperty("name");
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.toHaveLength", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Has length", function() {
				currier.expect([1, 2, 3]).toHaveLength(3);
				currier.expect("hello").toHaveLength(5);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expect.not negates assertions", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Not equal", function() {
				currier.expect(5).not.toBe(10);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})
}

func TestAssertions_ResponseTests(t *testing.T) {
	t.Run("test response status", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)

		_, err := scope.Execute(context.Background(), `
			currier.test("Status is 200", function() {
				currier.expect(currier.response.status).toBe(200);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("test response body JSON", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseBody(`{"user": {"name": "John", "age": 30}}`)

		_, err := scope.Execute(context.Background(), `
			currier.test("User name is John", function() {
				var data = currier.response.json();
				currier.expect(data.user.name).toBe("John");
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("test response time", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseTime(150)

		_, err := scope.Execute(context.Background(), `
			currier.test("Response time is under 500ms", function() {
				currier.expect(currier.response.time).toBeLessThan(500);
			})
		`)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})
}

func TestAssertions_ClearResults(t *testing.T) {
	t.Run("clear test results", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, _ = scope.Execute(context.Background(), `currier.test("Test 1", true)`)
		require.Len(t, scope.GetTestResults(), 1)

		scope.ClearTestResults()
		require.Len(t, scope.GetTestResults(), 0)
	})
}

func TestAssertions_Summary(t *testing.T) {
	t.Run("get test summary", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		_, err := scope.Execute(context.Background(), `
			currier.test("Test 1", true);
			currier.test("Test 2", false);
			currier.test("Test 3", true);
		`)

		require.NoError(t, err)
		summary := scope.GetTestSummary()
		assert.Equal(t, 3, summary.Total)
		assert.Equal(t, 2, summary.Passed)
		assert.Equal(t, 1, summary.Failed)
	})
}

func TestAssertEqual(t *testing.T) {
	t.Run("returns true for equal values", func(t *testing.T) {
		assert.True(t, AssertEqual(5, 5))
		assert.True(t, AssertEqual("hello", "hello"))
		assert.True(t, AssertEqual([]int{1, 2}, []int{1, 2}))
	})

	t.Run("returns false for unequal values", func(t *testing.T) {
		assert.False(t, AssertEqual(5, 10))
		assert.False(t, AssertEqual("hello", "world"))
	})
}

func TestAssertContains(t *testing.T) {
	t.Run("string contains substring", func(t *testing.T) {
		assert.True(t, AssertContains("hello world", "world"))
		assert.False(t, AssertContains("hello", "world"))
	})

	t.Run("slice contains element", func(t *testing.T) {
		slice := []interface{}{1, 2, 3}
		assert.True(t, AssertContains(slice, 2))
		assert.False(t, AssertContains(slice, 5))
	})

	t.Run("returns false for non-container types", func(t *testing.T) {
		assert.False(t, AssertContains(123, 1))
	})
}

func TestAssertMatch(t *testing.T) {
	t.Run("matches valid regex", func(t *testing.T) {
		assert.True(t, AssertMatch("hello123", `hello\d+`))
		assert.True(t, AssertMatch("test@example.com", `.*@.*\.com`))
	})

	t.Run("returns false for non-match", func(t *testing.T) {
		assert.False(t, AssertMatch("hello", `\d+`))
	})

	t.Run("returns false for invalid regex", func(t *testing.T) {
		assert.False(t, AssertMatch("test", `[invalid`))
	})
}

func TestAssertJSONEqual(t *testing.T) {
	t.Run("equal JSON objects", func(t *testing.T) {
		assert.True(t, AssertJSONEqual(`{"a": 1}`, `{"a": 1}`))
		assert.True(t, AssertJSONEqual(`{"a": 1, "b": 2}`, `{"b": 2, "a": 1}`))
	})

	t.Run("unequal JSON objects", func(t *testing.T) {
		assert.False(t, AssertJSONEqual(`{"a": 1}`, `{"a": 2}`))
	})

	t.Run("returns false for invalid JSON", func(t *testing.T) {
		assert.False(t, AssertJSONEqual("not json", `{"a": 1}`))
		assert.False(t, AssertJSONEqual(`{"a": 1}`, "not json"))
	})
}

func TestFormatTestResults(t *testing.T) {
	t.Run("formats passing tests", func(t *testing.T) {
		results := []TestResult{
			{Name: "Test 1", Passed: true},
			{Name: "Test 2", Passed: true},
		}
		output := FormatTestResults(results)
		assert.Contains(t, output, "✓ Test 1")
		assert.Contains(t, output, "✓ Test 2")
	})

	t.Run("formats failing tests", func(t *testing.T) {
		results := []TestResult{
			{Name: "Test 1", Passed: false, Error: "assertion failed"},
		}
		output := FormatTestResults(results)
		assert.Contains(t, output, "✗ Test 1")
		assert.Contains(t, output, "Error: assertion failed")
	})

	t.Run("formats mixed results", func(t *testing.T) {
		results := []TestResult{
			{Name: "Test 1", Passed: true},
			{Name: "Test 2", Passed: false, Error: "failed"},
		}
		output := FormatTestResults(results)
		assert.Contains(t, output, "✓ Test 1")
		assert.Contains(t, output, "✗ Test 2")
	})

	t.Run("handles empty results", func(t *testing.T) {
		output := FormatTestResults([]TestResult{})
		assert.Empty(t, output)
	})
}
