package redis_test

import (
	"errors"
	"testing"

	iredis "github.com/bright-room/idem/redis"
	goredis "github.com/redis/go-redis/v9"
)

func TestNew_sentinelErrors(t *testing.T) {
	t.Parallel()

	// stub client for validation-only tests (no actual Redis connection needed)
	stub := goredis.NewClient(&goredis.Options{})

	t.Run("returns ErrNilClient for nil client", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(nil)
		if !errors.Is(err, iredis.ErrNilClient) {
			t.Errorf("error = %v, want %v", err, iredis.ErrNilClient)
		}
	})

	t.Run("returns ErrEmptyKeyPrefix for empty key prefix", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(stub, iredis.WithKeyPrefix(""))
		if !errors.Is(err, iredis.ErrEmptyKeyPrefix) {
			t.Errorf("error = %v, want %v", err, iredis.ErrEmptyKeyPrefix)
		}
	})

	t.Run("returns ErrEmptyLockPrefix for empty lock prefix", func(t *testing.T) {
		t.Parallel()

		_, err := iredis.New(stub, iredis.WithLockPrefix(""))
		if !errors.Is(err, iredis.ErrEmptyLockPrefix) {
			t.Errorf("error = %v, want %v", err, iredis.ErrEmptyLockPrefix)
		}
	})
}
