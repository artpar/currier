// Package interfaces defines all core abstractions for the Currier application.
// All components communicate through these interfaces for loose coupling.
package interfaces

import (
	"context"
	"io"
	"time"
)

// Requester is the core interface for sending requests.
// Implemented by: HTTPClient, WebSocketClient, GRPCClient, GraphQLClient, etc.
type Requester interface {
	// Send executes a request and returns the response.
	Send(ctx context.Context, req Request) (Response, error)

	// Protocol returns the protocol identifier (e.g., "http", "websocket", "grpc").
	Protocol() string
}

// StreamRequester extends Requester for protocols that support streaming responses.
type StreamRequester interface {
	Requester

	// SendStream executes a request and returns a stream of responses.
	SendStream(ctx context.Context, req Request) (ResponseStream, error)
}

// ConnectionManager manages persistent connections for stateful protocols.
type ConnectionManager interface {
	// Connect establishes a connection to the given endpoint.
	Connect(ctx context.Context, endpoint string, opts ConnectionOptions) (Connection, error)

	// Disconnect closes the connection with the given ID.
	Disconnect(id string) error

	// ListConnections returns all active connections.
	ListConnections() []ConnectionInfo

	// GetConnection returns a specific connection by ID.
	GetConnection(id string) (Connection, error)
}

// Connection represents an active connection for stateful protocols.
type Connection interface {
	// ID returns the unique connection identifier.
	ID() string

	// Endpoint returns the connection endpoint.
	Endpoint() string

	// State returns the current connection state.
	State() ConnectionState

	// Send sends a message on this connection.
	Send(ctx context.Context, data []byte) error

	// Receive receives a message from this connection.
	Receive(ctx context.Context) ([]byte, error)

	// Close closes the connection.
	Close() error
}

// ConnectionOptions contains options for establishing a connection.
type ConnectionOptions struct {
	Headers     map[string]string
	Timeout     time.Duration
	TLSInsecure bool
}

// ConnectionInfo provides metadata about a connection.
type ConnectionInfo struct {
	ID        string
	Endpoint  string
	State     ConnectionState
	CreatedAt time.Time
	Protocol  string
}

// ConnectionState represents the state of a connection.
type ConnectionState int

const (
	ConnectionStateConnecting ConnectionState = iota
	ConnectionStateConnected
	ConnectionStateDisconnecting
	ConnectionStateDisconnected
	ConnectionStateError
)

func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateConnecting:
		return "connecting"
	case ConnectionStateConnected:
		return "connected"
	case ConnectionStateDisconnecting:
		return "disconnecting"
	case ConnectionStateDisconnected:
		return "disconnected"
	case ConnectionStateError:
		return "error"
	default:
		return "unknown"
	}
}

// Request represents a protocol-agnostic request.
type Request interface {
	// ID returns the unique request identifier.
	ID() string

	// Protocol returns the protocol type (e.g., "http", "websocket", "grpc").
	Protocol() string

	// Method returns the request method (e.g., "GET", "POST", "SUBSCRIBE").
	Method() string

	// Endpoint returns the target endpoint (URL, topic, service/method).
	Endpoint() string

	// Headers returns the request headers.
	Headers() Headers

	// Body returns the request body.
	Body() Body

	// Metadata returns protocol-specific metadata.
	Metadata() map[string]any

	// Clone creates a deep copy of the request.
	Clone() Request

	// Validate checks if the request is valid.
	Validate() error

	// SetHeader sets a header value.
	SetHeader(key, value string)

	// SetBody sets the request body.
	SetBody(body Body)

	// SetMetadata sets a metadata value.
	SetMetadata(key string, value any)
}

// Response represents a protocol-agnostic response.
type Response interface {
	// ID returns the unique response identifier.
	ID() string

	// RequestID returns the ID of the originating request.
	RequestID() string

	// Protocol returns the protocol type.
	Protocol() string

	// Status returns the response status.
	Status() Status

	// Headers returns the response headers.
	Headers() Headers

	// Body returns the response body.
	Body() Body

	// Timing returns timing information.
	Timing() TimingInfo

	// Metadata returns protocol-specific metadata.
	Metadata() map[string]any
}

// ResponseStream provides streaming access to responses.
type ResponseStream interface {
	// Next returns the next response in the stream.
	// Returns io.EOF when the stream is exhausted.
	Next() (Response, error)

	// Close closes the stream.
	Close() error
}

// Headers provides access to header values.
type Headers interface {
	// Get returns the first value for the given key.
	Get(key string) string

	// GetAll returns all values for the given key.
	GetAll(key string) []string

	// Set sets the value for the given key, replacing any existing values.
	Set(key, value string)

	// Add adds a value for the given key.
	Add(key, value string)

	// Del removes all values for the given key.
	Del(key string)

	// Keys returns all header keys.
	Keys() []string

	// Clone creates a deep copy of the headers.
	Clone() Headers

	// ToMap returns the headers as a map.
	ToMap() map[string][]string
}

// Body represents request or response body content.
type Body interface {
	// Type returns the body type (e.g., "json", "form", "raw", "graphql", "protobuf").
	Type() string

	// ContentType returns the MIME content type.
	ContentType() string

	// Bytes returns the raw body bytes.
	Bytes() []byte

	// String returns the body as a string.
	String() string

	// Reader returns an io.Reader for the body.
	Reader() io.Reader

	// Size returns the body size in bytes.
	Size() int64

	// IsEmpty returns true if the body is empty.
	IsEmpty() bool

	// JSON attempts to parse the body as JSON and returns the result.
	JSON() (any, error)
}

// Status represents a response status.
type Status interface {
	// Code returns the numeric status code.
	Code() int

	// Text returns the status text.
	Text() string

	// IsSuccess returns true if the status indicates success.
	IsSuccess() bool

	// IsError returns true if the status indicates an error.
	IsError() bool
}

// TimingInfo contains request/response timing information.
type TimingInfo struct {
	// StartTime is when the request started.
	StartTime time.Time

	// EndTime is when the response was fully received.
	EndTime time.Time

	// DNSLookup is the time spent on DNS lookup.
	DNSLookup time.Duration

	// TCPConnection is the time spent establishing TCP connection.
	TCPConnection time.Duration

	// TLSHandshake is the time spent on TLS handshake.
	TLSHandshake time.Duration

	// TimeToFirstByte is the time until the first byte was received.
	TimeToFirstByte time.Duration

	// ContentTransfer is the time spent receiving the response body.
	ContentTransfer time.Duration

	// Total is the total request duration.
	Total time.Duration
}
