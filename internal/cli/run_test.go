package cli

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/artpar/currier/internal/runner"
	"github.com/artpar/currier/internal/script"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	t.Run("formats milliseconds for sub-second durations", func(t *testing.T) {
		d := 500 * time.Millisecond
		result := formatDuration(d)
		assert.Equal(t, "500ms", result)
	})

	t.Run("formats seconds for longer durations", func(t *testing.T) {
		d := 2500 * time.Millisecond
		result := formatDuration(d)
		assert.Equal(t, "2.50s", result)
	})

	t.Run("formats zero duration", func(t *testing.T) {
		d := 0 * time.Millisecond
		result := formatDuration(d)
		assert.Equal(t, "0ms", result)
	})
}

func TestFormatSlice(t *testing.T) {
	t.Run("formats empty slice", func(t *testing.T) {
		result := formatSlice([]map[string]any{})
		assert.Equal(t, "[]", result)
	})

	t.Run("formats slice with items", func(t *testing.T) {
		slice := []map[string]any{
			{"key": "value1"},
			{"key": "value2"},
			{"key": "value3"},
		}
		result := formatSlice(slice)
		assert.Equal(t, "[...3 items...]", result)
	})
}

func TestOutputJSONResult(t *testing.T) {
	t.Run("outputs string values", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		result := map[string]any{
			"status": "success",
		}
		err := outputJSONResult(cmd, result)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "status")
		assert.Contains(t, buf.String(), "success")
	})

	t.Run("outputs int values", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		result := map[string]any{
			"count": 42,
		}
		err := outputJSONResult(cmd, result)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "count")
		assert.Contains(t, buf.String(), "42")
	})

	t.Run("outputs slice values", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		result := map[string]any{
			"items": []map[string]any{{"a": "b"}},
		}
		err := outputJSONResult(cmd, result)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "items")
	})
}

func TestFormatTestResults(t *testing.T) {
	t.Run("formats test results", func(t *testing.T) {
		results := []script.TestResult{
			{Name: "Test 1", Passed: true},
			{Name: "Test 2", Passed: false, Error: "expected 200, got 404"},
		}
		output := FormatTestResults(results)
		assert.NotEmpty(t, output)
	})

	t.Run("handles empty results", func(t *testing.T) {
		results := []script.TestResult{}
		output := FormatTestResults(results)
		assert.Equal(t, "", output)
	})
}

func TestOutputRunResultsJSON(t *testing.T) {
	t.Run("outputs successful run as JSON", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		summary := &runner.RunSummary{
			CollectionName: "Test Collection",
			TotalRequests:  2,
			Executed:       2,
			Passed:         2,
			Failed:         0,
			TotalTests:     4,
			TestsPassed:    4,
			TestsFailed:    0,
			TotalDuration:  500 * time.Millisecond,
			Results: []runner.RunResult{
				{
					RequestName: "Get Users",
					Method:      "GET",
					URL:         "https://api.example.com/users",
					Status:      200,
					StatusText:  "OK",
					Duration:    200 * time.Millisecond,
					TestResults: []script.TestResult{
						{Name: "Status is 200", Passed: true},
						{Name: "Has users", Passed: true},
					},
				},
				{
					RequestName: "Create User",
					Method:      "POST",
					URL:         "https://api.example.com/users",
					Status:      201,
					StatusText:  "Created",
					Duration:    300 * time.Millisecond,
					TestResults: []script.TestResult{
						{Name: "Status is 201", Passed: true},
						{Name: "Has ID", Passed: true},
					},
				},
			},
		}

		err := outputRunResultsJSON(cmd, summary)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Test Collection")
		assert.Contains(t, output, "total_requests")
		// Results are truncated to [...N items...]
		assert.Contains(t, output, "...2 items...")
	})

	t.Run("outputs run with errors as JSON", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		summary := &runner.RunSummary{
			CollectionName: "Test Collection",
			TotalRequests:  1,
			Executed:       1,
			Passed:         0,
			Failed:         1,
			TotalDuration:  100 * time.Millisecond,
			Results: []runner.RunResult{
				{
					RequestName: "Failing Request",
					Method:      "GET",
					URL:         "https://api.example.com/fail",
					Duration:    100 * time.Millisecond,
					Error:       errors.New("connection refused"),
				},
			},
		}

		err := outputRunResultsJSON(cmd, summary)
		require.NoError(t, err)

		output := buf.String()
		// Results are truncated, check for summary fields
		assert.Contains(t, output, "failed")
		assert.Contains(t, output, "...1 items...")
	})

	t.Run("outputs run with failed tests as JSON", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		summary := &runner.RunSummary{
			CollectionName: "Test Collection",
			TotalRequests:  1,
			Executed:       1,
			Passed:         1,
			Failed:         0,
			TotalTests:     2,
			TestsPassed:    1,
			TestsFailed:    1,
			TotalDuration:  100 * time.Millisecond,
			Results: []runner.RunResult{
				{
					RequestName: "Test Request",
					Method:      "GET",
					URL:         "https://api.example.com/test",
					Status:      404,
					StatusText:  "Not Found",
					Duration:    100 * time.Millisecond,
					TestResults: []script.TestResult{
						{Name: "Response received", Passed: true},
						{Name: "Status is 200", Passed: false, Error: "expected 200, got 404"},
					},
				},
			},
		}

		err := outputRunResultsJSON(cmd, summary)
		require.NoError(t, err)

		output := buf.String()
		// Check summary fields, not inner results (which are truncated)
		assert.Contains(t, output, "tests_failed")
		assert.Contains(t, output, "1")
	})
}

