package idem

import (
	"fmt"
	"regexp"
	"time"
)

// MaxTTL returns a Validator that rejects a TTL longer than max.
func MaxTTL(limit time.Duration) Validator {
	return func(cfg Config) error {
		if cfg.TTL > limit {
			return fmt.Errorf("idem: ttl %v exceeds maximum %v", cfg.TTL, limit)
		}
		return nil
	}
}

// MinTTL returns a Validator that rejects a TTL shorter than min.
func MinTTL(limit time.Duration) Validator {
	return func(cfg Config) error {
		if cfg.TTL < limit {
			return fmt.Errorf("idem: ttl %v is shorter than minimum %v", cfg.TTL, limit)
		}
		return nil
	}
}

// KeyHeaderPattern returns a Validator that requires the key header
// name to match the given regular expression pattern.
func KeyHeaderPattern(pattern *regexp.Regexp) Validator {
	return func(cfg Config) error {
		if pattern == nil {
			return ErrNilKeyHeaderPattern
		}
		if !pattern.MatchString(cfg.KeyHeader) {
			return fmt.Errorf("idem: key header %q does not match pattern %s", cfg.KeyHeader, pattern)
		}
		return nil
	}
}

// AllowedKeyHeaders returns a Validator that requires the key header
// name to be one of the specified allowed values.
func AllowedKeyHeaders(allowed ...string) Validator {
	return func(cfg Config) error {
		for _, a := range allowed {
			if cfg.KeyHeader == a {
				return nil
			}
		}
		return fmt.Errorf("idem: key header %q is not in allowed list %v", cfg.KeyHeader, allowed)
	}
}
