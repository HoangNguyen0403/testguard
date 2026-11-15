import XCTest

private func add(_ a: Int, _ b: Int) -> Int { a + b }

final class DuplicatesTests: XCTestCase {
    func testAddsTwoNumbers() {
        XCTAssertEqual(add(1, 2), 3)
    }

    func testAddsTwoNumbersAgain() {
        XCTAssertEqual(add(1, 2), 3)
    }
}
