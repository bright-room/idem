package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bright-room/idem"
	goredis "github.com/redis/go-redis/v9"
)

const defaultKeyPrefix = "idem:"

// Storage is a Redis-backed implementation of idem.Storage.
type Storage struct {
	client    goredis.Cmdable
	keyPrefix string
}

// Option configures a Redis Storage.
type Option func(*Storage)

// WithKeyPrefix sets the prefix for Redis keys.
// Default is "idem:".
func WithKeyPrefix(prefix string) Option {
	return func(s *Storage) {
		s.keyPrefix = prefix
	}
}

// New creates a new Redis Storage.
func New(client goredis.Cmdable, opts ...Option) *Storage {
	s := &Storage{
		client:    client,
		keyPrefix: defaultKeyPrefix,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Get returns the cached response for the given key.
// If the key does not exist, it returns nil, nil.
func (s *Storage) Get(ctx context.Context, key string) (*idem.Response, error) {
	data, err := s.client.Get(ctx, s.keyPrefix+key).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var res idem.Response
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
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
