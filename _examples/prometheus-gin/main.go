package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/bright-room/idem"
	idemgin "github.com/bright-room/idem/gin"
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
	wrap := idemgin.WrapMiddleware(idempotency)

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
