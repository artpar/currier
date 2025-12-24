package harness

import (
	"bytes"
	"context"
	"time"

	"github.com/artpar/currier/internal/cli"
)

// CLIResult holds CLI execution results.
type CLIResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// CLIRunner executes CLI commands.
type CLIRunner struct {
	harness *E2EHarness
}

// Run executes a CLI command with the given arguments.
func (r *CLIRunner) Run(args ...string) (*CLIResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.harness.timeout)
	defer cancel()

	start := time.Now()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := cli.NewRootCommand("test")
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	err := cmd.ExecuteContext(ctx)

	result := &CLIResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if err != nil {
		result.ExitCode = 1
	}

	return result, err
}

// Send is a convenience method for the send command.
func (r *CLIRunner) Send(method, url string, opts ...string) (*CLIResult, error) {
	args := []string{"send", method, url}
	args = append(args, opts...)
	return r.Run(args...)
}

// SendWithHeaders sends a request with headers.
func (r *CLIRunner) SendWithHeaders(method, url string, headers map[string]string) (*CLIResult, error) {
	args := []string{"send", method, url}
	for k, v := range headers {
		args = append(args, "--header", k+":"+v)
	}
	return r.Run(args...)
}

// SendWithBody sends a request with a body.
func (r *CLIRunner) SendWithBody(method, url, body string, headers map[string]string) (*CLIResult, error) {
	args := []string{"send", method, url, "--body", body}
	for k, v := range headers {
		args = append(args, "--header", k+":"+v)
	}
	return r.Run(args...)
}

// SendJSON sends a request expecting JSON output.
func (r *CLIRunner) SendJSON(method, url string) (*CLIResult, error) {
	return r.Run("send", method, url, "--json")
}
