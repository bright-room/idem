package main

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/bright-room/idem"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	idempotency := idem.New()

	e := echo.New()
	e.Use(middleware.Logger())

	var orderCount atomic.Int64
	var paymentCount atomic.Int64

	// Pattern 1: Apply to all routes via global middleware.
	// echo.WrapMiddleware converts func(http.Handler) http.Handler
	// directly into Echo middleware — no custom helper needed.
	e.Use(echo.WrapMiddleware(idempotency.Handler()))

	e.POST("/orders", func(c echo.Context) error {
		n := orderCount.Add(1)
		return c.JSON(http.StatusCreated, map[string]string{
			"order_id": fmt.Sprintf("order-%d", n),
			"message":  "order created",
		})
	})

	// Pattern 2: Apply to a route group only.
	api := e.Group("/api")
	api.Use(echo.WrapMiddleware(idempotency.Handler()))
	api.POST("/payments", func(c echo.Context) error {
		n := paymentCount.Add(1)
		return c.JSON(http.StatusCreated, map[string]string{
			"payment_id": fmt.Sprintf("payment-%d", n),
			"message":    "payment processed",
		})
	})

	// This endpoint has no idempotency middleware applied.
	// Requests without an Idempotency-Key header pass through unchanged.
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
