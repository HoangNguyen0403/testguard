import XCTest

final class LexicalTests: XCTestCase {
    func testOk() {
        let s = "unterminated
        XCTAssertEqual(s.count, 3)
    }
}
