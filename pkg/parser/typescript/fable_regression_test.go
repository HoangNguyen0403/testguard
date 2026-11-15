package typescript_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
	ts "github.com/HoangNguyen0403/testguard/pkg/parser/typescript"
)

func locateTests(t *testing.T, path string) []parser.TestUnit {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	units, err := ts.NewAdapter().LocateTests(content, path)
	if err != nil {
		t.Fatalf("LocateTests(%s): %v", path, err)
	}
	return units
}

func unitNames(units []parser.TestUnit) []string {
	names := make([]string, 0, len(units))
	for _, unit := range units {
		names = append(names, unit.Name)
	}
	return names
}

func findUnit(t *testing.T, units []parser.TestUnit, name string) parser.TestUnit {
	t.Helper()
	for _, unit := range units {
		if unit.Name == name {
			return unit
		}
	}
	t.Fatalf("unit %q not found in %v", name, unitNames(units))
	return parser.TestUnit{}
}

// TestF1_DeclarationsOnly is the regression for the review finding that any
// it/test/fit call became a test unit: Playwright hooks, fixture configuration
// and local functions named test each collected a false zero-assertion blocker.
func TestF1_DeclarationsOnly(t *testing.T) {
	playwright := locateTests(t, "testdata/fable/playwright_hooks.spec.ts")
	wantPlaywright := []string{
		"L15 adds item",
		"L20 steps",
		"L31 conditional skip in body",
		"L36 soft and poll",
		"L41 waitForURL is an implicit check",
	}
	got := unitNames(playwright)
	if len(got) != len(wantPlaywright) {
		t.Fatalf("playwright fixture: expected %d units, got %d: %v", len(wantPlaywright), len(got), got)
	}
	for i, name := range wantPlaywright {
		if got[i] != name {
			t.Errorf("playwright fixture unit[%d]: expected %q, got %q", i, name, got[i])
		}
	}
	if unit := findUnit(t, playwright, "L20 steps"); len(unit.Assertions) != 1 {
		t.Errorf("L20 steps: expected the in-step assertion only, got %v", unit.Assertions)
	}
	for _, unit := range playwright {
		if unit.Scope != "checkout" {
			t.Errorf("playwright fixture %q: expected scope 'checkout', got %q", unit.Name, unit.Scope)
		}
	}

	curried := locateTests(t, "testdata/fable/curried_modifiers.test.ts")
	wantCurried := []string{
		"L17 conditional skipIf",
		"L21 conditional runIf",
		"L25 concurrent",
		"L29 expected failure",
		"L33 regex literal mentioning it.skip( and it.only(",
		"L38 partial mock keeps real service",
		"L42 waits with real timer zero and uses test_data",
		"L48 multiline declaration",
		"L57 suppressed smoke",
		"L61 unicode title — naïve café ✓",
	}
	got = unitNames(curried)
	if len(got) != len(wantCurried) {
		t.Fatalf("curried fixture: expected %d units, got %d: %v", len(wantCurried), len(got), got)
	}
	for i, name := range wantCurried {
		if got[i] != name {
			t.Errorf("curried fixture unit[%d]: expected %q, got %q", i, name, got[i])
		}
	}
	for _, unit := range curried {
		if unit.Name == "L57 suppressed smoke" {
			continue
		}
		if len(unit.Assertions) == 0 {
			t.Errorf("curried fixture %q: expected the real body to be analysed", unit.Name)
		}
	}

	declarations := locateTests(t, "testdata/fable/declaration_shapes.test.js")
	wantDeclarations := []string{
		"L16 uses local function named test",
		"L20 comparison chain in js file",
		"L26 object matcher",
	}
	got = unitNames(declarations)
	if len(got) != len(wantDeclarations) {
		t.Fatalf("declaration fixture: expected %d units, got %d: %v", len(wantDeclarations), len(got), got)
	}
	for i, name := range wantDeclarations {
		if got[i] != name {
			t.Errorf("declaration fixture unit[%d]: expected %q, got %q", i, name, got[i])
		}
		if len(declarations[i].Assertions) == 0 {
			t.Errorf("declaration fixture %q: expected assertions", declarations[i].Name)
		}
	}
}

