package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/bright-room/idem"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	idempotency, err := idem.New()
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	var orderCount atomic.Int64
	var paymentCount atomic.Int64

	// Pattern 1: Apply to a specific route inline using r.With().
	// Chi is net/http compatible, so mw.Handler() works directly
	// — no wrapper or conversion needed.
	r.With(idempotency.Handler()).Post("/orders", func(w http.ResponseWriter, r *http.Request) {
		n := orderCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"order_id": fmt.Sprintf("order-%d", n),
			"message":  "order created",
		})
	})

	// Pattern 2: Apply to a route group.
	r.Route("/api", func(api chi.Router) {
		api.Use(idempotency.Handler())
		api.Post("/payments", func(w http.ResponseWriter, r *http.Request) {
			n := paymentCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"payment_id": fmt.Sprintf("payment-%d", n),
				"message":    "payment processed",
			})
		})
	})

	// This endpoint has no idempotency middleware.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.ListenAndServe(":8080", r)
}
