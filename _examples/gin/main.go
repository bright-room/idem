package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/bright-room/idem"
	idemgin "github.com/bright-room/idem/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	idempotency, err := idem.New()
	if err != nil {
		log.Fatal(err)
	}
	wrap := idemgin.WrapMiddleware(idempotency)

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

	// Debug endpoint: serves the middleware configuration as JSON.
	r.GET("/debug/idem/config", gin.WrapH(idempotency.ConfigHandler()))

	// This endpoint has no idempotency middleware.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.Run(":8080")
}
