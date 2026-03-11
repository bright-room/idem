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
| `WithOnError(fn)` | `nil` | Callback invoked when a storage operation fails |

```go
mw, err := idem.New(
	idem.WithKeyHeader("X-Request-Id"),
	idem.WithTTL(1 * time.Hour),
	idem.WithStorage(redisStore),
	idem.WithOnError(func(err error) {
		log.Printf("storage error: %v", err)
	}),
)
if err != nil {
	log.Fatal(err)
}
```

`New` validates the configuration and returns an error for invalid values such as an empty key header or a non-positive TTL.

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
  ├─ Store response in Storage with TTL
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

### Redis

For multi-instance deployments where idempotency state must be shared across processes.

```go
import (
	"github.com/bright-room/idem"
	idemredis "github.com/bright-room/idem/redis"
	goredis "github.com/redis/go-redis/v9"
)

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
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

### Gin

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

## Roadmap

| Phase | Status | Description |
|-------|--------|-------------|
| v0.1 | Planned | Core middleware + in-memory storage |
| v0.2 | **Done** | Redis storage |
| v0.3 | **Done** | Framework examples (Gin / Echo / Chi) |
| v0.4 | **Done** | Concurrent request handling (lock mechanism) |
| v1.0 | Planned | Documentation + stable release |
