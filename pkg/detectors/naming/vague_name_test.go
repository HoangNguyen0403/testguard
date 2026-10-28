package naming_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/naming"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

func TestDetectVagueTestNames(t *testing.T) {
	tests := []struct {
		name       string
		shouldFlag bool
	}{
		{"test 1", true},
		{"sample", true},
		{"dummy", true},
		{"foo", true},
		{"test", true},
		{"   ", true},
		{"FetchCart_WhenTokenExpired_EmitsUnauthorizedFailureState", false},
		{"returns empty list when items is null", false},
		{"calculates discount with rebate applied", false},
		{"login_success_scenario", false},
	}

	detector := naming.New()
	for _, tt := range tests {
		unit := parser.TestUnit{
			Name:      tt.name,
			FilePath:  "sample_test.dart",
			StartLine: 1,
		}
		violations := detector.Check([]parser.TestUnit{unit})
		isFlagged := len(violations) > 0

		if isFlagged != tt.shouldFlag {
			t.Errorf("name %q: flagged = %v, expected %v", tt.name, isFlagged, tt.shouldFlag)
		}
	}
}
