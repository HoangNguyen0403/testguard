import XCTest

final class PerformanceEmptyTests: XCTestCase {
    func testTrulyEmpty() {
        _ = Array(0..<1000).shuffled().sorted()
    }
}
