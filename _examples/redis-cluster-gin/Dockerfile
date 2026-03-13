FROM golang:1.26.1 AS builder

WORKDIR /src

# Copy root module (idem library)
COPY go.mod go.sum ./
COPY *.go ./
COPY redis/ redis/

# Copy example module
COPY _examples/redis-cluster-gin/ ./_examples/redis-cluster-gin/

WORKDIR /src/_examples/redis-cluster-gin
RUN go mod download
RUN CGO_ENABLED=0 go build -o /app .

FROM alpine:3
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
