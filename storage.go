package idem

import (
	"context"
	"time"
)

// Storage abstracts the persistence of idempotency keys and their associated responses.
type Storage interface {
	// Get returns the cached response for the given key.
	// If the key does not exist, it returns nil, nil.
	Get(ctx context.Context, key string) (*Response, error)

	// Set stores the response for the given key.
	// ttl specifies the time-to-live for the cache entry.
	Set(ctx context.Context, key string, res *Response, ttl time.Duration) error
}
