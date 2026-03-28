package idem

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMiddleware_Handler(t *testing.T) {
	t.Parallel()

	t.Run("passes through request without idempotency key", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t)
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if !called {
			t.Error("handler was not called")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("caches response on first request and returns cache on second", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		})

		mw := newTestMiddleware(t)
		wrapped := mw.Handler()(handler)

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/", nil)
		req1.Header.Set("Idempotency-Key", "key-1")
		rec1 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec1, req1)

		if callCount != 1 {
			t.Fatalf("first request: handler call count = %d, want 1", callCount)
		}

		if rec1.Code != http.StatusCreated {
			t.Errorf("first request: status = %d, want %d", rec1.Code, http.StatusCreated)
		}

		// Second request with same key
		req2 := httptest.NewRequest(http.MethodPost, "/", nil)
		req2.Header.Set("Idempotency-Key", "key-1")
		rec2 := httptest.NewRecorder()
		wrapped.ServeHTTP(rec2, req2)

		if callCount != 1 {
			t.Errorf("second request: handler call count = %d, want 1", callCount)
		}

		if rec2.Code != http.StatusCreated {
			t.Errorf("second request: status = %d, want %d", rec2.Code, http.StatusCreated)
		}

		if rec2.Header().Get("Content-Type") != "application/json" {
			t.Errorf("second request: Content-Type = %q, want %q",
				rec2.Header().Get("Content-Type"), "application/json")
		}

		if rec2.Body.String() != `{"id":1}` {
			t.Errorf("second request: body = %q, want %q", rec2.Body.String(), `{"id":1}`)
		}
	})

	t.Run("handles different keys independently", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t)
		wrapped := mw.Handler()(handler)

		for _, key := range []string{"key-a", "key-b", "key-c"} {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Idempotency-Key", key)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}

		if callCount != 3 {
			t.Errorf("handler call count = %d, want 3", callCount)
		}
	})

	t.Run("uses custom key header from WithKeyHeader", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithKeyHeader("X-Request-Id"))
		wrapped := mw.Handler()(handler)

		req1 := httptest.NewRequest(http.MethodPost, "/", nil)
		req1.Header.Set("X-Request-Id", "req-1")
		wrapped.ServeHTTP(httptest.NewRecorder(), req1)

		req2 := httptest.NewRequest(http.MethodPost, "/", nil)
		req2.Header.Set("X-Request-Id", "req-1")
		wrapped.ServeHTTP(httptest.NewRecorder(), req2)

		if callCount != 1 {
			t.Errorf("handler call count = %d, want 1", callCount)
		}
	})

	t.Run("respects TTL from WithTTL", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithTTL(time.Nanosecond))
		wrapped := mw.Handler()(handler)

		req1 := httptest.NewRequest(http.MethodPost, "/", nil)
		req1.Header.Set("Idempotency-Key", "ttl-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req1)

		time.Sleep(time.Millisecond)

		req2 := httptest.NewRequest(http.MethodPost, "/", nil)
		req2.Header.Set("Idempotency-Key", "ttl-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req2)

		if callCount != 2 {
			t.Errorf("handler call count = %d, want 2", callCount)
		}
	})

	t.Run("uses custom storage from WithStorage", func(t *testing.T) {
		t.Parallel()

		store := &spyStorage{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithStorage(store))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "spy-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if got := store.getCalls.Load(); got != 1 {
			t.Errorf("storage Get calls = %d, want 1", got)
		}

		if got := store.setCalls.Load(); got != 1 {
			t.Errorf("storage Set calls = %d, want 1", got)
		}
	})

	t.Run("falls through to handler when storage Get returns error", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		store := &errorStorage{getErr: errors.New("storage unavailable")}
		mw := newTestMiddleware(t, WithStorage(store))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-key")
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if !called {
			t.Error("handler was not called on storage error")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("calls onError callback when storage Get fails", func(t *testing.T) {
		t.Parallel()

		getErr := errors.New("get failed")
		var gotKey string
		var gotErr error
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		store := &errorStorage{getErr: getErr}
		mw := newTestMiddleware(t, WithStorage(store), WithOnError(func(key string, err error) {
			gotKey = key
			gotErr = err
		}))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if gotKey != "err-key" {
			t.Errorf("onError key = %q, want %q", gotKey, "err-key")
		}

		if !errors.Is(gotErr, getErr) {
			t.Errorf("onError received %v, want %v", gotErr, getErr)
		}
	})

	t.Run("calls onError callback when storage Set fails", func(t *testing.T) {
		t.Parallel()

		setErr := errors.New("set failed")
		var gotKey string
		var gotErr error
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		store := &errorStorage{setErr: setErr}
		mw := newTestMiddleware(t, WithStorage(store), WithOnError(func(key string, err error) {
			gotKey = key
			gotErr = err
		}))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-key")
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if gotKey != "err-key" {
			t.Errorf("onError key = %q, want %q", gotKey, "err-key")
		}

		if !errors.Is(gotErr, setErr) {
			t.Errorf("onError received %v, want %v", gotErr, setErr)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("acquires lock when storage implements Locker", func(t *testing.T) {
		t.Parallel()

		store := &spyLockerStorage{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithStorage(store))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "lock-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if store.lockCalls != 1 {
			t.Errorf("Lock calls = %d, want 1", store.lockCalls)
		}

		if store.unlockCalls != 1 {
			t.Errorf("Unlock calls = %d, want 1", store.unlockCalls)
		}
	})

	t.Run("returns 409 Conflict when lock acquisition fails", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		lockErr := errors.New("lock failed")
		store := &errorLockerStorage{lockErr: lockErr}
		mw := newTestMiddleware(t, WithStorage(store))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "conflict-key")
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if called {
			t.Error("handler was called, want not called")
		}

		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
		}
	})

	t.Run("does not call onError callback when lock acquisition fails", func(t *testing.T) {
		t.Parallel()

		lockErr := errors.New("lock failed")
		var onErrorCalled bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		store := &errorLockerStorage{lockErr: lockErr}
		mw := newTestMiddleware(t, WithStorage(store), WithOnError(func(_ string, _ error) { onErrorCalled = true }))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-lock-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if onErrorCalled {
			t.Error("WithOnError callback was called on lock contention, want not called")
		}
	})

	t.Run("calls OnCacheMiss on first request and OnCacheHit on second", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		var hitKey, missKey string
		mw := newTestMiddleware(t, WithMetrics(Metrics{
			OnCacheHit:  func(key string) { hitKey = key },
			OnCacheMiss: func(key string) { missKey = key },
		}))
		wrapped := mw.Handler()(handler)

		// First request — cache miss
		req1 := httptest.NewRequest(http.MethodPost, "/", nil)
		req1.Header.Set("Idempotency-Key", "metrics-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req1)

		if missKey != "metrics-key" {
			t.Errorf("OnCacheMiss key = %q, want %q", missKey, "metrics-key")
		}

		if hitKey != "" {
			t.Errorf("OnCacheHit key = %q, want empty on first request", hitKey)
		}

		// Second request — cache hit
		req2 := httptest.NewRequest(http.MethodPost, "/", nil)
		req2.Header.Set("Idempotency-Key", "metrics-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req2)

		if hitKey != "metrics-key" {
			t.Errorf("OnCacheHit key = %q, want %q", hitKey, "metrics-key")
		}
	})

	t.Run("calls Metrics.OnError when storage Get fails", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		getErr := errors.New("get failed")
		var gotKey string
		var gotErr error
		store := &errorStorage{getErr: getErr}
		mw := newTestMiddleware(t, WithStorage(store), WithMetrics(Metrics{
			OnError: func(key string, err error) {
				gotKey = key
				gotErr = err
			},
		}))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-get-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if gotKey != "err-get-key" {
			t.Errorf("OnError key = %q, want %q", gotKey, "err-get-key")
		}

		if !errors.Is(gotErr, getErr) {
			t.Errorf("OnError err = %v, want %v", gotErr, getErr)
		}
	})

	t.Run("calls Metrics.OnError when storage Set fails", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		setErr := errors.New("set failed")
		var gotKey string
		var gotErr error
		store := &errorStorage{setErr: setErr}
		mw := newTestMiddleware(t, WithStorage(store), WithMetrics(Metrics{
			OnError: func(key string, err error) {
				gotKey = key
				gotErr = err
			},
		}))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-set-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if gotKey != "err-set-key" {
			t.Errorf("OnError key = %q, want %q", gotKey, "err-set-key")
		}

		if !errors.Is(gotErr, setErr) {
			t.Errorf("OnError err = %v, want %v", gotErr, setErr)
		}
	})

	t.Run("calls Metrics.OnLockContention when lock acquisition fails", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		lockErr := errors.New("lock failed")
		var gotKey string
		var gotErr error
		store := &errorLockerStorage{lockErr: lockErr}
		mw := newTestMiddleware(t, WithStorage(store), WithMetrics(Metrics{
			OnLockContention: func(key string, err error) {
				gotKey = key
				gotErr = err
			},
		}))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-lock-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if gotKey != "err-lock-key" {
			t.Errorf("OnLockContention key = %q, want %q", gotKey, "err-lock-key")
		}

		if !errors.Is(gotErr, lockErr) {
			t.Errorf("OnLockContention err = %v, want %v", gotErr, lockErr)
		}
	})

	t.Run("does not call Metrics.OnError when lock acquisition fails", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		lockErr := errors.New("lock failed")
		var onErrorCalled bool
		store := &errorLockerStorage{lockErr: lockErr}
		mw := newTestMiddleware(t, WithStorage(store), WithMetrics(Metrics{
			OnError: func(_ string, _ error) { onErrorCalled = true },
		}))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-lock-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if onErrorCalled {
			t.Error("Metrics.OnError was called on lock contention, want not called")
		}
	})

	t.Run("does not call Metrics.OnLockContention on storage errors", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		var onLockContentionCalled bool
		store := &errorStorage{getErr: errors.New("get failed")}
		mw := newTestMiddleware(t, WithStorage(store), WithMetrics(Metrics{
			OnLockContention: func(_ string, _ error) { onLockContentionCalled = true },
		}))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "storage-err-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if onLockContentionCalled {
			t.Error("Metrics.OnLockContention was called on storage error, want not called")
		}
	})

	t.Run("calls both WithOnError and Metrics.OnError on storage Get failure", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		getErr := errors.New("get failed")
		var onErrorCalled bool
		var metricsErrorCalled bool
		store := &errorStorage{getErr: getErr}
		mw := newTestMiddleware(t,
			WithStorage(store),
			WithOnError(func(_ string, _ error) { onErrorCalled = true }),
			WithMetrics(Metrics{
				OnError: func(_ string, _ error) { metricsErrorCalled = true },
			}),
		)
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "both-err-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if !onErrorCalled {
			t.Error("WithOnError callback was not called")
		}

		if !metricsErrorCalled {
			t.Error("Metrics.OnError callback was not called")
		}
	})

	t.Run("does not panic with nil callback fields in Metrics", func(t *testing.T) {
		t.Parallel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Only OnCacheMiss is set; OnCacheHit and OnError are nil
		var missKey string
		mw := newTestMiddleware(t, WithMetrics(Metrics{
			OnCacheMiss: func(key string) { missKey = key },
		}))
		wrapped := mw.Handler()(handler)

		// First request — triggers OnCacheMiss, skips nil OnCacheHit/OnError
		req1 := httptest.NewRequest(http.MethodPost, "/", nil)
		req1.Header.Set("Idempotency-Key", "partial-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req1)

		if missKey != "partial-key" {
			t.Errorf("OnCacheMiss key = %q, want %q", missKey, "partial-key")
		}

		// Second request — triggers nil OnCacheHit without panic
		req2 := httptest.NewRequest(http.MethodPost, "/", nil)
		req2.Header.Set("Idempotency-Key", "partial-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req2)
	})

	t.Run("returns 400 Bad Request when key exceeds WithKeyMaxLength", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithKeyMaxLength(10))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "12345678901") // 11 chars > 10
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if called {
			t.Error("handler was called, want not called")
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("allows key with length equal to WithKeyMaxLength", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithKeyMaxLength(10))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "1234567890") // exactly 10 chars
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if !called {
			t.Error("handler was not called")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("allows key shorter than WithKeyMaxLength", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithKeyMaxLength(10))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "short") // 5 chars < 10
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if !called {
			t.Error("handler was not called")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("disables key length check when WithKeyMaxLength is 0", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t, WithKeyMaxLength(0))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", strings.Repeat("a", 1000)) // very long key
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if !called {
			t.Error("handler was not called")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("concurrent requests with same key execute handler only once", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			callCount++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		})

		store := NewMemoryStorage()
		mw := newTestMiddleware(t, WithStorage(store))
		wrapped := mw.Handler()(handler)

		var wg sync.WaitGroup
		const goroutines = 10

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.Header.Set("Idempotency-Key", "concurrent-key")
				rec := httptest.NewRecorder()
				wrapped.ServeHTTP(rec, req)
			}()
		}

		wg.Wait()

		mu.Lock()
		count := callCount
		mu.Unlock()

		if count != 1 {
			t.Errorf("handler call count = %d, want 1", count)
		}
	})
}

