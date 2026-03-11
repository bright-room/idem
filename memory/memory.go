package memory

import (
	"context"
	"sync"
	"time"

	"github.com/bright-room/idem"
)

type entry struct {
	res       *idem.Response
	expiresAt time.Time
}

// Storage is an in-memory implementation of idem.Storage.
// It also implements idem.Locker for per-key mutual exclusion.
type Storage struct {
	mu      sync.RWMutex
	entries map[string]*entry
	locks   sync.Map
}

// New creates a new in-memory Storage.
func New() *Storage {
	return &Storage{
		entries: make(map[string]*entry),
	}
}

// Get returns the cached response for the given key.
// If the key does not exist or has expired, it returns nil, nil.
func (s *Storage) Get(_ context.Context, key string) (*idem.Response, error) {
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
func (s *Storage) Set(_ context.Context, key string, res *idem.Response, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = &entry{
		res:       res,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Lock acquires a per-key mutex lock.
// The TTL parameter is ignored for in-memory locking since the mutex
// is released explicitly via the returned unlock function.
func (s *Storage) Lock(ctx context.Context, key string, _ time.Duration) (func(), error) {
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
