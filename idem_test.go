package idem

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("applies default config when no options given", func(t *testing.T) {
		t.Parallel()

		mw, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if mw.cfg.keyHeader != DefaultKeyHeader {
			t.Errorf("keyHeader = %q, want %q", mw.cfg.keyHeader, DefaultKeyHeader)
		}

		if mw.cfg.ttl != DefaultTTL {
			t.Errorf("ttl = %v, want %v", mw.cfg.ttl, DefaultTTL)
		}

		if _, ok := mw.cfg.storage.(*MemoryStorage); !ok {
			t.Errorf("storage type = %T, want *MemoryStorage", mw.cfg.storage)
		}
	})

	t.Run("applies options", func(t *testing.T) {
		t.Parallel()

		custom := &stubStorage{}
		mw, err := New(
			WithKeyHeader("X-Custom-Key"),
			WithTTL(5*time.Minute),
			WithStorage(custom),
		)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

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

	t.Run("MemoryStorage implements Locker", func(t *testing.T) {
		t.Parallel()

		mw, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if _, ok := mw.cfg.storage.(Locker); !ok {
			t.Error("MemoryStorage does not implement Locker")
		}
	})

	t.Run("returns error for empty keyHeader", func(t *testing.T) {
		t.Parallel()

		_, err := New(WithKeyHeader(""))
		if err == nil {
			t.Fatal("New() error = nil, want error")
		}
	})

	t.Run("returns error for zero ttl", func(t *testing.T) {
		t.Parallel()

		_, err := New(WithTTL(0))
		if err == nil {
			t.Fatal("New() error = nil, want error")
		}
	})

	t.Run("returns error for negative ttl", func(t *testing.T) {
		t.Parallel()

		_, err := New(WithTTL(-time.Second))
		if err == nil {
			t.Fatal("New() error = nil, want error")
		}
	})
}

func TestMemoryStorage_Get(t *testing.T) {
	t.Parallel()

	res := &Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(`{"ok":true}`),
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T, s *MemoryStorage)
		key     string
		wantRes *Response
	}{
		{
			name: "returns cached response for existing key",
			setup: func(t *testing.T, s *MemoryStorage) {
				t.Helper()
				if err := s.Set(context.Background(), "key-1", res, time.Hour); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
			},
			key:     "key-1",
			wantRes: res,
		},
		{
			name:    "returns nil for non-existent key",
			setup:   func(t *testing.T, _ *MemoryStorage) { t.Helper() },
			key:     "unknown",
			wantRes: nil,
		},
		{
			name: "returns nil after TTL has expired",
			setup: func(t *testing.T, s *MemoryStorage) {
				t.Helper()
				if err := s.Set(context.Background(), "expired", res, time.Nanosecond); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
				time.Sleep(time.Millisecond)
			},
			key:     "expired",
			wantRes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewMemoryStorage()
			tt.setup(t, s)

			got, err := s.Get(context.Background(), tt.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if tt.wantRes == nil {
				if got != nil {
					t.Errorf("Get() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("Get() = nil, want non-nil")
			}
			if got.StatusCode != tt.wantRes.StatusCode {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, tt.wantRes.StatusCode)
			}
			if got.Header.Get("Content-Type") != tt.wantRes.Header.Get("Content-Type") {
				t.Errorf("Header Content-Type = %q, want %q",
					got.Header.Get("Content-Type"), tt.wantRes.Header.Get("Content-Type"))
			}
			if string(got.Body) != string(tt.wantRes.Body) {
				t.Errorf("Body = %q, want %q", got.Body, tt.wantRes.Body)
			}
		})
	}
}

