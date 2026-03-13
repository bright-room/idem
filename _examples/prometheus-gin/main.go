package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/bright-room/idem"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "idem_cache_hits_total",
		Help: "Total number of idempotency cache hits.",
	})
	cacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "idem_cache_misses_total",
		Help: "Total number of idempotency cache misses.",
	})
	lockContentions = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "idem_lock_contentions_total",
		Help: "Total number of lock contentions (409 Conflict).",
	})
	storageErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "idem_storage_errors_total",
		Help: "Total number of storage operation errors.",
	})
)

func init() {
	prometheus.MustRegister(cacheHits, cacheMisses, lockContentions, storageErrors)
}

func main() {
	idempotency, err := idem.New(
		idem.WithMetrics(idem.Metrics{
			OnCacheHit: func(key string) {
				cacheHits.Inc()
			},
			OnCacheMiss: func(key string) {
				cacheMisses.Inc()
			},
			OnLockContention: func(key string, err error) {
				lockContentions.Inc()
			},
			OnError: func(key string, err error) {
				storageErrors.Inc()
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	wrap := wrapMiddleware(idempotency)

	r := gin.Default()

	var orderCount atomic.Int64

	r.POST("/orders", wrap, func(c *gin.Context) {
		n := orderCount.Add(1)
		c.JSON(http.StatusCreated, gin.H{
			"order_id": fmt.Sprintf("order-%d", n),
			"message":  "order created",
		})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("starting server on :8080")
	r.Run(":8080")
}

// wrapMiddleware converts idem.Middleware into a gin.HandlerFunc.
func wrapMiddleware(m *idem.Middleware) gin.HandlerFunc {
	handler := m.Handler()
	return func(c *gin.Context) {
		var innerCalled bool
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			innerCalled = true
			c.Request = r
			// Replace Gin's writer so that c.JSON() writes through
			// idem's responseRecorder, enabling response capture.
			c.Writer = &recorderGinWriter{
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

// recorderGinWriter wraps idem's responseRecorder while satisfying
// gin.ResponseWriter so that Gin helpers (c.JSON, etc.) work correctly.
type recorderGinWriter struct {
	http.ResponseWriter
	ginWriter gin.ResponseWriter
}

func (w *recorderGinWriter) WriteHeader(code int) {
	w.ginWriter.WriteHeader(code)
	w.ResponseWriter.WriteHeader(code)
}

func (w *recorderGinWriter) Write(data []byte) (int, error) {
	w.ginWriter.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *recorderGinWriter) WriteString(s string) (int, error) {
	w.ginWriter.WriteString(s)
	return w.ResponseWriter.Write([]byte(s))
}

func (w *recorderGinWriter) Status() int {
	return w.ginWriter.Status()
}

func (w *recorderGinWriter) Size() int {
	return w.ginWriter.Size()
}

func (w *recorderGinWriter) Written() bool {
	return w.ginWriter.Written()
}

func (w *recorderGinWriter) WriteHeaderNow() {
	w.ginWriter.WriteHeaderNow()
}

func (w *recorderGinWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ginWriter.Hijack()
}

func (w *recorderGinWriter) Flush() {
	w.ginWriter.Flush()
}

func (w *recorderGinWriter) CloseNotify() <-chan bool {
	return w.ginWriter.CloseNotify()
}

func (w *recorderGinWriter) Pusher() http.Pusher {
	return w.ginWriter.Pusher()
}
