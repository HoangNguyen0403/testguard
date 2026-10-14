package zeroassert_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/zeroassert"
	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

func TestDetectZeroAssertion(t *testing.T) {
	unitWithoutAssert := parser.TestUnit{
		Name:       "unasserted_test",
		FilePath:   "math_test.dart",
		StartLine:  10,
		Kind:       "unit",
		Assertions: []string{},
	}

	unitWithAssert := parser.TestUnit{
		Name:       "asserted_test",
		FilePath:   "math_test.dart",
		StartLine:  20,
		Kind:       "unit",
		Assertions: []string{"expect"},
	}

	detector := zeroassert.New()
	violations := detector.Check([]parser.TestUnit{unitWithoutAssert, unitWithAssert})

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.RuleID != "GUARD-UNIV-ZERO" {
		t.Errorf("expected rule GUARD-UNIV-ZERO, got %s", v.RuleID)
	}
	if v.Severity != models.SeverityBlocker {
		t.Errorf("expected Blocker severity, got %s", v.Severity)
	}
}
