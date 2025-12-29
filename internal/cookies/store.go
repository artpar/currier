package cookies

import (
	"context"
	"errors"
)

// Common errors.
var (
	ErrNotFound    = errors.New("cookie not found")
	ErrStoreClosed = errors.New("cookie store is closed")
)

// Store defines the interface for cookie persistence.
type Store interface {
	// Set stores or updates a cookie.
	Set(ctx context.Context, cookie *Cookie) error

	// Get retrieves a cookie by domain, path, and name.
	Get(ctx context.Context, domain, path, name string) (*Cookie, error)

	// List returns cookies matching the query options.
	List(ctx context.Context, opts QueryOptions) ([]*Cookie, error)

	// Delete removes a specific cookie.
	Delete(ctx context.Context, domain, path, name string) error

	// DeleteByDomain removes all cookies for a domain.
	DeleteByDomain(ctx context.Context, domain string) error

	// DeleteExpired removes all expired cookies and returns count.
	DeleteExpired(ctx context.Context) (int64, error)

	// Clear removes all cookies.
	Clear(ctx context.Context) error

	// Count returns total number of cookies.
	Count(ctx context.Context) (int64, error)

	// Close closes the store.
	Close() error
}
