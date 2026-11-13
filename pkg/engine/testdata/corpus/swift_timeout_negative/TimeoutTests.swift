import XCTest

final class SwiftTimeoutNegativeTests: XCTestCase {
    func testAsyncExpectation() {
        let done = expectation(description: "loaded")
        Loader().load { _ in done.fulfill() }
        wait(for: [done], timeout: 1.0)
    }
}
