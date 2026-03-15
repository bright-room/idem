package idem_test

import (
	"errors"
	"testing"

	"github.com/bright-room/idem"
)

func TestNew_sentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrEmptyKeyHeader for empty key header", func(t *testing.T) {
		t.Parallel()

		_, err := idem.New(idem.WithKeyHeader(""))
		if !errors.Is(err, idem.ErrEmptyKeyHeader) {
			t.Errorf("error = %v, want %v", err, idem.ErrEmptyKeyHeader)
		}
	})

	t.Run("returns ErrInvalidTTL for zero TTL", func(t *testing.T) {
		t.Parallel()

		_, err := idem.New(idem.WithTTL(0))
		if !errors.Is(err, idem.ErrInvalidTTL) {
			t.Errorf("error = %v, want %v", err, idem.ErrInvalidTTL)
		}
	})

	t.Run("returns ErrNilKeyHeaderPattern for nil pattern", func(t *testing.T) {
		t.Parallel()

		_, err := idem.New(idem.WithValidation(idem.KeyHeaderPattern(nil)))
		if !errors.Is(err, idem.ErrNilKeyHeaderPattern) {
			t.Errorf("error = %v, want %v", err, idem.ErrNilKeyHeaderPattern)
		}
	})
}
