package idem

import (
	"errors"
	"regexp"
	"testing"
	"time"
)

func TestMaxTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		max     time.Duration
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "exactly at maximum",
			max:     24 * time.Hour,
			ttl:     24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "below maximum",
			max:     24 * time.Hour,
			ttl:     12 * time.Hour,
			wantErr: false,
		},
		{
			name:    "exceeds maximum",
			max:     24 * time.Hour,
			ttl:     48 * time.Hour,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := MaxTTL(tt.max)
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(tt.ttl)})
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxTTL(%v)() error = %v, wantErr %v", tt.max, err, tt.wantErr)
			}
		})
	}
}

func TestMinTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		min     time.Duration
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "exactly at minimum",
			min:     1 * time.Minute,
			ttl:     1 * time.Minute,
			wantErr: false,
		},
		{
			name:    "above minimum",
			min:     1 * time.Minute,
			ttl:     5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "below minimum",
			min:     1 * time.Minute,
			ttl:     30 * time.Second,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := MinTTL(tt.min)
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(tt.ttl)})
			if (err != nil) != tt.wantErr {
				t.Errorf("MinTTL(%v)() error = %v, wantErr %v", tt.min, err, tt.wantErr)
			}
		})
	}
}

func TestTTLRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		min          time.Duration
		max          time.Duration
		ttl          time.Duration
		wantErr      bool
		wantSentinel error
	}{
		{
			name:    "exactly at minimum",
			min:     1 * time.Minute,
			max:     24 * time.Hour,
			ttl:     1 * time.Minute,
			wantErr: false,
		},
		{
			name:    "exactly at maximum",
			min:     1 * time.Minute,
			max:     24 * time.Hour,
			ttl:     24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "within range",
			min:     1 * time.Minute,
			max:     24 * time.Hour,
			ttl:     1 * time.Hour,
			wantErr: false,
		},
		{
			name:    "below minimum",
			min:     1 * time.Minute,
			max:     24 * time.Hour,
			ttl:     30 * time.Second,
			wantErr: true,
		},
		{
			name:    "exceeds maximum",
			min:     1 * time.Minute,
			max:     24 * time.Hour,
			ttl:     48 * time.Hour,
			wantErr: true,
		},
		{
			name:    "min equals max and TTL matches",
			min:     1 * time.Hour,
			max:     1 * time.Hour,
			ttl:     1 * time.Hour,
			wantErr: false,
		},
		{
			name:    "min equals max and TTL does not match",
			min:     1 * time.Hour,
			max:     1 * time.Hour,
			ttl:     2 * time.Hour,
			wantErr: true,
		},
		{
			name:         "min exceeds max",
			min:          24 * time.Hour,
			max:          1 * time.Minute,
			ttl:          1 * time.Hour,
			wantErr:      true,
			wantSentinel: ErrInvalidTTLRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := TTLRange(tt.min, tt.max)
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(tt.ttl)})
			if (err != nil) != tt.wantErr {
				t.Errorf("TTLRange(%v, %v)() error = %v, wantErr %v", tt.min, tt.max, err, tt.wantErr)
			}
			if tt.wantSentinel != nil && !errors.Is(err, tt.wantSentinel) {
				t.Errorf("TTLRange(%v, %v)() error = %v, want %v", tt.min, tt.max, err, tt.wantSentinel)
			}
		})
	}
}

func TestKeyHeaderPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pattern   *regexp.Regexp
		keyHeader string
		wantErr   bool
	}{
		{
			name:      "matches pattern",
			pattern:   regexp.MustCompile(`^X-`),
			keyHeader: "X-Request-Id",
			wantErr:   false,
		},
		{
			name:      "does not match pattern",
			pattern:   regexp.MustCompile(`^X-`),
			keyHeader: "Idempotency-Key",
			wantErr:   true,
		},
		{
			name:    "nil pattern returns error",
			pattern: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := KeyHeaderPattern(tt.pattern)
			err := v.Validate(Config{KeyHeader: tt.keyHeader, TTL: Duration(DefaultTTL)})
			if (err != nil) != tt.wantErr {
				t.Errorf("KeyHeaderPattern(%s)() error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestAllowedKeyHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		allowed   []string
		keyHeader string
		wantErr   bool
	}{
		{
			name:      "header in allowed list",
			allowed:   []string{"Idempotency-Key", "X-Request-Id"},
			keyHeader: "Idempotency-Key",
			wantErr:   false,
		},
		{
			name:      "header not in allowed list",
			allowed:   []string{"Idempotency-Key", "X-Request-Id"},
			keyHeader: "X-Custom-Key",
			wantErr:   true,
		},
		{
			name:      "single allowed header matches",
			allowed:   []string{"X-Request-Id"},
			keyHeader: "X-Request-Id",
			wantErr:   false,
		},
		{
			name:      "single allowed header does not match",
			allowed:   []string{"X-Request-Id"},
			keyHeader: "Idempotency-Key",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := AllowedKeyHeaders(tt.allowed...)
			err := v.Validate(Config{KeyHeader: tt.keyHeader, TTL: Duration(DefaultTTL)})
			if (err != nil) != tt.wantErr {
				t.Errorf("AllowedKeyHeaders(%v)() error = %v, wantErr %v", tt.allowed, err, tt.wantErr)
			}
		})
	}
}

func TestNew_withPresetValidators(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when all preset validators pass", func(t *testing.T) {
		t.Parallel()

		m, err := New(
			WithTTL(1*time.Hour),
			WithValidation(
				MaxTTL(24*time.Hour),
				MinTTL(1*time.Minute),
			),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})

	t.Run("returns error when MaxTTL fails", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithTTL(48*time.Hour),
			WithValidation(MaxTTL(24*time.Hour)),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when MinTTL fails", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithTTL(30*time.Second),
			WithValidation(MinTTL(1*time.Minute)),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when AllowedKeyHeaders fails", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithKeyHeader("X-Custom"),
			WithValidation(AllowedKeyHeaders("Idempotency-Key", "X-Request-Id")),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when KeyHeaderPattern receives nil pattern", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithValidation(KeyHeaderPattern(nil)),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("succeeds when TTLRange passes", func(t *testing.T) {
		t.Parallel()

		m, err := New(
			WithTTL(1*time.Hour),
			WithValidation(TTLRange(1*time.Minute, 24*time.Hour)),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})

	t.Run("returns error when TTLRange fails", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithTTL(48*time.Hour),
			WithValidation(TTLRange(1*time.Minute, 24*time.Hour)),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("combines multiple preset validators", func(t *testing.T) {
		t.Parallel()

		m, err := New(
			WithTTL(1*time.Hour),
			WithKeyHeader("Idempotency-Key"),
			WithValidation(
				MaxTTL(24*time.Hour),
				MinTTL(1*time.Minute),
				AllowedKeyHeaders("Idempotency-Key", "X-Request-Id"),
			),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})
}

func TestPresetValidator_WithMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		v       *PresetValidator
		cfg     Config
		wantMsg string
	}{
		{
			name:    "MaxTTL with custom message",
			v:       MaxTTL(1 * time.Hour).WithMessage("TTL is too long for this service"),
			cfg:     Config{TTL: Duration(2 * time.Hour)},
			wantMsg: "TTL is too long for this service",
		},
		{
			name:    "MinTTL with custom message",
			v:       MinTTL(1 * time.Hour).WithMessage("TTL is too short"),
			cfg:     Config{TTL: Duration(30 * time.Minute)},
			wantMsg: "TTL is too short",
		},
		{
			name:    "TTLRange with custom message",
			v:       TTLRange(1*time.Minute, 1*time.Hour).WithMessage("TTL out of acceptable range"),
			cfg:     Config{TTL: Duration(2 * time.Hour)},
			wantMsg: "TTL out of acceptable range",
		},
		{
			name:    "KeyHeaderPattern with custom message",
			v:       KeyHeaderPattern(regexp.MustCompile(`^X-`)).WithMessage("header must start with X-"),
			cfg:     Config{KeyHeader: "Idempotency-Key", TTL: Duration(DefaultTTL)},
			wantMsg: "header must start with X-",
		},
		{
			name:    "AllowedKeyHeaders with custom message",
			v:       AllowedKeyHeaders("Idempotency-Key").WithMessage("unsupported header"),
			cfg:     Config{KeyHeader: "X-Custom", TTL: Duration(DefaultTTL)},
			wantMsg: "unsupported header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.v.Validate(tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestPresetValidator_WithMessage_defaultMessageWithoutWithMessage(t *testing.T) {
	t.Parallel()

	v := MaxTTL(1 * time.Hour)
	err := v.Validate(Config{TTL: Duration(2 * time.Hour)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "idem: ttl 2h0m0s exceeds maximum 1h0m0s" {
		t.Errorf("unexpected default message: %q", err.Error())
	}
}

func TestPresetValidator_WithMessage_nilOnSuccess(t *testing.T) {
	t.Parallel()

	v := MaxTTL(1 * time.Hour).WithMessage("custom error")
	err := v.Validate(Config{TTL: Duration(30 * time.Minute)})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestPresetValidator_WithMessage_doesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	original := MaxTTL(1 * time.Hour)
	_ = original.WithMessage("custom error")

	err := original.Validate(Config{TTL: Duration(2 * time.Hour)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "custom error" {
		t.Error("WithMessage mutated the original validator")
	}
}

func TestAll(t *testing.T) {
	t.Parallel()

	alwaysPass := ValidatorFunc(func(_ Config) error { return nil })
	alwaysFail := ValidatorFunc(func(_ Config) error { return errors.New("fail") })
	alwaysFail2 := ValidatorFunc(func(_ Config) error { return errors.New("fail2") })

	tests := []struct {
		name       string
		validators []Validator
		wantErr    bool
		wantMsg    string
	}{
		{
			name:       "all validators pass",
			validators: []Validator{alwaysPass, alwaysPass},
			wantErr:    false,
		},
		{
			name:       "first validator fails",
			validators: []Validator{alwaysFail, alwaysPass},
			wantErr:    true,
			wantMsg:    "fail",
		},
		{
			name:       "second validator fails",
			validators: []Validator{alwaysPass, alwaysFail2},
			wantErr:    true,
			wantMsg:    "fail2",
		},
		{
			name:       "no validators",
			validators: []Validator{},
			wantErr:    false,
		},
		{
			name:       "single passing validator",
			validators: []Validator{alwaysPass},
			wantErr:    false,
		},
		{
			name:       "single failing validator",
			validators: []Validator{alwaysFail},
			wantErr:    true,
			wantMsg:    "fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := All(tt.validators...)
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
			if (err != nil) != tt.wantErr {
				t.Errorf("All() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantMsg != "" && err != nil && err.Error() != tt.wantMsg {
				t.Errorf("All() error = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestAll_WithMessage(t *testing.T) {
	t.Parallel()

	alwaysFail := ValidatorFunc(func(_ Config) error { return errors.New("fail") })

	v := All(alwaysFail).WithMessage("custom all error")
	err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "custom all error" {
		t.Errorf("error = %q, want %q", err.Error(), "custom all error")
	}
}

func TestAny(t *testing.T) {
	t.Parallel()

	alwaysPass := ValidatorFunc(func(_ Config) error { return nil })
	alwaysFail := ValidatorFunc(func(_ Config) error { return errors.New("fail") })

	tests := []struct {
		name         string
		validators   []Validator
		wantErr      bool
		wantSentinel error
	}{
		{
			name:       "first validator passes",
			validators: []Validator{alwaysPass, alwaysFail},
			wantErr:    false,
		},
		{
			name:       "last validator passes",
			validators: []Validator{alwaysFail, alwaysPass},
			wantErr:    false,
		},
		{
			name:       "all pass",
			validators: []Validator{alwaysPass, alwaysPass},
			wantErr:    false,
		},
		{
			name:         "all fail",
			validators:   []Validator{alwaysFail, alwaysFail},
			wantErr:      true,
			wantSentinel: ErrAllValidatorsFailed,
		},
		{
			name:         "no validators",
			validators:   []Validator{},
			wantErr:      true,
			wantSentinel: ErrAllValidatorsFailed,
		},
		{
			name:       "single passing validator",
			validators: []Validator{alwaysPass},
			wantErr:    false,
		},
		{
			name:         "single failing validator",
			validators:   []Validator{alwaysFail},
			wantErr:      true,
			wantSentinel: ErrAllValidatorsFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := Any(tt.validators...)
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
			if (err != nil) != tt.wantErr {
				t.Errorf("Any() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantSentinel != nil && !errors.Is(err, tt.wantSentinel) {
				t.Errorf("Any() error = %v, want %v", err, tt.wantSentinel)
			}
		})
	}
}

func TestAny_WithMessage(t *testing.T) {
	t.Parallel()

	alwaysFail := ValidatorFunc(func(_ Config) error { return errors.New("fail") })

	v := Any(alwaysFail).WithMessage("custom any error")
	err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "custom any error" {
		t.Errorf("error = %q, want %q", err.Error(), "custom any error")
	}
}

func TestAll_Any_nested(t *testing.T) {
	t.Parallel()

	alwaysPass := ValidatorFunc(func(_ Config) error { return nil })
	alwaysFail := ValidatorFunc(func(_ Config) error { return errors.New("fail") })

	t.Run("All with nested Any succeeds", func(t *testing.T) {
		t.Parallel()

		v := All(Any(alwaysFail, alwaysPass), alwaysPass)
		err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("All with nested Any fails when Any fails", func(t *testing.T) {
		t.Parallel()

		v := All(Any(alwaysFail, alwaysFail), alwaysPass)
		err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Any with nested All succeeds", func(t *testing.T) {
		t.Parallel()

		v := Any(All(alwaysFail, alwaysPass), alwaysPass)
		err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Any with nested All fails when all fail", func(t *testing.T) {
		t.Parallel()

		v := Any(All(alwaysFail, alwaysPass), alwaysFail)
		err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAll_WithMessage_doesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	alwaysFail := ValidatorFunc(func(_ Config) error { return errors.New("original") })
	original := All(alwaysFail)
	_ = original.WithMessage("custom")

	err := original.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "custom" {
		t.Error("WithMessage mutated the original validator")
	}
}

func TestAny_WithMessage_doesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	alwaysFail := ValidatorFunc(func(_ Config) error { return errors.New("fail") })
	original := Any(alwaysFail)
	_ = original.WithMessage("custom")

	err := original.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: Duration(DefaultTTL)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "custom" {
		t.Error("WithMessage mutated the original validator")
	}
}

func TestNew_withAllAndAny(t *testing.T) {
	t.Parallel()

	t.Run("succeeds with All validator", func(t *testing.T) {
		t.Parallel()

		m, err := New(
			WithTTL(1*time.Hour),
			WithValidation(
				All(MaxTTL(24*time.Hour), MinTTL(1*time.Minute)),
			),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})

	t.Run("fails with All validator", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithTTL(48*time.Hour),
			WithValidation(
				All(MaxTTL(24*time.Hour), MinTTL(1*time.Minute)),
			),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("succeeds with Any validator", func(t *testing.T) {
		t.Parallel()

		m, err := New(
			WithTTL(30*time.Second),
			WithValidation(
				Any(MinTTL(1*time.Minute), MaxTTL(1*time.Hour)),
			),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})

	t.Run("fails with Any validator", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithTTL(48*time.Hour),
			WithValidation(
				Any(MaxTTL(24*time.Hour), MaxTTL(12*time.Hour)),
			),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestNew_withPresetValidatorWithMessage(t *testing.T) {
	t.Parallel()

	t.Run("returns custom error message", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			WithTTL(48*time.Hour),
			WithValidation(MaxTTL(24*time.Hour).WithMessage("TTL exceeds limit")),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "TTL exceeds limit" {
			t.Errorf("error = %q, want %q", err.Error(), "TTL exceeds limit")
		}
	})

	t.Run("succeeds when validation passes with WithMessage set", func(t *testing.T) {
		t.Parallel()

		m, err := New(
			WithTTL(1*time.Hour),
			WithValidation(MaxTTL(24*time.Hour).WithMessage("TTL exceeds limit")),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})
}
