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
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: tt.ttl})
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
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: tt.ttl})
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
			err := v.Validate(Config{KeyHeader: DefaultKeyHeader, TTL: tt.ttl})
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
			err := v.Validate(Config{KeyHeader: tt.keyHeader, TTL: DefaultTTL})
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
			err := v.Validate(Config{KeyHeader: tt.keyHeader, TTL: DefaultTTL})
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
			cfg:     Config{TTL: 2 * time.Hour},
			wantMsg: "TTL is too long for this service",
		},
		{
			name:    "MinTTL with custom message",
			v:       MinTTL(1 * time.Hour).WithMessage("TTL is too short"),
			cfg:     Config{TTL: 30 * time.Minute},
			wantMsg: "TTL is too short",
		},
		{
			name:    "TTLRange with custom message",
			v:       TTLRange(1*time.Minute, 1*time.Hour).WithMessage("TTL out of acceptable range"),
			cfg:     Config{TTL: 2 * time.Hour},
			wantMsg: "TTL out of acceptable range",
		},
		{
			name:    "KeyHeaderPattern with custom message",
			v:       KeyHeaderPattern(regexp.MustCompile(`^X-`)).WithMessage("header must start with X-"),
			cfg:     Config{KeyHeader: "Idempotency-Key", TTL: DefaultTTL},
			wantMsg: "header must start with X-",
		},
		{
			name:    "AllowedKeyHeaders with custom message",
			v:       AllowedKeyHeaders("Idempotency-Key").WithMessage("unsupported header"),
			cfg:     Config{KeyHeader: "X-Custom", TTL: DefaultTTL},
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
	err := v.Validate(Config{TTL: 2 * time.Hour})
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
	err := v.Validate(Config{TTL: 30 * time.Minute})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestPresetValidator_WithMessage_doesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	original := MaxTTL(1 * time.Hour)
	_ = original.WithMessage("custom error")

	err := original.Validate(Config{TTL: 2 * time.Hour})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "custom error" {
		t.Error("WithMessage mutated the original validator")
	}
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
