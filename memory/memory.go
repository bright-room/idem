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
type Storage struct {
	mu      sync.RWMutex
	entries map[string]*entry
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
