package redis_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func newTestClusterClient(t *testing.T) goredis.Cmdable {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	addrs := os.Getenv("REDIS_CLUSTER_ADDRS")
	if addrs == "" {
		t.Skip("REDIS_CLUSTER_ADDRS not set, skipping cluster test")
	}

	client := goredis.NewClusterClient(&goredis.ClusterOptions{
		Addrs: strings.Split(addrs, ","),
	})
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func newTestSentinelClient(t *testing.T) goredis.Cmdable {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	addrs := os.Getenv("REDIS_SENTINEL_ADDRS")
	if addrs == "" {
		t.Skip("REDIS_SENTINEL_ADDRS not set, skipping sentinel test")
	}

	masterName := os.Getenv("REDIS_SENTINEL_MASTER")
	if masterName == "" {
		masterName = "mymaster"
	}

	client := goredis.NewFailoverClient(&goredis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: strings.Split(addrs, ","),
	})
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func newTestStorage(t *testing.T, client goredis.Cmdable, opts ...iredis.Option) *iredis.Storage {
	t.Helper()

	s, err := iredis.New(client, opts...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return s
}

// clientFactory creates a Redis client for testing.
type clientFactory func(t *testing.T) goredis.Cmdable

// runStorageTests runs the common Storage and Locker integration tests
// using the given client factory. The name parameter is used as a subtest
// prefix and as part of Redis key names to avoid collisions between environments.
func runStorageTests(t *testing.T, name string, factory clientFactory) {
	t.Helper()

	keyPrefix := strings.ToLower(name)

	t.Run(name, func(t *testing.T) {
		t.Run("SetAndGet", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)
			ctx := context.Background()
			key := keyPrefix + "-set-and-get"

			t.Cleanup(func() { client.Del(ctx, "idem:"+key) })

			want := &idem.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
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
			if got.Header.Get("Content-Type") != want.Header.Get("Content-Type") {
				t.Errorf("Header Content-Type = %q, want %q",
					got.Header.Get("Content-Type"), want.Header.Get("Content-Type"))
			}
			if string(got.Body) != string(want.Body) {
				t.Errorf("Body = %q, want %q", got.Body, want.Body)
			}
		})

		t.Run("GetReturnsNilForNonExistentKey", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)

			got, err := s.Get(context.Background(), keyPrefix+"-non-existent-key")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if got != nil {
				t.Errorf("Get() = %v, want nil", got)
			}
		})

		t.Run("GetReturnsNilAfterTTLExpired", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)
			ctx := context.Background()
			key := keyPrefix + "-ttl-expired"

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
		})

		t.Run("Delete", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)
			ctx := context.Background()
			key := keyPrefix + "-delete"

			t.Cleanup(func() { client.Del(ctx, "idem:"+key) })

			res := &idem.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       []byte(`{"ok":true}`),
			}

			if err := s.Set(ctx, key, res, time.Hour); err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			if err := s.Delete(ctx, key); err != nil {
				t.Fatalf("Delete() error = %v", err)
			}

			got, err := s.Get(ctx, key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if got != nil {
				t.Errorf("Get() = %v, want nil", got)
			}
		})

		t.Run("DeleteNonExistentKey", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)

			if err := s.Delete(context.Background(), keyPrefix+"-non-existent-delete-key"); err != nil {
				t.Errorf("Delete() error = %v, want nil", err)
			}
		})

		t.Run("LockAndUnlock", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)
			ctx := context.Background()
			key := keyPrefix + "-lock-basic"

			t.Cleanup(func() { client.Del(ctx, "idem:lock:"+key) })

			unlock, err := s.Lock(ctx, key, 5*time.Second)
			if err != nil {
				t.Fatalf("Lock() error = %v", err)
			}

			unlock()
		})

		t.Run("LockBlocksConcurrentAccess", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)
			ctx := context.Background()
			key := keyPrefix + "-lock-concurrent"

			t.Cleanup(func() { client.Del(ctx, "idem:lock:"+key) })

			var concurrent atomic.Int32
			var maxConcurrent atomic.Int32

			var wg sync.WaitGroup
			const goroutines = 5

			for range goroutines {
				wg.Add(1)
				go func() {
					defer wg.Done()

					unlock, err := s.Lock(ctx, key, 5*time.Second)
					if err != nil {
						t.Errorf("Lock() error = %v", err)
						return
					}

					cur := concurrent.Add(1)
					for {
						old := maxConcurrent.Load()
						if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
							break
						}
					}
					time.Sleep(10 * time.Millisecond)
					concurrent.Add(-1)

					unlock()
				}()
			}

			wg.Wait()

			if got := maxConcurrent.Load(); got != 1 {
				t.Errorf("max concurrent locks = %d, want 1", got)
			}
		})

		t.Run("LockRespectsContextCancellation", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)
			ctx := context.Background()
			key := keyPrefix + "-lock-cancel"

			t.Cleanup(func() { client.Del(ctx, "idem:lock:"+key) })

			// Hold the lock
			unlock, err := s.Lock(ctx, key, 5*time.Second)
			if err != nil {
				t.Fatalf("Lock() error = %v", err)
			}
			defer unlock()

			// Try to acquire with short timeout
			cancelCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()

			_, err = s.Lock(cancelCtx, key, 5*time.Second)
			if err == nil {
				t.Fatal("Lock() error = nil, want context error")
			}
		})

		t.Run("LockTTLExpiration", func(t *testing.T) {
			client := factory(t)
			s := newTestStorage(t, client)
			ctx := context.Background()
			key := keyPrefix + "-lock-ttl"

			t.Cleanup(func() { client.Del(ctx, "idem:lock:"+key) })

			// Acquire lock with short TTL (don't unlock manually)
			_, err := s.Lock(ctx, key, 200*time.Millisecond)
			if err != nil {
				t.Fatalf("first Lock() error = %v", err)
			}

			// Wait for TTL to expire
			time.Sleep(300 * time.Millisecond)

			// Should be able to acquire lock now
			unlock, err := s.Lock(ctx, key, 5*time.Second)
			if err != nil {
				t.Fatalf("second Lock() error = %v", err)
			}
			unlock()
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	// stub client for validation-only tests (no actual Redis connection needed)
	stub := goredis.NewClient(&goredis.Options{})

	t.Run("returns error for nil client", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(nil)
		if err == nil {
			t.Fatal("New(nil) error = nil, want error")
		}
	})

	t.Run("accepts default config", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(stub)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
	})

	t.Run("accepts valid custom prefixes", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(stub, iredis.WithKeyPrefix("custom:"), iredis.WithLockPrefix("custom:lock:"))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
	})

	t.Run("returns error for empty keyPrefix", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(stub, iredis.WithKeyPrefix(""))
		if err == nil {
			t.Fatal("New() error = nil, want error")
		}
	})

	t.Run("returns error for empty lockPrefix", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(stub, iredis.WithLockPrefix(""))
		if err == nil {
			t.Fatal("New() error = nil, want error")
		}
	})
}

