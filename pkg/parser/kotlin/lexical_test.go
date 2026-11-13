package kotlin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/kotlin"
)

func TestKotlinAdapter_StringTemplateWithNestedStringKeepsScope(t *testing.T) {
	src := `package probe

class IsoC {
    @Test
    fun c3TemplateWithNestedString() {
        val text = "${listOf("}").joinToString("{")} end"
        assertEquals("} end", text)
    }

    @Test
    fun c9AfterTemplate() {
        assertEquals(3, add(1, 2))
    }
}
`
	units, err := kotlin.NewAdapter().LocateTests([]byte(src), "IsoTest.kt")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d: %+v", len(units), names(units))
	}

	first, second := units[0], units[1]
	if first.Name != "c3TemplateWithNestedString" || first.Scope != "IsoC" {
		t.Errorf("unexpected first unit: name=%q scope=%q", first.Name, first.Scope)
	}
	if first.EndLine != 8 {
		t.Errorf("expected first unit to end at line 8, got %d (body=%q)", first.EndLine, first.RawBody)
	}
	if len(first.Assertions) == 0 {
		t.Errorf("expected template test to keep its shouldBe assertion, got none (body=%q)", first.RawBody)
	}
	if second.Name != "c9AfterTemplate" || second.Scope != "IsoC" {
		t.Errorf("expected the test after the template to keep scope IsoC, got name=%q scope=%q", second.Name, second.Scope)
	}
}

func TestKotlinAdapter_NestedBlockCommentHidesIgnore(t *testing.T) {
	src := `class KtTest {
    /* @Test fun commented() { } /* nested */ @Ignore */
    // @Disabled
    @Test
    fun realTest() {
        assertEquals(3, add(1, 2))
    }

    @Test
    fun requireAndCheck() {
        val user = Service(repo).current()
        check(user.name == "a") { "name mismatch" }
    }
}
`
	units, err := kotlin.NewAdapter().LocateTests([]byte(src), "KtTest.kt")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d: %+v", len(units), names(units))
	}
	if units[0].Name != "realTest" || units[0].Scope != "KtTest" {
		t.Errorf("unexpected unit: name=%q scope=%q", units[0].Name, units[0].Scope)
	}
	if units[1].Name != "requireAndCheck" || units[1].Scope != "KtTest" {
		t.Errorf("unexpected unit: name=%q scope=%q", units[1].Name, units[1].Scope)
	}
	if len(units[1].Assertions) == 0 {
		t.Errorf("expected the bare check(...) call to count as an assertion, got none (body=%q)", units[1].RawBody)
	}
}

func TestKotlinAdapter_BacktickNameWithBraceKeepsScope(t *testing.T) {
	src := "class BraceTest {\n" +
		"    @Test\n" +
		"    fun `name with { brace } and quote \"`() {\n" +
		"        assertEquals(3, add(1, 2))\n" +
		"    }\n" +
		"\n" +
		"    @Test\n" +
		"    fun afterBacktick() {\n" +
		"        assertEquals(4, add(2, 2))\n" +
		"    }\n" +
		"}\n"
	units, err := kotlin.NewAdapter().LocateTests([]byte(src), "BraceTest.kt")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d: %+v", len(units), names(units))
	}
	if units[0].Name != `name with { brace } and quote "` {
		t.Errorf("expected backtick identifier name preserved, got %q", units[0].Name)
	}
	if units[1].Name != "afterBacktick" || units[1].Scope != "BraceTest" {
		t.Errorf("expected following test to keep scope BraceTest, got name=%q scope=%q", units[1].Name, units[1].Scope)
	}
}

func TestKotlinAdapter_RejectsStructurallyInvalidInput(t *testing.T) {
	cases := map[string]string{
		"unterminated string": `class A {
    @Test
    fun broken() {
        val s = "oops
        assertEquals(1, 1)
    }
}
`,
		"unbalanced brace": `class A {
    @Test
    fun broken() {
        assertEquals(1, 1)
}
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			units, err := kotlin.NewAdapter().LocateTests([]byte(src), "Broken.kt")
			if err == nil {
				t.Fatalf("expected a lexical error, got %d units: %+v", len(units), names(units))
			}
			if !strings.Contains(err.Error(), "kotlin:") {
				t.Errorf("expected error to name the language, got %q", err)
			}
		})
	}
}

func TestKotlinAdapter_LexicalEdgeFixture(t *testing.T) {
	fixturePath := filepath.Join("testdata", "LexicalEdgeTest.kt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	units, err := kotlin.NewAdapter().LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	want := []string{
		"charLiteralBrace",
		"charLiteralQuote",
		"templateWithNestedStringBrace",
		"afterTemplate",
		"afterCommentedIgnore",
		`backtick name with { brace } and quote "`,
		"afterBacktick",
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d: %+v", len(want), len(units), names(units))
	}
	for i, name := range want {
		if units[i].Name != name {
			t.Errorf("unit %d: expected name %q, got %q", i, name, units[i].Name)
		}
		if units[i].Scope != "LexicalEdgeTest" {
			t.Errorf("unit %q: expected scope LexicalEdgeTest, got %q", name, units[i].Scope)
		}
		if len(units[i].Assertions) == 0 {
			t.Errorf("unit %q: expected an assertion, got none (body=%q)", name, units[i].RawBody)
		}
	}
}

