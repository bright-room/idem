package idem

import (
	"context"
	"testing"
	"time"
)

type stubStorage struct{}

func (s *stubStorage) Get(_ context.Context, _ string) (*Response, error) { return nil, nil }

func (s *stubStorage) Set(_ context.Context, _ string, _ *Response, _ time.Duration) error {
	return nil
}

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
