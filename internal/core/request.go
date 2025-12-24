package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/google/uuid"
)

// Request implements the interfaces.Request interface.
type Request struct {
	id       string
	protocol string
	method   string
	endpoint string
	headers  *Headers
	body     Body
	metadata map[string]any
}

// NewRequest creates a new request with the given parameters.
func NewRequest(protocol, method, endpoint string) (*Request, error) {
	if method == "" {
		return nil, errors.New("method cannot be empty")
	}
	if endpoint == "" {
		return nil, errors.New("endpoint cannot be empty")
	}

	return &Request{
		id:       uuid.New().String(),
		protocol: protocol,
		method:   method,
		endpoint: endpoint,
		headers:  NewHeaders(),
		body:     NewEmptyBody(),
		metadata: make(map[string]any),
	}, nil
}

func (r *Request) ID() string {
	return r.id
}

func (r *Request) Protocol() string {
	return r.protocol
}

func (r *Request) Method() string {
	return r.method
}

func (r *Request) Endpoint() string {
	return r.endpoint
}

func (r *Request) Headers() *Headers {
	return r.headers
}

func (r *Request) Body() Body {
	return r.body
}

func (r *Request) Metadata() map[string]any {
	result := make(map[string]any)
	for k, v := range r.metadata {
		result[k] = v
	}
	return result
}

func (r *Request) Clone() *Request {
	clone := &Request{
		id:       uuid.New().String(),
		protocol: r.protocol,
		method:   r.method,
		endpoint: r.endpoint,
		headers:  r.headers.Clone(),
		body:     r.body,
		metadata: make(map[string]any),
	}
	for k, v := range r.metadata {
		clone.metadata[k] = v
	}
	return clone
}

func (r *Request) Validate() error {
	if r.method == "" {
		return errors.New("method cannot be empty")
	}
	if r.endpoint == "" {
		return errors.New("endpoint cannot be empty")
	}
	return nil
}

func (r *Request) SetHeader(key, value string) {
	r.headers.Set(key, value)
}

func (r *Request) SetBody(body Body) {
	r.body = body
}

func (r *Request) SetMetadata(key string, value any) {
	r.metadata[key] = value
}

// Headers implements a case-insensitive HTTP header store.
type Headers struct {
	data     map[string][]string
	keyOrder []string // Preserves original casing for keys
}

// NewHeaders creates an empty headers collection.
func NewHeaders() *Headers {
	return &Headers{
		data:     make(map[string][]string),
		keyOrder: make([]string, 0),
	}
}

func (h *Headers) normalize(key string) string {
	return strings.ToLower(key)
}

func (h *Headers) Set(key, value string) {
	normalized := h.normalize(key)
	if _, exists := h.data[normalized]; !exists {
		h.keyOrder = append(h.keyOrder, key)
	} else {
		// Update keyOrder with new casing
		for i, k := range h.keyOrder {
			if h.normalize(k) == normalized {
				h.keyOrder[i] = key
				break
			}
		}
	}
	h.data[normalized] = []string{value}
}

func (h *Headers) Add(key, value string) {
	normalized := h.normalize(key)
	if _, exists := h.data[normalized]; !exists {
		h.keyOrder = append(h.keyOrder, key)
	}
	h.data[normalized] = append(h.data[normalized], value)
}

