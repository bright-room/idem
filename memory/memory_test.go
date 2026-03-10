package memory_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/bright-room/idem"
	"github.com/bright-room/idem/memory"
)

func TestStorage_Get(t *testing.T) {
	t.Parallel()

	res := &idem.Response{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"ok":true}`),
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T, s *memory.Storage)
		key     string
		wantRes *idem.Response
	}{
		{
			name: "returns cached response for existing key",
			setup: func(t *testing.T, s *memory.Storage) {
				t.Helper()
				if err := s.Set(context.Background(), "key-1", res, time.Hour); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
			},
			key:     "key-1",
			wantRes: res,
		},
		{
			name:    "returns nil for non-existent key",
			setup:   func(t *testing.T, _ *memory.Storage) { t.Helper() },
			key:     "unknown",
			wantRes: nil,
		},
		{
			name: "returns nil after TTL has expired",
			setup: func(t *testing.T, s *memory.Storage) {
				t.Helper()
				if err := s.Set(context.Background(), "expired", res, time.Nanosecond); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
				time.Sleep(time.Millisecond)
			},
			key:     "expired",
			wantRes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := memory.New()
			tt.setup(t, s)

			got, err := s.Get(context.Background(), tt.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if tt.wantRes == nil {
				if got != nil {
					t.Errorf("Get() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("Get() = nil, want non-nil")
			}
			if got.StatusCode != tt.wantRes.StatusCode {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, tt.wantRes.StatusCode)
			}
			if http.Header(got.Header).Get("Content-Type") != http.Header(tt.wantRes.Header).Get("Content-Type") {
				t.Errorf("Header Content-Type = %q, want %q",
					http.Header(got.Header).Get("Content-Type"), http.Header(tt.wantRes.Header).Get("Content-Type"))
			}
			if string(got.Body) != string(tt.wantRes.Body) {
				t.Errorf("Body = %q, want %q", got.Body, tt.wantRes.Body)
			}
		})
	}
}

func TestStorage_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	s := memory.New()
	ctx := context.Background()

	var wg sync.WaitGroup
	const goroutines = 100

	for i := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			key := "key"
			res := &idem.Response{
				StatusCode: http.StatusOK + i,
				Body:       []byte("body"),
			}

			_ = s.Set(ctx, key, res, time.Hour)
			_, _ = s.Get(ctx, key)
		}()
	}

	wg.Wait()
}
