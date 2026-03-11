package idem

import (
	"bytes"
	"net/http"
)

// Middleware provides HTTP middleware for idempotency key handling.
type Middleware struct {
	cfg *config
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

			cached, err := m.cfg.storage.Get(r.Context(), key)
			if err != nil {
				if m.cfg.onError != nil {
					m.cfg.onError(err)
				}
				next.ServeHTTP(w, r)
				return
			}

			if cached != nil {
				writeResponse(w, cached)
				return
			}

			rec := newResponseRecorder(w)
			next.ServeHTTP(rec, r)

			res := rec.toResponse()
			if err := m.cfg.storage.Set(r.Context(), key, res, m.cfg.ttl); err != nil {
				if m.cfg.onError != nil {
					m.cfg.onError(err)
				}
			}

			rec.flush()
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	written    bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
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
	header := make(map[string][]string)
	for k, v := range r.Header() {
		header[k] = v
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
