# idem

Idempotency key middleware for Go HTTP applications.

> **"One header, zero duplicates."**
> — Prevent duplicate request execution with a single HTTP header.

## Features

- **Framework-agnostic** — Built on `net/http`, works with Gin, Echo, Chi, and any compatible router
- **Pluggable storage** — Interface-based design with built-in memory and Redis implementations
- **Zero config** — Works out of the box with sensible defaults
- **Streaming & WebSocket ready** — Preserves `http.Flusher`, `http.Hijacker`, and `io.ReaderFrom` interfaces through the middleware
- **Lightweight** — Minimal API surface, just wrap your handler

## Installation

```sh
go get github.com/bright-room/idem
```

For Gin framework integration, also install the adapter:

```sh
go get github.com/bright-room/idem/gin
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bright-room/idem"
)

func main() {
	mw, err := idem.New()
	if err != nil {
		log.Fatal(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "order created")
	})

	http.Handle("/orders", mw.Handler()(handler))
	http.ListenAndServe(":8080", nil)
}
```

Send a request with the `Idempotency-Key` header:

```sh
curl -X POST http://localhost:8080/orders \
  -H "Idempotency-Key: abc-123"
```

The first request executes the handler and caches the response. Subsequent requests with the same key return the cached response without re-executing the handler.

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithKeyHeader(h)` | `"Idempotency-Key"` | Header name to read the idempotency key from |
| `WithTTL(d)` | `24h` | Cache duration for stored responses |
| `WithStorage(s)` | In-memory | Storage backend for cached responses |
| `WithKeyMaxLength(n)` | `0` (no limit) | Maximum allowed idempotency key length; exceeding keys receive 400 Bad Request |
| `WithCacheable(fn)` | `DefaultCacheable` (excludes 5xx) | Function that determines whether a response should be cached based on status code |
| `WithOnError(fn)` | `nil` | Callback invoked when a storage operation fails (receives key and error) |
| `WithMetrics(m)` | `nil` | Callbacks for observing cache hits, misses, lock contention, cache skips, and errors |
| `WithValidation(v...)` | none | Custom validators run during `New()` after built-in checks |

```go
mw, err := idem.New(
	idem.WithKeyHeader("X-Request-Id"),
	idem.WithTTL(1 * time.Hour),
	idem.WithKeyMaxLength(64),
	idem.WithStorage(redisStore),
	idem.WithOnError(func(key string, err error) {
		log.Printf("storage error: key=%s err=%v", key, err)
	}),
)
if err != nil {
	log.Fatal(err)
}
```

`New` validates the configuration and returns an error for invalid values such as an empty key header or a non-positive TTL.

### Response Cacheability

By default, the middleware does not cache responses with 5xx status codes. This prevents server errors from being persisted and returned for subsequent requests with the same idempotency key.

Use `WithCacheable` to customize which responses are cached:

```go
// Cache all responses including 5xx (Stripe-style behavior)
mw, err := idem.New(
	idem.WithCacheable(func(statusCode int) bool {
		return true
	}),
)

