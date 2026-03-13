# Nginx + Redis + Gin Example

Demonstrates idempotency key handling behind an Nginx reverse proxy. Nginx distributes requests across two Gin application instances via round-robin, while Redis ensures cached responses are shared.

## Architecture

```
Client ──► Nginx (:8080) ──┬──► app-1 ──► Redis
                           └──► app-2 ──┘
```

## Run

```bash
docker compose up --build
```

This starts:
- **Nginx** on `localhost:8080` (reverse proxy)
- **app-1** and **app-2** (not exposed externally)
- **Redis** as the shared storage backend

## Try it

```bash
# All requests go through the single Nginx endpoint
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# Same key — cached response returned regardless of which backend Nginx selects
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# Different key — new response from whichever instance handles it
curl -X POST http://localhost:8080/orders -H "Idempotency-Key: key-456"
# => {"instance_id":"app-2","message":"order created","order_id":"order-1"}
```

## Stop

```bash
docker compose down
```