func TestMiddleware_Config(t *testing.T) {
	t.Parallel()

	t.Run("returns defaults with no options", func(t *testing.T) {
		t.Parallel()

		mw := newTestMiddleware(t)
		cfg := mw.Config()

		if cfg.KeyHeader != DefaultKeyHeader {
			t.Errorf("KeyHeader = %q, want %q", cfg.KeyHeader, DefaultKeyHeader)
		}
		if cfg.TTL != Duration(DefaultTTL) {
			t.Errorf("TTL = %v, want %v", cfg.TTL, DefaultTTL)
		}
		if cfg.StorageType != "*idem.MemoryStorage" {
			t.Errorf("StorageType = %q, want %q", cfg.StorageType, "*idem.MemoryStorage")
		}
		if !cfg.LockSupported {
			t.Error("LockSupported = false, want true")
		}
		if cfg.MetricsEnabled {
			t.Error("MetricsEnabled = true, want false")
		}
		if cfg.OnErrorEnabled {
			t.Error("OnErrorEnabled = true, want false")
		}
		if cfg.KeyMaxLength != 0 {
			t.Errorf("KeyMaxLength = %d, want 0", cfg.KeyMaxLength)
		}
		if cfg.ValidatorCount != 0 {
			t.Errorf("ValidatorCount = %d, want 0", cfg.ValidatorCount)
		}
	})

	t.Run("reflects custom options", func(t *testing.T) {
		t.Parallel()

		mw := newTestMiddleware(t,
			WithKeyHeader("X-Request-Id"),
			WithTTL(5*time.Minute),
			WithKeyMaxLength(64),
			WithStorage(&stubStorage{}),
			WithOnError(func(_ string, _ error) {}),
			WithMetrics(Metrics{}),
			WithValidation(ValidatorFunc(func(_ Config) error { return nil })),
		)
		cfg := mw.Config()

		if cfg.KeyHeader != "X-Request-Id" {
			t.Errorf("KeyHeader = %q, want %q", cfg.KeyHeader, "X-Request-Id")
		}
		if cfg.TTL != Duration(5*time.Minute) {
			t.Errorf("TTL = %v, want %v", cfg.TTL, 5*time.Minute)
		}
		if cfg.StorageType != "*idem.stubStorage" {
			t.Errorf("StorageType = %q, want %q", cfg.StorageType, "*idem.stubStorage")
		}
		if cfg.LockSupported {
			t.Error("LockSupported = true, want false")
		}
		if !cfg.MetricsEnabled {
			t.Error("MetricsEnabled = false, want true")
		}
		if !cfg.OnErrorEnabled {
			t.Error("OnErrorEnabled = false, want true")
		}
		if cfg.KeyMaxLength != 64 {
			t.Errorf("KeyMaxLength = %d, want 64", cfg.KeyMaxLength)
		}
		if cfg.ValidatorCount != 1 {
			t.Errorf("ValidatorCount = %d, want 1", cfg.ValidatorCount)
		}
	})

	t.Run("LockSupported reflects Locker implementation", func(t *testing.T) {
		t.Parallel()

		mw := newTestMiddleware(t, WithStorage(&spyLockerStorage{}))
		cfg := mw.Config()

		if !cfg.LockSupported {
			t.Error("LockSupported = false, want true for Locker-implementing storage")
		}
	})
}

