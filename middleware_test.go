package idem

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

		mw := New()
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

		mw := New()
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

		mw := New()
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

		mw := New(WithKeyHeader("X-Request-Id"))
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

		mw := New(WithTTL(time.Nanosecond))
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

		mw := New(WithStorage(store))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "spy-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if store.getCalls != 1 {
			t.Errorf("storage Get calls = %d, want 1", store.getCalls)
		}

		if store.setCalls != 1 {
			t.Errorf("storage Set calls = %d, want 1", store.setCalls)
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
		mw := New(WithStorage(store))
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
		var gotErr error
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		store := &errorStorage{getErr: getErr}
		mw := New(WithStorage(store), WithOnError(func(err error) { gotErr = err }))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if !errors.Is(gotErr, getErr) {
			t.Errorf("onError received %v, want %v", gotErr, getErr)
		}
	})

	t.Run("calls onError callback when storage Set fails", func(t *testing.T) {
		t.Parallel()

		setErr := errors.New("set failed")
		var gotErr error
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		store := &errorStorage{setErr: setErr}
		mw := New(WithStorage(store), WithOnError(func(err error) { gotErr = err }))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-key")
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if !errors.Is(gotErr, setErr) {
			t.Errorf("onError received %v, want %v", gotErr, setErr)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

// spyStorage tracks Get/Set call counts.
type spyStorage struct {
	getCalls int
	setCalls int
}

func (s *spyStorage) Get(_ context.Context, _ string) (*Response, error) {
	s.getCalls++
	return nil, nil
}

func (s *spyStorage) Set(_ context.Context, _ string, _ *Response, _ time.Duration) error {
	s.setCalls++
	return nil
}

// errorStorage returns errors from Get/Set.
type errorStorage struct {
	getErr error
	setErr error
}

func (s *errorStorage) Get(_ context.Context, _ string) (*Response, error) {
	return nil, s.getErr
}

func (s *errorStorage) Set(_ context.Context, _ string, _ *Response, _ time.Duration) error {
	return s.setErr
}
