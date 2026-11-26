package swift_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser/swift"
)

func TestSwiftAdapter_DisabledAndHelper(t *testing.T) {
	src := `
import Testing
import XCTest

@Suite("Account Tests")
struct AccountTests {
    private func assertAccountValid() {
        XCTAssertTrue(true)
    }

    @Test(.disabled("WIP"))
    func disabledFeature() {
        #expect(1 == 1)
    }

    @Test
    func withHelper() {
        assertAccountValid()
    }

    func testSkipFirst() {
        throw XCTSkip("skip me")
        XCTAssertTrue(true)
    }
}
`
	adapter := swift.NewAdapter()
	units, err := adapter.LocateTests([]byte(src), "AccountTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d (%+v)", len(units), units)
	}

	if units[0].Name != "withHelper" {
		t.Errorf("expected unit 'withHelper', got %s", units[0].Name)
	}
	if units[0].Scope != "AccountTests" {
		t.Errorf("expected scope 'AccountTests', got %s", units[0].Scope)
	}
	if len(units[0].Assertions) == 0 {
		t.Errorf("expected assertions via helper, got 0")
	}
}

// swiftlang/swift test/CAS/swift-scan-test.swift:58 declares a bare
// top-level `func testFunc() {}`. XCTest and Swift Testing only discover
// test-prefixed methods on test types, so that file-scope function is not a
// test unit. The paired controls keep every supported shape eligible.
func TestSwiftAdapter_TopLevelTestPrefixedFunctionIsNotDiscovered(t *testing.T) {
	src := `import XCTest
import Testing

func testFunc() {}

func testFuncWithBraces() {
    if true { print("}") }
    XCTAssertTrue(true)
}

@Test
func topLevelSwiftTesting() {
    #expect(1 == 1)
}

@Testing.Test
func qualifiedTopLevel() {
    #expect(2 == 2)
}

@Custom.Test
func customQualified() {}

@Test
func testPrefixedTopLevelEmpty() {}

final class GenuineTests: XCTestCase {
    func testReal() {
        XCTAssertTrue(true)
    }
}

struct SwiftTestingSuite {
    @Test func suiteCase() {
        #expect(3 == 3)
    }
}
`
	units, err := swift.NewAdapter().LocateTests([]byte(src), "swift-scan-test.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	got := map[string]string{}
	for _, u := range units {
		if _, dup := got[u.Name]; dup {
			t.Errorf("unit %q discovered twice (%+v)", u.Name, swiftNames(units))
		}
		got[u.Name] = u.Scope
	}

	for _, name := range []string{"testFunc", "testFuncWithBraces", "customQualified"} {
		if _, ok := got[name]; ok {
			t.Errorf("declaration %q is not an XCTest/Swift Testing test and must be excluded; got %+v", name, swiftNames(units))
		}
	}
	want := map[string]string{
		"topLevelSwiftTesting":      "",
		"qualifiedTopLevel":         "",
		"testPrefixedTopLevelEmpty": "",
		"testReal":                  "GenuineTests",
		"suiteCase":                 "SwiftTestingSuite",
	}
	for name, scope := range want {
		gotScope, ok := got[name]
		if !ok {
			t.Errorf("expected %q to stay eligible, got %+v", name, swiftNames(units))
			continue
		}
		if gotScope != scope {
			t.Errorf("%q: expected scope %q, got %q", name, scope, gotScope)
		}
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d (%+v)", len(want), len(units), swiftNames(units))
	}
}

// A closure in a member initialiser opens and closes a brace without leaving
// the enclosing type: the scope must follow depth, not brace count, or every
// following test is silently dropped.
func TestSwiftAdapter_TypeScopeSurvivesClosureInitialiser(t *testing.T) {
	src := `import XCTest
import Testing

protocol Resettable {
    func reset()
}

final class GenuineTests: XCTestCase {
    let fixture = [1].map { $0 }
    let lookup = { (x: Int) -> Int in
        if x > 0 { return x }
        return 0
    }

    func testReal() {
        XCTAssertTrue(true)
    }
}

struct NestedScopeTests {
    static let sorted = [3, 1].sorted { $0 < $1 }

    @Test func nestedCase() {
        #expect(NestedScopeTests.sorted == [1, 3])
    }

    final class Inner: XCTestCase {
        func testInner() {
            XCTAssertTrue(true)
        }
    }
}

final class OneLineTests: XCTestCase { let fixture = [1].map { $0 }; func testOneLine() { XCTAssertTrue(true) } }
`
	units, err := swift.NewAdapter().LocateTests([]byte(src), "ScopeTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	got := map[string]string{}
	for _, u := range units {
		got[u.Name] = u.Scope
	}
	want := map[string]string{
		"testReal":    "GenuineTests",
		"nestedCase":  "NestedScopeTests",
		"testInner":   "NestedScopeTests > Inner",
		"testOneLine": "OneLineTests",
	}
	for name, scope := range want {
		gotScope, ok := got[name]
		if !ok {
			t.Errorf("expected %q after a closure initialiser, got %+v", name, swiftNames(units))
			continue
		}
		if gotScope != scope {
			t.Errorf("%q: expected scope %q, got %q", name, scope, gotScope)
		}
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d (%+v)", len(want), len(units), swiftNames(units))
	}
	for _, u := range units {
		if len(u.Assertions) == 0 {
			t.Errorf("expected assertions for %s, got none (body=%q)", u.Name, u.RawBody)
		}
	}
}

// Only `@Test` and Swift Testing's module-qualified `@Testing.Test` mark a
// declaration as a test; another module's `@Custom.Test` does not.
func TestSwiftAdapter_QualifiedTestAttributeRecognition(t *testing.T) {
	src := `import Testing

@Test
func plainTest() {
    #expect(1 == 1)
}

@Testing.Test
func testQualifiedPrefixed() {}

@Custom.Test
func customQualified() {}

@Custom.Test
func testCustomPrefixed() {}
`
	units, err := swift.NewAdapter().LocateTests([]byte(src), "QualifiedTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	got := map[string]bool{}
	for _, u := range units {
		got[u.Name] = true
	}
	for _, name := range []string{"plainTest", "testQualifiedPrefixed"} {
		if !got[name] {
			t.Errorf("expected %q to be discovered through a Swift Testing attribute, got %+v", name, swiftNames(units))
		}
	}
	for _, name := range []string{"customQualified", "testCustomPrefixed"} {
		if got[name] {
			t.Errorf("@Custom.Test does not mark a test, so %q must be excluded; got %+v", name, swiftNames(units))
		}
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d (%+v)", len(units), swiftNames(units))
	}
}

// A file-scope function that asserts is not a test, so it must resolve as an
// asserting helper instead of losing its credit or becoming a unit.
func TestSwiftAdapter_TopLevelAssertingHelperIsCredited(t *testing.T) {
	src := `import Testing
import XCTest

func testShared() {
    XCTAssertTrue(true)
}

@Test
func realUsesShared() {
    testShared()
}
`
	units, err := swift.NewAdapter().LocateTests([]byte(src), "SharedHelperTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 1 || units[0].Name != "realUsesShared" {
		t.Fatalf("expected only realUsesShared as a unit, got %+v", swiftNames(units))
	}
	if len(units[0].Assertions) == 0 {
		t.Errorf("expected the file-scope asserting helper to be credited, got no assertions (body=%q)", units[0].RawBody)
	}

	// A test-prefixed method on a type is a test, so it is never a helper.
	memberSrc := `import XCTest

final class MemberTests: XCTestCase {
    func testZeroArgMember() {
        XCTAssertTrue(true)
    }

    func testUsesMemberTest() {
        testZeroArgMember()
    }
}
`
	memberUnits, err := swift.NewAdapter().LocateTests([]byte(memberSrc), "MemberTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(memberUnits) != 2 {
		t.Fatalf("expected 2 member units, got %+v", swiftNames(memberUnits))
	}
	for _, u := range memberUnits {
		if u.Name == "testUsesMemberTest" && len(u.Assertions) != 0 {
			t.Errorf("a member test must not be credited as an asserting helper, got %v", u.Assertions)
		}
	}
}

func TestSwiftAdapter_ClosureFormSwiftTestingMacros(t *testing.T) {
	src := `
import Testing

struct ClientProcessTests {
    @Test func closureFormExpect() async throws {
        #expect {
            try await process.run()
        } throws: { error in
            error is ProcessError
        }
    }

    @Test func closureFormRequire() {
        #require {
            try container.start()
        } throws: { _ in true }
    }
}
`
	adapter := swift.NewAdapter()
	units, err := adapter.LocateTests([]byte(src), "ClientProcessTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d (%+v)", len(units), units)
	}
	for _, u := range units {
		if len(u.Assertions) == 0 {
			t.Errorf("expected closure-form macro credited for %s, got 0 assertions", u.Name)
		}
	}
}

func TestSwiftAdapter_MacroTextOutsideCodeNotCredited(t *testing.T) {
	src := `
import Testing

struct LiteralTests {
    @Test func macroTextOnly() {
        let bareMacro = #expect
        let message = "#expect { value == 1 }"
        let other = """
        #require { try container.start() }
        """
        // #expect { value == 1 }
        /* #require { try container.start() } */
        print(message, other)
    }
}
`
	adapter := swift.NewAdapter()
	units, err := adapter.LocateTests([]byte(src), "LiteralTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d (%+v)", len(units), units)
	}
	if len(units[0].Assertions) != 0 {
		t.Errorf("expected no assertions from macro text in strings/comments, got %v", units[0].Assertions)
	}
}

func TestSwiftAdapter_NimbleAndSnapshot(t *testing.T) {
	src := `
import XCTest

class SnapshotTests: XCTestCase {
    func testNimble() {
        expect(1 + 1).to(equal(2))
    }

    func testSnapshot() {
        assertSnapshot(matching: view, as: .image)
    }
}
`
	adapter := swift.NewAdapter()
	units, err := adapter.LocateTests([]byte(src), "SnapshotTests.swift")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
	if len(units[0].Assertions) == 0 {
		t.Errorf("expected testNimble to have assertions")
	}
	if len(units[1].Assertions) == 0 {
		t.Errorf("expected testSnapshot to have assertions")
	}
}