func TestMiddleware_ConfigHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns defaults as JSON", func(t *testing.T) {
		t.Parallel()

		mw := newTestMiddleware(t)
		handler := mw.ConfigHandler()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/debug/idem/config", nil)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		ct := rec.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		// Duration.UnmarshalJSON decodes the human-readable string (e.g. "24h0m0s")
		// produced by Duration.MarshalJSON, so the round-trip preserves the value.
		var got Config
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		want := mw.Config()
		if got != want {
			t.Errorf("config = %+v, want %+v", got, want)
		}
	})

	t.Run("reflects custom options", func(t *testing.T) {
		t.Parallel()

		mw := newTestMiddleware(t,
			WithKeyHeader("X-Request-Id"),
			WithTTL(5*time.Minute),
			WithKeyMaxLength(64),
		)
		handler := mw.ConfigHandler()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/debug/idem/config", nil)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var got Config
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if got.KeyHeader != "X-Request-Id" {
			t.Errorf("KeyHeader = %q, want %q", got.KeyHeader, "X-Request-Id")
		}
		if got.TTL != Duration(5*time.Minute) {
			t.Errorf("TTL = %v, want %v", got.TTL, 5*time.Minute)
		}
		if got.KeyMaxLength != 64 {
			t.Errorf("KeyMaxLength = %d, want 64", got.KeyMaxLength)
		}
	})
}

