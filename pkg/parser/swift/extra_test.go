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
