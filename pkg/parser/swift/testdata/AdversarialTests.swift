import Nimble
import Testing
import XCTest
@testable import Demo

final class AdversarialTests: XCTestCase {
    // A multi-line string that looks like a test must not become a unit.
    let doc = """
    func testFakeInString() {
        XCTAssertTrue(true)
    }
    """

    private func makeUser() -> User { User(name: "alice") }

    private func assertValid(_ user: User, file: StaticString = #file, line: UInt = #line) {
        XCTAssertTrue(user.isValid, file: file, line: line)
    }

    // ---- correct tests: an assertion each, no rule finding ----

    func testAsyncThrows() async throws {
        let session = try await AuthService().login(user: "a", password: "b")
        XCTAssertEqual(session.userID, 1)
    }

    @MainActor
    func testMainActorAttribute() {
        XCTAssertFalse(ViewModel().isLoading)
    }

    func testUsesSameFileHelper() {
        assertValid(makeUser())
    }

    func testNimble() {
        expect(Calc.add(1, 2)).to(equal(3))
    }

    func testShortTimeout() {
        let done = expectation(description: "loaded")
        Loader().load { _ in done.fulfill() }
        wait(for: [done], timeout: 1.0)
    }

    func testPerformanceOfSort() {
        measure {
            _ = Sorter().sort(Array(0..<1000).shuffled())
        }
    }

    // ---- hollow tests: each must be reported ----

    func testOnlyPrints() {
        print(Calc.add(1, 2))
    }
}

@Suite struct MathSuite {
    @Test func adds() {
        #expect(Calc.add(1, 2) == 3)
    }

    @Test(arguments: [1, 2, 3])
    func positive(n: Int) {
        #expect(n > 0)
    }

    @Test(.disabled("flaky on CI"))
    func disabledTest() {
        #expect(Calc.add(1, 2) == 4)
    }

    @Test func emptySwiftTesting() {
        _ = Calc.add(1, 2)
    }
}
