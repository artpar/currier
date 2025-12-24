package core

import (
	"github.com/artpar/currier/internal/interfaces"
	"github.com/google/uuid"
)

// Response implements the interfaces.Response interface.
type Response struct {
	id        string
	requestID string
	protocol  string
	status    *Status
	headers   *Headers
	body      Body
	timing    interfaces.TimingInfo
	metadata  map[string]any
}

// NewResponse creates a new response with the given parameters.
func NewResponse(requestID, protocol string, status *Status) *Response {
	return &Response{
		id:        uuid.New().String(),
		requestID: requestID,
		protocol:  protocol,
		status:    status,
		headers:   NewHeaders(),
		body:      NewEmptyBody(),
		timing:    interfaces.TimingInfo{},
		metadata:  make(map[string]any),
	}
}

func (r *Response) ID() string {
	return r.id
}

func (r *Response) RequestID() string {
	return r.requestID
}

func (r *Response) Protocol() string {
	return r.protocol
}

func (r *Response) Status() *Status {
	return r.status
}

func (r *Response) Headers() *Headers {
	return r.headers
}

func (r *Response) Body() Body {
	return r.body
}

func (r *Response) Timing() interfaces.TimingInfo {
	return r.timing
}

func (r *Response) Metadata() map[string]any {
	result := make(map[string]any)
	for k, v := range r.metadata {
		result[k] = v
	}
	return result
}

// WithHeaders sets the response headers and returns the response for chaining.
func (r *Response) WithHeaders(h *Headers) *Response {
	r.headers = h
	return r
}

// WithBody sets the response body and returns the response for chaining.
func (r *Response) WithBody(b Body) *Response {
	r.body = b
	return r
}

// WithTiming sets the timing info and returns the response for chaining.
func (r *Response) WithTiming(t interfaces.TimingInfo) *Response {
	r.timing = t
	return r
}

// WithMetadata adds a metadata entry and returns the response for chaining.
func (r *Response) WithMetadata(key string, value any) *Response {
	r.metadata[key] = value
	return r
}
