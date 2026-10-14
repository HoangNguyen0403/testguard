package clone_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/clone"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

func TestDetectExactClones(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test1",
		FilePath:  "auth_test.dart",
		StartLine: 10,
		RawBody:   "expect(auth.login('foo', 'bar'), equals(true));",
	}

	test2 := parser.TestUnit{
		Name:      "test2",
		FilePath:  "auth_test.dart",
		StartLine: 25,
		RawBody:   "expect(auth.login('baz', 'qux'), equals(true));",
	}

	detector := clone.New()
	violations := detector.Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 clone violation, got %d", len(violations))
	}

	if violations[0].RuleID != "GUARD-UNIV-01" {
		t.Errorf("expected GUARD-UNIV-01, got %s", violations[0].RuleID)
	}
}

func TestDetectStructuralRepetition(t *testing.T) {
	var tests []parser.TestUnit
	for i := 1; i <= 5; i++ {
		tests = append(tests, parser.TestUnit{
			Name:      "repetition_case",
			FilePath:  "pricing_test.go",
			StartLine: i * 10,
			RawBody:   "var input = 100; result := Calc(input); assert.Equal(t, 100, result);",
		})
	}

	detector := clone.New()
	violations := detector.Check(tests)

	hasRepetition := false
	for _, v := range violations {
		if v.RuleID == "GUARD-UNIV-02" {
			hasRepetition = true
		}
	}

	if !hasRepetition {
		t.Errorf("expected GUARD-UNIV-02 to be triggered for 5 repetitive tests")
	}
}
