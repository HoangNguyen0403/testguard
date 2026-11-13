import XCTest

final class PerformanceMeasureTests: XCTestCase {
    func testPerformanceOfSort() {
        measure {
            _ = Array(0..<1000).shuffled().sorted()
        }
    }
}
