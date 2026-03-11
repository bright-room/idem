package idem

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("applies default config when no options given", func(t *testing.T) {
		t.Parallel()

		mw := New()

		if mw.cfg.keyHeader != DefaultKeyHeader {
			t.Errorf("keyHeader = %q, want %q", mw.cfg.keyHeader, DefaultKeyHeader)
		}

		if mw.cfg.ttl != DefaultTTL {
			t.Errorf("ttl = %v, want %v", mw.cfg.ttl, DefaultTTL)
		}

		if _, ok := mw.cfg.storage.(*defaultStorage); !ok {
			t.Errorf("storage type = %T, want *defaultStorage", mw.cfg.storage)
		}
	})

	t.Run("applies options", func(t *testing.T) {
		t.Parallel()

		custom := &stubStorage{}
		mw := New(
			WithKeyHeader("X-Custom-Key"),
			WithTTL(5*time.Minute),
			WithStorage(custom),
		)

		if mw.cfg.keyHeader != "X-Custom-Key" {
			t.Errorf("keyHeader = %q, want %q", mw.cfg.keyHeader, "X-Custom-Key")
		}

		if mw.cfg.ttl != 5*time.Minute {
			t.Errorf("ttl = %v, want %v", mw.cfg.ttl, 5*time.Minute)
		}

		if mw.cfg.storage != custom {
			t.Errorf("storage = %v, want %v", mw.cfg.storage, custom)
		}
	})
}