// TestF2_LexerKeepsEveryTest is the regression for the review finding that a
// regex after an operator, or a template nested inside an interpolation, derailed
// the lexer and silently dropped every following test.
func TestF2_LexerKeepsEveryTest(t *testing.T) {
	nested := locateTests(t, "testdata/fable/nested_template.test.ts")
	wantNested := []string{"L4 first", "L9 second", "L13 third"}
	got := unitNames(nested)
	if len(got) != len(wantNested) {
		t.Fatalf("nested template fixture: expected %d units, got %d: %v", len(wantNested), len(got), got)
	}
	for i, name := range wantNested {
		if got[i] != name {
			t.Errorf("nested template fixture unit[%d]: expected %q, got %q", i, name, got[i])
		}
		if len(nested[i].Assertions) == 0 {
			t.Errorf("nested template fixture %q: expected assertions", name)
		}
		if nested[i].Scope != "nested template with literal brace" {
			t.Errorf("nested template fixture %q: expected the describe scope, got %q", name, nested[i].Scope)
		}
	}

	libraries := locateTests(t, "testdata/fable/library_assertions.test.ts")
	if len(libraries) != 12 {
		t.Fatalf("library fixture: expected 12 units, got %d: %v", len(libraries), unitNames(libraries))
	}
	for _, unit := range libraries {
		if len(unit.Assertions) == 0 {
			t.Errorf("library fixture %q: expected an assertion", unit.Name)
		}
	}
	if libraries[8].Name != "L56 regex after arrow containing quote" {
		t.Errorf("library fixture unit[8]: expected the test after the arrow regex, got %q", libraries[8].Name)
	}

	valid := locateTests(t, "testdata/fable/valid_suite.test.ts")
	if len(valid) != 22 {
		t.Fatalf("valid suite fixture: expected 22 units, got %d: %v", len(valid), unitNames(valid))
	}
	for _, unit := range valid {
		if len(unit.Assertions) == 0 {
			t.Errorf("valid suite fixture %q: expected an assertion", unit.Name)
		}
	}

	jsx := locateTests(t, "testdata/fable/jsx_suite.test.tsx")
	if len(jsx) != 12 {
		t.Fatalf("jsx suite fixture: expected 12 units, got %d: %v", len(jsx), unitNames(jsx))
	}
	for _, unit := range jsx {
		if len(unit.Assertions) == 0 {
			t.Errorf("jsx suite fixture %q: expected an assertion", unit.Name)
		}
	}
	if jsx[0].Scope != "suite A" || jsx[3].Scope != "suite B" {
		t.Errorf("jsx suite fixture: expected suite scopes, got %q and %q", jsx[0].Scope, jsx[3].Scope)
	}
}

// TestF2_MalformedSourceFailsClosed pins the fail-closed contract: a truncated
// scan must surface as an adapter error so the engine reports parse_failed,
// never a shorter test list analysed as complete.
func TestF2_MalformedSourceFailsClosed(t *testing.T) {
	cases := map[string]string{
		"unclosed brace":        `it('t', () => { expect(1).toBe(1);`,
		"unclosed paren":        `describe('d', () => { it('t', () => { expect(1).toBe(1); });`,
		"unterminated string":   "it('t', () => { const s = 'oops\n expect(1).toBe(1); });",
		"unterminated template": "it('t', () => { const s = `oops\n expect(1).toBe(1); });",
		"unterminated comment":  "it('t', () => { /* oops\n expect(1).toBe(1); });",
		"unexpected close":      `it('t', () => { expect(1).toBe(1); }); }`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ts.NewAdapter().LocateTests([]byte(src), "broken.test.ts"); err == nil {
				t.Fatalf("expected a lexical error, got a successful scan")
			}
		})
	}
}

// TestF3_AssertionRecognition is the regression for the review finding that
// common chai, sinon-chai, chai-as-promised, testdouble and Angular verification
// shapes produced false zero-assertion blockers.
func TestF3_AssertionRecognition(t *testing.T) {
	units := locateTests(t, "testdata/fable/library_assertions.test.ts")
	expected := map[string][]string{
		"L18 generic expect call":          {"expect<number>(sum(1, 2)).toBe(3)"},
		"L22 chai getter empty":            {"expect(svc.list()).to.be.empty"},
		"L26 sinon-chai calledOnce getter": {"expect(spy).to.have.been.calledOnce"},
		"L32 chai-as-promised rejected":    {"expect(load('bad')).to.be.rejected"},
		"L36 testdouble verify":            {"verify(save(1))"},
		"L42 angular http mock verify":     {"expectOne('/api/items')", "verify()"},
		"L48 typed const helper":           {"typedCheck(sum(1, 2))"},
		"L52 page object method helper":    {"expectTitle('home')"},
	}
	for name, want := range expected {
		unit := findUnit(t, units, name)
		if len(unit.Assertions) != len(want) {
			t.Errorf("%s: expected assertions %v, got %v", name, want, unit.Assertions)
			continue
		}
		for i, raw := range want {
			if unit.Assertions[i] != raw {
				t.Errorf("%s: expected assertion[%d] %q, got %q", name, i, raw, unit.Assertions[i])
			}
		}
	}
}