func TestMemoryStorage_WithCleanupInterval(t *testing.T) {
	t.Parallel()

	t.Run("removes expired entries in background", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage(WithCleanupInterval(10 * time.Millisecond))
		defer func() {
			if err := s.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		}()

		ctx := context.Background()
		res := &Response{StatusCode: http.StatusOK, Body: []byte("data")}

		if err := s.Set(ctx, "key-1", res, 20*time.Millisecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Wait for the entry to expire and cleanup to run
		time.Sleep(50 * time.Millisecond)

		s.mu.RLock()
		_, exists := s.entries["key-1"]
		s.mu.RUnlock()

		if exists {
			t.Error("expected expired entry to be removed by background cleanup")
		}
	})

	t.Run("removes expired lock entries in background", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage(WithCleanupInterval(10 * time.Millisecond))
		defer func() {
			if err := s.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		}()

		ctx := context.Background()
		res := &Response{StatusCode: http.StatusOK, Body: []byte("data")}

		if err := s.Set(ctx, "key-1", res, 20*time.Millisecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Acquire and release lock to create lock entry
		unlock, err := s.Lock(ctx, "key-1", time.Hour)
		if err != nil {
			t.Fatalf("Lock() error = %v", err)
		}
		unlock()

		// Verify lock entry exists
		if _, ok := s.locks.Load("key-1"); !ok {
			t.Fatal("expected lock entry to exist before cleanup")
		}

		// Wait for the entry to expire and cleanup to run
		time.Sleep(50 * time.Millisecond)

		s.mu.RLock()
		_, entryExists := s.entries["key-1"]
		s.mu.RUnlock()

		if entryExists {
			t.Error("expected expired entry to be removed by background cleanup")
		}

		if _, lockExists := s.locks.Load("key-1"); lockExists {
			t.Error("expected expired lock entry to be removed by background cleanup")
		}
	})

	t.Run("Close stops cleanup goroutine", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage(WithCleanupInterval(10 * time.Millisecond))
		ctx := context.Background()
		res := &Response{StatusCode: http.StatusOK, Body: []byte("data")}

		if err := s.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		// Add entry after Close
		if err := s.Set(ctx, "key-after-close", res, time.Nanosecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Wait long enough for cleanup to have run if it were active
		time.Sleep(50 * time.Millisecond)

		// Expired entry should remain because cleanup is stopped
		s.mu.RLock()
		_, exists := s.entries["key-after-close"]
		s.mu.RUnlock()

		if !exists {
			t.Error("expected expired entry to remain after Close (cleanup should be stopped)")
		}
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage(WithCleanupInterval(time.Minute))

		if err := s.Close(); err != nil {
			t.Fatalf("first Close() error = %v", err)
		}

		if err := s.Close(); err != nil {
			t.Fatalf("second Close() error = %v", err)
		}
	})

	t.Run("zero interval does not start cleanup", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage(WithCleanupInterval(0))
		defer func() {
			if err := s.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		}()

		ctx := context.Background()
		res := &Response{StatusCode: http.StatusOK, Body: []byte("data")}

		if err := s.Set(ctx, "key-1", res, time.Nanosecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		// Cleanup goroutine should not have started, so expired entry remains
		s.mu.RLock()
		_, exists := s.entries["key-1"]
		s.mu.RUnlock()

		if !exists {
			t.Error("expected expired entry to remain when cleanup interval is zero")
		}
	})
}

func TestMemoryStorage_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing key", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage()
		ctx := context.Background()
		res := &Response{StatusCode: http.StatusOK, Body: []byte("data")}

		if err := s.Set(ctx, "key-1", res, time.Hour); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if err := s.Delete(ctx, "key-1"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		got, err := s.Get(ctx, "key-1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got != nil {
			t.Errorf("Get() = %v, want nil", got)
		}
	})

	t.Run("returns nil for non-existent key", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage()

		if err := s.Delete(context.Background(), "unknown"); err != nil {
			t.Errorf("Delete() error = %v, want nil", err)
		}
	})
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	s := NewMemoryStorage()
	ctx := context.Background()

	var wg sync.WaitGroup
	const goroutines = 100

	for i := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			key := "key"
			res := &Response{
				StatusCode: http.StatusOK + i,
				Body:       []byte("body"),
			}

			_ = s.Set(ctx, key, res, time.Hour)
			_, _ = s.Get(ctx, key)
			_ = s.Delete(ctx, key)
		}()
	}

	wg.Wait()
}

func TestMemoryStorage_Lock(t *testing.T) {
	t.Parallel()

	t.Run("acquires and releases lock", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage()
		ctx := context.Background()

		unlock, err := s.Lock(ctx, "key-1", time.Hour)
		if err != nil {
			t.Fatalf("Lock() error = %v", err)
		}

		unlock()
	})

	t.Run("provides mutual exclusion for same key", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage()
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
				for {
					old := maxConcurrent.Load()
					if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
						break
					}
				}
				time.Sleep(time.Millisecond)
				concurrent.Add(-1)

				unlock()
			}()
		}

		wg.Wait()

		if got := maxConcurrent.Load(); got != 1 {
			t.Errorf("max concurrent locks = %d, want 1", got)
		}
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage()
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

	t.Run("allows independent keys to lock concurrently", func(t *testing.T) {
		t.Parallel()

		s := NewMemoryStorage()
		ctx := context.Background()

		unlock1, err := s.Lock(ctx, "key-a", time.Hour)
		if err != nil {
			t.Fatalf("Lock(key-a) error = %v", err)
		}

		unlock2, err := s.Lock(ctx, "key-b", time.Hour)
		if err != nil {
			t.Fatalf("Lock(key-b) error = %v", err)
		}

		unlock1()
		unlock2()
	})

	t.Run("implements Locker interface", func(t *testing.T) {
		t.Parallel()

		var s interface{} = NewMemoryStorage()
		if _, ok := s.(Locker); !ok {
			t.Error("MemoryStorage does not implement Locker")
		}
	})
}
