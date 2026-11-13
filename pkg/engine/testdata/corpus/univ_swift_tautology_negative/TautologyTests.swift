import XCTest

class TautologyTests: XCTestCase {
    func testValid() {
        XCTAssertTrue(service.isReady)
        XCTAssertEqual(42, service.count)
    }
}
