# idem

Idempotency key middleware for Go HTTP applications.

> **"One header, zero duplicates."**
> — Prevent duplicate request execution with a single HTTP header.

## Features

- **Framework-agnostic** — Built on `net/http`, works with Gin, Echo, Chi, and any compatible router
- **Pluggable storage** — Interface-based design with built-in memory and Redis implementations
- **Zero config** — Works out of the box with sensible defaults
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
	"net/http"

	"github.com/bright-room/idem"
)

func main() {
	mw := idem.New()

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
mw := idem.New(
	idem.WithKeyHeader("X-Request-Id"),
	idem.WithTTL(1 * time.Hour),
	idem.WithStorage(redisStore),
	idem.WithOnError(func(err error) {
		log.Printf("storage error: %v", err)
	}),
)
```

## How It Works

```
Request
  │
  ▼
[Middleware]
  ├─ Extract Idempotency-Key header
  ├─ Look up key in Storage
  │     ├─ Hit  → Return cached response immediately
  │     └─ Miss → Pass to next handler
  │
  ▼
[Handler executes]
  │
  ▼
[Middleware (post-response)]
  └─ Store response in Storage with TTL
```

Requests without an `Idempotency-Key` header pass through the middleware unchanged.

## Storage

### In-memory (default)

Used automatically when no storage is specified. Suitable for development, testing, and single-instance deployments.

```go
mw := idem.New() // uses in-memory storage
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
store := idemredis.New(client)

mw := idem.New(idem.WithStorage(store))
```

### Custom Storage

Implement the `Storage` interface to use any backend:

```go
type Storage interface {
	Get(ctx context.Context, key string) (*idem.Response, error)
	Set(ctx context.Context, key string, res *idem.Response, ttl time.Duration) error
}
```

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
| v0.4 | Planned | Concurrent request handling (lock mechanism) |
| v1.0 | Planned | Documentation + stable release |
