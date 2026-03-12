# Prometheus + Gin Example

Demonstrates how to use `idem.WithMetrics` to export idempotency middleware metrics to Prometheus. The application uses in-memory storage and exposes counters for cache hits, misses, lock contentions, and storage errors.

## Run

```bash
docker compose up --build
```

This starts:
- **app** on `localhost:8080` — Gin application with idem middleware and `/metrics` endpoint
- **prometheus** on `localhost:9090` — Prometheus server scraping the app every 5 seconds

## Try it

```bash
# 1. Send a request — cache miss, handler executes
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}

# 2. Same key — cache hit, cached response returned
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"message":"order created","order_id":"order-1"}

# 3. Check metrics
curl http://localhost:8080/metrics | grep idem_
# => idem_cache_hits_total 1
# => idem_cache_misses_total 1
```

Open Prometheus UI at [http://localhost:9090](http://localhost:9090) and query:

- `idem_cache_hits_total`
- `idem_cache_misses_total`
- `idem_lock_contentions_total`
- `idem_storage_errors_total`

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `idem_cache_hits_total` | Counter | Idempotency cache hits (cached response returned) |
| `idem_cache_misses_total` | Counter | Idempotency cache misses (handler executed) |
| `idem_lock_contentions_total` | Counter | Lock contentions resulting in 409 Conflict |
| `idem_storage_errors_total` | Counter | Storage operation errors |

## Stop

```bash
docker compose down
```
