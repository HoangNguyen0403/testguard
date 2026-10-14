package updater_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/updater"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"0.1.0", "v0.1.1", true},
		{"0.1.0", "v0.2.0", true},
		{"0.1.0", "v1.0.0", true},
		{"0.1.0", "v0.1.0", false},
		{"0.2.0", "v0.1.0", false},
		{"v0.1.0", "v0.1.0", false},
	}

	for _, tt := range tests {
		result := updater.IsNewer(tt.current, tt.latest)
		if result != tt.expected {
			t.Errorf("IsNewer(%q, %q) = %v; expected %v", tt.current, tt.latest, result, tt.expected)
		}
	}
}
