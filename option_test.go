package idem

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type stubStorage struct{}

func (s *stubStorage) Get(_ context.Context, _ string) (*Response, error) { return nil, nil }

func (s *stubStorage) Set(_ context.Context, _ string, _ *Response, _ time.Duration) error {
	return nil
}

func (s *stubStorage) Delete(_ context.Context, _ string) error { return nil }

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	if cfg.keyHeader != DefaultKeyHeader {
		t.Errorf("keyHeader = %q, want %q", cfg.keyHeader, DefaultKeyHeader)
	}

	if cfg.ttl != DefaultTTL {
		t.Errorf("ttl = %v, want %v", cfg.ttl, DefaultTTL)
	}

	if cfg.storage != nil {
		t.Errorf("storage = %v, want nil", cfg.storage)
	}

	if cfg.onError != nil {
		t.Error("onError is non-nil, want nil")
	}

	if cfg.cacheable == nil {
		t.Error("cacheable = nil, want non-nil (DefaultCacheable)")
	}
}

func TestDefaultCacheable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{name: "100 is cacheable", statusCode: 100, want: true},
		{name: "200 is cacheable", statusCode: 200, want: true},
		{name: "201 is cacheable", statusCode: 201, want: true},
		{name: "301 is cacheable", statusCode: 301, want: true},
		{name: "400 is cacheable", statusCode: 400, want: true},
		{name: "404 is cacheable", statusCode: 404, want: true},
		{name: "499 is cacheable", statusCode: 499, want: true},
		{name: "500 is not cacheable", statusCode: 500, want: false},
		{name: "502 is not cacheable", statusCode: 502, want: false},
		{name: "503 is not cacheable", statusCode: 503, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DefaultCacheable(tt.statusCode); got != tt.want {
				t.Errorf("DefaultCacheable(%d) = %t, want %t", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestWithCacheable(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	custom := CacheableFunc(func(statusCode int) bool {
		return statusCode == 200
	})
	WithCacheable(custom)(cfg)

	if cfg.cacheable == nil {
		t.Fatal("cacheable = nil, want non-nil")
	}

	if !cfg.cacheable(200) {
		t.Error("cacheable(200) = false, want true")
	}

	if cfg.cacheable(201) {
		t.Error("cacheable(201) = true, want false")
	}
}

func TestWithCacheable_nil_falls_back_to_default(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithCacheable(nil)(cfg)

	if cfg.cacheable == nil {
		t.Fatal("cacheable = nil after WithCacheable(nil), want DefaultCacheable")
	}

	if !cfg.cacheable(200) {
		t.Error("cacheable(200) = false, want true (DefaultCacheable)")
	}

	if cfg.cacheable(500) {
		t.Error("cacheable(500) = true, want false (DefaultCacheable)")
	}
}

func TestWithKeyHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "sets custom header name",
			header: "X-Request-Id",
		},
		{
			name:   "sets another custom header name",
			header: "X-Idempotency-Token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaultConfig()
			WithKeyHeader(tt.header)(cfg)

			if cfg.keyHeader != tt.header {
				t.Errorf("keyHeader = %q, want %q", cfg.keyHeader, tt.header)
			}
		})
	}
}

func TestWithTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "sets short TTL",
			ttl:  5 * time.Minute,
		},
		{
			name: "sets long TTL",
			ttl:  72 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaultConfig()
			WithTTL(tt.ttl)(cfg)

			if cfg.ttl != tt.ttl {
				t.Errorf("ttl = %v, want %v", cfg.ttl, tt.ttl)
			}
		})
	}
}

func TestWithStorage(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	stub := &stubStorage{}
	WithStorage(stub)(cfg)

	if cfg.storage != stub {
		t.Errorf("storage = %v, want %v", cfg.storage, stub)
	}
}

func TestWithOnError(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	called := false
	fn := func(_ string, _ error) { called = true }
	WithOnError(fn)(cfg)

	if cfg.onError == nil {
		t.Fatal("onError = nil, want non-nil")
	}

	cfg.onError("test-key", nil)

	if !called {
		t.Error("callback was not called")
	}
}

func TestConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     defaultConfig(),
			wantErr: false,
		},
		{
			name: "valid custom config",
			cfg: &config{
				keyHeader: "X-Request-Id",
				ttl:       5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "empty keyHeader",
			cfg: &config{
				keyHeader: "",
				ttl:       DefaultTTL,
			},
			wantErr: true,
		},
		{
			name: "zero ttl",
			cfg: &config{
				keyHeader: DefaultKeyHeader,
				ttl:       0,
			},
			wantErr: true,
		},
		{
			name: "negative ttl",
			cfg: &config{
				keyHeader: DefaultKeyHeader,
				ttl:       -time.Second,
			},
			wantErr: true,
		},
		{
			name: "custom validator passes",
			cfg: func() *config {
				c := defaultConfig()
				WithValidation(ValidatorFunc(func(_ Config) error {
					return nil
				}))(c)
				return c
			}(),
			wantErr: false,
		},
		{
			name: "custom validator fails",
			cfg: func() *config {
				c := defaultConfig()
				WithValidation(ValidatorFunc(func(_ Config) error {
					return errors.New("custom: validation failed")
				}))(c)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "multiple validators stop at first error",
			cfg: func() *config {
				c := defaultConfig()
				second := false
				WithValidation(
					ValidatorFunc(func(_ Config) error {
						return errors.New("first fails")
					}),
					ValidatorFunc(func(_ Config) error {
						second = true
						_ = second
						return nil
					}),
				)(c)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "custom validator receives config snapshot",
			cfg: func() *config {
				c := &config{
					keyHeader: "X-Test",
					ttl:       5 * time.Minute,
				}
				WithValidation(ValidatorFunc(func(cfg Config) error {
					if cfg.KeyHeader != "X-Test" {
						return errors.New("unexpected KeyHeader")
					}
					if cfg.TTL != Duration(5*time.Minute) {
						return errors.New("unexpected TTL")
					}
					return nil
				}))(c)
				return c
			}(),
			wantErr: false,
		},
		{
			name: "built-in validation runs before custom validators",
			cfg: func() *config {
				c := &config{
					keyHeader: "",
					ttl:       DefaultTTL,
				}
				WithValidation(ValidatorFunc(func(_ Config) error {
					return errors.New("should not reach here")
				}))(c)
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWithValidation(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	called := false
	v := ValidatorFunc(func(_ Config) error {
		called = true
		return nil
	})
	WithValidation(v)(cfg)

	if len(cfg.validators) != 1 {
		t.Fatalf("validators length = %d, want 1", len(cfg.validators))
	}

	if err := cfg.validators[0].Validate(cfg.snapshot()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("validator was not called")
	}
}

func TestConfig_snapshot(t *testing.T) {
	t.Parallel()

	t.Run("basic fields", func(t *testing.T) {
		t.Parallel()

		cfg := &config{
			keyHeader: "X-Custom-Key",
			ttl:       10 * time.Minute,
		}

		snap := cfg.snapshot()

		if snap.KeyHeader != cfg.keyHeader {
			t.Errorf("KeyHeader = %q, want %q", snap.KeyHeader, cfg.keyHeader)
		}
		if snap.TTL != Duration(cfg.ttl) {
			t.Errorf("TTL = %v, want %v", snap.TTL, cfg.ttl)
		}
	})

	t.Run("StorageType is empty when storage is nil", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		snap := cfg.snapshot()

		if snap.StorageType != "" {
			t.Errorf("StorageType = %q, want empty", snap.StorageType)
		}
	})

	t.Run("StorageType reflects MemoryStorage type name", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		cfg.storage = NewMemoryStorage()
		snap := cfg.snapshot()

		if snap.StorageType != "*idem.MemoryStorage" {
			t.Errorf("StorageType = %q, want %q", snap.StorageType, "*idem.MemoryStorage")
		}
	})

	t.Run("LockSupported is true when storage implements Locker", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		cfg.storage = NewMemoryStorage()
		snap := cfg.snapshot()

		if !snap.LockSupported {
			t.Error("LockSupported = false, want true")
		}
	})

	t.Run("LockSupported is false when storage does not implement Locker", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		cfg.storage = &stubStorage{}
		snap := cfg.snapshot()

		if snap.LockSupported {
			t.Error("LockSupported = true, want false")
		}
	})

	t.Run("CacheableEnabled reflects cacheable presence", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		if !cfg.snapshot().CacheableEnabled {
			t.Error("CacheableEnabled = false with default cacheable, want true")
		}

		cfg.cacheable = nil
		if cfg.snapshot().CacheableEnabled {
			t.Error("CacheableEnabled = true without cacheable, want false")
		}
	})

	t.Run("MetricsEnabled reflects metrics presence", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		if cfg.snapshot().MetricsEnabled {
			t.Error("MetricsEnabled = true without metrics, want false")
		}

		WithMetrics(Metrics{})(cfg)
		if !cfg.snapshot().MetricsEnabled {
			t.Error("MetricsEnabled = false with metrics, want true")
		}
	})

	t.Run("OnErrorEnabled reflects onError presence", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		if cfg.snapshot().OnErrorEnabled {
			t.Error("OnErrorEnabled = true without onError, want false")
		}

		WithOnError(func(_ string, _ error) {})(cfg)
		if !cfg.snapshot().OnErrorEnabled {
			t.Error("OnErrorEnabled = false with onError, want true")
		}
	})

	t.Run("ValidatorCount reflects registered validators", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		if cfg.snapshot().ValidatorCount != 0 {
			t.Errorf("ValidatorCount = %d, want 0", cfg.snapshot().ValidatorCount)
		}

		WithValidation(
			ValidatorFunc(func(_ Config) error { return nil }),
			ValidatorFunc(func(_ Config) error { return nil }),
		)(cfg)

		if cfg.snapshot().ValidatorCount != 2 {
			t.Errorf("ValidatorCount = %d, want 2", cfg.snapshot().ValidatorCount)
		}
	})
}

func TestConfig_String(t *testing.T) {
	t.Parallel()

	t.Run("contains all key fields", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			KeyHeader:        "Idempotency-Key",
			TTL:              Duration(24 * time.Hour),
			StorageType:      "*idem.MemoryStorage",
			LockSupported:    true,
			CacheableEnabled: true,
			MetricsEnabled:   true,
			OnErrorEnabled:   true,
			ValidatorCount:   3,
		}

		s := cfg.String()
		for _, want := range []string{
			"Idempotency-Key", "24h0m0s", "*idem.MemoryStorage",
			"LockSupported: true", "CacheableEnabled: true", "MetricsEnabled: true",
			"OnErrorEnabled: true", "ValidatorCount: 3",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("String() = %q, missing %q", s, want)
			}
		}
	})

	t.Run("does not panic on zero value", func(t *testing.T) {
		t.Parallel()

		var cfg Config
		_ = cfg.String()
	})
}

func TestNew_withCustomValidation(t *testing.T) {
	t.Parallel()

	t.Run("returns error from custom validator", func(t *testing.T) {
		t.Parallel()

		_, err := New(WithValidation(ValidatorFunc(func(_ Config) error {
			return errors.New("custom: not allowed")
		})))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("succeeds with passing custom validator", func(t *testing.T) {
		t.Parallel()

		m, err := New(WithValidation(ValidatorFunc(func(_ Config) error {
			return nil
		})))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})
}
