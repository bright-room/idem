package idem

import (
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
			err := v(Config{KeyHeader: DefaultKeyHeader, TTL: tt.ttl})
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
			err := v(Config{KeyHeader: DefaultKeyHeader, TTL: tt.ttl})
			if (err != nil) != tt.wantErr {
				t.Errorf("MinTTL(%v)() error = %v, wantErr %v", tt.min, err, tt.wantErr)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := KeyHeaderPattern(tt.pattern)
			err := v(Config{KeyHeader: tt.keyHeader, TTL: DefaultTTL})
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
			err := v(Config{KeyHeader: tt.keyHeader, TTL: DefaultTTL})
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