func TestNewResponseRecorder(t *testing.T) {
	t.Parallel()

	t.Run("delegates http.Flusher to underlying ResponseWriter", func(t *testing.T) {
		t.Parallel()

		fw := &flusherWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(fw)

		flusher, ok := rec.(http.Flusher)
		if !ok {
			t.Fatal("recorder does not implement http.Flusher")
		}

		flusher.Flush()

		if !fw.flushed {
			t.Error("Flush() was not delegated to the underlying writer")
		}
	})

	t.Run("delegates http.Hijacker to underlying ResponseWriter", func(t *testing.T) {
		t.Parallel()

		hw := &hijackerWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(hw)

		hijacker, ok := rec.(http.Hijacker)
		if !ok {
			t.Fatal("recorder does not implement http.Hijacker")
		}

		_, _, err := hijacker.Hijack()
		if err != nil {
			t.Fatalf("Hijack() error = %v", err)
		}

		if !hw.hijacked {
			t.Error("Hijack() was not delegated to the underlying writer")
		}
	})

	t.Run("delegates both http.Flusher and http.Hijacker", func(t *testing.T) {
		t.Parallel()

		fhw := &flusherHijackerWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(fhw)

		if _, ok := rec.(http.Flusher); !ok {
			t.Error("recorder does not implement http.Flusher")
		}

		if _, ok := rec.(http.Hijacker); !ok {
			t.Error("recorder does not implement http.Hijacker")
		}
	})

	t.Run("delegates io.ReaderFrom to underlying ResponseWriter", func(t *testing.T) {
		t.Parallel()

		rfw := &readerFromWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(rfw)

		rf, ok := rec.(io.ReaderFrom)
		if !ok {
			t.Fatal("recorder does not implement io.ReaderFrom")
		}

		_, _ = rf.ReadFrom(strings.NewReader("hello"))

		if !rfw.readFrom {
			t.Error("ReadFrom() was not delegated to the underlying writer")
		}
	})

	t.Run("delegates both http.Flusher and io.ReaderFrom", func(t *testing.T) {
		t.Parallel()

		frw := &flusherReaderFromWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(frw)

		if _, ok := rec.(http.Flusher); !ok {
			t.Error("recorder does not implement http.Flusher")
		}

		if _, ok := rec.(io.ReaderFrom); !ok {
			t.Error("recorder does not implement io.ReaderFrom")
		}
	})

	t.Run("delegates both http.Hijacker and io.ReaderFrom", func(t *testing.T) {
		t.Parallel()

		hrw := &hijackerReaderFromWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(hrw)

		if _, ok := rec.(http.Hijacker); !ok {
			t.Error("recorder does not implement http.Hijacker")
		}

		if _, ok := rec.(io.ReaderFrom); !ok {
			t.Error("recorder does not implement io.ReaderFrom")
		}
	})

	t.Run("delegates http.Flusher, http.Hijacker, and io.ReaderFrom", func(t *testing.T) {
		t.Parallel()

		fhrw := &flusherHijackerReaderFromWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(fhrw)

		if _, ok := rec.(http.Flusher); !ok {
			t.Error("recorder does not implement http.Flusher")
		}

		if _, ok := rec.(http.Hijacker); !ok {
			t.Error("recorder does not implement http.Hijacker")
		}

		if _, ok := rec.(io.ReaderFrom); !ok {
			t.Error("recorder does not implement io.ReaderFrom")
		}
	})

	t.Run("does not implement http.Flusher when underlying writer does not", func(t *testing.T) {
		t.Parallel()

		pw := &plainWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(pw)

		if _, ok := rec.(http.Flusher); ok {
			t.Error("recorder implements http.Flusher, want not implemented")
		}

		if _, ok := rec.(http.Hijacker); ok {
			t.Error("recorder implements http.Hijacker, want not implemented")
		}

		if _, ok := rec.(io.ReaderFrom); ok {
			t.Error("recorder implements io.ReaderFrom, want not implemented")
		}
	})

	t.Run("preserves response recording with ReaderFrom delegation", func(t *testing.T) {
		t.Parallel()

		rfw := &readerFromWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(rfw)

		rec.WriteHeader(http.StatusCreated)
		_, _ = rec.Write([]byte("hello"))

		rr := rec.(recorder)
		res := rr.toResponse()

		if res.StatusCode != http.StatusCreated {
			t.Errorf("status = %d, want %d", res.StatusCode, http.StatusCreated)
		}

		if string(res.Body) != "hello" {
			t.Errorf("body = %q, want %q", string(res.Body), "hello")
		}
	})

	t.Run("preserves response recording with Flusher delegation", func(t *testing.T) {
		t.Parallel()

		fw := &flusherWriter{ResponseWriter: httptest.NewRecorder()}
		rec := newResponseRecorder(fw)

		rec.WriteHeader(http.StatusCreated)
		_, _ = rec.Write([]byte("hello"))

		rr := rec.(recorder)
		res := rr.toResponse()

		if res.StatusCode != http.StatusCreated {
			t.Errorf("status = %d, want %d", res.StatusCode, http.StatusCreated)
		}

		if string(res.Body) != "hello" {
			t.Errorf("body = %q, want %q", string(res.Body), "hello")
		}
	})

	t.Run("preserves io.ReaderFrom interface through middleware handler", func(t *testing.T) {
		t.Parallel()

		var handlerReaderFromOK bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, handlerReaderFromOK = w.(io.ReaderFrom)
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t)
		wrapped := mw.Handler()(handler)

		rfw := &readerFromWriter{ResponseWriter: httptest.NewRecorder()}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "readerfrom-key")
		wrapped.ServeHTTP(rfw, req)

		if !handlerReaderFromOK {
			t.Error("io.ReaderFrom was not available inside handler")
		}
	})

	t.Run("preserves interface through middleware handler", func(t *testing.T) {
		t.Parallel()

		var handlerFlusherOK bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, handlerFlusherOK = w.(http.Flusher)
			w.WriteHeader(http.StatusOK)
		})

		mw := newTestMiddleware(t)
		wrapped := mw.Handler()(handler)

		fw := &flusherWriter{ResponseWriter: httptest.NewRecorder()}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "flusher-key")
		wrapped.ServeHTTP(fw, req)

		if !handlerFlusherOK {
			t.Error("http.Flusher was not available inside handler")
		}
	})
}

