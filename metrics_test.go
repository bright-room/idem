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
}
