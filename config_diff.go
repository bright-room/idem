package idem

import (
	"fmt"
	"strings"
)

// FieldDiff represents a single field difference between two Config values.
type FieldDiff struct {
	// Field is the name of the changed field.
	Field string

	// Old is the value in the base Config.
	Old string

	// New is the value in the other Config.
	New string
}

// ConfigDiff holds the differences between two Config values.
type ConfigDiff struct {
	// Diffs contains the individual field differences.
	// When the two configs are equal, Diffs is empty.
	Diffs []FieldDiff
}

// HasDiff reports whether there is at least one field difference.
func (d ConfigDiff) HasDiff() bool {
	return len(d.Diffs) > 0
}

// String returns a human-readable summary of the differences.
// Each changed field is printed on its own line with old and new values
// separated by " → ". When no differences exist it returns "(no differences)".
func (d ConfigDiff) String() string {
	if !d.HasDiff() {
		return "(no differences)"
	}

	var b strings.Builder
	for i, fd := range d.Diffs {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "  %s: %s → %s", fd.Field, fd.Old, fd.New)
	}

	return b.String()
}

// DiffConfig compares two Config values and returns their differences.
// Fields are compared in declaration order. Only fields with different
// values appear in the result.
func DiffConfig(a, b Config) ConfigDiff {
	var diffs []FieldDiff

	if a.KeyHeader != b.KeyHeader {
		diffs = append(diffs, FieldDiff{
			Field: "KeyHeader",
			Old:   fmt.Sprintf("%q", a.KeyHeader),
			New:   fmt.Sprintf("%q", b.KeyHeader),
		})
	}

	if a.TTL != b.TTL {
		diffs = append(diffs, FieldDiff{
			Field: "TTL",
			Old:   a.TTL.String(),
			New:   b.TTL.String(),
		})
	}

	if a.KeyMaxLength != b.KeyMaxLength {
		diffs = append(diffs, FieldDiff{
			Field: "KeyMaxLength",
			Old:   fmt.Sprintf("%d", a.KeyMaxLength),
			New:   fmt.Sprintf("%d", b.KeyMaxLength),
		})
	}

	if a.StorageType != b.StorageType {
		diffs = append(diffs, FieldDiff{
			Field: "StorageType",
			Old:   a.StorageType,
			New:   b.StorageType,
		})
	}

	if a.LockSupported != b.LockSupported {
		diffs = append(diffs, FieldDiff{
			Field: "LockSupported",
			Old:   fmt.Sprintf("%t", a.LockSupported),
			New:   fmt.Sprintf("%t", b.LockSupported),
		})
	}

	if a.MetricsEnabled != b.MetricsEnabled {
		diffs = append(diffs, FieldDiff{
			Field: "MetricsEnabled",
			Old:   fmt.Sprintf("%t", a.MetricsEnabled),
			New:   fmt.Sprintf("%t", b.MetricsEnabled),
		})
	}

	if a.OnErrorEnabled != b.OnErrorEnabled {
		diffs = append(diffs, FieldDiff{
			Field: "OnErrorEnabled",
			Old:   fmt.Sprintf("%t", a.OnErrorEnabled),
			New:   fmt.Sprintf("%t", b.OnErrorEnabled),
		})
	}

	if a.ValidatorCount != b.ValidatorCount {
		diffs = append(diffs, FieldDiff{
			Field: "ValidatorCount",
			Old:   fmt.Sprintf("%d", a.ValidatorCount),
			New:   fmt.Sprintf("%d", b.ValidatorCount),
		})
	}

	return ConfigDiff{Diffs: diffs}
}
