# Redis Cluster + Gin Example

A multi-instance setup demonstrating idempotency key sharing via a 3-node Redis Cluster. Two Gin application instances share a Redis Cluster backend, so a cached response from one instance is returned by the other — with data distributed across cluster nodes.

Unlike the [redis-gin](../redis-gin/) example (single Redis), this example uses a Redis Cluster for high availability and distributed caching. The `idem/redis` package accepts `goredis.Cmdable`, so switching from `*redis.Client` to `*redis.ClusterClient` requires no code changes to the idem integration.

## Run

```bash
docker compose up --build
```

This starts:
- **app-1** on `localhost:8081`
- **app-2** on `localhost:8082`
- **Redis Cluster** with 3 nodes (ports 7000–7002)
- **redis-cluster-init** initializes the cluster (runs once and exits)

## Try it

```bash
# 1. Send a request to app-1 — handler executes, response cached in Redis Cluster
curl -X POST http://localhost:8081/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# 2. Same key to app-2 — cached response returned from Redis Cluster (note: instance_id is still app-1)
curl -X POST http://localhost:8082/orders -H "Idempotency-Key: key-123"
# => {"instance_id":"app-1","message":"order created","order_id":"order-1"}

# 3. Different key to app-2 — new response from app-2
curl -X POST http://localhost:8082/orders -H "Idempotency-Key: key-456"
# => {"instance_id":"app-2","message":"order created","order_id":"order-1"}
```

## Stop

```bash
docker compose down
```
