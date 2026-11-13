package typescript_test

import (
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
	ts "github.com/HoangNguyen0403/testguard/pkg/parser/typescript"
)

func locateSource(t *testing.T, src string) []parser.TestUnit {
	t.Helper()
	units, err := ts.NewAdapter().LocateTests([]byte(src), "r2.test.ts")
	if err != nil {
		t.Fatalf("LocateTests: %v", err)
	}
	return units
}

// N1: a named function that is referenced by name, alias, callback or JSX prop
// keeps its assertions credited (still true after the R4 removal of inference).
func TestN1_ReferencedFunctionsAreNotInert(t *testing.T) {
	cases := map[string]string{
		"handler by reference": `it('a', (done) => {
  function onData(v) { expect(v).toBe(1); done(); }
  emitter.on('data', onData);
});`,
		"forEach reference": `it('a', () => {
  function checkItem(n) { expect(n).toBeGreaterThan(0); }
  [1, 2, 3].forEach(checkItem);
});`,
		"alias": `it('a', () => {
  function verifyTotal() { expect(total()).toBe(3); }
  const run = verifyTotal;
  run();
});`,
		"named callback": `it('a', function mochaStyle() { expect(1).toBe(1); });`,
		"named forEach callback": `it('a', () => {
  [1, 2].forEach(function checkItem(n) { expect(n).toBeTruthy(); });
});`,
		"named promise executor": `it('a', () => {
  return new Promise(function executor(resolve) { expect(1).toBe(1); resolve(); });
});`,
		"shorthand property": `it('a', () => {
  function check() { expect(1).toBe(1); }
  register({ check });
});`,
		"async declaration referenced": `it('a', async () => {
  async function check() { expect(1).toBe(1); }
  await Promise.all([check()]);
});`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			units := locateSource(t, src)
			if len(units) != 1 {
				t.Fatalf("expected 1 unit, got %d", len(units))
			}
			if len(units[0].Assertions) == 0 {
				t.Fatalf("assertion in referenced/expression function was discarded")
			}
		})
	}
}

// N3: identifier, member and call titles are discovered when a function
// literal callback accompanies them; the Name is the title expression source.
func TestN3_DynamicTitles(t *testing.T) {
	src := `import { it } from 'vitest';
cases.forEach(({ name, input }) => {
  it(name, () => { expect(run(input)).toBe(1); });
});
for (const c of cases) {
  test(c.name, async () => { expect(await go(c)).toBeTruthy(); });
}
it(TITLE, () => {});
it(title(1), function () { service.execute(); });
specify('mocha alias', () => { expect(1).toBe(1); });
specify(name, () => {});
test(name, { tag: '@x' }, async () => { expect(1).toBe(1); });
test(name, () => { expect(1).toBe(1); }, 5000);
`
	units := locateSource(t, src)
	want := []struct {
		name    string
		dynamic bool
		asserts int
	}{
		{"name", true, 1},
		{"c.name", true, 1},
		{"TITLE", true, 0},
		{"title(1)", true, 0},
		{"mocha alias", false, 1},
		{"name", true, 0},
		{"name", true, 1},
		{"name", true, 1},
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d: %v", len(want), len(units), unitNames(units))
	}
	for i, w := range want {
		u := units[i]
		if u.Name != w.name || (u.Meta["title_dynamic"] == "true") != w.dynamic || len(u.Assertions) != w.asserts {
			t.Errorf("unit %d = {%q dynamic=%q asserts=%d}, want %+v", i, u.Name, u.Meta["title_dynamic"], len(u.Assertions), w)
		}
		if u.Meta["body_start_line"] == "" {
			t.Errorf("unit %d lost body_start_line", i)
		}
	}
}

// N3 negatives: shapes that are configuration, hooks, locals or plain calls
// must not manufacture units.
func TestN3_DynamicTitleNegatives(t *testing.T) {
	cases := map[string]string{
		"no callback":        `test(1, 2);`,
		"array arg":          `fit([1, 2]);`,
		"use config":         `test.use({ locale: 'en' });`,
		"hook":               `test.beforeEach(async () => { await x(); });`,
		"skip with callback": `test.skip(cond, () => { expect(1).toBe(1); });`,
		"member call":        `suite.it(name, () => { expect(1).toBe(1); });`,
		"local function":     "function test(name, fn) { fn(); }\ntest(name, () => { expect(1).toBe(1); });",
		"local arrow":        "const it = (n, fn) => fn();\nit(name, () => { expect(1).toBe(1); });",
		"method definition":  "class A { test(name, fn) { return fn; } }",
		"title is callback":  `it(() => {}, () => {});`,
		"callback only":      `it(() => { expect(1).toBe(1); });`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if units := locateSource(t, src); len(units) != 0 {
				t.Fatalf("expected no units, got %v", unitNames(units))
			}
		})
	}
}

