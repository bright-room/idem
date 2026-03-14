package idem

import (
	"errors"
	"fmt"
	"time"
)

const (
	// DefaultKeyHeader is the default HTTP header name for the idempotency key.
	DefaultKeyHeader = "Idempotency-Key"

	// DefaultTTL is the default time-to-live for cached responses.
	DefaultTTL = 24 * time.Hour
)

// Config is a read-only snapshot of the middleware configuration.
// It is passed to Validator functions so they can inspect—but not modify—the
// settings that will be used by the middleware.
//
// Config is also available via Middleware.Config() for debug logging
// and configuration inspection endpoints.
type Config struct {
	// KeyHeader is the HTTP header name used to retrieve the idempotency key.
	KeyHeader string `json:"key_header"`

	// TTL is the time-to-live for cached responses.
	TTL time.Duration `json:"ttl"`

	// KeyMaxLength is the maximum allowed length for idempotency key values.
	// A value of 0 means no length limit.
	KeyMaxLength int `json:"key_max_length"`

	// StorageType is the Go type name of the storage backend (e.g. "*idem.MemoryStorage").
	StorageType string `json:"storage_type"`

	// LockSupported indicates whether the storage backend implements the Locker interface.
	LockSupported bool `json:"lock_supported"`

	// MetricsEnabled indicates whether metrics callbacks are configured.
	MetricsEnabled bool `json:"metrics_enabled"`

	// OnErrorEnabled indicates whether an error callback is configured.
	OnErrorEnabled bool `json:"on_error_enabled"`

	// ValidatorCount is the number of registered validators.
	ValidatorCount int `json:"validator_count"`
}

// Validator is a function that inspects the middleware configuration and
// returns an error if it does not meet application-specific requirements.
type Validator func(Config) error

type config struct {
	keyHeader    string
	ttl          time.Duration
	keyMaxLength int
	storage      Storage
	onError      func(key string, err error)
	metrics      *Metrics
	validators   []Validator
}

func (c *config) snapshot() Config {
	cfg := Config{
		KeyHeader:      c.keyHeader,
		TTL:            c.ttl,
		KeyMaxLength:   c.keyMaxLength,
		ValidatorCount: len(c.validators),
	}

	if c.storage != nil {
		cfg.StorageType = fmt.Sprintf("%T", c.storage)
		_, cfg.LockSupported = c.storage.(Locker)
	}

	cfg.MetricsEnabled = c.metrics != nil
	cfg.OnErrorEnabled = c.onError != nil

	return cfg
}

// String returns a human-readable summary of the configuration.
func (c Config) String() string {
	return fmt.Sprintf(
		"idem.Config{KeyHeader: %q, TTL: %v, KeyMaxLength: %d, StorageType: %s, LockSupported: %t, MetricsEnabled: %t}",
		c.KeyHeader, c.TTL, c.KeyMaxLength, c.StorageType, c.LockSupported, c.MetricsEnabled,
	)
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

// WithOnError specifies a callback function that is called when a storage
// operation (Get or Set) fails. The key is the idempotency key from the request.
// Lock contention is not reported here; use Metrics.OnLockContention instead.
func WithOnError(fn func(key string, err error)) Option {
	return func(c *config) {
		c.onError = fn
	}
}

// WithKeyMaxLength sets the maximum allowed length for idempotency key values.
// Requests with keys exceeding this length receive a 400 Bad Request response.
// A value of 0 (default) disables the length check.
func WithKeyMaxLength(n int) Option {
	return func(c *config) {
		c.keyMaxLength = n
	}
}

// WithValidation registers one or more custom validators that run during
// middleware initialization. Validators execute in registration order,
// after the built-in checks (non-empty keyHeader, positive ttl).
// Validation stops at the first error.
func WithValidation(validators ...Validator) Option {
	return func(c *config) {
		c.validators = append(c.validators, validators...)
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

	snap := c.snapshot()
	for _, v := range c.validators {
		if err := v(snap); err != nil {
			return err
		}
	}

	return nil
}
