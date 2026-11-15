package docs

import (
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

// RuleDoc provides self-contained documentation and fix guidance for a testguard rule.
type RuleDoc struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Severity     models.Severity `json:"severity"`
	Languages    []string        `json:"languages"`
	Description  string          `json:"description"`
	WhyItMatters string          `json:"why_it_matters"`
	Remediation  string          `json:"remediation"`
	GoodExample  string          `json:"good_example"`
	BadExample   string          `json:"bad_example"`
}

// universalLanguages lists every language a language-agnostic detector runs on.
// Universal rules are reported for these and only these languages.
var universalLanguages = []string{"Go", "Dart", "TypeScript", "Kotlin", "Swift"}

// LanguageLabel renders the languages a rule can fire in for compact CLI display.
// A rule covering every supported language reads "Universal"; a multi-language
// rule lists its languages in order of first emission.
func (d RuleDoc) LanguageLabel() string {
	if len(d.Languages) == len(universalLanguages) {
		same := true
		for i, lang := range d.Languages {
			if lang != universalLanguages[i] {
				same = false
				break
			}
		}
		if same {
			return "Universal"
		}
	}
	return strings.Join(d.Languages, ", ")
}

var catalog = map[string]RuleDoc{
	// ── 1. Universal Rules ──
	"GUARD-UNIV-ZERO": {
		ID:           "GUARD-UNIV-ZERO",
		Title:        "Zero Assertion Test",
		Severity:     models.SeverityBlocker,
		Languages:    universalLanguages,
		Description:  "The test body contains no assertion the language adapter recognizes as outcome verification.",
		WhyItMatters: "A test with no assertions provides zero regression protection and gives false confidence in CI.",
		Remediation:  "Add explicit assertions verifying observable return values, emitted states, or side-effects.",
		GoodExample:  "expect(result, equals(expectedValue));",
		BadExample:   "test('computes data', () { service.compute(); }); // No expect!",
	},
	"GUARD-UNIV-01": {
		ID:           "GUARD-UNIV-01",
		Title:        "Duplicate Test Body",
		Severity:     models.SeverityWarning,
		Languages:    universalLanguages,
		Description:  "Tests in the same describe/group/class scope share identical execution tokens (literals included).",
		WhyItMatters: "Identical duplicate tests add runtime and maintenance cost while appearing to cover a distinct scenario.",
		Remediation:  "Delete the redundant test or give it distinct inputs, preconditions and assertions.",
		GoodExample:  "it('rounds up', () => expect(price(0.5)).toBe(1));\nit('rounds down', () => expect(price(0.4)).toBe(0));",
		BadExample:   "test('case A', () { expect(Calc(1), 2); });\ntest('case B', () { expect(Calc(1), 2); });",
	},
	"GUARD-UNIV-02": {
		ID:           "GUARD-UNIV-02",
		Title:        "Structural Test Repetition",
		Severity:     models.SeverityAdvisory,
		Languages:    universalLanguages,
		Description:  "Four or more tests in one scope share the same normalized token structure and differ only in literal values.",
		WhyItMatters: "Large copy-paste families make a suite tedious to change and hide which equivalence classes are actually covered.",
		Remediation:  "Consolidate the repetitive tests into a parameterized table-driven test without dropping boundary cases.",
		GoodExample:  "for (final testCase in testCases) { expect(Calc(testCase.input), testCase.want); }",
		BadExample:   "4+ tests repeating the same arrange-act-assert flow with only a number changed.",
	},
	"GUARD-UNIV-TAUTOLOGY": {
		ID:           "GUARD-UNIV-TAUTOLOGY",
		Title:        "Tautological Assertion",
		Severity:     models.SeverityBlocker,
		Languages:    universalLanguages,
		Description:  "Asserts identical operands, constant arithmetic, negated distinct literals, or a self-evident boolean (expect(x, x); assert.Equal(t, x, x); XCTAssertTrue(true); expect(true).not.toBe(false)).",
		WhyItMatters: "A tautology passes for every possible implementation, so it cannot detect a regression.",
		Remediation:  "Assert the value produced by the system under test against a distinct expected value.",
		GoodExample:  "expect(calculator.add(1, 2)).toBe(3);",
		BadExample:   "expect(value).toBe(value); // Always passes regardless of implementation",
	},
	"GUARD-UNIV-SKIP": {
		ID:           "GUARD-UNIV-SKIP",
		Title:        "Skipped Test",
		Severity:     models.SeverityWarning,
		Languages:    universalLanguages,
		Description:  "A test declaration is disabled (it.skip, test.skip, skip: true, @Skip, @Ignore/@Disabled, throw XCTSkip, or @Test(.disabled)); conditional runtime skips are ignored.",
		WhyItMatters: "Skipped tests rot silently and conceal failing features from CI.",
		Remediation:  "Fix and unskip the test, or delete it and record why in the tracker.",
		GoodExample:  "it('computes total', () => { ... });",
		BadExample:   "it.skip('flaky checkout test', () => { ... });",
	},
	"GUARD-UNIV-FOCUS": {
		ID:           "GUARD-UNIV-FOCUS",
		Title:        "Focused Test",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript", "Dart"},
		Description:  "A committed focus modifier (it.only, test.only, describe.only, fit, fdescribe, .only.each, solo: true) stops the rest of the suite from running.",
		WhyItMatters: "A focused test silently excludes every other test, so CI passes while untested subsystems stay unverified.",
		Remediation:  "Remove the focus modifier before committing so the whole suite executes.",
		GoodExample:  "it('validates email format', () => { ... });",
		BadExample:   "it.only('validates email format', () => { ... });",
	},
	"GUARD-UNIV-NAME": {
		ID:           "GUARD-UNIV-NAME",
		Title:        "Vague / Placeholder Test Name",
		Severity:     models.SeverityAdvisory,
		Languages:    universalLanguages,
		Description:  "The test name is a hollow placeholder (test 1, sample, dummy, foo) that states no scenario or expectation.",
		WhyItMatters: "Vague names obscure failure investigation in CI and defeat living documentation.",
		Remediation:  "Describe the scenario and the expected behaviour, e.g. Method_Scenario_ExpectedBehavior.",
		GoodExample:  "FetchCart_WhenTokenExpired_EmitsUnauthorizedFailureState",
		BadExample:   "test('test 1', () { ... })",
	},

	// ── 2. Dart / Flutter Mobile Rules ──
	"GUARD-DART-SEED-ECHO": {
		ID:           "GUARD-DART-SEED-ECHO",
		Title:        "Seed Echo Assertion in BLoC Test",
		Severity:     models.SeverityWarning,
		Languages:    []string{"Dart"},
		Description:  "Every expect matcher only repeats top-level fields already established by the blocTest seed, so no state transition is verified.",
		WhyItMatters: "A test that asserts only unchanged seed fields passes without proving the dispatched event changed anything.",
		Remediation:  "Assert the fields the event mutates, or verify the error condition, rather than echoing the seed.",
		GoodExample:  "isA<UserState>().having((s) => s.status, 'status', Status.error)",
		BadExample:   "isA<UserState>().having((s) => s.name, 'name', 'bob') // seed already sets name: 'bob'",
	},
	"GUARD-DART-BLOC-DUPLICATE": {
		ID:           "GUARD-DART-BLOC-DUPLICATE",
		Title:        "Duplicate BLoC Test Scenario",
		Severity:     models.SeverityAdvisory,
		Languages:    []string{"Dart"},
		Description:  "Two blocTest cases in one file execute the exact same scenario (build/setUp/seed/act/expect/skip/verify, name excluded).",
		WhyItMatters: "Duplicate BLoC scenarios cost runtime without adding coverage or fault proof.",
		Remediation:  "Delete the redundant blocTest or test a distinct failure or edge-case branch.",
		GoodExample:  "blocTest('Success case', ...); blocTest('Timeout failure case', ...);",
		BadExample:   "blocTest('Search success', act: fetch, expect: data); blocTest('Auto-search success', act: fetch, expect: data);",
	},
	"GUARD-DART-MOCK-ECHO": {
		ID:           "GUARD-DART-MOCK-ECHO",
		Title:        "Pass-Through Mock Echo in Dart",
		Severity:     models.SeverityWarning,
		Languages:    []string{"Dart"},
		Description:  "A service or repository unit test stubs a collaborator (thenAnswer/thenReturn, Right or Left) and asserts the same token back with zero transformation.",
		WhyItMatters: "Mock echoing tests only that the mocking library works, not domain logic, mapping, or error translation.",
		Remediation:  "Assert domain entity mapping, calculation outcomes, or error translation.",
		GoodExample:  "expect(result, Right(fakeUser)); // datasource returned fakeUserModel",
		BadExample:   "when(() => repo.fetch()).thenAnswer((_) async => Right(fakeUser)); expect(result, Right(fakeUser));",
	},
	"GUARD-DART-VACUOUS-STATE": {
		ID:           "GUARD-DART-VACUOUS-STATE",
		Title:        "Vacuous State Assertion in BLoC Test",
		Severity:     models.SeverityWarning,
		Languages:    []string{"Dart"},
		Description:  "A typed blocTest expects only bare isA<S>() for the base state type, with no .having property matcher and no verify block.",
		WhyItMatters: "isA<BaseState>() matches every emitted state, so it passes whether the loaded, empty, or error state was emitted.",
		Remediation:  "Assert concrete state subclasses or specific properties with .having(...).",
		GoodExample:  "expect: () => [isA<UserLoaded>().having((s) => s.user.name, 'name', 'alice')]",
		BadExample:   "expect: () => [isA<UserState>()]",
	},

	// ── 3. Go Backend Rules ──
	"GUARD-GO-01": {
		ID:           "GUARD-GO-01",
		Title:        "Domain Test Mock Usage",
		Severity:     models.SeverityAdvisory,
		Languages:    []string{"Go"},
		Description:  "A test under a domain/ path instantiates a mock controller or generated mock.",
		WhyItMatters: "Pure domain logic is cheapest to verify with struct fixtures, though many teams mock domain ports by convention, so this is an advisory note.",
		Remediation:  "Consider pure domain fixtures if the team follows a mock-free domain test convention.",
		GoodExample:  "u := User{Name: \"Alice\", Age: 30}; if !u.IsValid() { t.Fatal(\"expected valid user\") }",
		BadExample:   "ctrl := gomock.NewController(t); repo := mocks.NewMockUserRepository(ctrl)",
	},
	"GUARD-GO-03": {
		ID:           "GUARD-GO-03",
		Title:        "Table-Test Duplicate Input Rows",
		Severity:     models.SeverityAdvisory,
		Languages:    []string{"Go"},
		Description:  "A Go table-driven test contains multiple rows with identical input values and expected results.",
		WhyItMatters: "Duplicate rows inflate the test count without testing a new equivalence class.",
		Remediation:  "Make each row cover a distinct equivalence class, boundary, or error scenario.",
		GoodExample:  "{name: 'valid price', in: 10, want: 10}, {name: 'negative price', in: -1, wantErr: true}",
		BadExample:   "{name: 'case 1', in: 10, want: 10}, {name: 'case 2', in: 10, want: 10}",
	},
	"GUARD-GO-SWALLOWED-ERR": {
		ID:           "GUARD-GO-SWALLOWED-ERR",
		Title:        "Swallowed Error via Bare Return",
		Severity:     models.SeverityWarning,
		Languages:    []string{"Go"},
		Description:  "A test function or t.Run closure handles an unexpected error with `if err != nil { return }` instead of failing the test.",
		WhyItMatters: "A bare return lets the test pass vacuously when setup or the call under test fails, hiding the regression from CI.",
		Remediation:  "Fail the test with t.Fatalf(\"unexpected error: %v\", err) or require.NoError(t, err).",
		GoodExample:  "if err != nil { t.Fatalf(\"Build failed: %v\", err) }",
		BadExample:   "if err != nil { return }",
	},
	"GUARD-GO-UNCHECKED-WANT": {
		ID:           "GUARD-GO-UNCHECKED-WANT",
		Title:        "Unchecked Want Field in Table Test",
		Severity:     models.SeverityWarning,
		Languages:    []string{"Go"},
		Description:  "A table row struct declares an expected-value field (want, wantErr, expected, ...) that the range loop body never reads.",
		WhyItMatters: "The author intended to verify that field; leaving it unread means the test asserts less than it appears to.",
		Remediation:  "Assert the expected field in the loop body, or delete the dead field from the row struct.",
		GoodExample:  "if (err != nil) != tt.wantErr { t.Fatalf(\"err=%v, wantErr=%v\", err, tt.wantErr) }",
		BadExample:   "got, _ := Calculate(tt.in)\nassert.Equal(t, tt.want, got) // tt.wantErr declared but never checked",
	},

	// ── 4. NestJS Backend Rules ──
	"GUARD-NEST-01": {
		ID:           "GUARD-NEST-01",
		Title:        "NestJS Pass-Through Controller Echo",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "A controller unit test asserts identity or equality with a mocked service value, inline or through an assigned result variable.",
		WhyItMatters: "1:1 pass-through mock echoing exercises no validation pipe, guard, interceptor, or business logic.",
		Remediation:  "Test validation pipes, guards, interceptors, or integration behaviour via supertest.",
		GoodExample:  "await expect(controller.create(invalidDto)).rejects.toThrow(BadRequestException);",
		BadExample:   "const result = await controller.findAll(); expect(result).toEqual(mockData);",
	},
	"GUARD-NEST-02": {
		ID:           "GUARD-NEST-02",
		Title:        "Unmocked Database Module in NestJS Unit Test",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "A unit-testing module imports a live database module (TypeOrmModule.forRoot, MongooseModule.forRoot, SequelizeModule.forRoot, PrismaModule) without an in-memory configuration or module override.",
		WhyItMatters: "Live database connections make unit tests slow, environment-dependent, and flaky in CI.",
		Remediation:  "Provide getRepositoryToken() stubs or .overrideProvider() in Test.createTestingModule.",
		GoodExample:  "providers: [{ provide: getRepositoryToken(User), useValue: mockUserRepository }]",
		BadExample:   "imports: [TypeOrmModule.forRoot({ type: 'postgres', host: 'db.prod' })]",
	},

	// ── 5. React & Next.js Frontend Rules ──
	"GUARD-REACT-02": {
		ID:           "GUARD-REACT-02",
		Title:        "Arbitrary Sleep / setTimeout Fragility",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "Real-timer sleeps > 0 ms in unit test bodies; ignores 0 ms flushes, mock latency, fake timers, and E2E files.",
		WhyItMatters: "Hardcoded sleeps slow the suite and flake under CI CPU variance.",
		Remediation:  "Use waitFor(...) or async findBy* queries instead of fixed sleeps.",
		GoodExample:  "await waitFor(() => expect(screen.getByText('Done')).toBeInTheDocument());",
		BadExample:   "await new Promise((r) => setTimeout(r, 1500));",
	},
	"GUARD-E2E-01": {
		ID:           "GUARD-E2E-01",
		Title:        "Hard Wait in E2E Test",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript", "Dart"},
		Description:  "An end-to-end or integration test uses a fixed delay (page.waitForTimeout, Future.delayed) instead of an event-driven wait.",
		WhyItMatters: "Hard waits slow E2E runs and flake under varying machine load.",
		Remediation:  "Wait for a DOM selector, network response, or state condition instead of a fixed duration.",
		GoodExample:  "await expect(page.getByText('Success')).toBeVisible();",
		BadExample:   "await page.waitForTimeout(3000);",
	},

	// ── 6. TypeScript / JavaScript Rule Pack ──
	"GUARD-TS-01": {
		ID:           "GUARD-TS-01",
		Title:        "Snapshot-Only Test Assertions",
		Severity:     models.SeverityAdvisory,
		Languages:    []string{"TypeScript"},
		Description:  "Every assertion in a non-E2E test is a snapshot matcher (toMatchSnapshot, toMatchInlineSnapshot, toMatchFileSnapshot).",
		WhyItMatters: "Snapshot-only tests are routinely refreshed with -u without anyone checking the domain invariant they were meant to protect.",
		Remediation:  "Add behavioural assertions on state or return values alongside or instead of the snapshot.",
		GoodExample:  "expect(screen.getByRole('button', { name: /checkout/i })).toBeEnabled();",
		BadExample:   "expect(render(<UserCard name=\"Alice\" />)).toMatchSnapshot();",
	},
	"GUARD-TS-FLOATING-ASSERT": {
		ID:           "GUARD-TS-FLOATING-ASSERT",
		Title:        "Floating Async Promise Assertion",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "A .resolves or .rejects assertion sits in an expression statement that is neither awaited nor returned.",
		WhyItMatters: "An un-awaited promise assertion runs in the background, so its rejection never fails the test.",
		Remediation:  "Await the assertion (await expect(...)) or return it.",
		GoodExample:  "await expect(fetchUser('1')).resolves.toEqual({ id: '1' });",
		BadExample:   "expect(fetchUser('1')).resolves.toEqual({ id: '1' });",
	},
	"GUARD-TS-CATCH-ONLY": {
		ID:           "GUARD-TS-CATCH-ONLY",
		Title:        "Swallowed Error Path in Catch Block",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "All assertions live inside a catch block while the try block has no failure guard (fail(), expect.fail(), throw).",
		WhyItMatters: "If the call unexpectedly succeeds, the catch block never runs and the test passes with zero assertions executed.",
		Remediation:  "Add expect.fail(...) at the end of the try block, or use expect(...).rejects.toThrow().",
		GoodExample:  "await expect(login('bad', 'pwd')).rejects.toThrow(UnauthorizedException);",
		BadExample:   "try { await login('bad', 'pwd'); } catch (e) { expect(e).toBeInstanceOf(UnauthorizedException); }",
	},
	"GUARD-TS-CONDITIONAL-ASSERT": {
		ID:           "GUARD-TS-CONDITIONAL-ASSERT",
		Title:        "Conditional Assertion Without Execution Guarantee",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "Some reachable branch of an if/switch/ternary/&&/|| construct asserts nothing, and the test declares no expect.hasAssertions()/expect.assertions() count. A branch that ends the test with throw, fail(), expect.fail(), assert.fail() or expect.unreachable() counts as total; a failure nested in an inner conditional does not.",
		WhyItMatters: "When the condition takes the silent path the assertions never execute, so the test passes without checking anything.",
		Remediation:  "Make the branches exhaustive with assertions in each, or declare expect.hasAssertions().",
		GoodExample:  "expect(token).toBeDefined(); expect(token.roles).toContain('admin');",
		BadExample:   "if (token) { expect(token.roles).toContain('admin'); }",
	},
	"GUARD-TS-EXISTENCE-ONLY": {
		ID:           "GUARD-TS-EXISTENCE-ONLY",
		Title:        "Existence-Only Test Assertions",
		Severity:     models.SeverityAdvisory,
		Languages:    []string{"TypeScript"},
		Description:  "Every assertion only checks presence or truthiness (toBeDefined, toBeTruthy, not.toBeNull, chai .to.exist/.to.be.ok).",
		WhyItMatters: "Presence is not correctness: the payload or domain state can be wrong and the test still passes.",
		Remediation:  "Assert exact expected values, types, or domain state.",
		GoodExample:  "expect(user).toEqual({ name: 'Bob', role: 'member' });",
		BadExample:   "expect(user).toBeDefined();",
	},
	"GUARD-TS-SELF-MOCK": {
		ID:           "GUARD-TS-SELF-MOCK",
		Title:        "Self-Mocking Direct Invocation Assertion",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "The test body itself invokes the mock it then asserts on (jest.fn(), vi.fn(), sinon.stub(), sinon.spy()). A mock invoked only by the SUT is not reported.",
		WhyItMatters: "Calling your own test double and asserting its configured return value tests the framework, not the application.",
		Remediation:  "Invoke the system under test, which uses the mock, and assert on its behaviour.",
		GoodExample:  "const user = await service.getUser(1); expect(user.id).toBe(1);",
		BadExample:   "const repo = { find: jest.fn().mockReturnValue(42) }; expect(repo.find()).toBe(42);",
	},
	"GUARD-TS-MOCKED-SUT": {
		ID:           "GUARD-TS-MOCKED-SUT",
		Title:        "Auto-Mocked System Under Test",
		Severity:     models.SeverityWarning,
		Languages:    []string{"TypeScript"},
		Description:  "The test file auto-mocks its own subject (jest.mock('./sut') or vi.mock('./sut')) without a factory argument.",
		WhyItMatters: "Mocking the unit under test replaces the production implementation with stubs, so the suite exercises no production code.",
		Remediation:  "Mock only the SUT's external dependencies, never the SUT itself.",
		GoodExample:  "jest.mock('./externalApi'); // SUT itself stays real",
		BadExample:   "jest.mock('./userService'); // file also imports UserService from './userService'",
	},

	// ── 7. Native Mobile Rules (Android & iOS) ──
	"GUARD-KT-01": {
		ID:           "GUARD-KT-01",
		Title:        "Kotlin Mock Echo (Unassertive Mock Verification)",
		Severity:     models.SeverityWarning,
		Languages:    []string{"Kotlin"},
		Description:  "A MockK stub (every/coEvery { ... } returns X) is compared unchanged with a property or accessor on a subject built from that mock.",
		WhyItMatters: "Asserting the unmodified stubbed value verifies the mock configuration, not the SUT's logic.",
		Remediation:  "Assert a transformed outcome or side effect instead of echoing the stub.",
		GoodExample:  "assertEquals(\"ALICE\", viewModel.name.uppercase())",
		BadExample:   "every { repo.name() } returns \"alice\"; assertEquals(\"alice\", viewModel.name)",
	},
	"GUARD-KT-02": {
		ID:           "GUARD-KT-02",
		Title:        "Unadvanced Kotlin Coroutine Virtual Time under StandardTestDispatcher",
		Severity:     models.SeverityWarning,
		Languages:    []string{"Kotlin"},
		Description:  "A test running under runTest with StandardTestDispatcher builds a subject from that dispatcher, calls it, then asserts on its state without advancing virtual time.",
		WhyItMatters: "Under StandardTestDispatcher coroutines do not run eagerly, so the assertion can observe the initial state and flake or pass vacuously.",
		Remediation:  "Call advanceUntilIdle(), advanceTimeBy(), runCurrent(), or await the emitted state before asserting.",
		GoodExample:  "viewModel.search(\"kotlin\"); advanceUntilIdle(); assertThat(state.loading).isFalse()",
		BadExample:   "viewModel.search(\"kotlin\"); assertThat(state.loading).isFalse()",
	},
	"GUARD-SWIFT-01": {
		ID:           "GUARD-SWIFT-01",
		Title:        "Long XCTest Expectation Timeout",
		Severity:     models.SeverityAdvisory,
		Languages:    []string{"Swift"},
		Description:  "An XCTest expectation wait or fulfillment uses a timeout greater than 2.0 seconds.",
		WhyItMatters: "A long timeout does not slow passing tests, but it stalls the feedback loop when a test genuinely fails or hangs.",
		Remediation:  "Prefer await fulfillment(of:) with async code and keep unit-test timeouts at or below about 2 seconds.",
		GoodExample:  "wait(for: [expectation], timeout: 1.0)",
		BadExample:   "wait(for: [expectation], timeout: 10.0)",
	},
}

// Explain returns the RuleDoc for a given rule ID.
func Explain(ruleID string) (RuleDoc, bool) {
	doc, found := catalog[strings.ToUpper(ruleID)]
	return doc, found
}

// AllRules returns all registered rule documentation items ordered by rule ID.
// The catalog is a map, so callers that render a list must not observe a
// different order between runs; sorting here keeps `testguard rules` stable.
func AllRules() []RuleDoc {
	rules := make([]RuleDoc, 0, len(catalog))
	for _, doc := range catalog {
		rules = append(rules, doc)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	return rules
}
