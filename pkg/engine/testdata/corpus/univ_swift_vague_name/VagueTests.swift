import XCTest

private func add(_ a: Int, _ b: Int) -> Int { a + b }

final class VagueTests: XCTestCase {
    func test1() {
        XCTAssertEqual(add(1, 2), 3)
    }
}
