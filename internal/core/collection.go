package core

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/artpar/currier/internal/interpolate"
)

// Collection represents a group of API requests.
type Collection struct {
	id          string
	name        string
	description string
	version     string
	variables   map[string]string
	folders     []*Folder
	requests    []*RequestDefinition
	websockets  []*WebSocketDefinition
	auth        AuthConfig
	preScript   string
	postScript  string
	createdAt   time.Time
	updatedAt   time.Time
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Type     string `json:"type" yaml:"type"`
	Token    string `json:"token,omitempty" yaml:"token,omitempty"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Key      string `json:"key,omitempty" yaml:"key,omitempty"`
	Value    string `json:"value,omitempty" yaml:"value,omitempty"`
	In       string `json:"in,omitempty" yaml:"in,omitempty"` // header, query

	// OAuth 2.0 configuration
	OAuth2 *OAuth2Config `json:"oauth2,omitempty" yaml:"oauth2,omitempty"`

	// AWS Signature v4 configuration
	AWS *AWSAuthConfig `json:"aws,omitempty" yaml:"aws,omitempty"`
}

// NewCollection creates a new collection with the given name.
func NewCollection(name string) *Collection {
	now := time.Now()
	return &Collection{
		id:         uuid.New().String(),
		name:       name,
		variables:  make(map[string]string),
		folders:    make([]*Folder, 0),
		requests:   make([]*RequestDefinition, 0),
		websockets: make([]*WebSocketDefinition, 0),
		createdAt:  now,
		updatedAt:  now,
	}
}

func (c *Collection) ID() string          { return c.id }
func (c *Collection) Name() string        { return c.name }
func (c *Collection) Description() string { return c.description }
func (c *Collection) Version() string     { return c.version }
func (c *Collection) CreatedAt() time.Time { return c.createdAt }
func (c *Collection) UpdatedAt() time.Time { return c.updatedAt }
func (c *Collection) Auth() AuthConfig    { return c.auth }
func (c *Collection) PreScript() string   { return c.preScript }
func (c *Collection) PostScript() string  { return c.postScript }

func (c *Collection) SetDescription(desc string) {
	c.description = desc
	c.touch()
}

func (c *Collection) SetName(name string) {
	c.name = name
	c.touch()
}

func (c *Collection) SetVersion(version string) {
	c.version = version
	c.touch()
}

func (c *Collection) SetAuth(auth AuthConfig) {
	c.auth = auth
	c.touch()
}

func (c *Collection) SetPreScript(script string) {
	c.preScript = script
	c.touch()
}

func (c *Collection) SetPostScript(script string) {
	c.postScript = script
	c.touch()
}

func (c *Collection) touch() {
	c.updatedAt = time.Now()
}

// Variables returns all collection variables.
func (c *Collection) Variables() map[string]string {
	result := make(map[string]string)
	for k, v := range c.variables {
		result[k] = v
	}
	return result
}

// GetVariable returns a variable value.
func (c *Collection) GetVariable(key string) string {
	return c.variables[key]
}

// SetVariable sets a variable value.
func (c *Collection) SetVariable(key, value string) {
	c.variables[key] = value
	c.touch()
}

// DeleteVariable removes a variable.
func (c *Collection) DeleteVariable(key string) {
	delete(c.variables, key)
	c.touch()
}

// Folders returns all top-level folders.
func (c *Collection) Folders() []*Folder {
	return c.folders
}

// AddFolder adds a new folder to the collection.
func (c *Collection) AddFolder(name string) *Folder {
	folder := NewFolder(name)
	c.folders = append(c.folders, folder)
	c.touch()
	return folder
}

// GetFolder returns a folder by ID.
func (c *Collection) GetFolder(id string) (*Folder, bool) {
	for _, f := range c.folders {
		if f.ID() == id {
			return f, true
		}
	}
	return nil, false
}

// GetFolderByName returns a folder by name.
func (c *Collection) GetFolderByName(name string) (*Folder, bool) {
	for _, f := range c.folders {
		if f.Name() == name {
			return f, true
		}
	}
	return nil, false
}

// RemoveFolder removes a folder by ID.
func (c *Collection) RemoveFolder(id string) {
	for i, f := range c.folders {
		if f.ID() == id {
			c.folders = append(c.folders[:i], c.folders[i+1:]...)
			c.touch()
			return
		}
	}
}

// FindFolder searches for a folder by ID recursively.
func (c *Collection) FindFolder(id string) *Folder {
	for _, f := range c.folders {
		if f.ID() == id {
			return f
		}
		if found := f.FindFolder(id); found != nil {
			return found
		}
	}
	return nil
}

// Requests returns all root-level requests.
func (c *Collection) Requests() []*RequestDefinition {
	return c.requests
}

// FirstRequest returns the first request in the collection (root or in folders).
func (c *Collection) FirstRequest() *RequestDefinition {
	// Check root-level requests first
	if len(c.requests) > 0 {
		return c.requests[0]
	}
	// Check folders recursively
	for _, f := range c.folders {
		if req := f.FirstRequest(); req != nil {
			return req
		}
	}
	return nil
}

// AddRequest adds a request to the collection root.
func (c *Collection) AddRequest(req *RequestDefinition) {
	c.requests = append(c.requests, req)
	c.touch()
}

// GetRequest returns a request by ID from root level.
func (c *Collection) GetRequest(id string) (*RequestDefinition, bool) {
	for _, r := range c.requests {
		if r.ID() == id {
			return r, true
		}
	}
	return nil, false
}

// FindRequest searches for a request by ID in the entire collection.
func (c *Collection) FindRequest(id string) (*RequestDefinition, bool) {
	// Check root requests
	if req, ok := c.GetRequest(id); ok {
		return req, true
	}
	// Check folders recursively
	for _, f := range c.folders {
		if req, ok := f.FindRequest(id); ok {
			return req, true
		}
	}
	return nil, false
}

// RemoveRequest removes a request by ID from root level.
func (c *Collection) RemoveRequest(id string) bool {
	for i, r := range c.requests {
		if r.ID() == id {
			c.requests = append(c.requests[:i], c.requests[i+1:]...)
			c.touch()
			return true
		}
	}
	return false
}

// RemoveRequestRecursive searches for and removes a request by ID from anywhere in the collection.
func (c *Collection) RemoveRequestRecursive(id string) bool {
	// Try root level first
	if c.RemoveRequest(id) {
		return true
	}
	// Try folders recursively
	for _, f := range c.folders {
		if f.RemoveRequestRecursive(id) {
			c.touch()
			return true
		}
	}
	return false
}

// WebSockets returns all WebSocket definitions.
func (c *Collection) WebSockets() []*WebSocketDefinition {
	return c.websockets
}

// AddWebSocket adds a WebSocket definition to the collection.
func (c *Collection) AddWebSocket(ws *WebSocketDefinition) {
	c.websockets = append(c.websockets, ws)
	c.touch()
}

// GetWebSocket returns a WebSocket definition by ID.
func (c *Collection) GetWebSocket(id string) (*WebSocketDefinition, bool) {
	for _, ws := range c.websockets {
		if ws.ID == id {
			return ws, true
		}
	}
	return nil, false
}

// GetWebSocketByName returns a WebSocket definition by name.
func (c *Collection) GetWebSocketByName(name string) (*WebSocketDefinition, bool) {
	for _, ws := range c.websockets {
		if ws.Name == name {
			return ws, true
		}
	}
	return nil, false
}

// RemoveWebSocket removes a WebSocket definition by ID.
func (c *Collection) RemoveWebSocket(id string) {
	for i, ws := range c.websockets {
		if ws.ID == id {
			c.websockets = append(c.websockets[:i], c.websockets[i+1:]...)
			c.touch()
			return
		}
	}
}

// Clone creates a deep copy of the collection.
func (c *Collection) Clone() *Collection {
	clone := NewCollection(c.name)
	clone.description = c.description
	clone.version = c.version
	clone.auth = c.auth
	clone.preScript = c.preScript
	clone.postScript = c.postScript

	for k, v := range c.variables {
		clone.variables[k] = v
	}

	for _, f := range c.folders {
		clone.folders = append(clone.folders, f.Clone())
	}

	for _, r := range c.requests {
		clone.requests = append(clone.requests, r.Clone())
	}

	for _, ws := range c.websockets {
		clone.websockets = append(clone.websockets, ws.Clone())
	}

	return clone
}

// Folder represents a folder within a collection.
type Folder struct {
	id          string
	name        string
	description string
	folders     []*Folder
	requests    []*RequestDefinition
}

// NewFolder creates a new folder.
func NewFolder(name string) *Folder {
	return &Folder{
		id:       uuid.New().String(),
		name:     name,
		folders:  make([]*Folder, 0),
		requests: make([]*RequestDefinition, 0),
	}
}

func (f *Folder) ID() string          { return f.id }
func (f *Folder) Name() string        { return f.name }
func (f *Folder) Description() string { return f.description }
func (f *Folder) Folders() []*Folder  { return f.folders }
func (f *Folder) Requests() []*RequestDefinition { return f.requests }

// FirstRequest returns the first request in the folder or its subfolders.
func (f *Folder) FirstRequest() *RequestDefinition {
	if len(f.requests) > 0 {
		return f.requests[0]
	}
	for _, folder := range f.folders {
		if req := folder.FirstRequest(); req != nil {
			return req
		}
	}
	return nil
}

func (f *Folder) SetDescription(desc string) {
	f.description = desc
}

func (f *Folder) AddFolder(name string) *Folder {
	folder := NewFolder(name)
	f.folders = append(f.folders, folder)
	return folder
}

func (f *Folder) AddRequest(req *RequestDefinition) {
	f.requests = append(f.requests, req)
}

func (f *Folder) GetRequest(id string) (*RequestDefinition, bool) {
	for _, r := range f.requests {
		if r.ID() == id {
			return r, true
		}
	}
	return nil, false
}

func (f *Folder) FindRequest(id string) (*RequestDefinition, bool) {
	if req, ok := f.GetRequest(id); ok {
		return req, true
	}
	for _, folder := range f.folders {
		if req, ok := folder.FindRequest(id); ok {
			return req, true
		}
	}
	return nil, false
}

// RemoveRequest removes a request by ID from this folder.
func (f *Folder) RemoveRequest(id string) bool {
	for i, r := range f.requests {
		if r.ID() == id {
			f.requests = append(f.requests[:i], f.requests[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveRequestRecursive searches for and removes a request by ID from this folder or any subfolder.
func (f *Folder) RemoveRequestRecursive(id string) bool {
	// Try this folder first
	if f.RemoveRequest(id) {
		return true
	}
	// Try subfolders recursively
	for _, sub := range f.folders {
		if sub.RemoveRequestRecursive(id) {
			return true
		}
	}
	return false
}

// FindFolder searches for a folder by ID recursively.
func (f *Folder) FindFolder(id string) *Folder {
	for _, sf := range f.folders {
		if sf.ID() == id {
			return sf
		}
		if found := sf.FindFolder(id); found != nil {
			return found
		}
	}
	return nil
}

func (f *Folder) Clone() *Folder {
	clone := NewFolder(f.name)
	clone.description = f.description

	for _, folder := range f.folders {
		clone.folders = append(clone.folders, folder.Clone())
	}

	for _, req := range f.requests {
		clone.requests = append(clone.requests, req.Clone())
	}

	return clone
}

// RequestDefinition represents a saved request definition.
type RequestDefinition struct {
	id          string
	name        string
	description string
	method      string
	url         string
	headers     map[string]string
	queryParams map[string]string
	bodyType    string
	bodyContent string
	auth        *AuthConfig
	preScript   string
	postScript  string
}

// NewRequestDefinition creates a new request definition.
func NewRequestDefinition(name, method, url string) *RequestDefinition {
	return &RequestDefinition{
		id:          uuid.New().String(),
		name:        name,
		method:      method,
		url:         url,
		headers:     make(map[string]string),
		queryParams: make(map[string]string),
	}
}

func (r *RequestDefinition) ID() string          { return r.id }
func (r *RequestDefinition) Name() string        { return r.name }
func (r *RequestDefinition) Description() string { return r.description }
func (r *RequestDefinition) Method() string      { return r.method }
func (r *RequestDefinition) URL() string { return r.url }

// FullURL returns the URL with query parameters appended.
func (r *RequestDefinition) FullURL() string {
	if len(r.queryParams) == 0 {
		return r.url
	}
	parsed, err := url.Parse(r.url)
	if err != nil {
		return r.url
	}
	q := parsed.Query()
	for k, v := range r.queryParams {
		q.Set(k, v)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
func (r *RequestDefinition) BodyType() string    { return r.bodyType }
func (r *RequestDefinition) BodyContent() string { return r.bodyContent }
func (r *RequestDefinition) PreScript() string   { return r.preScript }
func (r *RequestDefinition) PostScript() string  { return r.postScript }

func (r *RequestDefinition) SetDescription(desc string) {
	r.description = desc
}

func (r *RequestDefinition) SetName(name string) {
	r.name = name
}

func (r *RequestDefinition) SetHeader(key, value string) {
	r.headers[key] = value
}

func (r *RequestDefinition) GetHeader(key string) string {
	return r.headers[key]
}

func (r *RequestDefinition) Headers() map[string]string {
	result := make(map[string]string)
	for k, v := range r.headers {
		result[k] = v
	}
	return result
}

func (r *RequestDefinition) SetBodyJSON(data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	r.bodyType = "json"
	r.bodyContent = string(bytes)
	return nil
}

func (r *RequestDefinition) SetBodyRaw(content, contentType string) {
	r.bodyType = "raw"
	r.bodyContent = content
}

func (r *RequestDefinition) SetPreScript(script string) {
	r.preScript = script
}

func (r *RequestDefinition) SetPostScript(script string) {
	r.postScript = script
}

func (r *RequestDefinition) SetAuth(auth AuthConfig) {
	r.auth = &auth
}

// ToRequest converts the definition to a core.Request for execution.
func (r *RequestDefinition) ToRequest() (*Request, error) {
	// Start with the base URL
	finalURL := r.url

	// Collect auth headers
	var authHeaders map[string]string

	// Apply authentication - may modify URL for query params
	if r.auth != nil && r.auth.IsConfigured() {
		authHeaders = make(map[string]string)
		authQueryParams := r.auth.ApplyToHeaders(authHeaders)

		// Add auth query params to URL if any
		if len(authQueryParams) > 0 {
			newURL, err := r.auth.ApplyToURL(finalURL)
			if err == nil {
				finalURL = newURL
			}
		}
	}

	req, err := NewRequest("http", r.method, finalURL)
	if err != nil {
		return nil, err
	}

	// Apply regular headers
	for key, value := range r.headers {
		req.SetHeader(key, value)
	}

	// Apply auth headers
	for key, value := range authHeaders {
		req.SetHeader(key, value)
	}

	if r.bodyContent != "" {
		var contentType string
		switch r.bodyType {
		case "json":
			contentType = "application/json"
		case "raw":
			contentType = "text/plain"
		default:
			contentType = "text/plain"
		}
		req.SetBody(NewRawBody([]byte(r.bodyContent), contentType))
	}

	return req, nil
}

// ToRequestWithEnv converts the definition to a core.Request with variable interpolation.
func (r *RequestDefinition) ToRequestWithEnv(engine *interpolate.Engine) (*Request, error) {
	// Interpolate URL
	finalURL, err := engine.Interpolate(r.url)
	if err != nil {
		return nil, err
	}

	// Collect auth headers
	var authHeaders map[string]string

	// Apply authentication (with interpolation for tokens/credentials)
	if r.auth != nil && r.auth.IsConfigured() {
		// Clone auth to interpolate values
		authCopy := *r.auth
		if authCopy.Token != "" {
			authCopy.Token, _ = engine.Interpolate(authCopy.Token)
		}
		if authCopy.Username != "" {
			authCopy.Username, _ = engine.Interpolate(authCopy.Username)
		}
		if authCopy.Password != "" {
			authCopy.Password, _ = engine.Interpolate(authCopy.Password)
		}
		if authCopy.Value != "" {
			authCopy.Value, _ = engine.Interpolate(authCopy.Value)
		}

		authHeaders = make(map[string]string)
		authQueryParams := authCopy.ApplyToHeaders(authHeaders)

		// Add auth query params to URL if any
		if len(authQueryParams) > 0 {
			newURL, err := authCopy.ApplyToURL(finalURL)
			if err == nil {
				finalURL = newURL
			}
		}
	}

	req, err := NewRequest("http", r.method, finalURL)
	if err != nil {
		return nil, err
	}

	// Interpolate and apply regular headers
	for key, value := range r.headers {
		interpolatedValue, err := engine.Interpolate(value)
		if err != nil {
			return nil, err
		}
		req.SetHeader(key, interpolatedValue)
	}

	// Apply auth headers
	for key, value := range authHeaders {
		req.SetHeader(key, value)
	}

	// Interpolate body
	if r.bodyContent != "" {
		interpolatedBody, err := engine.Interpolate(r.bodyContent)
		if err != nil {
			return nil, err
		}

		var contentType string
		switch r.bodyType {
		case "json":
			contentType = "application/json"
		case "raw":
			contentType = "text/plain"
		default:
			contentType = "text/plain"
		}
		req.SetBody(NewRawBody([]byte(interpolatedBody), contentType))
	}

	return req, nil
}

func (r *RequestDefinition) Clone() *RequestDefinition {
	clone := NewRequestDefinition(r.name, r.method, r.url)
	clone.description = r.description
	clone.bodyType = r.bodyType
	clone.bodyContent = r.bodyContent
	clone.preScript = r.preScript
	clone.postScript = r.postScript

	for k, v := range r.headers {
		clone.headers[k] = v
	}

	if r.auth != nil {
		authCopy := *r.auth
		clone.auth = &authCopy
	}

	return clone
}

// Auth returns the auth configuration.
func (r *RequestDefinition) Auth() *AuthConfig {
	return r.auth
}

// Body returns the body content.
func (r *RequestDefinition) Body() string {
	return r.bodyContent
}

// SetBody sets the body content.
func (r *RequestDefinition) SetBody(content string) {
	r.bodyContent = content
}

// SetURL sets the request URL and parses any query parameters from it.
func (r *RequestDefinition) SetURL(rawURL string) {
	// Parse query params from the URL
	if parsed, err := url.Parse(rawURL); err == nil {
		// Extract query params
		for key, values := range parsed.Query() {
			if len(values) > 0 {
				r.SetQueryParam(key, values[0])
			}
		}
		// Store the base URL without query string
		parsed.RawQuery = ""
		r.url = parsed.String()
		// Remove trailing ? if present
		r.url = strings.TrimSuffix(r.url, "?")
	} else {
		// If parsing fails, just store the raw URL
		r.url = rawURL
	}
}

// SetMethod sets the HTTP method.
func (r *RequestDefinition) SetMethod(method string) {
	r.method = method
}

// RemoveHeader removes a header by key.
func (r *RequestDefinition) RemoveHeader(key string) {
	delete(r.headers, key)
}

// QueryParams returns query parameters.
func (r *RequestDefinition) QueryParams() map[string]string {
	if r.queryParams == nil {
		return make(map[string]string)
	}
	result := make(map[string]string)
	for k, v := range r.queryParams {
		result[k] = v
	}
	return result
}

// GetQueryParam returns a single query parameter value.
func (r *RequestDefinition) GetQueryParam(key string) string {
	if r.queryParams == nil {
		return ""
	}
	return r.queryParams[key]
}

// SetQueryParam sets a query parameter.
func (r *RequestDefinition) SetQueryParam(key, value string) {
	if r.queryParams == nil {
		r.queryParams = make(map[string]string)
	}
	r.queryParams[key] = value
}

// RemoveQueryParam removes a query parameter.
func (r *RequestDefinition) RemoveQueryParam(key string) {
	if r.queryParams != nil {
		delete(r.queryParams, key)
	}
}

// NewCollectionWithID creates a collection with a specific ID (for loading from storage).
func NewCollectionWithID(id, name string) *Collection {
	now := time.Now()
	return &Collection{
		id:         id,
		name:       name,
		variables:  make(map[string]string),
		folders:    make([]*Folder, 0),
		requests:   make([]*RequestDefinition, 0),
		websockets: make([]*WebSocketDefinition, 0),
		createdAt:  now,
		updatedAt:  now,
	}
}

// AddExistingWebSocket adds an already-created WebSocket definition to the collection.
func (c *Collection) AddExistingWebSocket(ws *WebSocketDefinition) {
	c.websockets = append(c.websockets, ws)
}

// SetTimestamps sets created and updated timestamps (for loading from storage).
func (c *Collection) SetTimestamps(created, updated time.Time) {
	c.createdAt = created
	c.updatedAt = updated
}

// AddExistingFolder adds an already-created folder to the collection.
func (c *Collection) AddExistingFolder(f *Folder) {
	c.folders = append(c.folders, f)
}

// NewFolderWithID creates a folder with a specific ID (for loading from storage).
func NewFolderWithID(id, name string) *Folder {
	return &Folder{
		id:       id,
		name:     name,
		folders:  make([]*Folder, 0),
		requests: make([]*RequestDefinition, 0),
	}
}

// AddExistingFolder adds an already-created folder to this folder.
func (f *Folder) AddExistingFolder(sf *Folder) {
	f.folders = append(f.folders, sf)
}

// NewRequestDefinitionWithID creates a request definition with a specific ID.
func NewRequestDefinitionWithID(id, name, method, url string) *RequestDefinition {
	return &RequestDefinition{
		id:          id,
		name:        name,
		method:      method,
		url:         url,
		headers:     make(map[string]string),
		queryParams: make(map[string]string),
	}
}
