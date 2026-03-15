package redis

import "errors"

// Sentinel errors returned by New when the configuration is invalid.
var (
	// ErrNilClient is returned when the Redis client is nil.
	ErrNilClient = errors.New("redis: client must not be nil")

	// ErrEmptyKeyPrefix is returned when the key prefix is empty.
	ErrEmptyKeyPrefix = errors.New("redis: keyPrefix must not be empty")

	// ErrEmptyLockPrefix is returned when the lock prefix is empty.
	ErrEmptyLockPrefix = errors.New("redis: lockPrefix must not be empty")
)
