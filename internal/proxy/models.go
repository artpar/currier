package proxy

import (
	"time"
)

// CapturedRequest represents a single captured HTTP request/response pair.
type CapturedRequest struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`

	// Request data
	Method         string              `json:"method"`
	URL            string              `json:"url"`
	Host           string              `json:"host"`
	Path           string              `json:"path"`
	RequestHeaders map[string][]string `json:"request_headers,omitempty"`
	RequestBody    []byte              `json:"request_body,omitempty"`
	RequestSize    int64               `json:"request_size"`

	// Response data
	StatusCode      int                 `json:"status_code"`
	StatusText      string              `json:"status_text"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
	ResponseBody    []byte              `json:"response_body,omitempty"`
	ResponseSize    int64               `json:"response_size"`

	// Timing
	Duration time.Duration `json:"duration"`

	// Connection info
	IsHTTPS        bool   `json:"is_https"`
	TLSVersion     string `json:"tls_version,omitempty"`
	TLSCipherSuite string `json:"tls_cipher_suite,omitempty"`
	SourceIP       string `json:"source_ip"`
	SourcePort     int    `json:"source_port"`

	// Error info (if request failed)
	Error string `json:"error,omitempty"`
}

// ContentType returns the response content-type header.
func (c *CapturedRequest) ContentType() string {
	if c.ResponseHeaders == nil {
		return ""
	}
	if ct, ok := c.ResponseHeaders["Content-Type"]; ok && len(ct) > 0 {
		return ct[0]
	}
	return ""
}

// IsSuccess returns true if the response status is 2xx.
func (c *CapturedRequest) IsSuccess() bool {
	return c.StatusCode >= 200 && c.StatusCode < 300
}

// IsRedirect returns true if the response status is 3xx.
func (c *CapturedRequest) IsRedirect() bool {
	return c.StatusCode >= 300 && c.StatusCode < 400
}

// IsClientError returns true if the response status is 4xx.
func (c *CapturedRequest) IsClientError() bool {
	return c.StatusCode >= 400 && c.StatusCode < 500
}

// IsServerError returns true if the response status is 5xx.
func (c *CapturedRequest) IsServerError() bool {
	return c.StatusCode >= 500 && c.StatusCode < 600
}

// CaptureListener receives real-time capture events.
type CaptureListener interface {
	OnCapture(capture *CapturedRequest)
}

// CaptureListenerFunc is a function adapter for CaptureListener.
type CaptureListenerFunc func(*CapturedRequest)

func (f CaptureListenerFunc) OnCapture(capture *CapturedRequest) {
	f(capture)
}

// FilterOptions specifies filters for querying captures.
type FilterOptions struct {
	// Method filter (GET, POST, etc.)
	Method string

	// Host filter (supports wildcards like *.example.com)
	Host string

	// Path filter (supports wildcards)
	Path string

	// Status code range
	StatusMin int
	StatusMax int

	// Content-type filter
	ContentType string

	// Full-text search (searches URL, headers, body)
	Search string

	// Size filters
	MinSize int64
	MaxSize int64

	// Time range
	After  time.Time
	Before time.Time

	// Protocol filter
	HTTPSOnly bool
	HTTPOnly  bool

	// Pagination
	Limit  int
	Offset int
}
