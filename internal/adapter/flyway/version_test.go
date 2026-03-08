package flyway

import (
	"testing"
	"time"
)

func TestIncrementVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "single digit",
			version:  "1",
			expected: "2",
		},
		{
			name:     "zero padded",
			version:  "003",
			expected: "004",
		},
		{
			name:     "semantic versioning",
			version:  "1.2.9",
			expected: "1.2.10",
		},
		{
			name:     "semantic version zero padding",
			version:  "1.02.09",
			expected: "1.02.10",
		},
		{
			name:     "hybrid with letters",
			version:  "v1-prod",
			expected: "v2-prod",
		},
		{
			name:     "no numbers", // edge case fallback
			version:  "init",
			expected: "init.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IncrementVersion(tt.version)
			if got != tt.expected {
				t.Errorf("IncrementVersion(%q) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}

	// Test case for timestamp scenario
	t.Run("timestamp format", func(t *testing.T) {
		ts := "20230101150000"
		got := IncrementVersion(ts)
		nowStr := time.Now().Format("20060102150405")
		// It should output a timestamp greater than or equal to nowStr
		// (likely equal, or if execution takes a sec >. Also it must be > old ts).
		if got < nowStr && got != "20230101150001" {
			t.Errorf("IncrementVersion timestamp failed, got %s, want recent timestamp", got)
		}
	})
}
