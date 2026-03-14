package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/bright-room/idem"
	goredis "github.com/redis/go-redis/v9"
)

const (
	defaultKeyPrefix  = "idem:"
	defaultLockPrefix = "idem:lock:"
	lockRetryInterval = 50 * time.Millisecond
)

// luaUnlockScript atomically deletes a lock key only if its value matches.
var luaUnlockScript = goredis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

// Storage is a Redis-backed implementation of idem.Storage and idem.Locker.
type Storage struct {
	client     goredis.Cmdable
	keyPrefix  string
	lockPrefix string
}

// Option configures a Redis Storage.
type Option func(*Storage)

// WithKeyPrefix sets the prefix for Redis keys.
// Default is "idem."
func WithKeyPrefix(prefix string) Option {
	return func(s *Storage) {
		s.keyPrefix = prefix
	}
}

// WithLockPrefix sets the prefix for Redis lock keys.
// Default is "idem:lock:".
func WithLockPrefix(prefix string) Option {
	return func(s *Storage) {
		s.lockPrefix = prefix
	}
}

// New creates a new Redis Storage.
// It returns an error if the configuration is invalid
// (e.g. empty keyPrefix or lockPrefix).
func New(client goredis.Cmdable, opts ...Option) (*Storage, error) {
	s := &Storage{
		client:     client,
		keyPrefix:  defaultKeyPrefix,
		lockPrefix: defaultLockPrefix,
	}
	for _, opt := range opts {
		opt(s)
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	return s, nil
}

// validate checks the Storage configuration for invalid values.
func (s *Storage) validate() error {
	if s.client == nil {
		return errors.New("redis: client must not be nil")
	}

	if s.keyPrefix == "" {
		return errors.New("redis: keyPrefix must not be empty")
	}

	if s.lockPrefix == "" {
		return errors.New("redis: lockPrefix must not be empty")
	}

	return nil
}

// Get returns the cached response for the given key.
// If the key does not exist, it returns nil, nil.
func (s *Storage) Get(ctx context.Context, key string) (*idem.Response, error) {
	data, err := s.client.Get(ctx, s.keyPrefix+key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var res idem.Response
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	// StatusCode 0 is not a valid HTTP status. This guards against
	// corrupt or legacy cache entries (e.g. serialized without JSON
	// tags) where all fields decode to zero values.
	if res.StatusCode == 0 {
		return nil, nil
	}

	return &res, nil
}

// Set stores the response for the given key with the specified TTL.
func (s *Storage) Set(ctx context.Context, key string, res *idem.Response, ttl time.Duration) error {
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, s.keyPrefix+key, data, ttl).Err()
}

// Delete removes the cached response for the given key.
// If the key does not exist, it returns nil.
func (s *Storage) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.keyPrefix+key).Err()
}

// Lock acquires a distributed lock for the given key using Redis SET NX.
// It retries until the lock is acquired or the context is cancelled.
// The returned unlock function releases the lock atomically using a Lua script,
// ensuring only the owner can release the lock.
func (s *Storage) Lock(ctx context.Context, key string, ttl time.Duration) (func(), error) {
	lockKey := s.lockPrefix + key
	lockValue := generateLockValue()

	for {
		ok, err := s.client.SetNX(ctx, lockKey, lockValue, ttl).Result()
		if err != nil {
			return nil, err
		}
		if ok {
			return func() {
				// Best-effort unlock: the Locker interface defines unlock as func()
				// so errors cannot be propagated. If the lock TTL has already expired,
				// the Lua script returns 0 (no-op) which is safe.
				_ = luaUnlockScript.Run(context.Background(), s.client, []string{lockKey}, lockValue).Err()
			}, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockRetryInterval):
		}
	}
}

func generateLockValue() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("idem/redis: crypto/rand.Read failed: " + err.Error())
	}

	return hex.EncodeToString(b)
}
