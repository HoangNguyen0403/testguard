import XCTest

final class SwiftTimeoutPositiveTests: XCTestCase {
    func testAsyncExpectation() {
        let done = expectation(description: "loaded")
        Loader().load { _ in done.fulfill() }
        wait(for: [done], timeout: 10.0)
    }
}
