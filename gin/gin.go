package gin

import (
	"bufio"
	"net"
	"net/http"

	"github.com/bright-room/idem"
	"github.com/gin-gonic/gin"
)

// WrapMiddleware converts idem.Middleware into a gin.HandlerFunc.
// It bridges idem's net/http middleware with Gin's request handling,
// ensuring that Gin helpers (c.JSON, etc.) work correctly while
// idem captures the response for caching.
//
// On a cache hit the inner handler is not called and c.Abort() stops
// the Gin handler chain.
func WrapMiddleware(m *idem.Middleware) gin.HandlerFunc {
	handler := m.Handler()
	return func(c *gin.Context) {
		var innerCalled bool
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			innerCalled = true
			c.Request = r
			// Replace Gin's writer so that c.JSON() writes through
			// idem's responseRecorder, enabling response capture.
			// The responseWriter only forwards data to the recorder
			// (not to the original Gin writer) to avoid a double
			// write when the recorder flushes the buffered response.
			c.Writer = &responseWriter{
				ResponseWriter: w,
				ginWriter:      c.Writer,
			}
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)

		if !innerCalled {
			// Cache hit: idem already wrote the response.
			// Abort prevents Gin from running subsequent handlers.
			c.Abort()
		}
	}
}

// responseWriter wraps idem's responseRecorder while satisfying
// gin.ResponseWriter so that Gin helpers (c.JSON, etc.) work correctly.
//
// Response data (Write, WriteHeader) flows only to the embedded
// ResponseWriter (idem's recorder). State queries (Status, Size,
// Written) are tracked locally. Connection-level operations
// (Hijack, Flush, etc.) delegate to the original Gin writer.
type responseWriter struct {
	http.ResponseWriter
	ginWriter    gin.ResponseWriter
	status       int
	size         int
	headerWriten bool
	written      bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.headerWriten {
		w.status = code
		w.headerWriten = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.headerWriten {
		w.status = http.StatusOK
		w.headerWriten = true
	}
	w.written = true
	n, err := w.ResponseWriter.Write(data)
	w.size += n
	return n, err
}

func (w *responseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Size() int {
	return w.size
}

func (w *responseWriter) Written() bool {
	return w.written
}

func (w *responseWriter) WriteHeaderNow() {
	if !w.written {
		w.written = true
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ginWriter.Hijack()
}

func (w *responseWriter) Flush() {
	w.ginWriter.Flush()
}

func (w *responseWriter) CloseNotify() <-chan bool {
	return w.ginWriter.CloseNotify()
}

func (w *responseWriter) Pusher() http.Pusher {
	return w.ginWriter.Pusher()
}