// Only cache successful responses (2xx)
mw, err := idem.New(
	idem.WithCacheable(func(statusCode int) bool {
		return statusCode >= 200 && statusCode < 300
	}),
)
```

### Custom Validation

Use `WithValidation` to enforce application-specific rules on the middleware configuration:

```go
mw, err := idem.New(
	idem.WithTTL(30 * time.Minute),
	idem.WithValidation(idem.ValidatorFunc(func(cfg idem.Config) error {
		if cfg.TTL > 1*time.Hour {
			return fmt.Errorf("TTL must not exceed 1 hour, got %v", cfg.TTL)
		}
		return nil
	})),
)
```

Validators receive a read-only `Config` snapshot and run after the built-in checks. Multiple validators execute in registration order; validation stops at the first error.

#### Preset Validators

Common validation rules are available as factory functions:

| Validator | Description |
|-----------|-------------|
| `MaxTTL(max)` | Rejects a TTL longer than `max` |
| `MinTTL(min)` | Rejects a TTL shorter than `min` |
| `TTLRange(min, max)` | Rejects a TTL outside the `[min, max]` range |
| `KeyHeaderPattern(re)` | Requires the key header name to match the regular expression |
| `AllowedKeyHeaders(h...)` | Requires the key header name to be one of the allowed values |

```go
mw, err := idem.New(
	idem.WithTTL(1 * time.Hour),
	idem.WithValidation(
		idem.MaxTTL(24 * time.Hour),
		idem.MinTTL(1 * time.Minute),
		idem.AllowedKeyHeaders("Idempotency-Key", "X-Request-Id"),
	),
)
```

Preset validators support custom error messages via `WithMessage`:

```go
mw, err := idem.New(
	idem.WithTTL(1 * time.Hour),
	idem.WithValidation(
		idem.MaxTTL(24 * time.Hour).WithMessage("TTL is too long for this service"),
		idem.MinTTL(1 * time.Minute).WithMessage("TTL is too short"),
	),
)
```

#### Validator Composition

Use `All` and `Any` to combine validators with AND / OR logic:

```go
mw, err := idem.New(
	idem.WithTTL(1 * time.Hour),
	idem.WithValidation(
		// All: every validator must pass (AND)
		idem.All(
			idem.MinTTL(1 * time.Minute),
			idem.MaxTTL(24 * time.Hour),
		),
		// Any: at least one validator must pass (OR)
		idem.Any(
			idem.AllowedKeyHeaders("Idempotency-Key"),
			idem.KeyHeaderPattern(regexp.MustCompile(`^X-`)),
		),
	),
)
```

`All` and `Any` return `*PresetValidator`, so they support `.WithMessage()` and can be nested arbitrarily:

```go
idem.All(
	idem.Any(v1, v2),
	v3,
).WithMessage("validation failed")
```

### Metrics

Use `WithMetrics` to observe cache hits, misses, lock contention, and errors — for example, to export to Prometheus:

```go
mw, err := idem.New(
	idem.WithMetrics(idem.Metrics{
		OnCacheHit: func(key string) {
			cacheHits.WithLabelValues(key).Inc()
		},
		OnCacheMiss: func(key string) {
			cacheMisses.WithLabelValues(key).Inc()
		},
		OnCacheSkip: func(key string, statusCode int) {
			cacheSkips.WithLabelValues(key, strconv.Itoa(statusCode)).Inc()
		},
		OnLockContention: func(key string, err error) {
			lockContentions.WithLabelValues(key).Inc()
		},
		OnError: func(key string, err error) {
			cacheErrors.WithLabelValues(key).Inc()
		},
	}),
)
```

All callback fields are optional — nil callbacks are never invoked and add no overhead. Lock contention (409 Conflict) is reported exclusively via `OnLockContention` and does not trigger `OnError`. Requests without an idempotency key bypass the middleware entirely and do not trigger any metrics callbacks.

### Configuration Inspection

Use `Middleware.Config()` to retrieve a read-only snapshot of the current configuration. This is useful for debug logging and configuration inspection endpoints.

```go
mw, _ := idem.New(
	idem.WithTTL(1 * time.Hour),
	idem.WithStorage(redisStore),
)

// Debug logging
log.Printf("idem config: %s", mw.Config())

// JSON endpoint
http.HandleFunc("/debug/idem/config", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mw.Config())
})
```

The `Config` struct includes JSON tags and implements `fmt.Stringer` for convenient serialization. The `TTL` field uses the `Duration` type, which serializes as a human-readable string (e.g. `"1h0m0s"`) instead of integer nanoseconds.

### Configuration Diff

Use `DiffConfig` to compare two `Config` snapshots and get a structured, human-readable summary of differences:

```go
old := mw.Config()

// ... reconfigure middleware ...
new := mw.Config()

diff := idem.DiffConfig(old, new)
if diff.HasDiff() {
	log.Printf("config changed:\n%s", diff)
}
// Output:
//   TTL: 24h0m0s → 1h0m0s
//   StorageType: *idem.MemoryStorage → *redis.Storage
```

## Error Handling

`New` returns sentinel errors for invalid configuration, so callers can identify specific error conditions programmatically using `errors.Is`:

```go
_, err := idem.New(idem.WithKeyHeader(""))
if errors.Is(err, idem.ErrEmptyKeyHeader) {
	// handle empty key header
}
```

### Sentinel Errors

| Package | Error | Condition |
|---------|-------|-----------|
| `idem` | `ErrEmptyKeyHeader` | Key header is empty |
| `idem` | `ErrInvalidTTL` | TTL is zero or negative |
| `idem` | `ErrNilKeyHeaderPattern` | `KeyHeaderPattern` receives a nil regexp |
| `idem/redis` | `ErrNilClient` | Redis client is nil |
| `idem/redis` | `ErrEmptyKeyPrefix` | Key prefix is empty |
| `idem/redis` | `ErrEmptyLockPrefix` | Lock prefix is empty |

## How It Works

```
Request
  │
  ▼
