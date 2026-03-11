package idem

import (
	"context"
	"sync"
	"sync/atomic"
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

	t.Run("defaultStorage implements Locker", func(t *testing.T) {
		t.Parallel()

		mw := New()
		if _, ok := mw.cfg.storage.(Locker); !ok {
			t.Error("defaultStorage does not implement Locker")
		}
	})
}

func TestDefaultStorage_Lock(t *testing.T) {
	t.Parallel()

	t.Run("acquires and releases lock", func(t *testing.T) {
		t.Parallel()

		s := newDefaultStorage()
		ctx := context.Background()

		unlock, err := s.Lock(ctx, "key-1", time.Hour)
		if err != nil {
			t.Fatalf("Lock() error = %v", err)
		}

		unlock()
	})

	t.Run("provides mutual exclusion for same key", func(t *testing.T) {
		t.Parallel()

		s := newDefaultStorage()
		ctx := context.Background()

		var concurrent atomic.Int32
		var maxConcurrent atomic.Int32

		var wg sync.WaitGroup
		const goroutines = 10

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				unlock, err := s.Lock(ctx, "shared-key", time.Hour)
				if err != nil {
					t.Errorf("Lock() error = %v", err)
					return
				}

				cur := concurrent.Add(1)
				if cur > maxConcurrent.Load() {
					maxConcurrent.Store(cur)
				}
				time.Sleep(time.Millisecond)
				concurrent.Add(-1)

				unlock()
			}()
		}

		wg.Wait()

		if max := maxConcurrent.Load(); max != 1 {
			t.Errorf("max concurrent locks = %d, want 1", max)
		}
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		t.Parallel()

		s := newDefaultStorage()
		ctx := context.Background()

		// Hold the lock
		unlock, err := s.Lock(ctx, "cancel-key", time.Hour)
		if err != nil {
			t.Fatalf("Lock() error = %v", err)
		}
		defer unlock()

		// Try to acquire with cancelled context
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err = s.Lock(cancelCtx, "cancel-key", time.Hour)
		if err == nil {
			t.Fatal("Lock() error = nil, want context error")
		}

		if err != context.Canceled {
			t.Errorf("Lock() error = %v, want %v", err, context.Canceled)
		}
	})
}
