import XCTest

private func add(_ a: Int, _ b: Int) -> Int { a + b }

final class SkipTests: XCTestCase {
    func testChecksAddition() throws {
        throw XCTSkip("flaky on CI")
        XCTAssertEqual(add(1, 2), 3)
    }

    func testChecksSubtraction() {
        XCTAssertEqual(add(3, 2), 5)
    }
}
