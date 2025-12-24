package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/artpar/currier/internal/app"
	"github.com/artpar/currier/internal/core"
	httpclient "github.com/artpar/currier/internal/protocol/http"
	"github.com/spf13/cobra"
)

// SendOptions holds options for the send command.
type SendOptions struct {
	Headers []string
	Body    string
	JSON    bool
	Timeout time.Duration
}

// NewSendCommand creates the send command.
func NewSendCommand() *cobra.Command {
	opts := &SendOptions{}

	cmd := &cobra.Command{
		Use:   "send METHOD URL",
		Short: "Send an HTTP request",
		Long:  "Send an HTTP request to the specified URL with the given method.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			url := args[1]
			return runSend(cmd, method, url, opts)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.Headers, "header", "H", nil, "Request headers (format: Key:Value)")
	cmd.Flags().StringVarP(&opts.Body, "body", "d", "", "Request body")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output response as JSON")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 30*time.Second, "Request timeout")

	return cmd
}

func runSend(cmd *cobra.Command, method, url string, opts *SendOptions) error {
	// Create the app with HTTP protocol
	application := app.New(
		app.WithProtocol("http", httpclient.NewClient(
			httpclient.WithTimeout(opts.Timeout),
		)),
	)

	// Create request
	req, err := core.NewRequest("http", method, url)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	headers := parseHeaders(opts.Headers)
	for key, value := range headers {
		req.SetHeader(key, value)
	}

	// Add body
	if opts.Body != "" {
		contentType := headers["Content-Type"]
		if contentType == "" {
			contentType = "text/plain"
		}
		req.SetBody(core.NewRawBody([]byte(opts.Body), contentType))
	}

	// Send request
	ctx := context.Background()
	resp, err := application.Send(ctx, req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	// Output response
	if opts.JSON {
		return outputJSON(cmd, resp)
	}
	return outputHuman(cmd, resp)
}

func outputJSON(cmd *cobra.Command, resp *core.Response) error {
	result := map[string]any{
		"status":      resp.Status().Code(),
		"status_text": resp.Status().Text(),
		"headers":     resp.Headers().ToMap(),
		"body":        resp.Body().String(),
		"timing_ms":   resp.Timing().Total.Milliseconds(),
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputHuman(cmd *cobra.Command, resp *core.Response) error {
	out := cmd.OutOrStdout()

	// Status line
	fmt.Fprintf(out, "HTTP %d %s\n", resp.Status().Code(), resp.Status().Text())
	fmt.Fprintf(out, "Time: %dms\n", resp.Timing().Total.Milliseconds())
	fmt.Fprintln(out)

	// Headers
	fmt.Fprintln(out, "Headers:")
	for _, key := range resp.Headers().Keys() {
		for _, value := range resp.Headers().GetAll(key) {
			fmt.Fprintf(out, "  %s: %s\n", key, value)
		}
	}
	fmt.Fprintln(out)

	// Body
	if !resp.Body().IsEmpty() {
		fmt.Fprintln(out, "Body:")
		fmt.Fprintln(out, resp.Body().String())
	}

	return nil
}

// parseHeaders converts header strings to a map.
func parseHeaders(headerStrs []string) map[string]string {
	headers := make(map[string]string)
	for _, h := range headerStrs {
		idx := strings.Index(h, ":")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(h[:idx])
		value := strings.TrimSpace(h[idx+1:])
		headers[key] = value
	}
	return headers
}
