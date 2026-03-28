package idem

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a [time.Duration] that serializes to and from a human-readable
// string in JSON (e.g. "1h30m0s") instead of integer nanoseconds.
type Duration time.Duration

// MarshalJSON encodes d as a quoted duration string (e.g. "1h0m0s").
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON decodes a quoted duration string (e.g. "1h0m0s") or an
// integer nanosecond value into d.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return err
		}

		*d = Duration(parsed)

		return nil
	}

	var ns int64
	if err := json.Unmarshal(b, &ns); err != nil {
		return fmt.Errorf("idem: Duration must be a string or integer, got %s", string(b))
	}

	*d = Duration(time.Duration(ns))

	return nil
}

// String returns the duration formatted as a string (e.g. "1h0m0s").
func (d Duration) String() string {
	return time.Duration(d).String()
}
