package idem

import "errors"

// Sentinel errors returned by New when the configuration is invalid.
var (
	// ErrEmptyKeyHeader is returned when the keyHeader is empty.
	ErrEmptyKeyHeader = errors.New("idem: keyHeader must not be empty")

	// ErrInvalidTTL is returned when the TTL is zero or negative.
	ErrInvalidTTL = errors.New("idem: ttl must be positive")

	// ErrNilKeyHeaderPattern is returned by KeyHeaderPattern when the
	// pattern argument is nil.
	ErrNilKeyHeaderPattern = errors.New("idem: key header pattern must not be nil")

	// ErrInvalidTTLRange is returned by TTLRange when min exceeds max.
	ErrInvalidTTLRange = errors.New("idem: TTLRange min must not exceed max")

	// ErrAllValidatorsFailed is returned by Any when every validator fails.
	ErrAllValidatorsFailed = errors.New("idem: all validators failed")
)
