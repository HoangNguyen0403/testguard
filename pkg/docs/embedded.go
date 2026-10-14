package docs

import (
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

// RuleDoc provides self-contained documentation and fix guidance for a testguard rule.
type RuleDoc struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Severity    models.Severity `json:"severity"`
	Language    string          `json:"language"`
	Description string          `json:"description"`
	WhyItMatters string         `json:"why_it_matters"`
	Remediation string          `json:"remediation"`
	GoodExample string          `json:"good_example"`
	BadExample  string          `json:"bad_example"`
}

var catalog = map[string]RuleDoc{
	"GUARD-UNIV-ZERO": {
		ID:          "GUARD-UNIV-ZERO",
		Title:       "Zero Assertion Test",
		Severity:    models.SeverityBlocker,
		Language:    "Universal",
		Description: "The test body contains zero expect(), assert, or outcome verification calls.",
		WhyItMatters: "A test with no assertions provides zero regression protection and gives false confidence in CI.",
		Remediation: "Add explicit assertions verifying observable return values, emitted states, or side-effects.",
		GoodExample: "expect(result, equals(expectedValue));",
		BadExample:  "test('computes data', () { service.compute(); }); // No expect!",
	},
	"GUARD-UNIV-01": {
		ID:          "GUARD-UNIV-01",
		Title:       "Exact Test Clone",
		Severity:    models.SeverityAdvisory,
		Language:    "Universal",
		Description: "Two tests share identical AST token execution sequences.",
		WhyItMatters: "Exact duplicate tests bloat test suite runtime without providing distinct behavioral coverage.",
		Remediation: "Eliminate the redundant test case or consolidate varied inputs into a table-driven parameterized test.",
		GoodExample: "for (final testCase in testCases) { ... }",
		BadExample:  "test('case A', () { ... });\ntest('case B', () { /* identical body */ });",
	},
	"GUARD-DART-SEED-ECHO": {
		ID:          "GUARD-DART-SEED-ECHO",
		Title:       "Seed Echo Assertion in BLoC Test",
		Severity:    models.SeverityWarning,
		Language:    "Dart",
		Description: "Expected state properties merely echo values already established in the seed baseline.",
		WhyItMatters: "Asserting unchanged fields over-specifies tests and breaks resistance to refactoring.",
		Remediation: "Use base state fixtures and assert only the mutated fields (delta) in state transitions.",
		GoodExample: "baseState.copyWith(isFetching: true)",
		BadExample:  "isA<State>().having((s) => s.userInfo, 'userInfo', fakeUser) // when userInfo is invariant",
	},
	"GUARD-DART-UNCOVERED-MUTATION": {
		ID:          "GUARD-DART-UNCOVERED-MUTATION",
		Title:       "Uncovered State Mutation in BLoC Test",
		Severity:    models.SeverityBlocker,
		Language:    "Dart",
		Description: "A BLoC event handler writes a field (e.g. isFetching = true), but the expected state transition omits asserting it.",
		WhyItMatters: "Leaves critical loading or error state transitions unverified, allowing regressions to ship undetected.",
		Remediation: "Assert all mutated state transition properties in the expected sequence.",
		GoodExample: "expect: () => [baseState.copyWith(isFetching: true), baseState.copyWith(isFetching: false, data: result)]",
		BadExample:  "expect: () => [isA<State>()] // omitted isFetching check",
	},
	"GUARD-GO-01": {
		ID:          "GUARD-GO-01",
		Title:       "Domain Purity (Zero Mocks)",
		Severity:    models.SeverityBlocker,
		Language:    "Go",
		Description: "A domain entity or model test imports mock controllers or database drivers.",
		WhyItMatters: "Clean Architecture requires domain models to be pure business logic tested with real instances.",
		Remediation: "Remove mocks from domain tests. Test business logic using pure domain struct fixtures.",
		GoodExample: "item := domain.NewItem(100); assert.True(t, item.IsValid())",
		BadExample:  "mockRepo := mocks.NewMockRepo(ctrl); // mocks in domain test",
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
