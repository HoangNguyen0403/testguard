import XCTest

private func add(_ a: Int, _ b: Int) -> Int { a + b }

final class VariantsTests: XCTestCase {
    func testAddsOneAndOne() {
        let input = 1
        let expected = 2
        let result = add(input, input)
        XCTAssertEqual(result, expected)
    }

    func testAddsTwoAndTwo() {
        let input = 2
        let expected = 4
        let result = add(input, input)
        XCTAssertEqual(result, expected)
    }

    func testAddsThreeAndThree() {
        let input = 3
        let expected = 6
        let result = add(input, input)
        XCTAssertEqual(result, expected)
    }

    func testAddsFourAndFour() {
        let input = 4
        let expected = 8
        let result = add(input, input)
        XCTAssertEqual(result, expected)
    }
}
