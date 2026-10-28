package docs

import (
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

// RuleDoc provides self-contained documentation and fix guidance for a testguard rule.
type RuleDoc struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Severity     models.Severity `json:"severity"`
	Language     string          `json:"language"`
	Description  string          `json:"description"`
	WhyItMatters string          `json:"why_it_matters"`
	Remediation  string          `json:"remediation"`
	GoodExample  string          `json:"good_example"`
	BadExample   string          `json:"bad_example"`
}

var catalog = map[string]RuleDoc{
	// ── 1. Universal Rules ──
	"GUARD-UNIV-ZERO": {
		ID:           "GUARD-UNIV-ZERO",
		Title:        "Zero Assertion Test",
		Severity:     models.SeverityBlocker,
		Language:     "Universal",
		Description:  "The test body contains zero expect(), assert, or outcome verification calls.",
		WhyItMatters: "A test with no assertions provides zero regression protection and gives false confidence in CI.",
		Remediation:  "Add explicit assertions verifying observable return values, emitted states, or side-effects.",
		GoodExample:  "expect(result, equals(expectedValue));",
		BadExample:   "test('computes data', () { service.compute(); }); // No expect!",
	},
	"GUARD-UNIV-01": {
		ID:           "GUARD-UNIV-01",
		Title:        "Exact Test Clone",
		Severity:     models.SeverityAdvisory,
		Language:     "Universal",
		Description:  "Two tests share identical AST token execution sequences.",
		WhyItMatters: "Exact duplicate tests bloat test suite runtime without providing distinct behavioral coverage.",
		Remediation:  "Eliminate the redundant test case or consolidate varied inputs into a table-driven parameterized test.",
		GoodExample:  "for (final testCase in testCases) { ... }",
		BadExample:   "test('case A', () { ... });\ntest('case B', () { /* identical body */ });",
	},
	"GUARD-UNIV-02": {
		ID:           "GUARD-UNIV-02",
		Title:        "Structural Repetition Smell",
		Severity:     models.SeverityAdvisory,
		Language:     "Universal",
		Description:  "Four or more tests share identical AST execution structure with only literal values swapped (exact copy-paste groups now reported as GUARD-UNIV-01).",
		WhyItMatters: "High structural duplication increases maintenance overhead when test setup changes.",
		Remediation:  "Consolidate repetitive test structures into a parameterized table-driven test loop.",
		GoodExample:  "var tests = []struct{ input, want string }{ ... }; for _, tt := range tests { ... }",
		BadExample:   "4+ tests copy-pasting the exact same 15-line arrange-act-assert flow with 1 number changed.",
	},
	"GUARD-UNIV-TAUTOLOGY": {
		ID:           "GUARD-UNIV-TAUTOLOGY",
		Title:        "Tautological Assertion",
		Severity:     models.SeverityBlocker,
		Language:     "Universal",
		Description:  "The test asserts identity between identical literals or matches a boolean literal directly to itself (e.g. expect(true).toBe(true), assert.True(t, true)).",
		WhyItMatters: "Tautological assertions never fail on product regressions and inflate perceived test quality.",
		Remediation:  "Assert actual system-under-test variables and return values against expected invariants.",
		GoodExample:  "expect(calculator.add(1, 2)).toBe(3);",
		BadExample:   "expect(true).toBe(true); // Always passes regardless of implementation",
	},
	"GUARD-UNIV-SKIP": {
		ID:           "GUARD-UNIV-SKIP",
		Title:        "Committed Skipped Test",
		Severity:     models.SeverityWarning,
		Language:     "Universal",
		Description:  "The test file contains permanently skipped or disabled test declarations (e.g. it.skip, test.skip, skip: true, or unconditional t.Skip).",
		WhyItMatters: "Silently skipped tests rot over time and conceal failing features from CI pipelines.",
		Remediation:  "Fix and unskip the test, or delete obsolete test cases with accompanying rationale.",
		GoodExample:  "it('computes total', () => { ... });",
		BadExample:   "it.skip('flaky checkout test', () => { ... });",
	},
	"GUARD-UNIV-NAME": {
		ID:           "GUARD-UNIV-NAME",
		Title:        "Vague / Placeholder Test Name",
		Severity:     models.SeverityAdvisory,
		Language:     "Universal",
		Description:  "Test name consists solely of hollow placeholder words (e.g., 'test 1', 'sample', 'dummy', 'foo').",
		WhyItMatters: "Vague test names obscure failure investigation in CI and defeat living documentation.",
		Remediation:  "Use descriptive names following Method_Scenario_ExpectedBehavior or descriptive behavior phrases.",
		GoodExample:  "FetchCart_WhenTokenExpired_EmitsUnauthorizedFailureState",
		BadExample:   "test('test 1', () { ... })",
	},

	// ── 2. Dart / Flutter Mobile Rules ──
	"GUARD-DART-SEED-ECHO": {
		ID:           "GUARD-DART-SEED-ECHO",
		Title:        "Seed Echo Assertion in BLoC Test",
		Severity:     models.SeverityWarning,
		Language:     "Dart",
		Description:  "Expected state properties merely echo values already established in the seed baseline.",
		WhyItMatters: "Asserting unchanged fields over-specifies tests and breaks resistance to refactoring.",
		Remediation:  "Use base state fixtures and assert only the mutated fields (delta) in state transitions.",
		GoodExample:  "baseState.copyWith(isFetching: true)",
		BadExample:   "isA<State>().having((s) => s.userInfo, 'userInfo', fakeUser) // when userInfo is invariant",
	},
	"GUARD-DART-BLOC-DUPLICATE": {
		ID:           "GUARD-DART-BLOC-DUPLICATE",
		Title:        "Duplicate BLoC Test Scenario",
		Severity:     models.SeverityAdvisory,
		Language:     "Dart",
		Description:  "Two blocTest cases in the same file execute the exact same whole scenario (build/setUp/seed/act/expect/skip/verify minus test name).",
		WhyItMatters: "Duplicate BLoC scenarios duplicate test runtime without increasing code coverage or fault proof.",
		Remediation:  "Remove redundant duplicate blocTest case or test a distinct failure or edge-case branch.",
		GoodExample:  "blocTest('Success case', ...); blocTest('Timeout failure case', ...);",
		BadExample:   "blocTest('Search success', act: fetch, expect: data); blocTest('Auto-search success', act: fetch, expect: data);",
	},
	"GUARD-DART-MOCK-ECHO": {
		ID:           "GUARD-DART-MOCK-ECHO",
		Title:        "Pass-Through Mock Echo in Dart",
		Severity:     models.SeverityWarning,
		Language:     "Dart",
		Description:  "A repository or service test mocks a datasource returning X and asserts only Right(X) with zero transformation.",
		WhyItMatters: "Mock echoing tests only that the mocking library works, providing zero business verification.",
		Remediation:  "Assert domain entity mapping, calculation outcomes, or error translation.",
		GoodExample:  "when(() => api.fetch()).thenThrow(DioException.badResponse(...)); expect(repo.fetch(), Left(ApiFailure.notFound()));",
		BadExample:   "when(() => api.fetch()).thenAnswer((_) => Right(mockUser)); expect(repo.fetch(), Right(mockUser));",
	},

	// ── 3. Go Backend Rules ──
	"GUARD-GO-01": {
		ID:           "GUARD-GO-01",
		Title:        "Domain Purity (Zero Mocks)",
		Severity:     models.SeverityBlocker,
		Language:     "Go",
		Description:  "A domain entity or model test imports mock controllers or database drivers.",
		WhyItMatters: "Clean Architecture requires domain models to be pure business logic tested with real instances.",
		Remediation:  "Remove mocks from domain tests. Test business logic using pure domain struct fixtures.",
		GoodExample:  "item := domain.NewItem(100); assert.True(t, item.IsValid())",
		BadExample:   "mockRepo := mocks.NewMockRepo(ctrl); // mocks in domain test",
	},
	"GUARD-GO-03": {
		ID:           "GUARD-GO-03",
		Title:        "Table-Test Duplicate Input Rows",
		Severity:     models.SeverityAdvisory,
		Language:     "Go",
		Description:  "A Go table-driven test contains multiple test rows with identical input values and expected results.",
		WhyItMatters: "Duplicate table test cases create noise and false test counts without testing new equivalence classes.",
		Remediation:  "Ensure each table test case tests a distinct equivalence class, boundary condition, or error scenario.",
		GoodExample:  "{name: 'valid price', in: 10, want: 10}, {name: 'negative price', in: -1, wantErr: true}",
		BadExample:   "{name: 'case 1', in: 10, want: 10}, {name: 'case 2', in: 10, want: 10}",
	},

	// ── 4. NestJS Backend Rules ──
	"GUARD-NEST-01": {
		ID:           "GUARD-NEST-01",
		Title:        "Pass-Through Controller Echo",
		Severity:     models.SeverityWarning,
		Language:     "NestJS",
		Description:  "A NestJS controller unit test mocks service returning X and asserts only controller returns X.",
		WhyItMatters: "Controllers in NestJS are thin routing wrappers. Testing 1:1 pass-through mocks provides zero validation.",
		Remediation:  "Test validation pipes, guards, interceptors, or integration behavior via supertest.",
		GoodExample:  "request(app.getHttpServer()).post('/items').send({ invalid: true }).expect(400)",
		BadExample:   "jest.spyOn(service, 'find').mockResolvedValue(result); expect(await controller.find()).toBe(result);",
	},
	"GUARD-NEST-02": {
		ID:           "GUARD-NEST-02",
		Title:        "Unmocked Database Module Leak",
		Severity:     models.SeverityBlocker,
		Language:     "NestJS",
		Description:  "A NestJS unit test imports real database modules (TypeOrmModule, MongooseModule, HttpModule) instead of test doubles.",
		WhyItMatters: "Connecting to real external databases during unit tests causes slow, non-deterministic CI execution.",
		Remediation:  "Use getRepositoryToken() or provide mock providers in Test.createTestingModule.",
		GoodExample:  "{ provide: getRepositoryToken(User), useValue: mockUserRepository }",
		BadExample:   "imports: [TypeOrmModule.forRoot(...)] // inside .spec.ts unit test",
	},

	// ── 5. React & Next.js Frontend Rules ──
	"GUARD-REACT-01": {
		ID:           "GUARD-REACT-01",
		Title:        "Shallow Mock Text Echo",
		Severity:     models.SeverityWarning,
		Language:     "React",
		Description:  "Test mocks an API query with literal string X and asserts only screen.getByText(X) without user interaction or state logic.",
		WhyItMatters: "Verifies only that React rendered mock text, not that user interactions or component state work.",
		Remediation:  "Simulate user interaction via userEvent (click, type, submit) and assert resulting behavior.",
		GoodExample:  "await userEvent.click(button); expect(mockSubmit).toHaveBeenCalledWith(...);",
		BadExample:   "render(<UserProfile />); expect(screen.getByText('mock-user')).toBeInTheDocument(); // no interaction",
	},
	"GUARD-REACT-02": {
		ID:           "GUARD-REACT-02",
		Title:        "Arbitrary Sleep / setTimeout Fragility",
		Severity:     models.SeverityWarning,
		Language:     "React",
		Description:  "Real-timer sleeps > 0 ms in unit test bodies; ignores 0 ms flushes, mock latency, fake timers; unit tests only.",
		WhyItMatters: "Hardcoded timer sleeps cause slow test suites and flaky CI failures due to runner CPU variances.",
		Remediation:  "Use React Testing Library's waitFor(...) or async findBy* queries instead of fixed sleeps.",
		GoodExample:  "await waitFor(() => expect(screen.getByText('Done')).toBeInTheDocument());",
		BadExample:   "await new Promise((r) => setTimeout(r, 1500)); expect(...)",
	},
	"GUARD-E2E-01": {
		ID:           "GUARD-E2E-01",
		Title:        "Hard Wait in E2E Test",
		Severity:     models.SeverityWarning,
		Language:     "E2E",
		Description:  "End-to-end / UI integration test uses fixed arbitrary time delays (e.g. page.waitForTimeout, Future.delayed) instead of event-driven wait conditions.",
		WhyItMatters: "Hard waits in E2E tests slow down test runs and lead to flaky CI runs under varying machine loads.",
		Remediation:  "Wait for expected DOM selectors, network responses, or state conditions rather than fixed milliseconds.",
		GoodExample:  "await expect(page.getByText('Success')).toBeVisible();",
		BadExample:   "await page.waitForTimeout(3000);",
	},
	"GUARD-TS-01": {
		ID:           "GUARD-TS-01",
		Title:        "Massive Snapshot-Only Test Smell",
		Severity:     models.SeverityAdvisory,
		Language:     "TypeScript",
		Description:  "Test relies solely on large JSON/DOM snapshots (>50 lines) without explicit behavioral assertions.",
		WhyItMatters: "Large snapshots create review fatigue where developers reflexively approve diffs with -u without reviewing.",
		Remediation:  "Assert specific semantic elements and contracts rather than snapshotting entire DOM trees.",
		GoodExample:  "expect(screen.getByRole('button', { name: /checkout/i })).toBeEnabled();",
		BadExample:   "expect(container).toMatchSnapshot(); // 200 lines DOM snapshot",
	},

	// ── 6. Native Mobile Rules (Android & iOS) ──
	"GUARD-KT-01": {
		ID:           "GUARD-KT-01",
		Title:        "MockK Pass-Through Echo",
		Severity:     models.SeverityWarning,
		Language:     "Kotlin",
		Description:  "ViewModel unit test mocks repository returning X and asserts StateFlow emits X with zero transformation.",
		WhyItMatters: "Provides zero fault proof of ViewModel state reduction or business validation.",
		Remediation:  "Assert transformation logic, error state emission, or intent side-effects.",
		GoodExample:  "viewModel.handle(Intent.Fetch); assertEquals(ViewState.Success(transformed), viewModel.state.value)",
		BadExample:   "every { repo.getUser() } returns user; assertEquals(user, viewModel.user.value) // no logic",
	},
	"GUARD-KT-02": {
		ID:           "GUARD-KT-02",
		Title:        "Unadvanced Coroutine Virtual Time",
		Severity:     models.SeverityBlocker,
		Language:     "Kotlin",
		Description:  "Calling delayed suspend functions inside runTest without advanceUntilIdle() or advanceTimeBy().",
		WhyItMatters: "Assertions execute prematurely before asynchronous coroutine state mutations have completed.",
		Remediation:  "Call advanceUntilIdle() or testDispatcher.scheduler.advanceUntilIdle() before asserting state.",
		GoodExample:  "viewModel.search('query'); advanceUntilIdle(); assertEquals(expectedList, viewModel.state.value)",
		BadExample:   "viewModel.search('query'); assertEquals(expectedList, viewModel.state.value) // fails without advance",
	},
	"GUARD-SWIFT-01": {
		ID:           "GUARD-SWIFT-01",
		Title:        "Excessive Expectation Timeout",
		Severity:     models.SeverityWarning,
		Language:     "Swift",
		Description:  "XCTest expectation uses excessive timeout (>2.0s) in local unit test.",
		WhyItMatters: "High expectation timeouts cause test suites to freeze for long intervals on genuine regression failures.",
		Remediation:  "Use immediate async/await or fulfill expectations within a tight sub-second timeout (<= 1.0s).",
		GoodExample:  "await fulfillment(of: [expectation], timeout: 0.5)",
		BadExample:   "wait(for: [expectation], timeout: 10.0) // 10s timeout in unit test",
	},
}

// Explain returns the RuleDoc for a given rule ID.
func Explain(ruleID string) (RuleDoc, bool) {
	doc, found := catalog[strings.ToUpper(ruleID)]
	return doc, found
}

// AllRules returns all registered rule documentation items.
func AllRules() []RuleDoc {
	rules := make([]RuleDoc, 0, len(catalog))
	for _, doc := range catalog {
		rules = append(rules, doc)
	}
	return rules
}
