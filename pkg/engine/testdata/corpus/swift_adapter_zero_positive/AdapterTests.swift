import XCTest
import Testing

final class SwiftAdapterZeroTests: XCTestCase {
    func testTrulyEmpty() {
        _ = Calc.add(1, 2)
    }
}

@Test func swiftTestingEmpty() {
    _ = Calc.add(1, 2)
}