// A framework fixture named test (base.extend) is not a local function.
func TestN3_FixtureTestStillDiscovered(t *testing.T) {
	src := "const test = base.extend({});\ntest(name, async ({ page }) => { await expect(page).toHaveURL('/'); });"
	if units := locateSource(t, src); len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}
}

// Titles built from a literal plus more tokens are expressions: the Name keeps
// the complete source and the unit is marked dynamic. Plain literals and static
// templates stay literal.
func TestN3_ConcatenatedAndInterpolatedTitlesAreDynamic(t *testing.T) {
	src := "it('foo' + suffix, () => { expect(1).toBe(1); });\n" +
		"it(`foo${suffix}`, () => { expect(1).toBe(1); });\n" +
		"it('a' + f(1, 2) + 'b', () => { expect(1).toBe(1); });\n" +
		"it('foo', () => { expect(1).toBe(1); });\n" +
		"it(`static`, () => { expect(1).toBe(1); });\n"
	units := locateSource(t, src)
	want := []struct {
		name    string
		dynamic bool
	}{
		{"'foo' + suffix", true},
		{"`foo${suffix}`", true},
		{"'a' + f(1, 2) + 'b'", true},
		{"foo", false},
		{"static", false},
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d: %v", len(want), len(units), unitNames(units))
	}
	for i, w := range want {
		u := units[i]
		if u.Name != w.name || (u.Meta["title_dynamic"] == "true") != w.dynamic {
			t.Errorf("unit %d = {%q dynamic=%q}, want %+v", i, u.Name, u.Meta["title_dynamic"], w)
		}
		if len(u.Assertions) != 1 || !strings.HasPrefix(u.RawBody, "() =>") {
			t.Errorf("unit %d body/assertions shifted: %q %v", i, u.RawBody, u.Assertions)
		}
	}
}

func locateSourceAs(t *testing.T, name, src string) []parser.TestUnit {
	t.Helper()
	units, err := ts.NewAdapter().LocateTests([]byte(src), name)
	if err != nil {
		t.Fatalf("LocateTests: %v", err)
	}
	return units
}

// R3-1: a function referenced only from a JSX attribute container may run.
func TestR3_JSXAttributeReferenceKeepsAssertions(t *testing.T) {
	credited := map[string]string{
		"attribute reference": `it('a', () => {
  function handleClick(e) { expect(e.type).toBe('click'); }
  render(<Button onClick={handleClick}>Go</Button>);
});`,
		"self-closing, nested expression": `it('a', () => {
  function handleClick(e) { expect(e.type).toBe('click'); }
  render(<Button label="x" onClick={(e) => handleClick(e)} />);
});`,
		"child expression": `it('a', () => {
  function renderRow(n) { expect(n).toBeGreaterThan(0); return <li>{n}</li>; }
  render(<ul>{[1, 2].map(renderRow)}</ul>);
});`,
	}
	for name, src := range credited {
		t.Run(name, func(t *testing.T) {
			units := locateSourceAs(t, "c.test.tsx", src)
			if len(units) != 1 || len(units[0].Assertions) == 0 {
				t.Fatalf("assertion in JSX-referenced function was discarded: %d units", len(units))
			}
		})
	}
}

// R4-1: the inert-function inference was removed. A nested function's assertions
// are credited even when nothing visibly calls it — components used only as JSX
// tags, refs, props and callbacks cannot be proven unreachable lexically. The
// accepted trade-off is that a genuinely unreferenced nested function with
// assertions no longer yields a ZERO finding.
func TestR4_NestedFunctionAssertionsAreCredited(t *testing.T) {
	cases := map[string]string{
		"component used only as a self-closing JSX tag": `it('a', () => {
  function Probe() { const v = useContext(Ctx); expect(v).toBe(1); return null; }
  render(<Provider value={1}>
    <Probe />
  </Provider>);
});`,
		"component used only as an opening tag with children": `it('a', () => {
  function Shell(props) { expect(props.children).toBeTruthy(); return <div>{props.children}</div>; }
  render(<Shell>text</Shell>);
});`,
		"unreferenced declaration (documented recall gap)": `it('a', () => {
  function neverUsed() { expect(1).toBe(1); }
});`,
		"typed-return recursive function": `it('a', () => {
  function walk(n: number): void { expect(n).toBeGreaterThanOrEqual(0); if (n > 0) walk(n - 1); }
  sum(1, 2);
});`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			units := locateSourceAs(t, "c.test.tsx", src)
			if len(units) != 1 || len(units[0].Assertions) != 1 {
				t.Fatalf("expected 1 unit with 1 credited assertion, got %d units", len(units))
			}
		})
	}

	// The conservative credit does not extend to what is not an assertion: a
	// nested function that only logs still yields no assertion.
	units := locateSourceAs(t, "c.test.tsx", `it('a', () => { function Probe() { console.log(1); return null; } render(<Probe />); });`)
	if len(units) != 1 || len(units[0].Assertions) != 0 {
		t.Fatalf("nested function without an assertion must stay uncredited, got %v", units)
	}
}

