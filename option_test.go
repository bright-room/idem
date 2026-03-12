package idem

import (
	"context"
	"errors"
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
	fn := func(_ error) { called = true }
	WithOnError(fn)(cfg)

	if cfg.onError == nil {
		t.Fatal("onError = nil, want non-nil")
	}

	cfg.onError(nil)

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
				WithValidation(func(_ Config) error {
					return nil
				})(c)
				return c
			}(),
			wantErr: false,
		},
		{
			name: "custom validator fails",
			cfg: func() *config {
				c := defaultConfig()
				WithValidation(func(_ Config) error {
					return errors.New("custom: validation failed")
				})(c)
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
					func(_ Config) error {
						return errors.New("first fails")
					},
					func(_ Config) error {
						second = true
						_ = second
						return nil
					},
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
				WithValidation(func(cfg Config) error {
					if cfg.KeyHeader != "X-Test" {
						return errors.New("unexpected KeyHeader")
					}
					if cfg.TTL != 5*time.Minute {
						return errors.New("unexpected TTL")
					}
					return nil
				})(c)
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
				WithValidation(func(_ Config) error {
					return errors.New("should not reach here")
				})(c)
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
	v := func(_ Config) error {
		called = true
		return nil
	}
	WithValidation(v)(cfg)

	if len(cfg.validators) != 1 {
		t.Fatalf("validators length = %d, want 1", len(cfg.validators))
	}

	if err := cfg.validators[0](cfg.snapshot()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("validator was not called")
	}
}

func TestConfig_snapshot(t *testing.T) {
	t.Parallel()

	cfg := &config{
		keyHeader: "X-Custom-Key",
		ttl:       10 * time.Minute,
	}

	snap := cfg.snapshot()

	if snap.KeyHeader != cfg.keyHeader {
		t.Errorf("KeyHeader = %q, want %q", snap.KeyHeader, cfg.keyHeader)
	}
	if snap.TTL != cfg.ttl {
		t.Errorf("TTL = %v, want %v", snap.TTL, cfg.ttl)
	}
}

func TestNew_withCustomValidation(t *testing.T) {
	t.Parallel()

	t.Run("returns error from custom validator", func(t *testing.T) {
		t.Parallel()

		_, err := New(WithValidation(func(_ Config) error {
			return errors.New("custom: not allowed")
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("succeeds with passing custom validator", func(t *testing.T) {
		t.Parallel()

		m, err := New(WithValidation(func(_ Config) error {
			return nil
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("middleware is nil")
		}
	})
}
