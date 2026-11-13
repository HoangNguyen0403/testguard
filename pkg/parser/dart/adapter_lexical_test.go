package dart_test

import (
	"os"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser/dart"
)

// testUnitShape is the subset of parser.TestUnit fields the lexical
// regressions pin: name, kind, start line and discovered assertions.
type testUnitShape struct {
	name       string
	kind       string
	startLine  int
	assertions []string
}

func locateFixture(t *testing.T, path string) []testUnitShape {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	units, err := dart.NewAdapter().LocateTests(source, path)
	if err != nil {
		t.Fatalf("locate tests in %s: %v", path, err)
	}
	shapes := make([]testUnitShape, 0, len(units))
	for _, u := range units {
		shapes = append(shapes, testUnitShape{
			name:       u.Name,
			kind:       u.Kind,
			startLine:  u.StartLine,
			assertions: u.Assertions,
		})
	}
	return shapes
}

func assertUnitsMatch(t *testing.T, got []testUnitShape, want []testUnitShape) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d test units, got %d: %+v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i].name != w.name {
			t.Errorf("unit[%d]: expected name %q, got %q", i, w.name, got[i].name)
		}
		if got[i].kind != w.kind {
			t.Errorf("unit %q: expected kind %q, got %q", w.name, w.kind, got[i].kind)
		}
		if got[i].startLine != w.startLine {
			t.Errorf("unit %q: expected start line %d, got %d", w.name, w.startLine, got[i].startLine)
		}
		if len(got[i].assertions) != len(w.assertions) {
			t.Errorf("unit %q: expected assertions %v, got %v", w.name, w.assertions, got[i].assertions)
			continue
		}
		for j, a := range w.assertions {
			if got[i].assertions[j] != a {
				t.Errorf("unit %q: expected assertion[%d] = %q, got %q", w.name, j, a, got[i].assertions[j])
			}
		}
	}
}

func TestLocateTests_SkipsCommentedAndDocCommentedDeclarations(t *testing.T) {
	got := locateFixture(t, "testdata/commented_tests_test.dart")
	assertUnitsMatch(t, got, []testUnitShape{
		{name: "A4 real test", kind: "unit", startLine: 4, assertions: []string{"expect"}},
		{name: "L17 json fixture", kind: "unit", startLine: 16, assertions: []string{"expect"}},
		{name: "L22 after", kind: "unit", startLine: 21, assertions: []string{"expect"}},
	})
}

func TestLocateTests_InterpolatedAndRawStringLexicalAwareness(t *testing.T) {
	got := locateFixture(t, "testdata/interpolation_test.dart")
	assertUnitsMatch(t, got, []testUnitShape{
		{name: "B4 raw string with fake test", kind: "unit", startLine: 4, assertions: []string{"expect"}},
		{name: "B9 triple quoted with fake test and skip", kind: "unit", startLine: 9, assertions: []string{"expect"}},
		{name: "B17 interpolation with brace key", kind: "unit", startLine: 17, assertions: []string{"expect"}},
		{name: "B23 after interpolation", kind: "unit", startLine: 23, assertions: []string{"expect"}},
		{name: "B27 plain string with brace", kind: "unit", startLine: 27, assertions: []string{"expect"}},
		{name: "B32 escaped dollar", kind: "unit", startLine: 32, assertions: []string{"expect"}},
		{name: "B37 nested interpolation string", kind: "unit", startLine: 37, assertions: []string{"expect"}},
		{name: "B42 interpolation with closure braces", kind: "unit", startLine: 42, assertions: []string{"expect"}},
	})
}

func TestLocateTests_KeepsRawLiteralValuesAndOffsets(t *testing.T) {
	source, err := os.ReadFile("testdata/interpolation_test.dart")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	units, err := dart.NewAdapter().LocateTests(source, "testdata/interpolation_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantBodies := map[string]string{
		"B17 interpolation with brace key":         `final s = '${m['}']} end';`,
		"B9 triple quoted with fake test and skip": "      test('B11 fake', () {});",
	}
	for _, u := range units {
		want, ok := wantBodies[u.Name]
		if !ok {
			continue
		}
		if !strings.Contains(u.RawBody, want) {
			t.Errorf("unit %q: expected RawBody to contain %q verbatim, got:\n%s", u.Name, want, u.RawBody)
		}
	}
}

func TestLocateTests_CommentedUnbalancedDeclarationDoesNotFail(t *testing.T) {
	source := []byte(`void main() {
  // test('unbalanced comment', () {
  test('real', () {
    expect(1, 1);
  });
}
`)
	units, err := dart.NewAdapter().LocateTests(source, "unbalanced_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 1 || units[0].Name != "real" {
		t.Fatalf("expected only the real test, got %+v", units)
	}
}
