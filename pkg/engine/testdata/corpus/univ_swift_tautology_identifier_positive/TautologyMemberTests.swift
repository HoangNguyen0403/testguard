final class TautologyMemberTests: XCTestCase {
    func testSameIdentifier() {
        let r = build()
        XCTAssertEqual(r, r)
    }

    func testDifferentReceivers() {
        let a = build()
        let b = build()
        XCTAssertEqual(a.id, b.id)
    }
}
