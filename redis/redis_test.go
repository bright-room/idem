package redis_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bright-room/idem"
	iredis "github.com/bright-room/idem/redis"
	goredis "github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) goredis.Cmdable {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { client.Close() })

	return client
}

func TestIntegration_Storage_SetAndGet(t *testing.T) {
	client := newTestClient(t)
	s := iredis.New(client)
	ctx := context.Background()
	key := "test-set-and-get"

	t.Cleanup(func() { client.Del(ctx, "idem:"+key) })

	want := &idem.Response{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"ok":true}`),
	}

	if err := s.Set(ctx, key, want, time.Hour); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil {
		t.Fatal("Get() = nil, want non-nil")
	}
	if got.StatusCode != want.StatusCode {
		t.Errorf("StatusCode = %d, want %d", got.StatusCode, want.StatusCode)
	}
	if http.Header(got.Header).Get("Content-Type") != http.Header(want.Header).Get("Content-Type") {
		t.Errorf("Header Content-Type = %q, want %q",
			http.Header(got.Header).Get("Content-Type"), http.Header(want.Header).Get("Content-Type"))
	}
	if string(got.Body) != string(want.Body) {
		t.Errorf("Body = %q, want %q", got.Body, want.Body)
	}
}

func TestIntegration_Storage_GetReturnsNilForNonExistentKey(t *testing.T) {
	client := newTestClient(t)
	s := iredis.New(client)

	got, err := s.Get(context.Background(), "non-existent-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Errorf("Get() = %v, want nil", got)
	}
}

func TestIntegration_Storage_GetReturnsNilAfterTTLExpired(t *testing.T) {
	client := newTestClient(t)
	s := iredis.New(client)
	ctx := context.Background()
	key := "test-ttl-expired"

	t.Cleanup(func() { client.Del(ctx, "idem:"+key) })

	res := &idem.Response{
		StatusCode: http.StatusOK,
		Body:       []byte("expired"),
	}

	if err := s.Set(ctx, key, res, 100*time.Millisecond); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	got, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Errorf("Get() = %v, want nil", got)
	}
}

func TestIntegration_Storage_WithKeyPrefix(t *testing.T) {
	client := newTestClient(t)
	prefix := "custom:"
	s := iredis.New(client, iredis.WithKeyPrefix(prefix))
	ctx := context.Background()
	key := "test-prefix"

	t.Cleanup(func() { client.Del(ctx, prefix+key) })

	res := &idem.Response{
		StatusCode: http.StatusCreated,
		Body:       []byte("created"),
	}

	if err := s.Set(ctx, key, res, time.Hour); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	keys, err := client.Keys(ctx, prefix+"*").Result()
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	found := false
	for _, k := range keys {
		if k == prefix+key {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected key %q in Redis, got keys: %v", prefix+key, keys)
	}
}

func TestIntegration_Storage_GetReturnsErrorOnConnectionFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := goredis.NewClient(&goredis.Options{Addr: "localhost:1"})
	t.Cleanup(func() { client.Close() })

	s := iredis.New(client)

	_, err := s.Get(context.Background(), "any-key")
	if err == nil {
		t.Fatal("Get() error = nil, want connection error")
	}
}
