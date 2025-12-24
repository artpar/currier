package interfaces

import (
	"context"
	"time"
)

// CollectionStore manages collection persistence.
type CollectionStore interface {
	// List returns all collections.
	List(ctx context.Context) ([]CollectionMeta, error)

	// Get retrieves a collection by ID.
	Get(ctx context.Context, id string) (Collection, error)

	// GetByPath retrieves a collection by file path.
	GetByPath(ctx context.Context, path string) (Collection, error)

	// Save persists a collection.
	Save(ctx context.Context, c Collection) error

	// Delete removes a collection.
	Delete(ctx context.Context, id string) error

	// Search finds collections matching the query.
	Search(ctx context.Context, query string) ([]CollectionMeta, error)
}

// RequestStore manages individual request persistence.
type RequestStore interface {
	// Get retrieves a request by ID.
	Get(ctx context.Context, id string) (RequestDefinition, error)

	// Save persists a request.
	Save(ctx context.Context, r RequestDefinition) error

	// Delete removes a request.
	Delete(ctx context.Context, id string) error

	// ListByCollection returns all requests in a collection.
	ListByCollection(ctx context.Context, collectionID string) ([]RequestMeta, error)
}

// EnvironmentStore manages environment persistence.
type EnvironmentStore interface {
	// List returns all environments.
	List(ctx context.Context) ([]EnvironmentMeta, error)

	// Get retrieves an environment by ID.
	Get(ctx context.Context, id string) (Environment, error)

	// GetByName retrieves an environment by name.
	GetByName(ctx context.Context, name string) (Environment, error)

	// Save persists an environment.
	Save(ctx context.Context, e Environment) error

	// Delete removes an environment.
	Delete(ctx context.Context, id string) error

	// GetActive returns the currently active environment.
	GetActive(ctx context.Context) (Environment, error)

	// SetActive sets the active environment.
	SetActive(ctx context.Context, id string) error
}

// HistoryStore manages request history.
type HistoryStore interface {
	// Add adds a history entry.
	Add(ctx context.Context, entry HistoryEntry) error

	// List returns history entries matching the options.
	List(ctx context.Context, opts HistoryQueryOpts) ([]HistoryEntry, error)

	// Get retrieves a history entry by ID.
	Get(ctx context.Context, id string) (HistoryEntry, error)

	// Clear removes history entries older than the given time.
	Clear(ctx context.Context, before time.Time) error

	// Search finds history entries matching the query.
	Search(ctx context.Context, query string) ([]HistoryEntry, error)

	// Count returns the total number of history entries.
	Count(ctx context.Context) (int64, error)
}

// Collection represents a collection of requests.
type Collection interface {
	// ID returns the collection identifier.
	ID() string

	// Name returns the collection name.
	Name() string

	// Description returns the collection description.
	Description() string

	// Version returns the collection version.
	Version() string

	// Variables returns collection-level variables.
	Variables() map[string]string

	// Auth returns the default authentication for the collection.
	Auth() AuthConfig

	// PreScript returns the pre-request script.
	PreScript() string

	// Folders returns the collection folders.
	Folders() []Folder

	// Requests returns top-level requests (not in folders).
	Requests() []RequestDefinition

	// Path returns the file system path.
	Path() string

	// CreatedAt returns the creation timestamp.
	CreatedAt() time.Time

	// UpdatedAt returns the last update timestamp.
	UpdatedAt() time.Time
}

// CollectionMeta contains collection metadata without full content.
type CollectionMeta struct {
	ID          string
	Name        string
	Description string
	Path        string
	RequestCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Folder represents a folder within a collection.
type Folder interface {
	// Name returns the folder name.
	Name() string

	// Description returns the folder description.
	Description() string

	// Requests returns the requests in this folder.
	Requests() []RequestDefinition

	// Folders returns nested folders.
	Folders() []Folder

	// Auth returns folder-level authentication override.
	Auth() AuthConfig

	// PreScript returns folder-level pre-request script.
	PreScript() string
}

// RequestDefinition represents a saved request definition.
type RequestDefinition interface {
	// ID returns the request identifier.
	ID() string

	// Name returns the request name.
	Name() string

	// Description returns the request description.
	Description() string

	// Protocol returns the protocol type.
	Protocol() string

	// Method returns the HTTP method or equivalent.
	Method() string

	// URL returns the request URL (may contain variables).
	URL() string

	// Headers returns the request headers.
	Headers() map[string]string

	// QueryParams returns query parameters.
	QueryParams() map[string]string

	// Body returns the request body configuration.
	Body() BodyConfig

	// Auth returns the authentication configuration.
	Auth() AuthConfig

	// PreScript returns the pre-request script.
	PreScript() string

	// PostScript returns the post-response script.
	PostScript() string

	// Tests returns the test definitions.
	Tests() []TestDefinition

	// Options returns request options.
	Options() RequestOptions

	// CollectionID returns the parent collection ID.
	CollectionID() string

	// FolderPath returns the folder path within the collection.
	FolderPath() string

	// CreatedAt returns the creation timestamp.
	CreatedAt() time.Time

	// UpdatedAt returns the last update timestamp.
	UpdatedAt() time.Time
}

// RequestMeta contains request metadata without full content.
type RequestMeta struct {
	ID           string
	Name         string
	Method       string
	URL          string
	CollectionID string
	FolderPath   string
	UpdatedAt    time.Time
}

// BodyConfig represents body configuration.
type BodyConfig struct {
	Type        string // "json", "form", "multipart", "raw", "graphql", "protobuf"
	Content     any
	ContentType string
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Type   string // "none", "basic", "bearer", "oauth2", "apikey", "aws"
	Params map[string]string
}

// TestDefinition represents a test assertion.
type TestDefinition struct {
	Name   string
	Assert string // JavaScript expression
}

// RequestOptions contains request options.
type RequestOptions struct {
	Timeout          time.Duration
	FollowRedirects  bool
	VerifySSL        bool
	MaxRedirects     int
}

// Environment represents an environment with variables.
type Environment interface {
	// ID returns the environment identifier.
	ID() string

	// Name returns the environment name.
	Name() string

	// Variables returns the environment variables.
	Variables() map[string]string

	// Secrets returns the encrypted secrets.
	Secrets() map[string]string

	// IsActive returns true if this is the active environment.
	IsActive() bool

	// CreatedAt returns the creation timestamp.
	CreatedAt() time.Time

	// UpdatedAt returns the last update timestamp.
	UpdatedAt() time.Time

	// SetVariable sets a variable value.
	SetVariable(key, value string)

	// GetVariable gets a variable value.
	GetVariable(key string) (string, bool)

	// DeleteVariable removes a variable.
	DeleteVariable(key string)
}

// EnvironmentMeta contains environment metadata.
type EnvironmentMeta struct {
	ID        string
	Name      string
	IsActive  bool
	VarCount  int
	UpdatedAt time.Time
}

// HistoryEntry represents a request/response in history.
type HistoryEntry struct {
	ID          string
	RequestID   string
	RequestName string
	Method      string
	URL         string
	Status      int
	StatusText  string
	Duration    time.Duration
	Size        int64
	Timestamp   time.Time

	// Full request/response data (may be loaded on demand)
	Request  Request
	Response Response
}

// HistoryQueryOpts specifies options for querying history.
type HistoryQueryOpts struct {
	Limit      int
	Offset     int
	Method     string
	URL        string
	Status     int
	MinStatus  int
	MaxStatus  int
	Before     time.Time
	After      time.Time
	Search     string
	SortBy     string
	Descending bool
}
