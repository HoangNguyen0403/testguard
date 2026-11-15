package swift_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/swift"
)

func TestSwiftAdapter_LexicalEdgeFixture(t *testing.T) {
	fixturePath := filepath.Join("testdata", "LexicalEdgeTests.swift")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	units, err := swift.NewAdapter().LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	want := []struct {
		name  string
		scope string
	}{
		{"testRawString", "LexicalEdgeTests"},
		{"testAfterRawString", "LexicalEdgeTests"},
		{"testInterpolationWithNestedString", "LexicalEdgeTests"},
		{"testAfterInterpolation", "LexicalEdgeTests"},
		{"testAfterCommentedAttribute", "LexicalEdgeTests"},
		{"parameterized", "LexicalEdgeSwiftTesting"},
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d: %+v", len(want), len(units), swiftNames(units))
	}
	for i, w := range want {
		if units[i].Name != w.name {
			t.Errorf("unit %d: expected name %q, got %q", i, w.name, units[i].Name)
		}
		if units[i].Scope != w.scope {
			t.Errorf("unit %q: expected scope %q, got %q", w.name, w.scope, units[i].Scope)
		}
		if len(units[i].Assertions) == 0 {
			t.Errorf("unit %q: expected an assertion, got none (body=%q)", w.name, units[i].RawBody)
		}
	}
}

func swiftNames(units []parser.TestUnit) []string {
	out := make([]string, 0, len(units))
	for _, u := range units {
		out = append(out, u.Name)
	}
	return out
}

func TestSwiftAdapter_RawStringWithUnbalancedBraceKeepsScope(t *testing.T) {
	src := `final class IsoBTests: XCTestCase {
    func testBRawString() {
        let raw = #"quote " and brace }"#
        XCTAssertEqual(raw.count, 19)
    }

    func testBAfter() {
        XCTAssertEqual(add(1, 2), 3)
    }
}
`
	units, err := swift.NewAdapter().LocateTests([]byte(src), "IsoTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d: %+v", len(units), swiftNames(units))
	}
	first, second := units[0], units[1]
	if first.EndLine != 5 {
		t.Errorf("expected raw-string test to end at line 5, got %d (body=%q)", first.EndLine, first.RawBody)
	}
	if len(first.Assertions) == 0 {
		t.Errorf("expected raw-string test to keep its XCTAssertEqual, got none (body=%q)", first.RawBody)
	}
	if second.Name != "testBAfter" || second.Scope != "IsoBTests" {
		t.Errorf("expected following test to keep scope IsoBTests, got name=%q scope=%q", second.Name, second.Scope)
	}
}

func TestSwiftAdapter_InterpolationWithNestedStringKeepsScope(t *testing.T) {
	src := `final class IsoCTests: XCTestCase {
    func testCInterpolation() {
        let s = "\(["}"].joined(separator: "{")) ok"
        XCTAssertEqual(s, "} ok")
    }

    func testCAfter() {
        XCTAssertEqual(add(1, 2), 3)
    }
}
`
	units, err := swift.NewAdapter().LocateTests([]byte(src), "IsoTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d: %+v", len(units), swiftNames(units))
	}
	first, second := units[0], units[1]
	if first.EndLine != 5 {
		t.Errorf("expected interpolated test to end at line 5, got %d (body=%q)", first.EndLine, first.RawBody)
	}
	if len(first.Assertions) == 0 {
		t.Errorf("expected interpolated test to keep its XCTAssertEqual, got none (body=%q)", first.RawBody)
	}
	if second.Name != "testCAfter" || second.Scope != "IsoCTests" {
		t.Errorf("expected following test to keep scope IsoCTests, got name=%q scope=%q", second.Name, second.Scope)
	}
}

func TestSwiftAdapter_XCTestMethodWithParametersIsNotATest(t *testing.T) {
	src := `final class HelperTests: XCTestCase {
    func testHelper(named name: String) -> Int { return name.count }

    func testRealTest() {
        XCTAssertEqual(testHelper(named: "a"), 1)
    }
}

struct SwiftTestingSuite {
    @Test func parameterized(n: Int) {
        #expect(n > 0)
    }
}
`
	units, err := swift.NewAdapter().LocateTests([]byte(src), "HelperTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	got := swiftNames(units)
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d: %+v", len(units), got)
	}
	for _, name := range got {
		if name == "testHelper" {
			t.Errorf("func testHelper(named:) takes parameters and is not an XCTest method; got units %+v", got)
		}
	}
	if units[0].Name != "testRealTest" || units[0].Scope != "HelperTests" {
		t.Errorf("unexpected first unit: name=%q scope=%q", units[0].Name, units[0].Scope)
	}
	if units[1].Name != "parameterized" || units[1].Scope != "SwiftTestingSuite" {
		t.Errorf("unexpected second unit: name=%q scope=%q", units[1].Name, units[1].Scope)
	}
}

func TestSwiftAdapter_RejectsStructurallyInvalidInput(t *testing.T) {
	cases := map[string]string{
		"unterminated string": `final class A: XCTestCase {
    func testBroken() {
        let s = "oops
        XCTAssertEqual(1, 1)
    }
}
`,
		"unbalanced brace": `final class A: XCTestCase {
    func testBroken() {
        XCTAssertEqual(1, 1)
}
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			units, err := swift.NewAdapter().LocateTests([]byte(src), "Broken.swift")
			if err == nil {
				t.Fatalf("expected a lexical error, got %d units: %+v", len(units), swiftNames(units))
			}
			if !strings.Contains(err.Error(), "swift:") {
				t.Errorf("expected error to name the language, got %q", err)
			}
		})
	}
}
