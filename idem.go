package idem

import (
	"context"
	"sync"
	"time"
)

// New creates a new idempotency Middleware with the given options.
func New(opts ...Option) *Middleware {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.storage == nil {
		cfg.storage = NewMemoryStorage()
	}

	return &Middleware{cfg: cfg}
}

type memoryEntry struct {
	res       *Response
	expiresAt time.Time
}

// MemoryStorage is an in-memory implementation of Storage.
// It also implements Locker for per-key mutual exclusion.
type MemoryStorage struct {
	mu      sync.RWMutex
	entries map[string]*memoryEntry
	locks   sync.Map
}

// NewMemoryStorage creates a new in-memory Storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		entries: make(map[string]*memoryEntry),
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
		delete(s.entries, key)
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
