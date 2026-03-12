package idem

// Metrics holds callback functions for observing middleware events.
// All fields are optional; nil callbacks are never invoked.
type Metrics struct {
	// OnCacheHit is called when a cached response is found for the idempotency key.
	OnCacheHit func(key string)

	// OnCacheMiss is called when no cached response exists for the idempotency key.
	OnCacheMiss func(key string)

	// OnLockContention is called when lock acquisition fails for the idempotency key,
	// resulting in a 409 Conflict response. When this event fires, OnError is not called;
	// lock contention is treated as a normal concurrency-control signal, not an error.
	OnLockContention func(key string, err error)

	// OnError is called when a storage operation (Get, Set, or Delete) fails.
	// Lock contention is reported via OnLockContention instead.
	OnError func(key string, err error)
}

// WithMetrics specifies callbacks for observing middleware events such as
// cache hits, cache misses, and errors. Passing a zero-value Metrics is
// valid and results in no overhead.
func WithMetrics(m Metrics) Option {
	return func(c *config) {
		c.metrics = &m
	}
}
