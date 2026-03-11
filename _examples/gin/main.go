package main

import (
	"fmt"
	"log"
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
		handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	}
}
