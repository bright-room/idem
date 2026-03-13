package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/bright-room/idem"
	idemredis "github.com/bright-room/idem/redis"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	instanceID := os.Getenv("HOSTNAME")
	if instanceID == "" {
		instanceID = "unknown"
	}

	client := goredis.NewClient(&goredis.Options{Addr: redisAddr})

	store, err := idemredis.New(client)
	if err != nil {
		log.Fatal(err)
	}

	idempotency, err := idem.New(
		idem.WithStorage(store),
		idem.WithOnError(func(err error) {
			log.Printf("[idem] error: %v", err)
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
			"order_id":    fmt.Sprintf("order-%d", n),
			"message":     "order created",
			"instance_id": instanceID,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"instance_id": instanceID,
		})
	})

	log.Printf("starting server on :8080 (instance: %s)", instanceID)
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
