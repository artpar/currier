package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/artpar/currier/internal/script"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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