[Middleware]
  ├─ Extract Idempotency-Key header
  ├─ Acquire lock (if Storage implements Locker)
  │     └─ Fail → Return 409 Conflict
  ├─ Look up key in Storage
  │     ├─ Hit  → Return cached response immediately
  │     └─ Miss → Pass to next handler
  │
  ▼
[Handler executes]
  │
  ▼
[Middleware (post-response)]
  ├─ Check CacheableFunc(statusCode)
  │     ├─ true  → Store response in Storage with TTL
  │     └─ false → Skip caching (trigger OnCacheSkip if configured)
  └─ Release lock
```

Requests without an `Idempotency-Key` header pass through the middleware unchanged.

When the `Storage` implementation also implements the `Locker` interface, the middleware acquires a per-key lock before checking the cache. This prevents duplicate handler execution when concurrent requests arrive with the same idempotency key. The built-in memory and Redis storage backends implement `Locker` out of the box.

## Storage

### In-memory (default)

Used automatically when no storage is specified. Suitable for development, testing, and single-instance deployments.

```go
mw, err := idem.New() // uses in-memory storage
```

To prevent memory growth from expired entries, enable periodic background cleanup with `WithCleanupInterval`:

```go
store := idem.NewMemoryStorage(
	idem.WithCleanupInterval(5 * time.Minute),
)
mw, err := idem.New(idem.WithStorage(store))
```

When using `WithCleanupInterval`, call `Close()` to stop the background goroutine when the storage is no longer needed:

```go
defer store.Close()
```

### Redis

For multi-instance deployments where idempotency state must be shared across processes. The `idem/redis` package accepts `goredis.Cmdable`, so it works with standalone, cluster, and sentinel (failover) clients.

```go
import (
	"github.com/bright-room/idem"
	idemredis "github.com/bright-room/idem/redis"
	goredis "github.com/redis/go-redis/v9"
)

// Standalone
client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})

// Cluster
// client := goredis.NewClusterClient(&goredis.ClusterOptions{
// 	Addrs: []string{"localhost:7000", "localhost:7001", "localhost:7002"},
// })

// Sentinel (Failover)
// client := goredis.NewFailoverClient(&goredis.FailoverOptions{
// 	MasterName:    "mymaster",
// 	SentinelAddrs: []string{"localhost:26379", "localhost:26380", "localhost:26381"},
// })

store, err := idemredis.New(client)
if err != nil {
	log.Fatal(err)
}

mw, err := idem.New(idem.WithStorage(store))
```

### Custom Storage

Implement the `Storage` interface to use any backend:

```go
type Storage interface {
	Get(ctx context.Context, key string) (*idem.Response, error)
	Set(ctx context.Context, key string, res *idem.Response, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
```

To enable concurrent request locking, also implement the `Locker` interface on your storage:

```go
type Locker interface {
	Lock(ctx context.Context, key string, ttl time.Duration) (unlock func(), err error)
}
```

If your `Storage` does not implement `Locker`, the middleware operates without locking (v0.1 behavior).

## Examples

The [`_examples`](./_examples) directory contains runnable examples for popular frameworks.

### Framework Integration

| Framework | net/http compatible | Conversion method | Code example |
|-----------|:-------------------:|-------------------|--------------|
| **Chi** | ✅ | None — `mw.Handler()` works directly | `r.Use(idempotency.Handler())` |
| **Echo** | — | `echo.WrapMiddleware()` built-in adapter | `e.Use(echo.WrapMiddleware(idempotency.Handler()))` |
| **Gin** | — | `idemgin.WrapMiddleware()` official adapter | `r.POST("/orders", wrap, handler)` |

Chi is a `net/http` compatible router, so `mw.Handler()` works out of the box. Echo provides a built-in `echo.WrapMiddleware()` adapter to convert `func(http.Handler) http.Handler` into Echo middleware. Gin requires a `gin.HandlerFunc` signature; the `idem/gin` sub-module provides `WrapMiddleware()` to bridge the two.

### Gin

```go
import (
	"github.com/bright-room/idem"
	idemgin "github.com/bright-room/idem/gin"
	"github.com/gin-gonic/gin"
)

mw, _ := idem.New()
wrap := idemgin.WrapMiddleware(mw)

r := gin.Default()
r.POST("/orders", wrap, handler)
```

```bash
cd _examples/gin && go run main.go
```

```bash
# First request — handler executes and response is cached
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}

# Second request — cached response returned, handler is NOT re-executed
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}
```

See [`_examples/gin/main.go`](./_examples/gin/main.go) for the full source including per-endpoint and route-group middleware patterns.

### Echo

```bash
cd _examples/echo && go run main.go
```

```bash
# First request — handler executes and response is cached
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}

