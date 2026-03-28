package idem

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    Duration
		want string
	}{
		{
			name: "zero",
			d:    0,
			want: `"0s"`,
		},
		{
			name: "24 hours",
			d:    Duration(24 * time.Hour),
			want: `"24h0m0s"`,
		},
		{
			name: "5 minutes",
			d:    Duration(5 * time.Minute),
			want: `"5m0s"`,
		},
		{
			name: "mixed",
			d:    Duration(1*time.Hour + 30*time.Minute + 15*time.Second),
			want: `"1h30m15s"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.d)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Duration
		wantErr bool
	}{
		{
			name:  "zero",
			input: `"0s"`,
			want:  0,
		},
		{
			name:  "24 hours",
			input: `"24h0m0s"`,
			want:  Duration(24 * time.Hour),
		},
		{
			name:  "5 minutes shorthand",
			input: `"5m"`,
			want:  Duration(5 * time.Minute),
		},
		{
			name:    "invalid string",
			input:   `"invalid"`,
			wantErr: true,
		},
		{
			name:    "non-string value",
			input:   `123`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got Duration
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDuration_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    Duration
		want string
	}{
		{
			name: "zero",
			d:    0,
			want: "0s",
		},
		{
			name: "24 hours",
			d:    Duration(24 * time.Hour),
			want: "24h0m0s",
		},
		{
			name: "5 minutes",
			d:    Duration(5 * time.Minute),
			want: "5m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.d.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDuration_roundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    Duration
	}{
		{name: "zero", d: 0},
		{name: "1 hour", d: Duration(1 * time.Hour)},
		{name: "24 hours", d: Duration(24 * time.Hour)},
		{name: "complex", d: Duration(2*time.Hour + 15*time.Minute + 30*time.Second)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(tt.d)
			if err != nil {
				t.Fatalf("Marshal error = %v", err)
			}

			var got Duration
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("Unmarshal error = %v", err)
			}

			if got != tt.d {
				t.Errorf("round-trip = %v, want %v", got, tt.d)
			}
		})
	}
}
