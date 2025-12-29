package runner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interpolate"
	httpclient "github.com/artpar/currier/internal/protocol/http"
	"github.com/artpar/currier/internal/script"
)

// RunResult represents the result of a single request execution.
type RunResult struct {
	RequestID   string
	RequestName string
	Method      string
	URL         string
	Status      int
	StatusText  string
	Duration    time.Duration
	TestResults []script.TestResult
	Error       error
}

// RunSummary represents the summary of a collection run.
type RunSummary struct {
	CollectionName string
	TotalRequests  int
	Executed       int
	Passed         int
	Failed         int
	TotalTests     int
	TestsPassed    int
	TestsFailed    int
	TotalDuration  time.Duration
	Results        []RunResult
	StartTime      time.Time
	EndTime        time.Time
}

// ProgressCallback is called after each request is executed.
type ProgressCallback func(current int, total int, result *RunResult)

// Runner executes all requests in a collection.
type Runner struct {
	collection *core.Collection
	env        *core.Environment
	engine     *interpolate.Engine
	httpClient *httpclient.Client
	cookieJar  http.CookieJar
	onProgress ProgressCallback
}

// Option configures the Runner.
type Option func(*Runner)

// WithEnvironment sets the environment for variable interpolation.
func WithEnvironment(env *core.Environment) Option {
	return func(r *Runner) {
		r.env = env
		if env != nil {
			r.engine.SetVariables(env.ExportAll())
		}
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *httpclient.Client) Option {
	return func(r *Runner) {
		r.httpClient = client
	}
}

// WithCookieJar sets a cookie jar for cookie persistence across requests.
func WithCookieJar(jar http.CookieJar) Option {
	return func(r *Runner) {
		r.cookieJar = jar
	}
}

// WithProgressCallback sets a callback for progress updates.
func WithProgressCallback(cb ProgressCallback) Option {
	return func(r *Runner) {
		r.onProgress = cb
	}
}

// NewRunner creates a new collection runner.
func NewRunner(collection *core.Collection, opts ...Option) *Runner {
	// Create cookie jar for this run
	jar, _ := cookiejar.New(nil)

	r := &Runner{
		collection: collection,
		engine:     interpolate.NewEngine(),
		cookieJar:  jar,
	}

	// Create default HTTP client with cookie jar
	r.httpClient = httpclient.NewClient(
		httpclient.WithCookieJar(r.cookieJar),
		httpclient.WithTimeout(30*time.Second),
	)

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Run executes all requests in the collection sequentially.
func (r *Runner) Run(ctx context.Context) *RunSummary {
	summary := &RunSummary{
		CollectionName: r.collection.Name(),
		StartTime:      time.Now(),
		Results:        make([]RunResult, 0),
	}

	// Collect all requests from collection
	requests := r.walkRequests(r.collection)
	summary.TotalRequests = len(requests)

	for i, reqDef := range requests {
		// Check for context cancellation
		if ctx.Err() != nil {
			break
		}

		result := r.executeRequest(ctx, reqDef)
		summary.Results = append(summary.Results, result)
		summary.Executed++

		// Update statistics
		if result.Error == nil {
			summary.Passed++
		} else {
			summary.Failed++
		}

		for _, tr := range result.TestResults {
			summary.TotalTests++
			if tr.Passed {
				summary.TestsPassed++
			} else {
				summary.TestsFailed++
			}
		}

		// Call progress callback
		if r.onProgress != nil {
			r.onProgress(i+1, len(requests), &result)
		}
	}

	summary.EndTime = time.Now()
	summary.TotalDuration = summary.EndTime.Sub(summary.StartTime)

	return summary
}

// walkRequests recursively collects all requests from a collection.
func (r *Runner) walkRequests(coll *core.Collection) []*core.RequestDefinition {
	var requests []*core.RequestDefinition

	// Add requests at this level
	requests = append(requests, coll.Requests()...)

	// Recursively process folders
	for _, folder := range coll.Folders() {
		folderRequests := r.walkFolder(folder)
		requests = append(requests, folderRequests...)
	}

	return requests
}

// walkFolder recursively collects all requests from a folder.
func (r *Runner) walkFolder(folder *core.Folder) []*core.RequestDefinition {
	var requests []*core.RequestDefinition

	// Add requests in this folder
	requests = append(requests, folder.Requests()...)

	// Recursively process subfolders
	for _, subfolder := range folder.Folders() {
		subRequests := r.walkFolder(subfolder)
		requests = append(requests, subRequests...)
	}

	return requests
}

// executeRequest executes a single request and returns the result.
func (r *Runner) executeRequest(ctx context.Context, reqDef *core.RequestDefinition) RunResult {
	result := RunResult{
		RequestID:   reqDef.ID(),
		RequestName: reqDef.Name(),
		Method:      reqDef.Method(),
	}

	startTime := time.Now()

	// Create script scope for this request
	scriptScope := script.NewScopeWithAssertions()

	// Set up script context with environment variables
	if r.env != nil {
		scriptScope.SetEnvironmentName(r.env.Name())
		for k, v := range r.env.ExportAll() {
			scriptScope.SetVariable(k, v)
			scriptScope.SetEnvironmentVariable(k, v)
		}
	}

	// Run pre-request script
	if preScript := reqDef.PreScript(); preScript != "" {
		if _, err := scriptScope.Execute(ctx, preScript); err != nil {
			result.Error = fmt.Errorf("pre-request script error: %w", err)
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// Convert RequestDefinition to Request with interpolation
	req, err := reqDef.ToRequestWithEnv(r.engine)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.URL = req.Endpoint()

	// Execute request
	resp, err := r.httpClient.Send(ctx, req)
	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.Status = resp.Status().Code()
	result.StatusText = resp.Status().Text()

	// Run post-request script (tests)
	if postScript := reqDef.PostScript(); postScript != "" {
		// Set response context in script scope
		scriptScope.SetResponseStatus(resp.Status().Code())
		scriptScope.SetResponseStatusText(resp.Status().Text())
		scriptScope.SetResponseBody(resp.Body().String())
		scriptScope.SetResponseTime(resp.Timing().Total.Milliseconds())
		scriptScope.SetResponseSize(resp.Body().Size())

		// Set response headers
		respHeaders := make(map[string]string)
		for _, key := range resp.Headers().Keys() {
			respHeaders[key] = resp.Headers().Get(key)
		}
		scriptScope.SetResponseHeaders(respHeaders)

		if _, err := scriptScope.Execute(ctx, postScript); err != nil {
			result.Error = fmt.Errorf("test script error: %w", err)
		}

		// Collect test results
		result.TestResults = scriptScope.GetTestResults()
	}

	result.Duration = time.Since(startTime)
	return result
}

// IsSuccess returns true if the result had no errors.
func (r *RunResult) IsSuccess() bool {
	return r.Error == nil
}

// AllTestsPassed returns true if all tests passed.
func (r *RunResult) AllTestsPassed() bool {
	for _, tr := range r.TestResults {
		if !tr.Passed {
			return false
		}
	}
	return true
}

// IsSuccess returns true if all requests passed.
func (s *RunSummary) IsSuccess() bool {
	return s.Failed == 0
}

// AllTestsPassed returns true if all tests in the run passed.
func (s *RunSummary) AllTestsPassed() bool {
	return s.TestsFailed == 0
}
