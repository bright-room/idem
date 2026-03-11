package idem

import (
	"context"
	"time"
)

// Locker provides mutual exclusion for idempotency keys.
// Storage implementations may optionally implement this interface
// to enable concurrent request handling for the same key.
type Locker interface {
	// Lock acquires a lock for the given key with the specified TTL.
	// It returns an unlock function that must be called to release the lock.
	// If the lock cannot be acquired within the context deadline,
	// the context error is returned.
	Lock(ctx context.Context, key string, ttl time.Duration) (unlock func(), err error)
}
