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
		cfg.storage = newDefaultStorage()
	}

	return &Middleware{cfg: cfg}
}

type defaultEntry struct {
	res       *Response
	expiresAt time.Time
}

type defaultStorage struct {
	mu      sync.RWMutex
	entries map[string]*defaultEntry
}

func newDefaultStorage() *defaultStorage {
	return &defaultStorage{
		entries: make(map[string]*defaultEntry),
	}
}

func (s *defaultStorage) Get(_ context.Context, key string) (*Response, error) {
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

func (s *defaultStorage) Set(_ context.Context, key string, res *Response, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = &defaultEntry{
		res:       res,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}
