package idem

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDiffConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		a         Config
		b         Config
		wantDiff  bool
		wantCount int
		wantField string
	}{
		{
			name:      "identical configs have no diff",
			a:         Config{KeyHeader: "Idempotency-Key", TTL: Duration(24 * time.Hour)},
			b:         Config{KeyHeader: "Idempotency-Key", TTL: Duration(24 * time.Hour)},
			wantDiff:  false,
			wantCount: 0,
		},
		{
			name:      "zero value configs have no diff",
			a:         Config{},
			b:         Config{},
			wantDiff:  false,
			wantCount: 0,
		},
		{
			name:      "different KeyHeader",
			a:         Config{KeyHeader: "Idempotency-Key"},
			b:         Config{KeyHeader: "X-Request-Id"},
			wantDiff:  true,
			wantCount: 1,
			wantField: "KeyHeader",
		},
		{
			name:      "different TTL",
			a:         Config{TTL: Duration(24 * time.Hour)},
			b:         Config{TTL: Duration(1 * time.Hour)},
			wantDiff:  true,
			wantCount: 1,
			wantField: "TTL",
		},
		{
			name:      "different KeyMaxLength",
			a:         Config{KeyMaxLength: 0},
			b:         Config{KeyMaxLength: 64},
			wantDiff:  true,
			wantCount: 1,
			wantField: "KeyMaxLength",
		},
		{
			name:      "different StorageType",
			a:         Config{StorageType: "*idem.MemoryStorage"},
			b:         Config{StorageType: "*redis.Storage"},
			wantDiff:  true,
			wantCount: 1,
			wantField: "StorageType",
		},
		{
			name:      "different LockSupported",
			a:         Config{LockSupported: false},
			b:         Config{LockSupported: true},
			wantDiff:  true,
			wantCount: 1,
			wantField: "LockSupported",
		},
		{
			name:      "different CacheableEnabled",
			a:         Config{CacheableEnabled: false},
			b:         Config{CacheableEnabled: true},
			wantDiff:  true,
			wantCount: 1,
			wantField: "CacheableEnabled",
		},
		{
			name:      "different MetricsEnabled",
			a:         Config{MetricsEnabled: false},
			b:         Config{MetricsEnabled: true},
			wantDiff:  true,
			wantCount: 1,
			wantField: "MetricsEnabled",
		},
		{
			name:      "different OnErrorEnabled",
			a:         Config{OnErrorEnabled: false},
			b:         Config{OnErrorEnabled: true},
			wantDiff:  true,
			wantCount: 1,
			wantField: "OnErrorEnabled",
		},
		{
			name:      "different ValidatorCount",
			a:         Config{ValidatorCount: 0},
			b:         Config{ValidatorCount: 3},
			wantDiff:  true,
			wantCount: 1,
			wantField: "ValidatorCount",
		},
		{
			name: "all fields differ",
			a: Config{
				KeyHeader:        "Idempotency-Key",
				TTL:              Duration(24 * time.Hour),
				KeyMaxLength:     0,
				StorageType:      "*idem.MemoryStorage",
				LockSupported:    true,
				CacheableEnabled: true,
				MetricsEnabled:   true,
				OnErrorEnabled:   true,
				ValidatorCount:   2,
			},
			b: Config{
				KeyHeader:        "X-Request-Id",
				TTL:              Duration(1 * time.Hour),
				KeyMaxLength:     64,
				StorageType:      "*redis.Storage",
				LockSupported:    false,
				CacheableEnabled: false,
				MetricsEnabled:   false,
				OnErrorEnabled:   false,
				ValidatorCount:   0,
			},
			wantDiff:  true,
			wantCount: 9,
		},
		{
			name: "multiple fields differ partially",
			a: Config{
				KeyHeader:    "Idempotency-Key",
				TTL:          Duration(24 * time.Hour),
				KeyMaxLength: 0,
			},
			b: Config{
				KeyHeader:    "X-Request-Id",
				TTL:          Duration(24 * time.Hour),
				KeyMaxLength: 128,
			},
			wantDiff:  true,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diff := DiffConfig(tt.a, tt.b)

			if diff.HasDiff() != tt.wantDiff {
				t.Errorf("HasDiff() = %t, want %t", diff.HasDiff(), tt.wantDiff)
			}

			if len(diff.Diffs) != tt.wantCount {
				t.Errorf("len(Diffs) = %d, want %d", len(diff.Diffs), tt.wantCount)
			}

			if tt.wantField != "" && len(diff.Diffs) > 0 {
				if diff.Diffs[0].Field != tt.wantField {
					t.Errorf("Diffs[0].Field = %q, want %q", diff.Diffs[0].Field, tt.wantField)
				}
			}
		})
	}
}

