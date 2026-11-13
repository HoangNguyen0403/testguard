import XCTest

// Regression fixture for the Swift lexer: raw strings with unbalanced quotes
// and braces, interpolation with a nested string brace, and a parameterised
// func that only looks like an XCTest method. Every real test must stay
// discovered inside its class scope and keep its assertion.
final class LexicalEdgeTests: XCTestCase {
    func testRawString() {
        let raw = #"quote " and brace }"#
        XCTAssertEqual(raw.count, 19)
    }

    func testAfterRawString() {
        XCTAssertEqual(add(1, 2), 3)
    }

    func testInterpolationWithNestedString() {
        let s = "\(["}"].joined(separator: "{")) ok"
        XCTAssertEqual(s, "} ok")
    }

    func testAfterInterpolation() {
        XCTAssertEqual(add(2, 3), 5)
    }

    func testHelper(named name: String) -> Int { return name.count }

    /* /* nested */ @Test(.disabled) throw XCTSkip("x") */
    func testAfterCommentedAttribute() {
        XCTAssertEqual(testHelper(named: "a"), 1)
    }
}

@Suite struct LexicalEdgeSwiftTesting {
    @Test func parameterized(n: Int) {
        #expect(n > 0)
    }
}
