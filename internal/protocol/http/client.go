package http

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
)

// Client implements the Requester interface for HTTP protocol.
type Client struct {
	httpClient *http.Client
	config     Config
}

// Config holds HTTP client configuration.
type Config struct {
	Timeout       time.Duration
	FollowRedirect bool
}

// Option is a function that configures the Client.
type Option func(*Client)

// NewClient creates a new HTTP client with the given options.
func NewClient(opts ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: Config{
			Timeout:       30 * time.Second,
			FollowRedirect: true,
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.config.Timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithTransport sets a custom HTTP transport.
func WithTransport(transport *http.Transport) Option {
	return func(c *Client) {
		c.httpClient.Transport = transport
	}
}

// WithNoRedirects disables automatic redirect following.
func WithNoRedirects() Option {
	return func(c *Client) {
		c.config.FollowRedirect = false
		c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
}

// Protocol returns the protocol identifier.
func (c *Client) Protocol() string {
	return "http"
}

// Send executes an HTTP request and returns the response.
func (c *Client) Send(ctx context.Context, req *core.Request) (*core.Response, error) {
	startTime := time.Now()

	// Create HTTP request
	httpReq, err := c.toHTTPRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	endTime := time.Now()

	// Read response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// Create response
	return c.fromHTTPResponse(req, httpResp, bodyBytes, startTime, endTime), nil
}

// toHTTPRequest converts a core.Request to an http.Request.
func (c *Client) toHTTPRequest(ctx context.Context, req *core.Request) (*http.Request, error) {
	var bodyReader io.Reader
	if !req.Body().IsEmpty() {
		bodyReader = req.Body().Reader()
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method(), req.Endpoint(), bodyReader)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for _, key := range req.Headers().Keys() {
		for _, value := range req.Headers().GetAll(key) {
			httpReq.Header.Add(key, value)
		}
	}

	return httpReq, nil
}

// fromHTTPResponse converts an http.Response to a core.Response.
func (c *Client) fromHTTPResponse(req *core.Request, httpResp *http.Response, bodyBytes []byte, startTime, endTime time.Time) *core.Response {
	// Create status
	status := core.NewStatus(httpResp.StatusCode, httpResp.Status)

	// Create headers
	headers := core.NewHeaders()
	for key, values := range httpResp.Header {
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	// Create body
	var body core.Body
	if len(bodyBytes) > 0 {
		contentType := httpResp.Header.Get("Content-Type")
		body = core.NewRawBody(bodyBytes, contentType)
	} else {
		body = core.NewEmptyBody()
	}

	// Create timing info
	timing := interfaces.TimingInfo{
		StartTime: startTime,
		EndTime:   endTime,
		Total:     endTime.Sub(startTime),
	}

	// Build response
	return core.NewResponse(req.ID(), "http", status).
		WithHeaders(headers).
		WithBody(body).
		WithTiming(timing)
}