// --- Common integration tests (Standalone + Cluster) ---

func TestIntegration_Storage(t *testing.T) {
	runStorageTests(t, "Storage", newTestClient)
}

func TestIntegration_ClusterStorage(t *testing.T) {
	runStorageTests(t, "ClusterStorage", newTestClusterClient)
}

func TestIntegration_SentinelStorage(t *testing.T) {
	runStorageTests(t, "SentinelStorage", newTestSentinelClient)
}

// --- Standalone-only integration tests ---

func TestIntegration_Storage_WithKeyPrefix(t *testing.T) {
	client := newTestClient(t)
	prefix := "custom:"
	s := newTestStorage(t, client, iredis.WithKeyPrefix(prefix))
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

func TestIntegration_Storage_DeleteWithKeyPrefix(t *testing.T) {
	client := newTestClient(t)
	prefix := "custom:"
	s := newTestStorage(t, client, iredis.WithKeyPrefix(prefix))
	ctx := context.Background()
	key := "test-delete-prefix"

	t.Cleanup(func() { client.Del(ctx, prefix+key) })

	res := &idem.Response{
		StatusCode: http.StatusOK,
		Body:       []byte("data"),
	}

	if err := s.Set(ctx, key, res, time.Hour); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Errorf("Get() = %v, want nil", got)
	}
}

func TestIntegration_Storage_GetReturnsErrorOnConnectionFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := goredis.NewClient(&goredis.Options{Addr: "localhost:1"})
	t.Cleanup(func() { _ = client.Close() })

	s := newTestStorage(t, client)

	_, err := s.Get(context.Background(), "any-key")
	if err == nil {
		t.Fatal("Get() error = nil, want connection error")
	}
}

func TestIntegration_Storage_ImplementsLocker(t *testing.T) {
	client := newTestClient(t)

	var s interface{} = newTestStorage(t, client)
	if _, ok := s.(idem.Locker); !ok {
		t.Error("redis.Storage does not implement idem.Locker")
	}
}
