package idem

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
		var gotErr error
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		store := &errorStorage{getErr: getErr}
		mw := newTestMiddleware(t, WithStorage(store), WithOnError(func(err error) { gotErr = err }))
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
		mw := newTestMiddleware(t, WithStorage(store), WithOnError(func(err error) { gotErr = err }))
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

	t.Run("calls onError callback when lock acquisition fails", func(t *testing.T) {
		t.Parallel()

		lockErr := errors.New("lock failed")
		var gotErr error
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		store := &errorLockerStorage{lockErr: lockErr}
		mw := newTestMiddleware(t, WithStorage(store), WithOnError(func(err error) { gotErr = err }))
		wrapped := mw.Handler()(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "err-lock-key")
		wrapped.ServeHTTP(httptest.NewRecorder(), req)

		if !errors.Is(gotErr, lockErr) {
			t.Errorf("onError received %v, want %v", gotErr, lockErr)
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

// plainWriter implements only http.ResponseWriter.
type plainWriter struct {
	http.ResponseWriter
}

// flusherWriter implements http.ResponseWriter and http.Flusher.
type flusherWriter struct {
	http.ResponseWriter
	flushed bool
}

func (w *flusherWriter) Flush() {
	w.flushed = true
}

// hijackerWriter implements http.ResponseWriter and http.Hijacker.
type hijackerWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (w *hijackerWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}

// flusherHijackerWriter implements http.ResponseWriter, http.Flusher, and http.Hijacker.
type flusherHijackerWriter struct {
	http.ResponseWriter
	flushed  bool
	hijacked bool
}

func (w *flusherHijackerWriter) Flush() {
	w.flushed = true
}

func (w *flusherHijackerWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}

// readerFromWriter implements http.ResponseWriter and io.ReaderFrom.
type readerFromWriter struct {
	http.ResponseWriter
	readFrom bool
}

func (w *readerFromWriter) ReadFrom(r io.Reader) (int64, error) {
	w.readFrom = true
	return io.Copy(w.ResponseWriter, r)
}

// flusherReaderFromWriter implements http.ResponseWriter, http.Flusher, and io.ReaderFrom.
type flusherReaderFromWriter struct {
	http.ResponseWriter
	flushed  bool
	readFrom bool
}

func (w *flusherReaderFromWriter) Flush() {
	w.flushed = true
}

func (w *flusherReaderFromWriter) ReadFrom(r io.Reader) (int64, error) {
	w.readFrom = true
	return io.Copy(w.ResponseWriter, r)
}

// hijackerReaderFromWriter implements http.ResponseWriter, http.Hijacker, and io.ReaderFrom.
type hijackerReaderFromWriter struct {
	http.ResponseWriter
	hijacked bool
	readFrom bool
}

func (w *hijackerReaderFromWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}

func (w *hijackerReaderFromWriter) ReadFrom(r io.Reader) (int64, error) {
	w.readFrom = true
	return io.Copy(w.ResponseWriter, r)
}

// flusherHijackerReaderFromWriter implements http.ResponseWriter, http.Flusher,
// http.Hijacker, and io.ReaderFrom.
type flusherHijackerReaderFromWriter struct {
	http.ResponseWriter
	flushed  bool
	hijacked bool
	readFrom bool
}

func (w *flusherHijackerReaderFromWriter) Flush() {
	w.flushed = true
}

func (w *flusherHijackerReaderFromWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}

func (w *flusherHijackerReaderFromWriter) ReadFrom(r io.Reader) (int64, error) {
	w.readFrom = true
	return io.Copy(w.ResponseWriter, r)
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
	getCalls    int
	setCalls    int
	deleteCalls int
}

func (s *spyStorage) Get(_ context.Context, _ string) (*Response, error) {
	s.getCalls++
	return nil, nil
}

func (s *spyStorage) Set(_ context.Context, _ string, _ *Response, _ time.Duration) error {
	s.setCalls++
	return nil
}

func (s *spyStorage) Delete(_ context.Context, _ string) error {
	s.deleteCalls++
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