// R4-2: a type-argument list on a JSX element is skipped before the attributes.
func TestR4_GenericJSXElements(t *testing.T) {
	valid := map[string]string{
		"object type argument": `it('a', () => {
  render(<Table<{ n: number }> rows={[{ n: 1 }]} onRow={f} />);
  expect(1).toBe(1);
});
it('b', () => { expect(2).toBe(2); });`,
		"nested generic": `it('a', () => {
  render(<Table<Array<Map<string, number>>> rows={[]} />);
  expect(1).toBe(1);
});
it('b', () => { expect(2).toBe(2); });`,
		"function type and arrow": `it('a', () => {
  render(<Select<(v: Array<number>) => void, { k: '>' }> onPick={f}>x</Select>);
  expect(1).toBe(1);
});
it('b', () => { expect(2).toBe(2); });`,
		"multi-line with comment and string": `it('a', () => {
  render(
    <Table<
      // a > comment
      { s: "a>b" }
    >
      rows={[]}
    />,
  );
  expect(1).toBe(1);
});
it('b', () => { expect(2).toBe(2); });`,
		"non-generic still works": `it('a', () => { render(<Table rows={[]} />); expect(1).toBe(1); });
it('b', () => { expect(2).toBe(2); });`,
	}
	for name, src := range valid {
		t.Run(name, func(t *testing.T) {
			units := locateSourceAs(t, "g.test.tsx", src)
			if len(units) != 2 || len(units[0].Assertions) != 1 || len(units[1].Assertions) != 1 {
				t.Fatalf("expected 2 analysed units after a generic element, got %v", unitNames(units))
			}
		})
	}

	t.Run("malformed type arguments fail closed", func(t *testing.T) {
		src := `it('a', () => { render(<Table<{ n: number } rows={[]} />); expect(1).toBe(1); });
it('b', () => { expect(2).toBe(2); });`
		if _, err := ts.NewAdapter().LocateTests([]byte(src), "m.test.tsx"); err == nil {
			t.Fatal("expected a parse error for an unterminated type-argument list")
		}
	})
}

// R3-2: an imported or otherwise bound fit is a utility, not Jasmine's focused
// test; the global fit, and it/test/specify/fixtures, are still discovered.
func TestR3_BoundFitIsNotATest(t *testing.T) {
	notTests := map[string]string{
		"named import":      "import { fit } from './chart-layout';\nit('t', () => { const s = fit(bounds, () => 2); expect(s).toBe(2); });",
		"default import":    "import fit from './chart-layout';\nfit(bounds, () => 2);",
		"aliased-to import": "import { fitAll as fit } from './chart-layout';\nfit(bounds, () => 2);",
		"namespace import":  "import * as fit from './chart-layout';\nfit(bounds, () => 2);",
		"require":           "const fit = require('./chart-layout');\nfit(bounds, () => 2);",
		"require destruct":  "const { fit } = require('./chart-layout');\nfit(bounds, () => 2);",
		"local function":    "function fit(a, b) { return b(); }\nfit(bounds, () => 2);",
		"local const":       "const fit = (a, b) => b();\nfit(bounds, () => 2);",
	}
	for name, src := range notTests {
		t.Run(name, func(t *testing.T) {
			for _, u := range locateSource(t, src) {
				if u.Name == "bounds" || u.Name == "fit" {
					t.Fatalf("phantom unit from bound fit: %q", u.Name)
				}
			}
		})
	}

	global := locateSource(t, "fit(name, () => { expect(1).toBe(1); });")
	if len(global) != 1 || global[0].Name != "name" || len(global[0].Assertions) != 1 {
		t.Fatalf("global dynamic fit must stay discovered, got %v", unitNames(global))
	}

	// A bound fit does not affect other callees, nor a literal-title fit.
	mixed := locateSource(t, "import { fit as focus } from './x';\nit(name, () => { expect(1).toBe(1); });\nfit('lit', () => { expect(1).toBe(1); });")
	if got := unitNames(mixed); len(got) != 2 || got[0] != "name" || got[1] != "lit" {
		t.Fatalf("expected [name lit], got %v", got)
	}
}
