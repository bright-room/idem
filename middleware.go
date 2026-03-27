package idem

//go:generate go run ./internal/cmd/genrecorder

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// Middleware provides HTTP middleware for idempotency key handling.
type Middleware struct {
	cfg *config
}

// Config returns a read-only snapshot of the middleware configuration.
// This is useful for debug logging, health check endpoints, and
// configuration inspection.
func (m *Middleware) Config() Config {
	return m.cfg.snapshot()
}

// ConfigHandler returns an http.Handler that serves the current middleware
// configuration as JSON. This is intended for debug endpoints such as
// /debug/idem/config.
func (m *Middleware) ConfigHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(m.Config()); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(buf.Bytes())
	})
}

// Handler returns a net/http compatible middleware handler.
func (m *Middleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(m.cfg.keyHeader)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			if m.cfg.keyMaxLength > 0 && len(key) > m.cfg.keyMaxLength {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			if locker, ok := m.cfg.storage.(Locker); ok {
				unlock, err := locker.Lock(r.Context(), key, m.cfg.ttl)
				if err != nil {
					if m.cfg.metrics != nil && m.cfg.metrics.OnLockContention != nil {
						m.cfg.metrics.OnLockContention(key, err)
					}
					http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
					return
				}
				defer unlock()
			}

			cached, err := m.cfg.storage.Get(r.Context(), key)
			if err != nil {
				if m.cfg.onError != nil {
					m.cfg.onError(key, err)
				}
				if m.cfg.metrics != nil && m.cfg.metrics.OnError != nil {
					m.cfg.metrics.OnError(key, err)
				}
				next.ServeHTTP(w, r)
				return
			}

			if cached != nil {
				if m.cfg.metrics != nil && m.cfg.metrics.OnCacheHit != nil {
					m.cfg.metrics.OnCacheHit(key)
				}
				writeResponse(w, cached)
				return
			}

			if m.cfg.metrics != nil && m.cfg.metrics.OnCacheMiss != nil {
				m.cfg.metrics.OnCacheMiss(key)
			}

			rec := newResponseRecorder(w)
			next.ServeHTTP(rec, r)

			rr := rec.(recorder)
			res := rr.toResponse()
			if err := m.cfg.storage.Set(r.Context(), key, res, m.cfg.ttl); err != nil {
				if m.cfg.onError != nil {
					m.cfg.onError(key, err)
				}
				if m.cfg.metrics != nil && m.cfg.metrics.OnError != nil {
					m.cfg.metrics.OnError(key, err)
				}
			}

			rr.flush()
		})
	}
}

// recorder provides access to the underlying responseRecorder methods.
type recorder interface {
	toResponse() *Response
	flush()
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	written    bool
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.written {
		r.statusCode = code
		r.written = true
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.statusCode = http.StatusOK
		r.written = true
	}

	return r.body.Write(b)
}

func (r *responseRecorder) toResponse() *Response {
	header := make(http.Header)
	for k, v := range r.Header() {
		header[k] = append([]string(nil), v...)
	}

	return &Response{
		StatusCode: r.statusCode,
		Header:     header,
		Body:       r.body.Bytes(),
	}
}

func (r *responseRecorder) flush() {
	r.ResponseWriter.WriteHeader(r.statusCode)
	_, _ = r.ResponseWriter.Write(r.body.Bytes())
}

func writeResponse(w http.ResponseWriter, res *Response) {
	for k, vals := range res.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	_, _ = w.Write(res.Body)
}
