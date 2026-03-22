package idem

import (
	"context"
	"sync"
	"time"
)

// New creates a new idempotency Middleware with the given options.
// It returns an error if the configuration is invalid
// (e.g. empty keyHeader or non-positive ttl).
func New(opts ...Option) (*Middleware, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if cfg.storage == nil {
		cfg.storage = NewMemoryStorage()
	}

	return &Middleware{cfg: cfg}, nil
}

type memoryEntry struct {
	res       *Response
	expiresAt time.Time
}

// MemoryStorage is an in-memory implementation of Storage.
// It also implements Locker for per-key mutual exclusion.
//
// For production environments, consider using the redis.Storage backend
// which provides automatic TTL-based expiration without additional configuration.
type MemoryStorage struct {
	mu              sync.RWMutex
	entries         map[string]*memoryEntry
	locks           sync.Map
	cleanupInterval time.Duration
	done            chan struct{}
}

// MemoryStorageOption configures a MemoryStorage.
type MemoryStorageOption func(*MemoryStorage)

// WithCleanupInterval sets the interval for background cleanup of expired entries.
// When set to a positive duration, a background goroutine periodically removes
// expired entries to prevent memory growth from unused keys.
// Call Close to stop the background goroutine.
func WithCleanupInterval(d time.Duration) MemoryStorageOption {
	return func(s *MemoryStorage) {
		s.cleanupInterval = d
	}
}

// NewMemoryStorage creates a new in-memory Storage.
func NewMemoryStorage(opts ...MemoryStorageOption) *MemoryStorage {
	s := &MemoryStorage{
		entries: make(map[string]*memoryEntry),
		done:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.cleanupInterval > 0 {
		go s.startCleanup()
	}

	return s
}

// Close stops the background cleanup goroutine, if running.
func (s *MemoryStorage) Close() error {
	select {
	case <-s.done:
		// already closed
	default:
		close(s.done)
	}

	return nil
}

func (s *MemoryStorage) startCleanup() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.deleteExpired()
		case <-s.done:
			return
		}
	}
}

func (s *MemoryStorage) deleteExpired() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, key)
			s.locks.Delete(key)
		}
	}
}

// Get returns the cached response for the given key.
// If the key does not exist or has expired, it returns nil, nil.
func (s *MemoryStorage) Get(_ context.Context, key string) (*Response, error) {
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	if time.Now().After(e.expiresAt) {
		s.mu.Lock()
		if current, exists := s.entries[key]; exists && current == e {
			delete(s.entries, key)
		}
		s.mu.Unlock()

		return nil, nil
	}

	return e.res, nil
}

// Set stores the response for the given key with the specified TTL.
func (s *MemoryStorage) Set(_ context.Context, key string, res *Response, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = &memoryEntry{
		res:       res,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes the cached response for the given key.
// If the key does not exist, it returns nil.
func (s *MemoryStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, key)

	return nil
}

// Lock acquires a per-key mutex lock.
// The TTL parameter is ignored for in-memory locking since the mutex
// is released explicitly via the returned unlock function.
func (s *MemoryStorage) Lock(ctx context.Context, key string, _ time.Duration) (func(), error) {
	v, _ := s.locks.LoadOrStore(key, &sync.Mutex{})
	mu := v.(*sync.Mutex)

	locked := make(chan struct{})
	go func() {
		mu.Lock()
		close(locked)
	}()

	select {
	case <-locked:
		return func() { mu.Unlock() }, nil
	case <-ctx.Done():
		go func() {
			<-locked
			mu.Unlock()
		}()
		return nil, ctx.Err()
	}
}
