package idem

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// PresetValidator is a Validator returned by the built-in validator
// constructors (MaxTTL, MinTTL, etc.). It supports custom error messages
// via the WithMessage method.
type PresetValidator struct {
	validate func(Config) error
	msg      string
}

// Validate runs the validation logic. If a custom message has been set via
// WithMessage and the underlying check fails, the custom message is returned
// instead of the default error.
func (p *PresetValidator) Validate(cfg Config) error {
	if err := p.validate(cfg); err != nil {
		if p.msg != "" {
			return errors.New(p.msg)
		}
		return err
	}
	return nil
}

// WithMessage returns a copy of the PresetValidator that uses msg as the
// error message when validation fails.
func (p *PresetValidator) WithMessage(msg string) *PresetValidator {
	return &PresetValidator{
		validate: p.validate,
		msg:      msg,
	}
}

// MaxTTL returns a Validator that rejects a TTL longer than max.
func MaxTTL(limit time.Duration) *PresetValidator {
	return &PresetValidator{
		validate: func(cfg Config) error {
			if cfg.TTL > limit {
				return fmt.Errorf("idem: ttl %v exceeds maximum %v", cfg.TTL, limit)
			}
			return nil
		},
	}
}

// MinTTL returns a Validator that rejects a TTL shorter than min.
func MinTTL(limit time.Duration) *PresetValidator {
	return &PresetValidator{
		validate: func(cfg Config) error {
			if cfg.TTL < limit {
				return fmt.Errorf("idem: ttl %v is shorter than minimum %v", cfg.TTL, limit)
			}
			return nil
		},
	}
}

// TTLRange returns a Validator that rejects a TTL outside the [lower, upper] range.
func TTLRange(lower, upper time.Duration) *PresetValidator {
	return &PresetValidator{
		validate: func(cfg Config) error {
			if lower > upper {
				return ErrInvalidTTLRange
			}
			if cfg.TTL < lower || cfg.TTL > upper {
				return fmt.Errorf("idem: ttl %v is out of range [%v, %v]", cfg.TTL, lower, upper)
			}
			return nil
		},
	}
}

// KeyHeaderPattern returns a Validator that requires the key header
// name to match the given regular expression pattern.
func KeyHeaderPattern(pattern *regexp.Regexp) *PresetValidator {
	return &PresetValidator{
		validate: func(cfg Config) error {
			if pattern == nil {
				return ErrNilKeyHeaderPattern
			}
			if !pattern.MatchString(cfg.KeyHeader) {
				return fmt.Errorf("idem: key header %q does not match pattern %s", cfg.KeyHeader, pattern)
			}
			return nil
		},
	}
}

// AllowedKeyHeaders returns a Validator that requires the key header
// name to be one of the specified allowed values.
func AllowedKeyHeaders(allowed ...string) *PresetValidator {
	return &PresetValidator{
		validate: func(cfg Config) error {
			for _, a := range allowed {
				if cfg.KeyHeader == a {
					return nil
				}
			}
			return fmt.Errorf("idem: key header %q is not in allowed list %v", cfg.KeyHeader, allowed)
		},
	}
}
