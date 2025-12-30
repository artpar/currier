package starred

import (
	"context"
	"errors"
)

// Common errors.
var (
	ErrStoreClosed = errors.New("starred store is closed")
)

// Store defines the interface for starred request persistence.
// Starred status is user preference metadata, stored separately from request definitions.
type Store interface {
	// IsStarred checks if a request is starred.
	IsStarred(ctx context.Context, requestID string) (bool, error)

	// Star marks a request as starred.
	Star(ctx context.Context, requestID string) error

	// Unstar removes the starred status from a request.
	Unstar(ctx context.Context, requestID string) error

	// Toggle toggles the starred status and returns the new state.
	Toggle(ctx context.Context, requestID string) (bool, error)

	// ListStarred returns all starred request IDs.
	ListStarred(ctx context.Context) ([]string, error)

	// Clear removes all starred entries.
	Clear(ctx context.Context) error

	// Count returns total number of starred requests.
	Count(ctx context.Context) (int64, error)

	// Close closes the store.
	Close() error
}
