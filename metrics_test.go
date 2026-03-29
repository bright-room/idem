package idem

import "testing"

func TestWithMetrics(t *testing.T) {
	t.Parallel()

	t.Run("sets metrics on config", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()

		called := false
		WithMetrics(Metrics{
			OnCacheHit: func(_ string) { called = true },
		})(cfg)

		if cfg.metrics == nil {
			t.Fatal("metrics = nil, want non-nil")
		}

		cfg.metrics.OnCacheHit("test")

		if !called {
			t.Error("OnCacheHit callback was not called")
		}
	})

	t.Run("sets metrics with zero-value Metrics", func(t *testing.T) {
		t.Parallel()

		cfg := defaultConfig()
		WithMetrics(Metrics{})(cfg)

		if cfg.metrics == nil {
			t.Error("metrics = nil, want non-nil even for zero-value Metrics")
		}
	})

	t.Run("OnCacheSkip callback is invoked", func(t *testing.T) {
		t.Parallel()

		var gotKey string
		var gotCode int
		cfg := defaultConfig()
		WithMetrics(Metrics{
			OnCacheSkip: func(key string, statusCode int) {
				gotKey = key
				gotCode = statusCode
			},
		})(cfg)

		cfg.metrics.OnCacheSkip("test-key", 500)

		if gotKey != "test-key" {
			t.Errorf("OnCacheSkip key = %q, want %q", gotKey, "test-key")
		}

		if gotCode != 500 {
			t.Errorf("OnCacheSkip statusCode = %d, want %d", gotCode, 500)
		}
	})
}
