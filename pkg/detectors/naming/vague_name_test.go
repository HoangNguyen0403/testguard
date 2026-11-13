package naming_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/naming"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	tsparser "github.com/HoangNguyen0403/testguard/pkg/parser/typescript"
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

// TestDynamicTitleIsNotJudgedAsLiteralText pins the metadata guard: a test whose
// adapter recorded only the unevaluated title expression must not be reported as
// a placeholder, while a real literal with the same text still is.
func TestDynamicTitleIsNotJudgedAsLiteralText(t *testing.T) {
	detector := naming.New()
	units := []parser.TestUnit{
		{Name: "foo", FilePath: "dynamic.test.ts", StartLine: 2, Meta: map[string]string{"title_dynamic": "true"}},
		{Name: "foo", FilePath: "dynamic.test.ts", StartLine: 6},
	}

	violations := detector.Check(units)
	if len(violations) != 1 {
		t.Fatalf("expected exactly the literal title to be reported, got %d: %+v", len(violations), violations)
	}
	if violations[0].LineNumber != 6 {
		t.Errorf("expected the literal declaration at line 6, got line %d", violations[0].LineNumber)
	}
}

// TestDynamicTitleGuardThroughAdapter runs the real TypeScript adapter so the
// guard is pinned to the adapter metadata contract rather than a hand-built map.
func TestDynamicTitleGuardThroughAdapter(t *testing.T) {
	source := []byte(`import { sum } from './sut';

const foo = describeScenario();

it(foo, () => {
  expect(sum(1, 2)).toBe(3);
});

it('foo', () => {
  expect(sum(2, 2)).toBe(4);
});
`)
	units, err := tsparser.NewAdapter().LocateTests(source, "dynamic.test.ts")
	if err != nil {
		t.Fatalf("unexpected adapter error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 discovered tests, got %d", len(units))
	}

	violations := naming.New().Check(units)
	if len(violations) != 1 {
		t.Fatalf("expected only the literal 'foo' title to be reported, got %d: %+v", len(violations), violations)
	}
	if violations[0].LineNumber != 9 {
		t.Errorf("expected the literal declaration at line 9, got line %d", violations[0].LineNumber)
	}
}
