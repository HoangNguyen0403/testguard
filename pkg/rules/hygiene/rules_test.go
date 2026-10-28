package hygiene_test

import (
	"fmt"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/hygiene"
)

func TestTautologyRule_TS(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected int
	}{
		{"toBe literal number", `expect(1).toBe(1);`, 1},
		{"toEqual literal string", `expect("abc").toEqual("abc");`, 1},
		{"toStrictEqual literal bool", `expect(true).toStrictEqual(true);`, 1},
		{"toBeTruthy literal true", `expect(true).toBeTruthy();`, 1},
		{"toBeFalsy literal false", `expect(false).toBeFalsy();`, 1},
		{"toBe different numbers", `expect(1).toBe(2);`, 0},
		{"toBe non-literal variable", `expect(x).toBe(x);`, 0},
		{"commented tautology line", `// expect(true).toBe(true);`, 0},
		{"commented tautology block", `/* expect(true).toBeTruthy(); */`, 0},
		{"string literal tautology", `const s = "expect(true).toBe(true);";`, 0},
		{"template string interpolation", "expect(`value ${actual}`).toBe(`value ${actual}`);", 0},
	}

	rule := hygiene.NewTautologyRule()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      "test_unit",
				FilePath:  "example.test.ts",
				StartLine: 10,
				RawBody:   tc.body,
			}
			violations := rule.Evaluate(unit)
			if len(violations) != tc.expected {
				t.Fatalf("expected %d violations, got %d", tc.expected, len(violations))
			}
			if tc.expected > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-UNIV-TAUTOLOGY" {
					t.Errorf("expected rule ID GUARD-UNIV-TAUTOLOGY, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityBlocker {
					t.Errorf("expected Blocker severity, got %s", v.Severity)
				}
			}
		})
	}
}

func TestTautologyRule_Dart(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected int
	}{
		{"expect literal number", `expect(1, 1);`, 1},
		{"expect literal string", `expect("foo", "foo");`, 1},
		{"expect true isTrue", `expect(true, isTrue);`, 1},
		{"expect false isFalse", `expect(false, isFalse);`, 1},
		{"expect literal equals", `expect("bar", equals("bar"));`, 1},
		{"expect different numbers", `expect(1, 2);`, 0},
		{"expect non-literal variable", `expect(x, x);`, 0},
		{"commented tautology line", `// expect(true, isTrue);`, 0},
		{"commented tautology block", `/* expect(true, isTrue); */`, 0},
		{"string literal tautology", `final s = 'expect(true, isTrue);';`, 0},
	}

	rule := hygiene.NewTautologyRule()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      "dart_test",
				FilePath:  "widget_test.dart",
				StartLine: 20,
				RawBody:   tc.body,
			}
			violations := rule.Evaluate(unit)
			if len(violations) != tc.expected {
				t.Fatalf("expected %d violations, got %d", tc.expected, len(violations))
			}
			if tc.expected > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-UNIV-TAUTOLOGY" {
					t.Errorf("expected rule ID GUARD-UNIV-TAUTOLOGY, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityBlocker {
					t.Errorf("expected Blocker severity, got %s", v.Severity)
				}
			}
		})
	}
}

func TestTautologyRule_Go(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected int
	}{
		{"assert.True true", `assert.True(t, true)`, 1},
		{"require.True true", `require.True(t, true)`, 1},
		{"assert.False false", `assert.False(t, false)`, 1},
		{"assert.Equal same literal", `assert.Equal(t, 42, 42)`, 1},
		{"assert.Equal custom testing var", `assert.Equal(tt, "ok", "ok")`, 1},
		{"assert.Equal different numbers", `assert.Equal(t, 1, 2)`, 0},
		{"assert.Equal non-literal variable", `assert.Equal(t, x, x)`, 0},
		{"commented tautology line", `// assert.True(t, true)`, 0},
		{"string literal tautology", `msg := "assert.True(t, true)"`, 0},
	}

	rule := hygiene.NewTautologyRule()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      "TestSomething",
				FilePath:  "calc_test.go",
				StartLine: 5,
				RawBody:   tc.body,
			}
			violations := rule.Evaluate(unit)
			if len(violations) != tc.expected {
				t.Fatalf("expected %d violations, got %d", tc.expected, len(violations))
			}
			if tc.expected > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-UNIV-TAUTOLOGY" {
					t.Errorf("expected rule ID GUARD-UNIV-TAUTOLOGY, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityBlocker {
					t.Errorf("expected Blocker severity, got %s", v.Severity)
				}
			}
		})
	}
}

