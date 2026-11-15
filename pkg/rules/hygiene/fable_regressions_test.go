package hygiene_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/hygiene"
)

func TestSkipRule_Fixtures(t *testing.T) {
	rule := hygiene.NewSkipRule()

	t.Run("literal skip text produces no findings", func(t *testing.T) {
		src := readFixture(t, "literal_skips_test.dart")
		if violations := rule.CheckFile("literal_skips_test.dart", src); len(violations) != 0 {
			t.Fatalf("expected no findings for literal skip text, got %+v", violations)
		}
	})

	t.Run("real skip reported at its line", func(t *testing.T) {
		src := readFixture(t, "real_skips_test.dart")
		violations := rule.CheckFile("real_skips_test.dart", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %+v", violations)
		}
		if violations[0].RuleID != "GUARD-UNIV-SKIP" || violations[0].Severity != models.SeverityWarning {
			t.Errorf("expected GUARD-UNIV-SKIP Warning, got %s %s", violations[0].RuleID, violations[0].Severity)
		}
		if violations[0].LineNumber != 6 {
			t.Errorf("expected the skip on line 6, got %d", violations[0].LineNumber)
		}
	})
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return string(content)
}

func TestTautologyRule_KotlinSameOperandParity(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected int
	}{
		{"same plain identifier", `assertEquals(r, r)`, 1},
		{"same dotted member path", `assertEquals(user.profile.name, user.profile.name)`, 1},
		{"different receivers same field", `assertEquals(a.id, b.id)`, 0},
		{"identical call operands", `assertEquals(next(), next())`, 0},
		{"identical getter operands", `assertEquals(repo.name, repo.name())`, 0},
	}
	rule := hygiene.NewTautologyRule()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      "kotlin_test",
				FilePath:  "ExampleTest.kt",
				StartLine: 15,
				RawBody:   tc.body,
			}
			violations := rule.Evaluate(unit)
			if len(violations) != tc.expected {
				t.Fatalf("expected %d violations, got %d", tc.expected, len(violations))
			}
			if tc.expected > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-UNIV-TAUTOLOGY" || v.Severity != models.SeverityBlocker {
					t.Errorf("expected GUARD-UNIV-TAUTOLOGY Blocker, got %s %s", v.RuleID, v.Severity)
				}
				if v.LineNumber != 15 {
					t.Errorf("expected line 15, got %d", v.LineNumber)
				}
			}
		})
	}
}

func TestTautologyRule_SwiftSameOperandParity(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected int
	}{
		{"XCTAssertEqual same plain identifier", `XCTAssertEqual(r, r)`, 1},
		{"XCTAssertEqual same dotted member path", `XCTAssertEqual(user.profile.name, user.profile.name)`, 1},
		{"XCTAssertEqual different receivers", `XCTAssertEqual(a.id, b.id)`, 0},
		{"XCTAssertEqual identical calls", `XCTAssertEqual(next(), next())`, 0},
		{"expect equal same plain identifier", `#expect(r == r)`, 1},
		{"expect equal same dotted path", `#expect(user.profile.name == user.profile.name)`, 1},
		{"expect equal different receivers", `#expect(a.id == b.id)`, 0},
		{"expect equal different identifiers", `#expect(r == other)`, 0},
	}
	rule := hygiene.NewTautologyRule()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      "swift_test",
				FilePath:  "ExampleTests.swift",
				StartLine: 20,
				RawBody:   tc.body,
			}
			violations := rule.Evaluate(unit)
			if len(violations) != tc.expected {
				t.Fatalf("expected %d violations, got %d", tc.expected, len(violations))
			}
			if tc.expected > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-UNIV-TAUTOLOGY" || v.Severity != models.SeverityBlocker {
					t.Errorf("expected GUARD-UNIV-TAUTOLOGY Blocker, got %s %s", v.RuleID, v.Severity)
				}
				if v.LineNumber != 20 {
					t.Errorf("expected line 20, got %d", v.LineNumber)
				}
			}
		})
	}
}

func TestTautologyRule_BodyStartLineMetadata(t *testing.T) {
	const body = "\n        expect(r).toBe(r);\n"

	t.Run("metadata anchors the reported line", func(t *testing.T) {
		unit := parser.TestUnit{
			Name:      "multi-line declaration",
			FilePath:  "misc.test.ts",
			StartLine: 49,
			RawBody:   body,
			Meta:      map[string]string{"body_start_line": "50"},
		}
		violations := hygiene.NewTautologyRule().Evaluate(unit)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(violations))
		}
		if violations[0].LineNumber != 51 {
			t.Errorf("expected the tautology on line 51, got %d", violations[0].LineNumber)
		}
	})

	t.Run("missing metadata falls back to the declaration line", func(t *testing.T) {
		unit := parser.TestUnit{
			Name:      "legacy adapter",
			FilePath:  "misc.test.ts",
			StartLine: 49,
			RawBody:   body,
		}
		violations := hygiene.NewTautologyRule().Evaluate(unit)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(violations))
		}
		if violations[0].LineNumber != 50 {
			t.Errorf("expected line 50 without metadata, got %d", violations[0].LineNumber)
		}
	})
}

func TestSkipRule_LiteralTextInStringsAndNestedComments(t *testing.T) {
	rule := hygiene.NewSkipRule()

	cases := []struct {
		name string
		file string
		src  string
	}{
		{
			name: "dart skip inside triple-quoted string",
			file: "iso_b_test.dart",
			src:  "void main() {\n  test('B9', () {\n    const source = '''\n      skip: true\n    ''';\n    expect(source.length, 5);\n  });\n}",
		},
		{
			name: "dart annotation inside interpolated string",
			file: "iso_b_test.dart",
			src:  "void main() {\n  test('B17', () {\n    final m = {'}': 1};\n    final s = '${m['}']} @Skip(\"x\")';\n    expect(s, '1 end');\n  });\n}",
		},
		{
			name: "dart annotation inside raw string",
			file: "raw_test.dart",
			src:  "void main() {\n  test('raw', () {\n    const s = r'@Skip(\"fake\")';\n    expect(s.length, 2);\n  });\n}",
		},
		{
			name: "kotlin ignore inside nested block comment",
			file: "Nested.kt",
			src:  "class Nested {\n    /* @Test fun commented() { } /* nested */ @Ignore */\n    @Test\n    fun real() {\n        assertEquals(1, 1)\n    }\n}",
		},
		{
			name: "kotlin ignore inside string template",
			file: "Template.kt",
			src:  "class Template {\n    @Test\n    fun real() {\n        val text = \"${listOf(\"}\")} @Ignore\"\n        assertEquals(1, 1)\n    }\n}",
		},
		{
			name: "swift disabled attribute inside raw string",
			file: "RawTests.swift",
			src:  "final class RawTests: XCTestCase {\n    func testReal() {\n        let s = #\"@Test(.disabled) fake\"#\n        XCTAssertEqual(s.count, 4)\n    }\n}",
		},
		{
			name: "swift disabled attribute inside nested comment",
			file: "NestedTests.swift",
			src:  "final class NestedTests: XCTestCase {\n    /* /* nested */ @Test(.disabled) throw XCTSkip(\"x\") */\n    func testReal() {\n        XCTAssertEqual(1, 1)\n    }\n}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			violations := rule.CheckFile(tc.file, tc.src)
			if len(violations) != 0 {
				t.Fatalf("expected no findings for literal text, got %d: %+v", len(violations), violations)
			}
		})
	}
}
