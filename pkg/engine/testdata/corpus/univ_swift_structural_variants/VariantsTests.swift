import XCTest

private func add(_ a: Int, _ b: Int) -> Int { a + b }

final class VariantsTests: XCTestCase {
    func testAddsOneAndOne() {
        XCTAssertEqual(add(1, 1), 2)
    }

    func testAddsTwoAndTwo() {
        XCTAssertEqual(add(2, 2), 4)
    }

    func testAddsThreeAndThree() {
        XCTAssertEqual(add(3, 3), 6)
    }

    func testAddsFourAndFour() {
        XCTAssertEqual(add(4, 4), 8)
    }
}
