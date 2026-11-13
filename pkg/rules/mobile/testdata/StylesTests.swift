import XCTest
import Testing
@testable import Demo

final class StylesTests: XCTestCase {
    func testXCTAssert() {
        XCTAssertEqual(Calc.add(1, 2), 3)
    }

    func testUnwrapOnly() throws {
        let user = try XCTUnwrap(Store.shared.user(id: 1))
        _ = user
    }

    func testFailBranch() {
        guard case .success = Calc.parse("1") else {
            return XCTFail("expected success")
        }
    }

    func testAsyncExpectation() {
        let done = expectation(description: "loaded")
        Loader().load { _ in done.fulfill() }
        wait(for: [done], timeout: 10.0)
    }

    func testTrulyEmpty() {
        _ = Calc.add(1, 2)
    }

    func testTautology() {
        XCTAssertTrue(true)
    }

    func testSkipped() throws {
        throw XCTSkip("flaky on CI")
    }
}

@Test func swiftTestingExpect() {
    #expect(Calc.add(1, 2) == 3)
}

@Test("requires a value") func swiftTestingRequire() throws {
    let value = try #require(Calc.parseInt("7"))
    _ = value
}

@Test func swiftTestingEmpty() {
    _ = Calc.add(1, 2)
}
