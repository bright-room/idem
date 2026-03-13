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
)

func main() {
	idempotency, err := idem.New()
	if err != nil {
		log.Fatal(err)
	}
	wrap := wrapMiddleware(idempotency)

	r := gin.Default()

	var orderCount atomic.Int64
	var paymentCount atomic.Int64

	// Pattern 1: Apply to specific endpoints only.
	r.POST("/orders", wrap, func(c *gin.Context) {
		n := orderCount.Add(1)
		c.JSON(http.StatusCreated, gin.H{
			"order_id": fmt.Sprintf("order-%d", n),
			"message":  "order created",
		})
	})

	// Pattern 2: Apply to a route group.
	api := r.Group("/api", wrap)
	{
		api.POST("/payments", func(c *gin.Context) {
			n := paymentCount.Add(1)
			c.JSON(http.StatusCreated, gin.H{
				"payment_id": fmt.Sprintf("payment-%d", n),
				"message":    "payment processed",
			})
		})
	}

	// This endpoint has no idempotency middleware.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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
