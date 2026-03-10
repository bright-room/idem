package idem

import "time"

const (
	// DefaultKeyHeader is the default HTTP header name for the idempotency key.
	DefaultKeyHeader = "Idempotency-Key"

	// DefaultTTL is the default time-to-live for cached responses.
	DefaultTTL = 24 * time.Hour
)

type config struct {
	keyHeader string
	ttl       time.Duration
	storage   Storage
}

// defaultConfig returns a config with sensible defaults.
// storage is nil by default; the middleware constructor initializes it
// to memory.New() to avoid an import cycle between idem and idem/memory.
func defaultConfig() *config {
	return &config{
		keyHeader: DefaultKeyHeader,
		ttl:       DefaultTTL,
	}
}

// Option configures the middleware behavior.
type Option func(*config)

// WithKeyHeader specifies the HTTP header name used to retrieve the idempotency key.
func WithKeyHeader(header string) Option {
	return func(c *config) {
		c.keyHeader = header
	}
}

// WithTTL specifies the time-to-live for cached responses.
func WithTTL(ttl time.Duration) Option {
	return func(c *config) {
		c.ttl = ttl
	}
}

// WithStorage specifies the storage backend for caching responses.
func WithStorage(s Storage) Option {
	return func(c *config) {
		c.storage = s
	}
}