func TestOutputRunResultsHuman(t *testing.T) {
	t.Run("outputs successful run in human format", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		summary := &runner.RunSummary{
			CollectionName: "Test Collection",
			TotalRequests:  2,
			Executed:       2,
			Passed:         2,
			Failed:         0,
			TotalTests:     4,
			TestsPassed:    4,
			TestsFailed:    0,
			TotalDuration:  500 * time.Millisecond,
			Results: []runner.RunResult{
				{
					RequestName: "Get Users",
					Method:      "GET",
					Status:      200,
					Duration:    200 * time.Millisecond,
					TestResults: []script.TestResult{
						{Name: "Status is 200", Passed: true},
					},
				},
				{
					RequestName: "Create User",
					Method:      "POST",
					Status:      201,
					Duration:    300 * time.Millisecond,
				},
			},
		}

		err := outputRunResultsHuman(cmd, summary, false)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "✓")
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "Get Users")
		assert.Contains(t, output, "Requests: 2/2 passed")
		assert.Contains(t, output, "Tests: 4/4 passed")
	})

	t.Run("outputs failed run in human format", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		summary := &runner.RunSummary{
			CollectionName: "Test Collection",
			TotalRequests:  1,
			Executed:       1,
			Passed:         0,
			Failed:         1,
			TotalDuration:  100 * time.Millisecond,
			Results: []runner.RunResult{
				{
					RequestName: "Failing Request",
					Method:      "GET",
					Duration:    100 * time.Millisecond,
					Error:       errors.New("connection refused"),
				},
			},
		}

		err := outputRunResultsHuman(cmd, summary, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "1 requests failed")

		output := buf.String()
		assert.Contains(t, output, "✗")
	})

	t.Run("shows failed test details", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		summary := &runner.RunSummary{
			CollectionName: "Test Collection",
			TotalRequests:  1,
			Executed:       1,
			Passed:         1,
			Failed:         0,
			TotalTests:     2,
			TestsPassed:    1,
			TestsFailed:    1,
			TotalDuration:  100 * time.Millisecond,
			Results: []runner.RunResult{
				{
					RequestName: "Test Request",
					Method:      "GET",
					Status:      404,
					Duration:    100 * time.Millisecond,
					TestResults: []script.TestResult{
						{Name: "Response received", Passed: true},
						{Name: "Status is 200", Passed: false, Error: "expected 200, got 404"},
					},
				},
			},
		}

		err := outputRunResultsHuman(cmd, summary, false)
		require.Error(t, err)

		output := buf.String()
		assert.Contains(t, output, "Status is 200")
		assert.Contains(t, output, "expected 200, got 404")
	})

	t.Run("shows no test info when no tests", func(t *testing.T) {
		cmd := &cobra.Command{}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		summary := &runner.RunSummary{
			CollectionName: "Test Collection",
			TotalRequests:  1,
			Executed:       1,
			Passed:         1,
			Failed:         0,
			TotalTests:     0,
			TotalDuration:  100 * time.Millisecond,
			Results: []runner.RunResult{
				{
					RequestName: "Simple Request",
					Method:      "GET",
					Status:      200,
					Duration:    100 * time.Millisecond,
				},
			},
		}

		err := outputRunResultsHuman(cmd, summary, false)
		require.NoError(t, err)

		output := buf.String()
		assert.NotContains(t, output, "Tests:")
	})
}
