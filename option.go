package idem

import (
	"errors"
	"time"
)

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
	onError   func(error)
	metrics   *Metrics
}

// defaultConfig returns a config with sensible defaults.
// storage is nil by default; the middleware constructor initializes it
// to NewMemoryStorage().
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

// WithOnError specifies a callback function that is called when a storage operation fails.
func WithOnError(fn func(error)) Option {
	return func(c *config) {
		c.onError = fn
	}
}

// validate checks the config for invalid values.
func (c *config) validate() error {
	if c.keyHeader == "" {
		return errors.New("idem: keyHeader must not be empty")
	}

	if c.ttl <= 0 {
		return errors.New("idem: ttl must be positive")
	}

	return nil
}