func TestSkipRule_TS(t *testing.T) {
	rule := hygiene.NewSkipRule()

	t.Run("it.skip should be flagged", func(t *testing.T) {
		src := "describe('suite', () => {\n  it.skip('broken test', () => {});\n});"
		violations := rule.CheckFile("suite.test.ts", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(violations))
		}
		if violations[0].RuleID != "GUARD-UNIV-SKIP" {
			t.Errorf("expected GUARD-UNIV-SKIP, got %s", violations[0].RuleID)
		}
		if violations[0].Severity != models.SeverityWarning {
			t.Errorf("expected Warning severity, got %s", violations[0].Severity)
		}
		if violations[0].LineNumber != 2 {
			t.Errorf("expected line 2, got %d", violations[0].LineNumber)
		}
	})

	t.Run("commented it.skip should not be flagged", func(t *testing.T) {
		src := "describe('suite', () => {\n  // it.skip('broken test', () => {});\n});"
		violations := rule.CheckFile("suite.test.ts", src)
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations for commented it.skip, got %d", len(violations))
		}
	})
	t.Run("all TS skip variants flagged", func(t *testing.T) {
		variants := []string{"test.skip(", "describe.skip(", "xit(", "xtest(", "xdescribe("}
		for _, v := range variants {
			src := fmt.Sprintf("%s'name', () => {});", v)
			res := rule.CheckFile("a.test.js", src)
			if len(res) != 1 {
				t.Errorf("expected 1 violation for %s, got %d", v, len(res))
			}
		}
	})

	t.Run("string containing it.skip not flagged", func(t *testing.T) {
		src := `const msg = "it.skip('not a real skip')";`
		res := rule.CheckFile("a.test.ts", src)
		if len(res) != 0 {
			t.Errorf("expected 0 violations for string literal, got %d", len(res))
		}
	})
}

func TestSkipRule_Dart(t *testing.T) {
	rule := hygiene.NewSkipRule()

	t.Run("skip true should be flagged", func(t *testing.T) {
		src := "void main() {\n  test('flaky', () {}, skip: true);\n}"
		violations := rule.CheckFile("main_test.dart", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(violations))
		}
		if violations[0].RuleID != "GUARD-UNIV-SKIP" {
			t.Errorf("expected GUARD-UNIV-SKIP, got %s", violations[0].RuleID)
		}
		if violations[0].LineNumber != 2 {
			t.Errorf("expected line 2, got %d", violations[0].LineNumber)
		}
	})

	t.Run("commented skip should not be flagged", func(t *testing.T) {
		src := "void main() {\n  // test('flaky', () {}, skip: true);\n}"
		violations := rule.CheckFile("main_test.dart", src)
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(violations))
		}
	})
	t.Run("skip with string reasons flagged", func(t *testing.T) {
		singleQuote := "test('flaky', () {}, skip: 'ticket-123');"
		res1 := rule.CheckFile("a_test.dart", singleQuote)
		if len(res1) != 1 {
			t.Errorf("expected 1 violation for single quote reason, got %d", len(res1))
		}

		doubleQuote := `test('flaky', () {}, skip: "ticket-456");`
		res2 := rule.CheckFile("a_test.dart", doubleQuote)
		if len(res2) != 1 {
			t.Errorf("expected 1 violation for double quote reason, got %d", len(res2))
		}
	})

	t.Run("skip: false should not be flagged", func(t *testing.T) {
		src := "test('flaky', () {}, skip: false);"
		res := rule.CheckFile("a_test.dart", src)
		if len(res) != 0 {
			t.Errorf("expected 0 violations for skip: false, got %d", len(res))
		}
	})
	t.Run("Options skip true should not be flagged", func(t *testing.T) {
		src := "void main() { test('real', () { final options = Options(skip: true); expect(options.enabled, isTrue); }); }"
		res := rule.CheckFile("options_test.dart", src)
		if len(res) != 0 {
			t.Errorf("expected 0 violations for Options(skip: true), got %d: %+v", len(res), res)
		}
	})
}