// newTestMiddleware creates a Middleware with default options, failing the test on error.
func newTestMiddleware(t *testing.T, opts ...Option) *Middleware {
	t.Helper()

	mw, err := New(opts...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return mw
}

// spyStorage tracks Get/Set/Delete call counts.
type spyStorage struct {
	getCalls    atomic.Int32
	setCalls    atomic.Int32
	deleteCalls atomic.Int32
}

func (s *spyStorage) Get(_ context.Context, _ string) (*Response, error) {
	s.getCalls.Add(1)
	return nil, nil
}

func (s *spyStorage) Set(_ context.Context, _ string, _ *Response, _ time.Duration) error {
	s.setCalls.Add(1)
	return nil
}

func (s *spyStorage) Delete(_ context.Context, _ string) error {
	s.deleteCalls.Add(1)
	return nil
}

// errorStorage returns errors from Get/Set/Delete.
type errorStorage struct {
	getErr error
	setErr error
	delErr error
}

func (s *errorStorage) Get(_ context.Context, _ string) (*Response, error) {
	return nil, s.getErr
}

func (s *errorStorage) Set(_ context.Context, _ string, _ *Response, _ time.Duration) error {
	return s.setErr
}

func (s *errorStorage) Delete(_ context.Context, _ string) error {
	return s.delErr
}

// spyLockerStorage tracks Lock/Unlock call counts.
type spyLockerStorage struct {
	spyStorage
	lockCalls   int
	unlockCalls int
}

func (s *spyLockerStorage) Lock(_ context.Context, _ string, _ time.Duration) (func(), error) {
	s.lockCalls++
	return func() { s.unlockCalls++ }, nil
}

// errorLockerStorage returns errors from Lock.
type errorLockerStorage struct {
	errorStorage
	lockErr error
}

func (s *errorLockerStorage) Lock(_ context.Context, _ string, _ time.Duration) (func(), error) {
	return nil, s.lockErr
}