# Second request — cached response returned, handler is NOT re-executed
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}
```

See [`_examples/echo/main.go`](./_examples/echo/main.go) for the full source including global and route-group middleware patterns.

### Chi

```bash
cd _examples/chi && go run main.go
```

```bash
# First request — handler executes and response is cached
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}

# Second request — cached response returned, handler is NOT re-executed
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}
```

See [`_examples/chi/main.go`](./_examples/chi/main.go) for the full source including inline (`r.With()`) and route-group middleware patterns.

### Docker Compose (Multi-instance with Redis)

For a production-like setup with Redis storage shared across multiple instances:

```bash
cd _examples/redis-gin && docker compose up --build
```

```bash
# Request to instance 1 — handler executes, response cached in Redis
curl -X POST http://localhost:8081/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# Same key to instance 2 — cached response returned from Redis (same instance_id!)
curl -X POST http://localhost:8082/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}
```

See [`_examples/redis-gin/`](./_examples/redis-gin/) for the full setup.

### Docker Compose (Nginx Reverse Proxy)

For a production-like setup with Nginx load balancing across multiple instances:

```bash
cd _examples/nginx-redis-gin && docker compose up --build
```

```bash
# All requests go through the single Nginx endpoint
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# Same key — cached response returned regardless of which backend Nginx selects
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}
```

Nginx distributes requests via round-robin, while Redis ensures cached responses are shared across all instances.

See [`_examples/nginx-redis-gin/`](./_examples/nginx-redis-gin/) for the full setup.

### Docker Compose (Multi-instance with Redis Cluster)

For a high-availability setup with a 3-node Redis Cluster shared across multiple instances:

```bash
cd _examples/redis-cluster-gin && docker compose up --build
```

```bash
# Request to instance 1 — handler executes, response cached in Redis Cluster
curl -X POST http://localhost:8081/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# Same key to instance 2 — cached response returned from Redis Cluster (same instance_id!)
curl -X POST http://localhost:8082/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}
```

The `idem/redis` package accepts `goredis.Cmdable`, so switching from `*redis.Client` to `*redis.ClusterClient` requires no code changes.

See [`_examples/redis-cluster-gin/`](./_examples/redis-cluster-gin/) for the full setup.

### Redis Sentinel (Failover)

The `idem/redis` package also works with Redis Sentinel via `goredis.NewFailoverClient`:

```bash
cd _examples/redis-sentinel-gin && docker compose up --build
```

```go
client := goredis.NewFailoverClient(&goredis.FailoverOptions{
	MasterName:    "mymaster",
	SentinelAddrs: []string{"localhost:26379", "localhost:26380", "localhost:26381"},
})

store, err := idemredis.New(client)
```

```bash
# 1. Send a request to app-1 — handler executes, response cached via Sentinel master
curl -X POST http://localhost:8081/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# 2. Same key to app-2 — cached response returned (note: instance_id is still app-1)
curl -X POST http://localhost:8082/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}
```

See [`_examples/redis-sentinel-gin/`](./_examples/redis-sentinel-gin/) for the full setup.

### Docker Compose (Prometheus Metrics)

Export idempotency metrics to Prometheus using `WithMetrics`:

```bash
cd _examples/prometheus-gin && docker compose up --build
```

```bash
# Send requests to generate cache hits and misses
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"

# Check metrics
curl http://localhost:8080/metrics | grep idem_
```

Open Prometheus UI at [http://localhost:9090](http://localhost:9090) to query `idem_cache_hits_total`, `idem_cache_misses_total`, and more.

See [`_examples/prometheus-gin/`](./_examples/prometheus-gin/) for the full setup.