func TestSkipRule_Go(t *testing.T) {
	rule := hygiene.NewSkipRule()

	t.Run("t.Skip as first statement flagged", func(t *testing.T) {
		src := "package foo\n\nfunc TestWIP(t *testing.T) {\n\tt.Skip(\"wip\")\n\tdoSomething()\n}"
		violations := rule.CheckFile("foo_test.go", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(violations))
		}
		if violations[0].RuleID != "GUARD-UNIV-SKIP" {
			t.Errorf("expected GUARD-UNIV-SKIP, got %s", violations[0].RuleID)
		}
		if violations[0].LineNumber != 4 {
			t.Errorf("expected line 4, got %d", violations[0].LineNumber)
		}
	})

	t.Run("conditional t.Skip inside if not flagged", func(t *testing.T) {
		src := "package foo\n\nfunc TestShort(t *testing.T) {\n\tif testing.Short() {\n\t\tt.Skip(\"short mode\")\n\t}\n\tdoSomething()\n}"
		violations := rule.CheckFile("foo_test.go", src)
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations for conditional t.Skip, got %d", len(violations))
		}
	})
	t.Run("t.Skipf and t.SkipNow flagged as first statement", func(t *testing.T) {
		srcF := "package foo\n\nfunc TestSkipF(t *testing.T) {\n\tt.Skipf(\"reason %d\", 1)\n}"
		resF := rule.CheckFile("f_test.go", srcF)
		if len(resF) != 1 {
			t.Errorf("expected 1 violation for t.Skipf, got %d", len(resF))
		}

		srcNow := "package foo\n\nfunc TestSkipNow(t *testing.T) {\n\tt.SkipNow()\n}"
		resNow := rule.CheckFile("now_test.go", srcNow)
		if len(resNow) != 1 {
			t.Errorf("expected 1 violation for t.SkipNow, got %d", len(resNow))
		}
	})

	t.Run("t.Skip after comment flagged", func(t *testing.T) {
		src := "package foo\n\nfunc TestCommentFirst(t *testing.T) {\n\t// A comment\n\t/* Another comment */\n\tt.Skip(\"reason\")\n}"
		res := rule.CheckFile("c_test.go", src)
		if len(res) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(res))
		}
		if res[0].LineNumber != 6 {
			t.Errorf("expected line 6, got %d", res[0].LineNumber)
		}
	})
	t.Run("collection.Skip should not be flagged", func(t *testing.T) {
		src := "package foo\n\nimport \"testing\"\n\nfunc TestReal(t *testing.T) {\n\tcollection.Skip(2)\n\tif result != 2 { t.Fatal(result) }\n}"
		res := rule.CheckFile("skip_test.go", src)
		if len(res) != 0 {
			t.Errorf("expected 0 violations for collection.Skip, got %d: %+v", len(res), res)
		}
	})

	t.Run("Go raw string containing t.Skip should not be flagged", func(t *testing.T) {
		src := "package foo\n\nimport \"testing\"\n\nconst example = `\nfunc TestFake(t *testing.T) {\n\tt.Skip(\"example\")\n}\n`\n\nfunc TestReal(t *testing.T) { if result != 2 { t.Fatal(result) } }"
		res := rule.CheckFile("raw_test.go", src)
		if len(res) != 0 {
			t.Errorf("expected 0 violations for raw string containing t.Skip, got %d: %+v", len(res), res)
		}
	})
	t.Run("non-test receiver method calling Skip should not be flagged", func(t *testing.T) {
		src := `package sample
import "testing"
type collection struct{}
func (collection) Skip(int) {}
func (collection) TestUtility(c collection) { c.Skip(1) }
func TestReal(t *testing.T) { if false { t.Fatal("failure") } }`
		res := rule.CheckFile("skip_signature_test.go", src)
		if len(res) != 0 {
			t.Errorf("expected 0 violations for non-test receiver method Skip, got %d: %+v", len(res), res)
		}
	})
}