func (h *Headers) Get(key string) string {
	values := h.data[h.normalize(key)]
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func (h *Headers) GetAll(key string) []string {
	values := h.data[h.normalize(key)]
	if values == nil {
		return []string{}
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func (h *Headers) Del(key string) {
	normalized := h.normalize(key)
	delete(h.data, normalized)
	// Remove from keyOrder
	for i, k := range h.keyOrder {
		if h.normalize(k) == normalized {
			h.keyOrder = append(h.keyOrder[:i], h.keyOrder[i+1:]...)
			break
		}
	}
}

func (h *Headers) Keys() []string {
	result := make([]string, len(h.keyOrder))
	copy(result, h.keyOrder)
	return result
}

func (h *Headers) Clone() *Headers {
	clone := NewHeaders()
	for _, key := range h.keyOrder {
		normalized := h.normalize(key)
		for _, v := range h.data[normalized] {
			clone.Add(key, v)
		}
	}
	return clone
}

func (h *Headers) ToMap() map[string][]string {
	result := make(map[string][]string)
	for _, key := range h.keyOrder {
		normalized := h.normalize(key)
		result[key] = make([]string, len(h.data[normalized]))
		copy(result[key], h.data[normalized])
	}
	return result
}

// Body represents a request or response body.
type Body interface {
	Type() string
	ContentType() string
	IsEmpty() bool
	Size() int64
	Bytes() []byte
	String() string
	Reader() io.Reader
	JSON() (any, error)
}

// emptyBody represents an empty body.
type emptyBody struct{}

// NewEmptyBody creates an empty body.
func NewEmptyBody() Body {
	return &emptyBody{}
}

func (b *emptyBody) Type() string        { return "empty" }
func (b *emptyBody) ContentType() string { return "" }
func (b *emptyBody) IsEmpty() bool       { return true }
func (b *emptyBody) Size() int64         { return 0 }
func (b *emptyBody) Bytes() []byte       { return nil }
func (b *emptyBody) String() string      { return "" }
func (b *emptyBody) Reader() io.Reader   { return bytes.NewReader(nil) }
func (b *emptyBody) JSON() (any, error)  { return nil, errors.New("empty body") }

// jsonBody represents a JSON body.
type jsonBody struct {
	data    any
	encoded []byte
}

// NewJSONBody creates a JSON body from any value.
func NewJSONBody(data any) Body {
	encoded, _ := json.Marshal(data)
	return &jsonBody{
		data:    data,
		encoded: encoded,
	}
}

func (b *jsonBody) Type() string        { return "json" }
func (b *jsonBody) ContentType() string { return "application/json" }
func (b *jsonBody) IsEmpty() bool       { return len(b.encoded) == 0 }
func (b *jsonBody) Size() int64         { return int64(len(b.encoded)) }
func (b *jsonBody) Bytes() []byte       { return b.encoded }
func (b *jsonBody) String() string      { return string(b.encoded) }
func (b *jsonBody) Reader() io.Reader   { return bytes.NewReader(b.encoded) }
func (b *jsonBody) JSON() (any, error) {
	var result any
	err := json.Unmarshal(b.encoded, &result)
	return result, err
}

// rawBody represents a raw byte body.
type rawBody struct {
	content     []byte
	contentType string
}

// NewRawBody creates a raw body with the given content and content type.
func NewRawBody(content []byte, contentType string) Body {
	return &rawBody{
		content:     content,
		contentType: contentType,
	}
}

func (b *rawBody) Type() string        { return "raw" }
func (b *rawBody) ContentType() string { return b.contentType }
func (b *rawBody) IsEmpty() bool       { return len(b.content) == 0 }
func (b *rawBody) Size() int64         { return int64(len(b.content)) }
func (b *rawBody) Bytes() []byte       { return b.content }
func (b *rawBody) String() string      { return string(b.content) }
func (b *rawBody) Reader() io.Reader   { return bytes.NewReader(b.content) }
func (b *rawBody) JSON() (any, error) {
	var result any
	err := json.Unmarshal(b.content, &result)
	return result, err
}

// Status represents an HTTP status code and text.
type Status struct {
	code int
	text string
}

// NewStatus creates a new status.
func NewStatus(code int, text string) *Status {
	return &Status{
		code: code,
		text: text,
	}
}

func (s *Status) Code() int    { return s.code }
func (s *Status) Text() string { return s.text }

func (s *Status) IsSuccess() bool {
	return s.code >= 200 && s.code < 300
}

func (s *Status) IsError() bool {
	return s.code >= 400
}
