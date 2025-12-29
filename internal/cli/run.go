package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/runner"
	"github.com/artpar/currier/internal/script"
	"github.com/spf13/cobra"
)

// RunOptions holds options for the run command.
type RunOptions struct {
	EnvFiles []string
	Verbose  bool
	JSON     bool
}

// NewRunCommand creates the run command.
func NewRunCommand() *cobra.Command {
	opts := &RunOptions{}

	cmd := &cobra.Command{
		Use:   "run COLLECTION_FILE",
		Short: "Run all requests in a collection",
		Long:  "Execute all requests in a collection file sequentially and display results.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCollection(cmd, args[0], opts)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.EnvFiles, "env", "e", nil, "Environment file(s) for variable substitution")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Show detailed output for each request")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output results as JSON")

	return cmd
}

func runCollection(cmd *cobra.Command, collectionPath string, opts *RunOptions) error {
	// Read collection file
	data, err := os.ReadFile(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to read collection file: %w", err)
	}

	// Use the registry to detect format and import
	registry := newImporterRegistry()
	result, err := registry.DetectAndImport(context.Background(), data)
	if err != nil {
		return fmt.Errorf("failed to parse collection: %w", err)
	}
	collection := result.Collection

	// Load environment if provided
	var env *core.Environment
	if len(opts.EnvFiles) > 0 {
		env, err = core.LoadMultipleEnvironments(opts.EnvFiles)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}
		if env != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Using environment: %s\n", env.Name())
		}
	}

	// Create runner with options
	runnerOpts := []runner.Option{}
	if env != nil {
		runnerOpts = append(runnerOpts, runner.WithEnvironment(env))
	}

	// Progress callback
	out := cmd.OutOrStdout()
	runnerOpts = append(runnerOpts, runner.WithProgressCallback(func(current, total int, result *runner.RunResult) {
		if opts.Verbose {
			status := "✓"
			if result.Error != nil {
				status = "✗"
			}
			fmt.Fprintf(out, "%s %s %s (%dms)\n",
				status,
				result.Method,
				result.RequestName,
				result.Duration.Milliseconds())

			// Show test results
			for _, tr := range result.TestResults {
				testStatus := "  ✓"
				if !tr.Passed {
					testStatus = "  ✗"
				}
				fmt.Fprintf(out, "%s %s\n", testStatus, tr.Name)
				if tr.Error != "" {
					fmt.Fprintf(out, "    Error: %s\n", tr.Error)
				}
			}
		} else {
			fmt.Fprintf(out, "\rRunning: %d/%d", current, total)
		}
	}))

	r := runner.NewRunner(collection, runnerOpts...)

	// Run collection
	fmt.Fprintf(out, "Running collection: %s\n", collection.Name())
	ctx := context.Background()
	summary := r.Run(ctx)

	// Clear progress line if not verbose
	if !opts.Verbose {
		fmt.Fprintln(out)
	}

	// Output results
	if opts.JSON {
		return outputRunResultsJSON(cmd, summary)
	}
	return outputRunResultsHuman(cmd, summary, opts.Verbose)
}

func outputRunResultsJSON(cmd *cobra.Command, summary *runner.RunSummary) error {
	// Build JSON output
	results := make([]map[string]any, 0, len(summary.Results))
	for _, r := range summary.Results {
		result := map[string]any{
			"name":        r.RequestName,
			"method":      r.Method,
			"url":         r.URL,
			"status":      r.Status,
			"status_text": r.StatusText,
			"duration_ms": r.Duration.Milliseconds(),
		}
		if r.Error != nil {
			result["error"] = r.Error.Error()
		}
		if len(r.TestResults) > 0 {
			tests := make([]map[string]any, 0, len(r.TestResults))
			for _, tr := range r.TestResults {
				test := map[string]any{
					"name":   tr.Name,
					"passed": tr.Passed,
				}
				if tr.Error != "" {
					test["error"] = tr.Error
				}
				tests = append(tests, test)
			}
			result["tests"] = tests
		}
		results = append(results, result)
	}

	output := map[string]any{
		"collection":      summary.CollectionName,
		"total_requests":  summary.TotalRequests,
		"executed":        summary.Executed,
		"passed":          summary.Passed,
		"failed":          summary.Failed,
		"total_tests":     summary.TotalTests,
		"tests_passed":    summary.TestsPassed,
		"tests_failed":    summary.TestsFailed,
		"total_duration":  summary.TotalDuration.Milliseconds(),
		"results":         results,
	}

	return outputJSONResult(cmd, output)
}

func outputRunResultsHuman(cmd *cobra.Command, summary *runner.RunSummary, verbose bool) error {
	out := cmd.OutOrStdout()

	// If not verbose, show summary of each request
	if !verbose {
		fmt.Fprintln(out)
		for _, r := range summary.Results {
			status := "✓"
			if r.Error != nil {
				status = "✗"
			}
			testInfo := ""
			if len(r.TestResults) > 0 {
				passed := 0
				for _, tr := range r.TestResults {
					if tr.Passed {
						passed++
					}
				}
				testInfo = fmt.Sprintf(" - %d/%d tests", passed, len(r.TestResults))
			}
			fmt.Fprintf(out, "%s %s %s (%dms)%s\n",
				status,
				r.Method,
				r.RequestName,
				r.Duration.Milliseconds(),
				testInfo)

			// Show failed tests
			for _, tr := range r.TestResults {
				if !tr.Passed {
					fmt.Fprintf(out, "  ✗ %s\n", tr.Name)
					if tr.Error != "" {
						fmt.Fprintf(out, "    %s\n", tr.Error)
					}
				}
			}
		}
	}

	// Summary
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Summary:\n")
	fmt.Fprintf(out, "  Requests: %d/%d passed\n", summary.Passed, summary.TotalRequests)
	if summary.TotalTests > 0 {
		fmt.Fprintf(out, "  Tests: %d/%d passed\n", summary.TestsPassed, summary.TotalTests)
	}
	fmt.Fprintf(out, "  Total time: %s\n", formatDuration(summary.TotalDuration))

	// Exit with error if any failures
	if summary.Failed > 0 || summary.TestsFailed > 0 {
		return fmt.Errorf("%d requests failed, %d tests failed", summary.Failed, summary.TestsFailed)
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func outputJSONResult(cmd *cobra.Command, result map[string]any) error {
	// Simple JSON output
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "{\n")
	first := true
	for k, v := range result {
		if !first {
			fmt.Fprintf(out, ",\n")
		}
		first = false
		switch val := v.(type) {
		case string:
			fmt.Fprintf(out, "  %q: %q", k, val)
		case int, int64:
			fmt.Fprintf(out, "  %q: %v", k, val)
		case []map[string]any:
			fmt.Fprintf(out, "  %q: %v", k, formatSlice(val))
		default:
			fmt.Fprintf(out, "  %q: %v", k, val)
		}
	}
	fmt.Fprintf(out, "\n}\n")
	return nil
}

func formatSlice(slice []map[string]any) string {
	if len(slice) == 0 {
		return "[]"
	}
	return fmt.Sprintf("[...%d items...]", len(slice))
}

// FormatTestResults formats test results for CLI output.
func FormatTestResults(results []script.TestResult) string {
	return script.FormatTestResults(results)
}
