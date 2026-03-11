package idem

import (
	"bytes"
	"io"
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

			if locker, ok := m.cfg.storage.(Locker); ok {
				unlock, err := locker.Lock(r.Context(), key, m.cfg.ttl)
				if err != nil {
					if m.cfg.onError != nil {
						m.cfg.onError(err)
					}
					http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
					return
				}
				defer unlock()
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

			rr := rec.(recorder)
			res := rr.toResponse()
			if err := m.cfg.storage.Set(r.Context(), key, res, m.cfg.ttl); err != nil {
				if m.cfg.onError != nil {
					m.cfg.onError(err)
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

// responseRecorderFlusher delegates http.Flusher to the underlying ResponseWriter.
type responseRecorderFlusher struct {
	*responseRecorder
	http.Flusher
}

// responseRecorderHijacker delegates http.Hijacker to the underlying ResponseWriter.
type responseRecorderHijacker struct {
	*responseRecorder
	http.Hijacker
}

// responseRecorderFlusherHijacker delegates both http.Flusher and http.Hijacker
// to the underlying ResponseWriter.
type responseRecorderFlusherHijacker struct {
	*responseRecorder
	http.Flusher
	http.Hijacker
}

// responseRecorderReaderFrom delegates io.ReaderFrom to the underlying ResponseWriter.
type responseRecorderReaderFrom struct {
	*responseRecorder
	io.ReaderFrom
}

// responseRecorderFlusherReaderFrom delegates both http.Flusher and io.ReaderFrom
// to the underlying ResponseWriter.
type responseRecorderFlusherReaderFrom struct {
	*responseRecorder
	http.Flusher
	io.ReaderFrom
}

// responseRecorderHijackerReaderFrom delegates both http.Hijacker and io.ReaderFrom
// to the underlying ResponseWriter.
type responseRecorderHijackerReaderFrom struct {
	*responseRecorder
	http.Hijacker
	io.ReaderFrom
}

// responseRecorderFlusherHijackerReaderFrom delegates http.Flusher, http.Hijacker,
// and io.ReaderFrom to the underlying ResponseWriter.
type responseRecorderFlusherHijackerReaderFrom struct {
	*responseRecorder
	http.Flusher
	http.Hijacker
	io.ReaderFrom
}

func newResponseRecorder(w http.ResponseWriter) http.ResponseWriter {
	rec := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	flusher, canFlush := w.(http.Flusher)
	hijacker, canHijack := w.(http.Hijacker)
	readerFrom, canReadFrom := w.(io.ReaderFrom)

	switch {
	case canFlush && canHijack && canReadFrom:
		return &responseRecorderFlusherHijackerReaderFrom{rec, flusher, hijacker, readerFrom}
	case canFlush && canHijack:
		return &responseRecorderFlusherHijacker{rec, flusher, hijacker}
	case canFlush && canReadFrom:
		return &responseRecorderFlusherReaderFrom{rec, flusher, readerFrom}
	case canHijack && canReadFrom:
		return &responseRecorderHijackerReaderFrom{rec, hijacker, readerFrom}
	case canFlush:
		return &responseRecorderFlusher{rec, flusher}
	case canHijack:
		return &responseRecorderHijacker{rec, hijacker}
	case canReadFrom:
		return &responseRecorderReaderFrom{rec, readerFrom}
	default:
		return rec
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
