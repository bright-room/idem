package idem

// Metrics holds callback functions for observing middleware events.
// All fields are optional; nil callbacks are never invoked.
type Metrics struct {
	// OnCacheHit is called when a cached response is found for the idempotency key.
	OnCacheHit func(key string)

	// OnCacheMiss is called when no cached response exists for the idempotency key.
	OnCacheMiss func(key string)

	// OnError is called when a storage or lock operation fails.
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