// TestF3_PrecisionBoundaries pins the cases that must stay uncredited: an expect
// shadowed in the test body and a helper declared in a sibling describe block.
// An assertion inside an unreferenced nested function is deliberately credited
// again (inert-function inference was removed in R4).
func TestF3_PrecisionBoundaries(t *testing.T) {
	units := locateTests(t, "testdata/fable/precision_pos.test.ts")
	if len(units) != 15 {
		t.Fatalf("precision fixture: expected 15 units, got %d: %v", len(units), unitNames(units))
	}
	uncredited := map[string]bool{
		"L4 ZERO only logs":                                                 true,
		"L8 ZERO expect shadowed by local stub":                             true,
		"L13 ZERO helper with same name as asserting helper in other suite": true,
	}
	for _, unit := range units {
		if uncredited[unit.Name] {
			if len(unit.Assertions) != 0 {
				t.Errorf("%s: expected no credited assertion, got %v", unit.Name, unit.Assertions)
			}
			continue
		}
		if len(unit.Assertions) == 0 {
			t.Errorf("%s: expected an assertion", unit.Name)
		}
	}
	if unit := findUnit(t, units, "L85 valid uses verify"); len(unit.Assertions) != 1 {
		t.Errorf("L85: a helper declared in the same describe must stay credited, got %v", unit.Assertions)
	}
	findUnit(t, units, "L72 FOCUS fit")
}

// TestF3_TsMockitoVerification pins ts-mockito's bare `verify` form: it is a
// real assertion only when the identifier is imported from ts-mockito (directly,
// aliased, or through a namespace) and the call is terminated by a verification
// matcher such as `.once()`. An arbitrary or shadowed `verify` stays uncredited.
func TestF3_TsMockitoVerification(t *testing.T) {
	units := locateTests(t, "testdata/fable/ts_mockito_verify.test.ts")
	credited := map[string]string{
		"credits a call-terminated verify imported from ts-mockito": "verify(repository.save('item')).once()",
		"credits a pivot-count matcher":                             "verify(repository.save('item')).times(2)",
		"credits an aliased verify import":                          "verifyCalls(repository.save('item')).never()",
		"credits a namespace-qualified verify":                      "tsm.verify(repository.save('item')).atLeast(1)",
	}
	for name, want := range credited {
		unit := findUnit(t, units, name)
		if len(unit.Assertions) != 1 || unit.Assertions[0] != want {
			t.Errorf("%s: expected assertion %q, got %v", name, want, unit.Assertions)
		}
	}
	for _, name := range []string{
		"does not credit a verify call with no matcher",
		"does not credit a locally shadowed verify",
	} {
		if unit := findUnit(t, units, name); len(unit.Assertions) != 0 {
			t.Errorf("%s: expected no credited assertion, got %v", name, unit.Assertions)
		}
	}

	unrelated := locateTests(t, "testdata/fable/ts_mockito_verify_unrelated.test.ts")
	if len(unrelated) != 2 {
		t.Fatalf("unrelated verify fixture: expected 2 units, got %d: %v", len(unrelated), unitNames(unrelated))
	}
	for _, unit := range unrelated {
		if len(unit.Assertions) != 0 {
			t.Errorf("%s: a verify that does not come from ts-mockito must not be credited, got %v", unit.Name, unit.Assertions)
		}
	}
}

// TestBodyStartLineMapping pins the body_start_line metadata contract: it must
// name the original source line holding the first byte of RawBody, including for
// multi-line declarations.
func TestBodyStartLineMapping(t *testing.T) {
	const path = "testdata/fable/curried_modifiers.test.ts"
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	units := locateTests(t, path)
	lines := strings.Split(string(content), "\n")
	for _, unit := range units {
		raw, ok := unit.Meta["body_start_line"]
		if !ok {
			t.Fatalf("%q: body_start_line missing from Meta %v", unit.Name, unit.Meta)
		}
		line, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("%q: body_start_line %q is not decimal", unit.Name, raw)
		}
		if line < 1 || line > len(lines) {
			t.Fatalf("%q: body_start_line %d outside 1..%d", unit.Name, line, len(lines))
		}
		firstLine := strings.SplitN(unit.RawBody, "\n", 2)[0]
		if !strings.Contains(lines[line-1], firstLine) {
			t.Errorf("%q: body_start_line %d points at %q, which does not hold the RawBody start %q",
				unit.Name, line, lines[line-1], firstLine)
		}
		if unit.StartLine > line {
			t.Errorf("%q: StartLine %d must not follow body_start_line %d", unit.Name, unit.StartLine, line)
		}
	}

	multiline := findUnit(t, units, "L48 multiline declaration")
	if multiline.Meta["body_start_line"] != "49" {
		t.Errorf("multiline declaration: expected body_start_line 49, got %q", multiline.Meta["body_start_line"])
	}
	if multiline.StartLine != 47 {
		t.Errorf("multiline declaration: expected StartLine 47, got %d", multiline.StartLine)
	}
}