func TestKotlinAdapter_RawStringSurplusQuotesAreContent(t *testing.T) {
	cases := []struct {
		name       string
		literal    string
		wantTests  int
		wantAssert string
	}{
		{"raw string ending in quote", `private val fourQuotes = """ends with quote""""`, 1, "assertEquals"},
		{"quoted regex fragment", `private val regex = """\d+\s"(\w+)"""".toRegex()`, 1, "assertEquals"},
		{"plain raw string", `private val plain = """no surplus quotes"""`, 1, "assertEquals"},
		{"empty raw string", `private val empty = """"""`, 1, "assertEquals"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "class RawTest {\n    " + tc.literal + "\n\n    @Test\n    fun realTest() {\n        assertEquals(1, 1)\n    }\n}\n"
			units, err := kotlin.NewAdapter().LocateTests([]byte(src), "RawTest.kt")
			if err != nil {
				t.Fatalf("expected valid raw string to scan, got error: %v", err)
			}
			if len(units) != tc.wantTests {
				t.Fatalf("expected %d unit, got %d: %+v", tc.wantTests, len(units), names(units))
			}
			if units[0].Name != "realTest" || units[0].Scope != "RawTest" {
				t.Errorf("unexpected unit: name=%q scope=%q", units[0].Name, units[0].Scope)
			}
			if len(units[0].Assertions) == 0 || units[0].Assertions[0] != tc.wantAssert {
				t.Errorf("expected assertion %q, got %v", tc.wantAssert, units[0].Assertions)
			}
		})
	}
}

func TestKotlinAdapter_RawQuoteEdgeFixture(t *testing.T) {
	fixturePath := filepath.Join("testdata", "RawQuoteEdgeTest.kt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	units, err := kotlin.NewAdapter().LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	want := []struct {
		name  string
		line  int
		first string
	}{
		{"fourQuoteLength", 24, "assertEquals"},
		{"windowsPathLength", 29, "assertEquals"},
		{"regexWithQuotedFragment", 34, "assertTrue"},
		{"rawTemplateKeepsBraces", 39, "assertTrue"},
		{`name with ' apostrophe and ( paren`, 44, "assertEquals"},
		{"whenExpressionOnChars", 50, "assertEquals"},
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d: %+v", len(want), len(units), names(units))
	}
	for i, w := range want {
		if units[i].Name != w.name {
			t.Errorf("unit %d: expected name %q, got %q", i, w.name, units[i].Name)
		}
		if units[i].StartLine != w.line {
			t.Errorf("unit %q: expected start line %d, got %d", w.name, w.line, units[i].StartLine)
		}
		if units[i].Scope != "RawQuoteEdgeTest" {
			t.Errorf("unit %q: expected scope RawQuoteEdgeTest, got %q", w.name, units[i].Scope)
		}
		if len(units[i].Assertions) == 0 || units[i].Assertions[0] != w.first {
			t.Errorf("unit %q: expected first assertion %q, got %v", w.name, w.first, units[i].Assertions)
		}
	}
}

func TestKotlinAdapter_UnterminatedRawStringStillFails(t *testing.T) {
	src := "class A {\n    @Test\n    fun broken() {\n        val raw = \"\"\"no closing delimiter\n    }\n}\n"
	units, err := kotlin.NewAdapter().LocateTests([]byte(src), "Broken.kt")
	if err == nil {
		t.Fatalf("expected a lexical error for an unterminated raw string, got %d units: %+v", len(units), names(units))
	}
	if !strings.Contains(err.Error(), "kotlin: unterminated") {
		t.Errorf("expected an unterminated-construct error, got %q", err)
	}
}

func names(units []parser.TestUnit) []string {
	out := make([]string, 0, len(units))
	for _, u := range units {
		out = append(out, u.Name)
	}
	return out
}
