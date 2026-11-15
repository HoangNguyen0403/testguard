import XCTest

final class LexicalStructureTests: XCTestCase {
    func testRawStringWithBrace() {
        let s = #"quote " and brace }"#
        XCTAssertEqual(s.count, 20)
    }

    func testInterpolationWithNestedStringBrace() {
        let s = "value \(["a": "}"]["a"]!)"
        XCTAssertEqual(s, "value }")
    }

    func testHelperWithParametersIsNotATest(named: String) -> Int {
        return named.count
    }

    func testAfterLexicalShapes() {
        XCTAssertEqual(sum(1, 2), 3)
    }
}
