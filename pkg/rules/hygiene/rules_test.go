package hygiene_test

import (
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
		{"toBe plain identifier", `expect(x).toBe(x);`, 1},
		{"toEqual dotted member path", `expect(user.profile.name).toEqual(user.profile.name);`, 1},
		{"toStrictEqual plain identifier", `expect(x).toStrictEqual(x);`, 1},
		{"constant arithmetic addition", `expect(2 + 2).toBe(4);`, 1},
		{"constant arithmetic multiplication", `expect(10 * 5).toEqual(50);`, 1},
		{"negated distinct literal bool", `expect(true).not.toBe(false);`, 1},
		{"negated distinct literal number", `expect(1).not.toEqual(2);`, 1},
		{"call operand not tautology", `expect(a.next()).toBe(a.next());`, 0},
		{"negated non-literal not tautology", `expect(x).not.toBe(null);`, 0},
		{"different identifiers not tautology", `expect(x).toBe(y);`, 0},
		{"constant plus call not tautology", `expect(2 + add(1, 2)).toBe(5);`, 0},
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
		{"expect plain identifier", `expect(x, x);`, 1},
		{"expect equals plain identifier", `expect(x, equals(x));`, 1},
		{"expect equals dotted member path", `expect(user.profile.name, equals(user.profile.name));`, 1},
		{"expect call operand not tautology", `expect(a.next(), a.next());`, 0},
		{"expect different identifiers not tautology", `expect(x, y);`, 0},
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
		{"assert.Equal plain identifier", `assert.Equal(t, x, x)`, 1},
		{"require.Equal plain identifier", `require.Equal(t, x, x)`, 1},
		{"assert.Equal dotted member path", `assert.Equal(t, user.Name, user.Name)`, 1},
		{"assert.EqualValues plain identifier", `assert.EqualValues(t, x, x)`, 1},
		{"assert.Equalf plain identifier", `assert.Equalf(t, x, x, "msg")`, 1},
		{"assert.Equal different numbers", `assert.Equal(t, 1, 2)`, 0},
		{"assert.Equal different variables", `assert.Equal(t, x, y)`, 0},
		{"assert.Equal call operand not tautology", `assert.Equal(t, next(), next())`, 0},
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
func TestTautologyRule_Kotlin(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected int
	}{
		{"assertTrue literal true", `assertTrue(true)`, 1},
		{"assertTrue literal true with message", `assertTrue(true, "message")`, 1},
		{"assertFalse literal false", `assertFalse(false)`, 1},
		{"assert literal true", `assert(true)`, 1},
		{"assertEquals same literal int", `assertEquals(42, 42)`, 1},
		{"assertEquals same literal string", `assertEquals("hello", "hello")`, 1},
		{"assertTrue variable", `assertTrue(isValid)`, 0},
		{"assertFalse variable", `assertFalse(isValid)`, 0},
		{"assertEquals different literals", `assertEquals(1, 2)`, 0},
		{"commented tautology", `// assertTrue(true)`, 0},
		{"string literal containing tautology", `val s = "assertTrue(true)"`, 0},
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

func TestTautologyRule_Swift(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected int
	}{
		{"XCTAssertTrue true", `XCTAssertTrue(true)`, 1},
		{"XCTAssertTrue true with message", `XCTAssertTrue(true, "always true")`, 1},
		{"XCTAssertFalse false", `XCTAssertFalse(false)`, 1},
		{"XCTAssertEqual same literal int", `XCTAssertEqual(42, 42)`, 1},
		{"XCTAssertEqual same literal string", `XCTAssertEqual("abc", "abc")`, 1},
		{"#expect(true)", `#expect(true)`, 1},
		{"#expect same literal equals", `#expect(42 == 42)`, 1},
		{"#expect same literal string equals", `#expect("hello" == "hello")`, 1},
		{"XCTAssertTrue variable", `XCTAssertTrue(isValid)`, 0},
		{"XCTAssertFalse variable", `XCTAssertFalse(isValid)`, 0},
		{"XCTAssertEqual different literals", `XCTAssertEqual(1, 2)`, 0},
		{"#expect variable", `#expect(isValid)`, 0},
		{"#expect different literals", `#expect(1 == 2)`, 0},
		{"commented tautology", `// XCTAssertTrue(true)`, 0},
		{"string literal containing tautology", `let s = "XCTAssertTrue(true)"`, 0},
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
	t.Run("all TS declaration skip variants flagged", func(t *testing.T) {
		variants := []string{
			"it.skip('name', () => {});",
			"test.skip('name', () => {});",
			"describe.skip('name', () => {});",
			"test.describe.skip('name', () => {});",
			"xit('name', () => {});",
			"xtest('name', () => {});",
			"xdescribe('name', () => {});",
			"test.fixme('name', () => {});",
			"it.fixme('name', () => {});",
			"it.skip.each([1, 2])('name %s', () => {});",
			"test.skip.each([1, 2])('name %s', () => {});",
			"describe.skip.each([1, 2])('name %s', () => {});",
			"test.describe.skip.each([1, 2])('name %s', () => {});",
			"it.skip(`template title`, () => {});",
		}
		for _, src := range variants {
			res := rule.CheckFile("a.test.ts", src)
			if len(res) != 1 {
				t.Errorf("expected 1 violation for %q, got %d", src, len(res))
			} else {
				if res[0].RuleID != "GUARD-UNIV-SKIP" {
					t.Errorf("expected GUARD-UNIV-SKIP for %q, got %s", src, res[0].RuleID)
				}
				if res[0].Severity != models.SeverityWarning {
					t.Errorf("expected Warning severity for %q, got %s", src, res[0].Severity)
				}
			}
		}
	})

	t.Run("conditional skip and zero-arg skip not flagged", func(t *testing.T) {
		cases := []string{
			"test.skip(browserName === 'webkit', 'reason');",
			"test.skip(condition);",
			"test.skip();",
			"it.skip();",
		}
		for _, src := range cases {
			res := rule.CheckFile("a.test.ts", src)
			if len(res) != 0 {
				t.Errorf("expected 0 violations for %q, got %d (%+v)", src, len(res), res)
			}
		}
	})

	t.Run("skip probe file lines 9, 14, 19 only", func(t *testing.T) {
		src := `import { test, expect } from '@playwright/test';

test('conditional skip by browser', async ({ page, browserName }) => {
  test.skip(browserName === 'webkit', 'clipboard API unsupported on webkit');
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Home');
});

test.skip('declared skipped', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Home');
});

test.fixme('known broken', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Home');
});

test.describe.skip('skipped suite', () => {
  test('inner', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Home');
  });
});`
		res := rule.CheckFile("skip.spec.ts", src)
		if len(res) != 3 {
			t.Fatalf("expected 3 violations, got %d (%+v)", len(res), res)
		}
		expectedLines := []int{9, 14, 19}
		for i, exp := range expectedLines {
			if res[i].LineNumber != exp {
				t.Errorf("expected violation %d at line %d, got line %d", i, exp, res[i].LineNumber)
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

	t.Run("member access and non-declaration xit not flagged", func(t *testing.T) {
		cases := []string{
			"runner.xit('x', () => {});",
			"obj.it.skip('x', () => {});",
			"suite?.test.skip('x', () => {});",
		}
		for _, src := range cases {
			res := rule.CheckFile("a.test.ts", src)
			if len(res) != 0 {
				t.Errorf("expected 0 violations for member skip %q, got %d (%+v)", src, len(res), res)
			}
		}

		// Bare xit with string title must still be flagged
		res := rule.CheckFile("a.test.ts", "xit('x', () => {});")
		if len(res) != 1 {
			t.Errorf("expected 1 violation for bare xit, got %d", len(res))
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
func TestSkipRule_Dart_Annotation(t *testing.T) {
	rule := hygiene.NewSkipRule()

	t.Run("@Skip annotation flagged at line", func(t *testing.T) {
		src := "// File header\n@Skip('flaky suite')\nimport 'package:test/test.dart';\nvoid main() {}"
		violations := rule.CheckFile("suite_test.dart", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation for @Skip, got %d", len(violations))
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

	t.Run("@Skip without argument flagged", func(t *testing.T) {
		src := "@Skip()\nvoid main() {}"
		violations := rule.CheckFile("suite_test.dart", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation for @Skip(), got %d", len(violations))
		}
		if violations[0].LineNumber != 1 {
			t.Errorf("expected line 1, got %d", violations[0].LineNumber)
		}
	})

	t.Run("commented @Skip not flagged", func(t *testing.T) {
		src := "// @Skip('flaky suite')\nvoid main() {}"
		violations := rule.CheckFile("suite_test.dart", src)
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations for commented @Skip, got %d", len(violations))
		}
	})
}

func TestSkipRule_Kotlin(t *testing.T) {
	rule := hygiene.NewSkipRule()

	t.Run("@Ignore annotation flagged", func(t *testing.T) {
		src := "class FooTest {\n    @Ignore(\"reason\")\n    @Test\n    fun testBar() {}\n}"
		violations := rule.CheckFile("FooTest.kt", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation for @Ignore, got %d", len(violations))
		}
		if violations[0].LineNumber != 2 {
			t.Errorf("expected line 2, got %d", violations[0].LineNumber)
		}
		if violations[0].RuleID != "GUARD-UNIV-SKIP" {
			t.Errorf("expected GUARD-UNIV-SKIP, got %s", violations[0].RuleID)
		}
		if violations[0].Severity != models.SeverityWarning {
			t.Errorf("expected Warning severity, got %s", violations[0].Severity)
		}
	})

	t.Run("@Disabled annotation flagged", func(t *testing.T) {
		src := "class FooTest {\n    @Disabled\n    @Test\n    fun testBar() {}\n}"
		violations := rule.CheckFile("FooTest.kt", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation for @Disabled, got %d", len(violations))
		}
		if violations[0].LineNumber != 2 {
			t.Errorf("expected line 2, got %d", violations[0].LineNumber)
		}
	})

	t.Run("commented annotation not flagged", func(t *testing.T) {
		src := "class FooTest {\n    // @Ignore\n    @Test\n    fun testBar() {}\n}"
		violations := rule.CheckFile("FooTest.kt", src)
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations for commented @Ignore, got %d", len(violations))
		}
	})
}

func TestSkipRule_Swift(t *testing.T) {
	rule := hygiene.NewSkipRule()

	t.Run("throw XCTSkip as first statement flagged", func(t *testing.T) {
		src := `import XCTest
class MyTests: XCTestCase {
    func testExample() throws {
        throw XCTSkip("feature not supported")
        let x = 10
        XCTAssertEqual(x, 10)
    }
}`
		violations := rule.CheckFile("MyTests.swift", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation for first-statement XCTSkip, got %d (%+v)", len(violations), violations)
		}
		if violations[0].LineNumber != 4 {
			t.Errorf("expected line 4, got %d", violations[0].LineNumber)
		}
		if violations[0].RuleID != "GUARD-UNIV-SKIP" {
			t.Errorf("expected GUARD-UNIV-SKIP, got %s", violations[0].RuleID)
		}
		if violations[0].Severity != models.SeverityWarning {
			t.Errorf("expected Warning severity, got %s", violations[0].Severity)
		}
	})

	t.Run("throw XCTSkip inside if condition not flagged", func(t *testing.T) {
		src := `import XCTest
class MyTests: XCTestCase {
    func testExample() throws {
        let condition = false
        if condition {
            throw XCTSkip("conditional skip")
        }
        XCTAssertTrue(true)
    }
}`
		violations := rule.CheckFile("MyTests.swift", src)
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations for conditional XCTSkip, got %d (%+v)", len(violations), violations)
		}
	})

	t.Run("@Test(.disabled) flagged", func(t *testing.T) {
		src := `import Testing
struct MyTests {
    @Test(.disabled("issue-123"))
    func example() {
        #expect(1 == 1)
    }
}`
		violations := rule.CheckFile("MyTests.swift", src)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation for @Test(.disabled), got %d (%+v)", len(violations), violations)
		}
		if violations[0].LineNumber != 3 {
			t.Errorf("expected line 3, got %d", violations[0].LineNumber)
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
	t.Run("N3: named Suite field from arbitrary package not treated as suite", func(t *testing.T) {
		src := `package sample
import "example.test/custom"
type S struct{named custom.Suite}
func(s *S) T() *custom.T {return nil}
func(s *S) TestUtility(){s.T().Skip("not a test")}
`
		res := rule.CheckFile("suite_skip_test.go", src)
		if len(res) != 0 {
			t.Errorf("expected 0 violations for arbitrary package named Suite field, got %d: %+v", len(res), res)
		}
	})
}
func TestFocusRule_TS(t *testing.T) {
	rule := hygiene.NewFocusRule()

	t.Run("all TS focus variants flagged", func(t *testing.T) {
		variants := []string{
			"it.only('focus test', () => {});",
			"test.only('focus test', () => {});",
			"describe.only('focus suite', () => {});",
			"test.describe.only('focus suite', () => {});",
			"fit('focused it', () => {});",
			"fdescribe('focused describe', () => {});",
			"it.only.each([1, 2])('focus %s', () => {});",
			"test.only.each([1, 2])('focus %s', () => {});",
			"describe.only.each([1, 2])('focus %s', () => {});",
			"test.describe.only.each([1, 2])('focus %s', () => {});",
		}

		for _, src := range variants {
			res := rule.CheckFile("focus.test.ts", src)
			if len(res) != 1 {
				t.Errorf("expected 1 violation for %q, got %d", src, len(res))
			} else {
				v := res[0]
				if v.RuleID != "GUARD-UNIV-FOCUS" {
					t.Errorf("expected GUARD-UNIV-FOCUS, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityWarning {
					t.Errorf("expected SeverityWarning, got %s", v.Severity)
				}
				if v.LineNumber != 1 {
					t.Errorf("expected line 1, got %d", v.LineNumber)
				}
			}
		}
	})

	t.Run("commented focus not flagged", func(t *testing.T) {
		src := "// it.only('focus', () => {});\n/* fit('focus', () => {}); */"
		res := rule.CheckFile("focus.test.ts", src)
		if len(res) != 0 {
			t.Fatalf("expected 0 violations for commented focus, got %d", len(res))
		}
	})

	t.Run("string containing focus not flagged", func(t *testing.T) {
		src := `const s = "it.only('not real focus')";`
		res := rule.CheckFile("focus.test.ts", src)
		if len(res) != 0 {
			t.Fatalf("expected 0 violations for string containing focus, got %d", len(res))
		}
	})
	t.Run("member calls and non-string arguments not flagged as focus", func(t *testing.T) {
		nonFindings := []string{
			"const trained = model.fit([1, 2, 3]);",
			"fit(model, data);",
			"obj.it.only('x', () => {});",
			"obj?.it.only('x', () => {});",
			"module.describe.only('x', () => {});",
		}
		for _, src := range nonFindings {
			res := rule.CheckFile("focus.test.ts", src)
			if len(res) != 0 {
				t.Errorf("expected 0 violations for %q, got %d (%+v)", src, len(res), res)
			}
		}

		// Valid declarations must still be flagged
		positives := []string{
			"fit('x', () => {});",
			"fdescribe('x', () => {});",
		}
		for _, src := range positives {
			res := rule.CheckFile("focus.test.ts", src)
			if len(res) != 1 {
				t.Errorf("expected 1 violation for %q, got %d", src, len(res))
			}
		}
	})

	t.Run("adversarial hygiene probe: Focus line 55 only, Skip line 51 only", func(t *testing.T) {
		src := `import { test as pw } from '@playwright/test';
import { a, counter, model, user, x, y } from './sut';

describe('must produce no finding', () => {
  it('calls on both sides are not tautologies', () => {
    expect(counter.next()).toBe(counter.next());
  });

  it('indexed operands are not tautologies', () => {
    const arr = a();
    expect(arr[0]).toBe(arr[0]);
  });

  it('different identifiers', () => {
    expect(x).toBe(y);
  });

  it('arithmetic with a variable', () => {
    expect(1 + x).toBe(2);
  });

  it('negated against a variable', () => {
    expect(x).not.toBe(2);
  });

  it('mentions it.only and it.skip in a string', () => {
    const doc = "use it.only('x') or it.skip('y') while debugging";
    expect(doc).toContain('it.only');
  });

  it('member call named fit is not a focused test', () => {
    const trained = model.fit([1, 2, 3]);
    expect(trained.loss).toBeLessThan(1);
  });

  it('callback-form conditional skip', () => {
    pw.skip(({ browserName }) => browserName === 'webkit', 'unsupported');
    expect(x).toBe(1);
  });
});

describe('must produce a finding', () => {
  it('dotted path tautology', () => {
    expect(user.name).toBe(user.name);
  });

  it('negated distinct numbers', () => {
    expect(1).not.toBe(2);
  });

  it.skip(` + "`template title skip`" + `, () => {
    expect(x).toBe(1);
  });

  describe.only('focused suite', () => {
    it('inner', () => {
      expect(x).toBe(1);
    });
  });

  it('duplicate one', () => {
    expect(a().length).toBe(3);
  });

  it('duplicate two', () => {
    expect(a().length).toBe(3);
  });
});`

		focusRule := hygiene.NewFocusRule()
		focusViolations := focusRule.CheckFile("adv_hygiene.test.ts", src)
		if len(focusViolations) != 1 {
			t.Fatalf("expected exactly 1 focus violation, got %d (%+v)", len(focusViolations), focusViolations)
		}
		if focusViolations[0].LineNumber != 55 {
			t.Errorf("expected focus violation at line 55, got line %d", focusViolations[0].LineNumber)
		}
		if focusViolations[0].RuleID != "GUARD-UNIV-FOCUS" {
			t.Errorf("expected GUARD-UNIV-FOCUS, got %s", focusViolations[0].RuleID)
		}

		skipRule := hygiene.NewSkipRule()
		skipViolations := skipRule.CheckFile("adv_hygiene.test.ts", src)
		if len(skipViolations) != 1 {
			t.Fatalf("expected exactly 1 skip violation, got %d (%+v)", len(skipViolations), skipViolations)
		}
		if skipViolations[0].LineNumber != 51 {
			t.Errorf("expected skip violation at line 51, got line %d", skipViolations[0].LineNumber)
		}
		if skipViolations[0].RuleID != "GUARD-UNIV-SKIP" {
			t.Errorf("expected GUARD-UNIV-SKIP, got %s", skipViolations[0].RuleID)
		}
	})
}

func TestFocusRule_Dart(t *testing.T) {
	rule := hygiene.NewFocusRule()

	t.Run("solo true flagged in test, testWidgets, group, blocTest", func(t *testing.T) {
		calls := []struct {
			name string
			src  string
		}{
			{"test", "void main() {\n  test('focused', () {}, solo: true);\n}"},
			{"testWidgets", "void main() {\n  testWidgets('focused', (tester) async {}, solo: true);\n}"},
			{"group", "void main() {\n  group('focused group', () {}, solo: true);\n}"},
			{"blocTest", "void main() {\n  blocTest('focused bloc', build: () => bloc, solo: true);\n}"},
		}

		for _, tc := range calls {
			res := rule.CheckFile("main_test.dart", tc.src)
			if len(res) != 1 {
				t.Errorf("expected 1 violation for %s solo: true, got %d", tc.name, len(res))
			} else {
				v := res[0]
				if v.RuleID != "GUARD-UNIV-FOCUS" {
					t.Errorf("expected GUARD-UNIV-FOCUS for %s, got %s", tc.name, v.RuleID)
				}
				if v.Severity != models.SeverityWarning {
					t.Errorf("expected SeverityWarning for %s, got %s", tc.name, v.Severity)
				}
				if v.LineNumber != 2 {
					t.Errorf("expected line 2 for %s, got %d", tc.name, v.LineNumber)
				}
			}
		}
	})

	t.Run("solo false not flagged", func(t *testing.T) {
		src := "void main() {\n  test('not focused', () {}, solo: false);\n}"
		res := rule.CheckFile("main_test.dart", src)
		if len(res) != 0 {
			t.Fatalf("expected 0 violations for solo: false, got %d", len(res))
		}
	})

	t.Run("solo true in unrelated call not flagged", func(t *testing.T) {
		src := "void main() {\n  customHelper(solo: true);\n}"
		res := rule.CheckFile("main_test.dart", src)
		if len(res) != 0 {
			t.Fatalf("expected 0 violations for customHelper(solo: true), got %d", len(res))
		}
	})
}