func TestDiffConfig_fieldOrder(t *testing.T) {
	t.Parallel()

	a := Config{
		KeyHeader:        "A",
		TTL:              Duration(1 * time.Hour),
		KeyMaxLength:     10,
		StorageType:      "typeA",
		LockSupported:    false,
		CacheableEnabled: false,
		MetricsEnabled:   false,
		OnErrorEnabled:   false,
		ValidatorCount:   0,
	}
	b := Config{
		KeyHeader:        "B",
		TTL:              Duration(2 * time.Hour),
		KeyMaxLength:     20,
		StorageType:      "typeB",
		LockSupported:    true,
		CacheableEnabled: true,
		MetricsEnabled:   true,
		OnErrorEnabled:   true,
		ValidatorCount:   1,
	}

	diff := DiffConfig(a, b)

	wantOrder := []string{
		"KeyHeader", "TTL", "KeyMaxLength", "StorageType",
		"LockSupported", "CacheableEnabled", "MetricsEnabled", "OnErrorEnabled", "ValidatorCount",
	}
	if len(diff.Diffs) != len(wantOrder) {
		t.Fatalf("len(Diffs) = %d, want %d", len(diff.Diffs), len(wantOrder))
	}
	for i, fd := range diff.Diffs {
		if fd.Field != wantOrder[i] {
			t.Errorf("Diffs[%d].Field = %q, want %q", i, fd.Field, wantOrder[i])
		}
	}
}

func TestDiffConfig_fieldValues(t *testing.T) {
	t.Parallel()

	a := Config{
		KeyHeader:        "Idempotency-Key",
		TTL:              Duration(24 * time.Hour),
		KeyMaxLength:     0,
		StorageType:      "*idem.MemoryStorage",
		LockSupported:    false,
		CacheableEnabled: false,
		MetricsEnabled:   false,
		OnErrorEnabled:   false,
		ValidatorCount:   0,
	}
	b := Config{
		KeyHeader:        "X-Request-Id",
		TTL:              Duration(1 * time.Hour),
		KeyMaxLength:     64,
		StorageType:      "*redis.Storage",
		LockSupported:    true,
		CacheableEnabled: true,
		MetricsEnabled:   true,
		OnErrorEnabled:   true,
		ValidatorCount:   3,
	}

	diff := DiffConfig(a, b)

	wantValues := []struct {
		field string
		old   string
		new   string
	}{
		{"KeyHeader", `"Idempotency-Key"`, `"X-Request-Id"`},
		{"TTL", "24h0m0s", "1h0m0s"},
		{"KeyMaxLength", "0", "64"},
		{"StorageType", "*idem.MemoryStorage", "*redis.Storage"},
		{"LockSupported", "false", "true"},
		{"CacheableEnabled", "false", "true"},
		{"MetricsEnabled", "false", "true"},
		{"OnErrorEnabled", "false", "true"},
		{"ValidatorCount", "0", "3"},
	}

	if len(diff.Diffs) != len(wantValues) {
		t.Fatalf("len(Diffs) = %d, want %d", len(diff.Diffs), len(wantValues))
	}
	for i, want := range wantValues {
		fd := diff.Diffs[i]
		if fd.Old != want.old {
			t.Errorf("Diffs[%d] (%s) Old = %q, want %q", i, want.field, fd.Old, want.old)
		}
		if fd.New != want.new {
			t.Errorf("Diffs[%d] (%s) New = %q, want %q", i, want.field, fd.New, want.new)
		}
	}
}

func TestDiffConfig_coversAllFields(t *testing.T) {
	t.Parallel()

	a := Config{
		KeyHeader:        "A",
		TTL:              Duration(1 * time.Hour),
		KeyMaxLength:     10,
		StorageType:      "typeA",
		LockSupported:    false,
		CacheableEnabled: false,
		MetricsEnabled:   false,
		OnErrorEnabled:   false,
		ValidatorCount:   0,
	}
	b := Config{
		KeyHeader:        "B",
		TTL:              Duration(2 * time.Hour),
		KeyMaxLength:     20,
		StorageType:      "typeB",
		LockSupported:    true,
		CacheableEnabled: true,
		MetricsEnabled:   true,
		OnErrorEnabled:   true,
		ValidatorCount:   1,
	}

	diff := DiffConfig(a, b)
	numFields := reflect.TypeOf(Config{}).NumField()

	if len(diff.Diffs) != numFields {
		t.Errorf("DiffConfig covers %d fields, but Config has %d fields", len(diff.Diffs), numFields)
	}
}

func TestConfigDiff_String(t *testing.T) {
	t.Parallel()

	t.Run("no differences", func(t *testing.T) {
		t.Parallel()

		diff := ConfigDiff{}
		if diff.String() != "(no differences)" {
			t.Errorf("String() = %q, want %q", diff.String(), "(no differences)")
		}
	})

	t.Run("single difference", func(t *testing.T) {
		t.Parallel()

		diff := DiffConfig(
			Config{KeyHeader: "Idempotency-Key"},
			Config{KeyHeader: "X-Request-Id"},
		)

		got := diff.String()
		want := `  KeyHeader: "Idempotency-Key" → "X-Request-Id"`

		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("multiple differences", func(t *testing.T) {
		t.Parallel()

		diff := DiffConfig(
			Config{KeyHeader: "Idempotency-Key", TTL: Duration(24 * time.Hour), LockSupported: false},
			Config{KeyHeader: "X-Request-Id", TTL: Duration(1 * time.Hour), LockSupported: true},
		)

		got := diff.String()

		if !strings.Contains(got, "KeyHeader") {
			t.Error("String() missing KeyHeader diff")
		}
		if !strings.Contains(got, "TTL") {
			t.Error("String() missing TTL diff")
		}
		if !strings.Contains(got, "LockSupported") {
			t.Error("String() missing LockSupported diff")
		}
		if strings.Count(got, "\n") != 2 {
			t.Errorf("String() has %d newlines, want 2", strings.Count(got, "\n"))
		}
	})
}
