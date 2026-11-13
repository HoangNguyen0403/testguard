import XCTest
import Testing

final class SwiftAdapterPositiveTests: XCTestCase {
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
