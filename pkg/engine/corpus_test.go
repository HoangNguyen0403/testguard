package engine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/docs"
	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/models"
)

const (
	sevBlocker  = models.SeverityBlocker
	sevWarning  = models.SeverityWarning
	sevAdvisory = models.SeverityAdvisory
)

// expectedFinding is one corpus contract row: a finding must exist with exactly
// this rule, severity, path relative to the case directory and line.
type expectedFinding struct {
	RuleID   string
	Severity models.Severity
	Path     string
	Line     int
}

// excludedExpectation pins a path the discovery layer must exclude and why.
type excludedExpectation struct {
	Path   string
	Reason string
}

type corpusCase struct {
	name string
	dir  string
	// expected lists every finding the case may emit. Any finding not listed is
	// a false positive; any row not matched is a false negative.
	expected []expectedFinding
	// negativeFiles must be analysed (present in the scanned scope, not
	// excluded) and must emit no finding at all.
	negativeFiles []string
	// excludedFiles must never reach the analyser, with the stated reason.
	excludedFiles []excludedExpectation
	// analyzedTests pins the exact number of discovered tests when non-zero;
	// every ordinary case must analyse at least one test.
	analyzedTests int
	// emptyScope marks a scope that is intentionally empty (only skipped
	// declarations) rather than an ordinary no-findings case.
	emptyScope bool
	// parseFailure marks a deliberately malformed file: the adapter must fail
	// closed, the scan must report input_failure and the CLI must exit 2.
	parseFailure bool
}

type ruleLangKey struct {
	RuleID   string
	Language string
}

type metricCount struct {
	TP int
	FP int
	FN int
}

func detectLanguageFromExt(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "Go"
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".mts", ".cts":
		return "TypeScript"
	case ".dart":
		return "Dart"
	case ".kt":
		return "Kotlin"
	case ".swift":
		return "Swift"
	default:
		return "Unknown"
	}
}

// relativeToCase converts an engine-normalised path into the path relative to
// the corpus case directory, which is how the table rows identify files.
func relativeToCase(caseDir, findingPath string) string {
	slashDir := filepath.ToSlash(caseDir)
	slashPath := filepath.ToSlash(findingPath)
	return strings.TrimPrefix(slashPath, slashDir+"/")
}

// TestPrecisionCorpus asserts exact corpus contracts: rule, severity, relative
// path and line for every expected finding, plus the analysis status semantics
// that make an empty scan distinguishable from a clean one.
func TestPrecisionCorpus(t *testing.T) {
	cases := []corpusCase{
		// --- Universal rule coverage across every supported language ---
		{name: "Go: duplicate test body", dir: "testdata/corpus/univ_go_clone_duplicate",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "duplicates_test.go", Line: 13}}},
		{name: "Go: four literal variants", dir: "testdata/corpus/univ_go_structural_variants",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-02", Severity: sevAdvisory, Path: "variants_test.go", Line: 25}}},
		{name: "Go: tautological assertion", dir: "testdata/corpus/univ_go_tautology",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "tautology_test.go", Line: 13}}},
		{name: "Go: committed skip", dir: "testdata/corpus/univ_go_skip",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "skip_test.go", Line: 12}}},
		{name: "Go: vague test name", dir: "testdata/corpus/univ_go_vague_name",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-NAME", Severity: sevAdvisory, Path: "vague_test.go", Line: 11}}},
		{name: "Dart: duplicate test body", dir: "testdata/corpus/univ_dart_clone_duplicate",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "duplicates_test.dart", Line: 10}}},
		{name: "Dart: four literal variants", dir: "testdata/corpus/univ_dart_structural_variants",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-02", Severity: sevAdvisory, Path: "variants_test.dart", Line: 18}}},
		{name: "Dart: tautological assertion", dir: "testdata/corpus/univ_dart_tautology",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "tautology_test.dart", Line: 8}}},
		{name: "Dart: committed skip", dir: "testdata/corpus/univ_dart_skip",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "skip_test.dart", Line: 8}}},
		{name: "Dart: vague test name", dir: "testdata/corpus/univ_dart_vague_name",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-NAME", Severity: sevAdvisory, Path: "vague_test.dart", Line: 6}}},
		{name: "Kotlin: duplicate test body", dir: "testdata/corpus/univ_kotlin_clone_duplicate",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "DuplicatesTest.kt", Line: 13}}},
		{name: "Kotlin: four literal variants", dir: "testdata/corpus/univ_kotlin_structural_variants",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-02", Severity: sevAdvisory, Path: "VariantsTest.kt", Line: 23}}},
		{name: "Kotlin: committed skip", dir: "testdata/corpus/univ_kotlin_skip",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "SkipTest.kt", Line: 8}}},
		{name: "Kotlin: vague test name", dir: "testdata/corpus/univ_kotlin_vague_name",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-NAME", Severity: sevAdvisory, Path: "VagueTest.kt", Line: 8}}},
		{name: "Swift: duplicate test body", dir: "testdata/corpus/univ_swift_clone_duplicate",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "DuplicatesTests.swift", Line: 10}}},
		{name: "Swift: four literal variants", dir: "testdata/corpus/univ_swift_structural_variants",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-02", Severity: sevAdvisory, Path: "VariantsTests.swift", Line: 18}}},
		{name: "Swift: committed skip", dir: "testdata/corpus/univ_swift_skip",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "SkipTests.swift", Line: 7}}},
		{name: "Swift: vague test name", dir: "testdata/corpus/univ_swift_vague_name",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-NAME", Severity: sevAdvisory, Path: "VagueTests.swift", Line: 6}}},
		{name: "TypeScript: four literal variants", dir: "testdata/corpus/univ_ts_structural_variants",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-02", Severity: sevAdvisory, Path: "variants.test.ts", Line: 16}}},
		{name: "TypeScript: vague test name", dir: "testdata/corpus/univ_ts_vague_name",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-NAME", Severity: sevAdvisory, Path: "vague.test.ts", Line: 3}}},

		// --- Universal rule dedicated fixtures ---
		{name: "Universal: exact clone", dir: "testdata/corpus/univ_clone_exact_positive",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "clones.test.ts", Line: 7}}},
		{name: "Universal: clone across scopes", dir: "testdata/corpus/univ_clone_scope_negative"},
		{name: "Universal: literal variants below threshold", dir: "testdata/corpus/univ_clone_literals_negative"},
		{name: "Universal: tautology positive", dir: "testdata/corpus/univ_tautology_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "tautology.test.ts", Line: 4},
				{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "tautology.test.ts", Line: 8},
				{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "tautology.test.ts", Line: 12},
			}},
		{name: "Universal: tautology negative", dir: "testdata/corpus/univ_tautology_negative"},
		{name: "Universal: skip positive", dir: "testdata/corpus/univ_skip_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "skip.spec.ts", Line: 3},
				{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "skip.spec.ts", Line: 8},
				{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "skip.spec.ts", Line: 13},
			}},
		{name: "Universal: skip negative", dir: "testdata/corpus/univ_skip_negative"},
		{name: "Universal: focus positive", dir: "testdata/corpus/univ_focus_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-FOCUS", Severity: sevWarning, Path: "focus.test.ts", Line: 1},
				{RuleID: "GUARD-UNIV-FOCUS", Severity: sevWarning, Path: "focus.test.ts", Line: 8},
				{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "focus.test.ts", Line: 9},
			}},
		{name: "Universal: focus negative", dir: "testdata/corpus/univ_focus_negative"},
		{name: "Universal: Kotlin tautology positive", dir: "testdata/corpus/univ_kotlin_tautology_positive",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "TautologyTest.kt", Line: 4}}},
		{name: "Universal: Kotlin tautology negative", dir: "testdata/corpus/univ_kotlin_tautology_negative"},
		// F5 regression: identical plain identifiers/member paths are tautologies,
		// while different receivers are not.
		{name: "Universal: Kotlin identical-identifier tautology", dir: "testdata/corpus/univ_kotlin_tautology_identifier_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "TautologyMemberTest.kt", Line: 5}},
			analyzedTests: 2},
		{name: "Kotlin: nested comment and template lexical awareness", dir: "testdata/corpus/kt_lexical_comment_negative",
			analyzedTests: 2},
		// F8 regression: an unbalanced Kotlin file fails closed.
		{name: "Kotlin: unterminated body is a parse failure", dir: "testdata/corpus/kt_unterminated_body_parse_failed",
			parseFailure: true},
		// N2 regression: extra consecutive quotes after a raw-string close are part
		// of the literal, not a new unterminated string.
		{name: "Kotlin: raw string ending in a quote", dir: "testdata/corpus/kt_raw_string_four_quotes_negative",
			analyzedTests: 1},
		{name: "Kotlin: raw strings with trailing backslash and quotes", dir: "testdata/corpus/kt_raw_string_backslash_negative",
			analyzedTests: 2},
		{name: "Kotlin: lexical edge shapes stay analysable", dir: "testdata/corpus/kt_lexical_valid_negative",
			analyzedTests: 4},
		// N2 fail-closed counterpart: an actually unterminated raw string errors.
		{name: "Kotlin: unterminated raw string is a parse failure", dir: "testdata/corpus/kt_unterminated_raw_string_parse_failed",
			parseFailure: true},
		{name: "Universal: Swift tautology positive", dir: "testdata/corpus/univ_swift_tautology_positive",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "TautologyTests.swift", Line: 5}}},
		{name: "Universal: Swift tautology negative", dir: "testdata/corpus/univ_swift_tautology_negative"},
		// F5 regression: identical identifiers are tautologies in Swift too.
		{name: "Universal: Swift identical-identifier tautology", dir: "testdata/corpus/univ_swift_tautology_identifier_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "TautologyMemberTests.swift", Line: 4}},
			analyzedTests: 2},
		// F8 regression: raw strings, nested interpolation, and parameterised
		// `func test…` helpers do not break discovery or create tests.
		{name: "Universal: Swift lexical structure", dir: "testdata/corpus/swift_lexical_structure_negative",
			analyzedTests: 3},
		// F8 regression: an unterminated Swift string fails closed.
		{name: "Swift: unterminated string is a parse failure", dir: "testdata/corpus/swift_unterminated_string_parse_failed",
			parseFailure: true},
		{name: "Universal: Dart focus positive", dir: "testdata/corpus/univ_dart_focus_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "focus_test.dart", Line: 5},
				{RuleID: "GUARD-UNIV-FOCUS", Severity: sevWarning, Path: "focus_test.dart", Line: 6},
			}},
		{name: "Universal: Dart focus negative", dir: "testdata/corpus/univ_dart_focus_negative"},
		{name: "Universal: four identical TS tests", dir: "testdata/corpus/clones_four_identical",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "clones.test.ts", Line: 6},
				{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "clones.test.ts", Line: 11},
				{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "clones.test.ts", Line: 16},
			}},
		{name: "Universal: expect(true).toBe(true)", dir: "testdata/corpus/tautology_expect_true",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "tautology.test.ts", Line: 2}}},
		// A file whose only declarations are skipped has an intentionally empty
		// analysable scope: exit 2 and empty_scope must not be read as a clean pass.
		{name: "Universal: it.skip", dir: "testdata/corpus/skip_it_skip",
			expected:   []expectedFinding{{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "skipped.test.ts", Line: 1}},
			emptyScope: true},

		// --- Go: negative cases (former false positives) ---
		{name: "Go: t.Run table test", dir: "testdata/corpus/go_trun_table"},
		{name: "Go: testify suite runner", dir: "testdata/corpus/go_testify_suite"},
		{name: "Go: same-file asserting helper", dir: "testdata/corpus/go_asserting_helper"},
		{name: "Go: gomock EXPECT with NewController(t)", dir: "testdata/corpus/go_gomock_expect"},
		{name: "Go: test inside block comment", dir: "testdata/corpus/go_commented_test"},
		{name: "Go: sqlmock in repository test", dir: "testdata/corpus/go_sqlmock_repo"},
		{name: "Go: probe test with t.Run loop", dir: "testdata/corpus/go_probe_table"},
		{name: "Go: table test with mock map nested params", dir: "testdata/corpus/go_table_mock_map"},
		{name: "Go: gomock in closure with package-qualified constructor", dir: "testdata/corpus/go_gomock_closure_provenance"},
		{name: "Go: table with positional strings", dir: "testdata/corpus/go_table_positional_strings"},
		{name: "Go: gomock with cross-file helper", dir: "testdata/corpus/go_gomock_cross_file_helper"},
		{name: "Go: pure delegating runner", dir: "testdata/corpus/go_pure_delegating_runner"},
		{name: "Go: gomega assertions", dir: "testdata/corpus/go_gomega_assert"},
		{name: "Go: quicktest assertions", dir: "testdata/corpus/go_quicktest_assert"},
		{name: "Go: external assertion helper", dir: "testdata/corpus/go_external_helper"},
		{name: "Go: package assertion helper", dir: "testdata/corpus/go_package_helper_assert"},
		{name: "Go: swallowed error negative", dir: "testdata/corpus/go_swallowed_err_negative"},
		{name: "Go: unchecked want negative", dir: "testdata/corpus/go_unchecked_want_negative"},
		{name: "Go: named row unchecked want negative", dir: "testdata/corpus/go_unchecked_want_named_negative"},

		// --- Go: positive cases ---
		{name: "Go: zero assertions", dir: "testdata/corpus/go_zero_assert",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "no_assert_test.go", Line: 5}}},
		{name: "Go: hollow shadowed helper", dir: "testdata/corpus/go_hollow_shadow",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "shadow_test.go", Line: 9}}},
		{name: "Go: setup-only test drops its result", dir: "testdata/corpus/go_package_helper_negative",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "pkg_setup_test.go", Line: 5}}},
		{name: "Go: duplicate table rows", dir: "testdata/corpus/go_table_duplicate",
			expected: []expectedFinding{{RuleID: "GUARD-GO-03", Severity: sevAdvisory, Path: "table_duplicate_test.go", Line: 12}}},
		// Both copies are analysed. Only the domain/ copy can fire GUARD-GO-01
		// (the rule is path-gated); the directory-root copy is an asserted
		// path-gate negative so its silence is a contract, not an accident.
		{name: "Go: domain test mocks its port", dir: "testdata/corpus/go_domain_purity_advisory",
			expected: []expectedFinding{
				{RuleID: "GUARD-GO-01", Severity: sevAdvisory, Path: "domain/order_service_test.go", Line: 13},
				{RuleID: "GUARD-GO-01", Severity: sevAdvisory, Path: "domain/order_service_test.go", Line: 15},
			},
			negativeFiles: []string{"order_service_test.go"},
			analyzedTests: 2},
		{name: "Go: domain test with aliased gomock controller", dir: "testdata/corpus/go_domain_mock_advisory",
			expected: []expectedFinding{
				{RuleID: "GUARD-GO-01", Severity: sevAdvisory, Path: "domain/order_service_test.go", Line: 13},
				{RuleID: "GUARD-GO-01", Severity: sevAdvisory, Path: "domain/order_service_test.go", Line: 15},
			},
			analyzedTests: 1},
		{name: "Go: swallowed error", dir: "testdata/corpus/go_swallowed_err_positive",
			expected: []expectedFinding{{RuleID: "GUARD-GO-SWALLOWED-ERR", Severity: sevWarning, Path: "swallowed_err_test.go", Line: 11}}},
		{name: "Go: unchecked want field", dir: "testdata/corpus/go_unchecked_want_positive",
			expected: []expectedFinding{{RuleID: "GUARD-GO-UNCHECKED-WANT", Severity: sevWarning, Path: "table_unchecked_want_test.go", Line: 10}}},
		{name: "Go: unchecked want field on named row type", dir: "testdata/corpus/go_unchecked_want_named_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-GO-UNCHECKED-WANT", Severity: sevWarning, Path: "table_named_unchecked_test.go", Line: 14},
				{RuleID: "GUARD-GO-UNCHECKED-WANT", Severity: sevWarning, Path: "table_named_unchecked_test.go", Line: 15},
			}},

		// --- Dart ---
		{name: "Dart: asserting local helper", dir: "testdata/corpus/dart_asserting_helper"},
		{name: "Dart: two blocTests differing only in seed", dir: "testdata/corpus/dart_bloc_diff_seed"},
		{name: "Dart: blocTest with setUp semicolon", dir: "testdata/corpus/dart_bloc_setup_semicolon"},
		{name: "Dart: helper inside group with named params", dir: "testdata/corpus/dart_group_helper_named"},
		{name: "Dart: scope groups are not clones", dir: "testdata/corpus/dart_scope_groups"},
		// F7 regression: single-line, block and doc comments must not declare tests.
		{name: "Dart: commented-out declarations are not tests", dir: "testdata/corpus/dart_commented_tests_negative",
			analyzedTests: 3},
		// F7 regression: interpolation-aware scanning keeps the real tests,
		// their literal values and their assertions.
		{name: "Dart: interpolated and raw strings keep tests intact", dir: "testdata/corpus/dart_interpolation_negative",
			analyzedTests: 8},
		{name: "Dart: zero assertions", dir: "testdata/corpus/dart_zero_assert",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "empty_test.dart", Line: 4}}},
		{name: "Dart: hollow text-only helper", dir: "testdata/corpus/dart_hollow_helper",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "hollow_test.dart", Line: 8}}},
		{name: "Dart: seed echo", dir: "testdata/corpus/dart_seed_echo_positive",
			expected: []expectedFinding{{RuleID: "GUARD-DART-SEED-ECHO", Severity: sevWarning, Path: "seed_echo_test.dart", Line: 5}}},
		{name: "Dart: seed echo negative", dir: "testdata/corpus/dart_seed_echo_negative"},
		{name: "Dart: mock echo", dir: "testdata/corpus/dart_mock_echo_positive",
			expected: []expectedFinding{{RuleID: "GUARD-DART-MOCK-ECHO", Severity: sevWarning, Path: "mock_echo_test.dart", Line: 6}}},
		{name: "Dart: mock echo negative", dir: "testdata/corpus/dart_mock_echo_negative"},
		{name: "Dart: vacuous state", dir: "testdata/corpus/dart_vacuous_state_positive",
			expected: []expectedFinding{{RuleID: "GUARD-DART-VACUOUS-STATE", Severity: sevWarning, Path: "vacuous_state_test.dart", Line: 5}}},
		{name: "Dart: vacuous state negative", dir: "testdata/corpus/dart_vacuous_state_negative"},
		{name: "Dart: duplicate bloc scenario", dir: "testdata/corpus/dart_bloc_duplicate_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-DART-BLOC-DUPLICATE", Severity: sevAdvisory, Path: "duplicates_test.dart", Line: 9},
				{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "duplicates_test.dart", Line: 9},
			}},
		{name: "Dart: E2E Future.delayed", dir: "testdata/corpus/dart_e2e_future_delayed",
			expected: []expectedFinding{{RuleID: "GUARD-E2E-01", Severity: sevWarning, Path: "app_test.dart", Line: 8}}},

		// --- TypeScript / JavaScript ---
		{name: "TS: asserting helper", dir: "testdata/corpus/ts_asserting_helper"},
		{name: "TS: setTimeout(r, 0) flush", dir: "testdata/corpus/ts_sleep_zero"},
		{name: "TS: fake timers with setTimeout", dir: "testdata/corpus/ts_fake_timers"},
		{name: "TS: multiline expression arrow helper", dir: "testdata/corpus/ts_multiline_arrow"},
		{name: "TS: mocking latency helper stays existence-only", dir: "testdata/corpus/ts_mock_latency",
			expected: []expectedFinding{{RuleID: "GUARD-TS-EXISTENCE-ONLY", Severity: sevAdvisory, Path: "mock.test.ts", Line: 1}}},
		{name: "TS: commented-out helper declaration", dir: "testdata/corpus/ts_comment_helper_zero",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "comment_helper.test.ts", Line: 4}}},
		{name: "TS: empty body", dir: "testdata/corpus/ts_empty_body",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "empty.test.ts", Line: 1}}},
		{name: "TS: render-only test", dir: "testdata/corpus/ts_render_only",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "render.test.tsx", Line: 1}}},
		{name: "TS: hollow text-only helper", dir: "testdata/corpus/ts_hollow_helper",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "hollow.test.ts", Line: 5}}},
		{name: "TS: hollow shadowed helper", dir: "testdata/corpus/ts_hollow_shadow",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "shadow.test.ts", Line: 5}}},
		{name: "TS: real timer sleep", dir: "testdata/corpus/ts_real_sleep",
			expected: []expectedFinding{{RuleID: "GUARD-REACT-02", Severity: sevWarning, Path: "sleep.test.ts", Line: 2}}},
		{name: "TS: snapshot only", dir: "testdata/corpus/ts_snapshot_only",
			expected: []expectedFinding{{RuleID: "GUARD-TS-01", Severity: sevAdvisory, Path: "snapshot.test.ts", Line: 1}}},
		{name: "TS: snapshot with behaviour assertion", dir: "testdata/corpus/ts_snapshot_with_assertion"},
		{name: "TS: floating assertion", dir: "testdata/corpus/ts_floating_assert",
			expected: []expectedFinding{{RuleID: "GUARD-TS-FLOATING-ASSERT", Severity: sevWarning, Path: "floating.test.ts", Line: 2}}},
		{name: "TS: awaited assertion", dir: "testdata/corpus/ts_floating_assert_awaited"},
		{name: "TS: catch-only assertion", dir: "testdata/corpus/ts_catch_only",
			expected: []expectedFinding{{RuleID: "GUARD-TS-CATCH-ONLY", Severity: sevWarning, Path: "catch.test.ts", Line: 2}}},
		{name: "TS: guarded catch", dir: "testdata/corpus/ts_catch_only_guarded"},
		{name: "TS: conditional assertion", dir: "testdata/corpus/ts_conditional_assert",
			expected: []expectedFinding{{RuleID: "GUARD-TS-CONDITIONAL-ASSERT", Severity: sevWarning, Path: "conditional.test.ts", Line: 3}}},
		// F4 regression: branches that end the test via throw/fail are total.
		{name: "TS: conditional branches that fail the test are total", dir: "testdata/corpus/ts_conditional_failure_paths_negative",
			analyzedTests: 2},
		// F4 precision bound: a failure nested in an inner conditional does not
		// totalise the enclosing branch.
		{name: "TS: nested throw leaves the enclosing branch partial", dir: "testdata/corpus/ts_conditional_nested_throw_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-TS-CONDITIONAL-ASSERT", Severity: sevWarning, Path: "conditional_nested.test.ts", Line: 5}},
			analyzedTests: 1},
		// F4: a nested conditional whose every branch throws does totalise the
		// enclosing branch; only the sibling test with a silent path is reported.
		{name: "TS: exhaustive nested failure branches", dir: "testdata/corpus/ts_conditional_nested_exhaustive_negative",
			expected:      []expectedFinding{{RuleID: "GUARD-TS-CONDITIONAL-ASSERT", Severity: sevWarning, Path: "nested.test.ts", Line: 19}},
			analyzedTests: 2},
		// F12 regression: a multi-line title must not shift a body-relative line.
		{name: "TS: body-relative line attribution", dir: "testdata/corpus/ts_conditional_body_start_line_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-TS-CONDITIONAL-ASSERT", Severity: sevWarning, Path: "attribution.test.ts", Line: 8}},
			analyzedTests: 1},
		{name: "TS: unconditional assertion", dir: "testdata/corpus/ts_conditional_assert_unconditional"},
		{name: "TS: existence-only assertion", dir: "testdata/corpus/ts_existence_only",
			expected: []expectedFinding{{RuleID: "GUARD-TS-EXISTENCE-ONLY", Severity: sevAdvisory, Path: "existence.test.ts", Line: 1}}},
		{name: "TS: exact assertion", dir: "testdata/corpus/ts_existence_only_exact"},
		{name: "TS: self mock", dir: "testdata/corpus/ts_self_mock",
			expected: []expectedFinding{{RuleID: "GUARD-TS-SELF-MOCK", Severity: sevWarning, Path: "self_mock.test.ts", Line: 1}}},
		{name: "TS: mock handed to SUT", dir: "testdata/corpus/ts_self_mock_sut"},
		{name: "TS: mocked SUT", dir: "testdata/corpus/ts_mocked_sut",
			expected: []expectedFinding{
				{RuleID: "GUARD-TS-MOCKED-SUT", Severity: sevWarning, Path: "sut.test.ts", Line: 2},
				{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "sut.test.ts", Line: 5},
			}},
		{name: "TS: mocked dependency", dir: "testdata/corpus/ts_mocked_sut_dependency",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "sut.test.ts", Line: 5}}},
		{name: "TS: weak oracle family", dir: "testdata/corpus/ts_weak_oracles",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-FOCUS", Severity: sevWarning, Path: "weak.test.ts", Line: 38},
				{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "weak.test.ts", Line: 4},
				{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "weak.test.ts", Line: 47},
				{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "weak.test.ts", Line: 51},
				{RuleID: "GUARD-UNIV-01", Severity: sevWarning, Path: "weak.test.ts", Line: 51},
				{RuleID: "GUARD-TS-EXISTENCE-ONLY", Severity: sevAdvisory, Path: "weak.test.ts", Line: 9},
				{RuleID: "GUARD-TS-CATCH-ONLY", Severity: sevWarning, Path: "weak.test.ts", Line: 15},
				{RuleID: "GUARD-TS-FLOATING-ASSERT", Severity: sevWarning, Path: "weak.test.ts", Line: 23},
				{RuleID: "GUARD-TS-CONDITIONAL-ASSERT", Severity: sevWarning, Path: "weak.test.ts", Line: 28},
				{RuleID: "GUARD-TS-SELF-MOCK", Severity: sevWarning, Path: "weak.test.ts", Line: 33},
			}},
		{name: "TS: assertion libs", dir: "testdata/corpus/ts_assertion_libs",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "libs.test.ts", Line: 44}}},
		// F3 regression: chai/sinon-chai/chai-as-promised getters, testdouble
		// verify, Angular HttpTestingController verify and typed helpers all
		// count as verification.
		{name: "TS: assertion library shapes are recognized", dir: "testdata/corpus/ts_assertion_library_shapes_negative",
			analyzedTests: 13},
		// F3 compatibility: an imported ts-mockito `verify(...)` call is an assertion.
		{name: "TS: ts-mockito imported verify is credited", dir: "testdata/corpus/ts_mockito_verify_negative",
			analyzedTests: 1},
		// F3 precision: a locally shadowed `verify` is not a library assertion.
		{name: "TS: shadowed verify is not credited", dir: "testdata/corpus/ts_shadowed_verify_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "shadowed.test.ts", Line: 1}},
			analyzedTests: 1},
		// F1 regression: hooks, test.use/configure/setTimeout must not become tests.
		{name: "TS: Playwright hooks are not tests", dir: "testdata/corpus/ts_playwright_hooks_negative",
			analyzedTests: 3},
		// R4 semantic change: nested-function assertions are credited lexically,
		// with no execution-reachability guarantee. Named callbacks, references,
		// aliases and JSX use therefore produce no finding; a genuinely
		// unreferenced nested assertion is a documented recall gap, not a Blocker.
		{name: "TS: nested function assertions are credited", dir: "testdata/corpus/ts_named_callback_references_negative",
			analyzedTests: 5},
		// N3 regression: identifier/member/call titles, concatenated and template
		// titles, and mocha `specify` declare tests when a callback is supplied.
		// Discovering `specify` adds a fifth structurally similar declaration, so
		// the pre-existing GUARD-UNIV-02 advisory legitimately fires at L74.
		{name: "TS: dynamic titles and specify are declarations", dir: "testdata/corpus/ts_dynamic_titles_negative",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-02", Severity: sevAdvisory, Path: "decl.test.ts", Line: 74}},
			analyzedTests: 17},
		// N3 negative control: an unevaluated dynamic title still declares exactly
		// one analysable test, and a hollow one stays a zero-assertion blocker.
		{name: "TS: hollow dynamic-title test is analysed", dir: "testdata/corpus/ts_dynamic_hollow_title_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "hollow.test.ts", Line: 3}},
			analyzedTests: 1},
		// Naming guard: expression titles (identifier, concatenation, interpolated
		// template) are not judged by their unevaluated source text.
		{name: "TS: dynamic title expressions are not vague names", dir: "testdata/corpus/ts_dynamic_title_expression_negative",
			analyzedTests: 3},
		// Naming guard control: a real literal placeholder title still reports.
		{name: "TS: literal placeholder title is still reported", dir: "testdata/corpus/ts_literal_foo_title_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-NAME", Severity: sevAdvisory, Path: "literal.test.ts", Line: 3}},
			analyzedTests: 1},
		// R3-1/R4 regression: nested assertions are credited regardless of how the
		// function is mentioned; a nested function with no recognized assertion
		// still leaves the test hollow.
		{name: "TS: JSX attribute handler reference is credited", dir: "testdata/corpus/ts_jsx_attribute_handler_negative",
			analyzedTests: 3},
		{name: "TS: nested assertion credit boundary", dir: "testdata/corpus/ts_jsx_string_mention_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "stringmention.test.tsx", Line: 10}},
			analyzedTests: 2},
		// R4-1 regression: a component declared in the test and used only as a JSX
		// tag is credited; the Shell test keeps its genuine existence-only advisory.
		{name: "TS: JSX probe components are credited", dir: "testdata/corpus/ts_jsx_probe_component_negative",
			expected:      []expectedFinding{{RuleID: "GUARD-TS-EXISTENCE-ONLY", Severity: sevAdvisory, Path: "probe.test.tsx", Line: 19}},
			analyzedTests: 2},
		// R4-2 regression: a generic JSX element is valid and later tests survive.
		{name: "TS: generic JSX component syntax", dir: "testdata/corpus/ts_jsx_generic_component_negative",
			analyzedTests: 2},
		// R4-2 fail-closed counterpart: a genuinely malformed generic element errors.
		{name: "TS: malformed generic JSX is a parse failure", dir: "testdata/corpus/ts_jsx_generic_malformed_parse_failed",
			parseFailure: true},
		// R3-2 regression: an imported or required `fit` utility taking a callback
		// is a call, not a focused-test declaration.
		{name: "TS: imported fit utility is not a declaration", dir: "testdata/corpus/ts_imported_fit_utility_negative",
			analyzedTests: 2},
		{name: "TS: required fit utility is not a declaration", dir: "testdata/corpus/ts_required_fit_utility_negative",
			analyzedTests: 2},
		// Boundary retained: a genuine global focused `fit` declaration still
		// analyses as one test and reports the focus modifier.
		{name: "TS: global fit declaration still reports focus", dir: "testdata/corpus/ts_fit_focus_positive",
			expected:      []expectedFinding{{RuleID: "GUARD-UNIV-FOCUS", Severity: sevWarning, Path: "fitfocus.test.ts", Line: 3}},
			analyzedTests: 1},
		// F1 regression: local functions, object methods and calls named `test`
		// are not declarations.
		{name: "TS: non-declaration test calls are not tests", dir: "testdata/corpus/ts_non_declaration_calls_negative",
			analyzedTests: 3},
		// F1 regression: curried skipIf/runIf/each and modifiers still declare
		// exactly one real test each.
		{name: "TS: curried declaration modifiers", dir: "testdata/corpus/ts_curried_modifiers_negative",
			analyzedTests: 5},
		// F1 regression: a local helper named `fit` called with an array is not
		// a focused-test declaration.
		{name: "TS: local fit helper is not a declaration", dir: "testdata/corpus/ts_fit_helper_negative",
			analyzedTests: 1},
		// F2 regression: nested templates and operator-adjacent regex literals
		// keep every following test discoverable.
		{name: "TS: nested template and regex operators", dir: "testdata/corpus/ts_nested_template_negative",
			analyzedTests: 3},
		// F2 regression: an unterminated body fails closed instead of silently
		// reporting a truncated but complete scan.
		{name: "TS: unterminated body is a parse failure", dir: "testdata/corpus/ts_unterminated_body_parse_failed",
			parseFailure: true},
		{name: "TS: JSX text render only", dir: "testdata/corpus/ts_jsx_text_render_only",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "jsx_text.test.tsx", Line: 3}}},
		{name: "TS: JSX text asserted", dir: "testdata/corpus/ts_jsx_text_asserted"},
		{name: "NestJS: pass-through controller echo", dir: "testdata/corpus/nest_controller_echo_assigned",
			expected: []expectedFinding{{RuleID: "GUARD-NEST-01", Severity: sevWarning, Path: "orders.controller.spec.ts", Line: 1}}},
		{name: "NestJS: transformed controller result", dir: "testdata/corpus/nest_controller_echo_transformed"},
		{name: "NestJS: unmocked TypeORM module", dir: "testdata/corpus/nest_unmocked_db_typeorm",
			expected: []expectedFinding{
				{RuleID: "GUARD-NEST-02", Severity: sevWarning, Path: "orders.controller.spec.ts", Line: 7},
				{RuleID: "GUARD-UNIV-TAUTOLOGY", Severity: sevBlocker, Path: "orders.controller.spec.ts", Line: 12},
			}},
		{name: "NestJS: in-memory sqlite integration test", dir: "testdata/corpus/nest_inmemory_sqlite",
			expected: []expectedFinding{{RuleID: "GUARD-TS-EXISTENCE-ONLY", Severity: sevAdvisory, Path: "users.integration.spec.ts", Line: 5}}},
		{name: "Playwright: hard wait", dir: "testdata/corpus/playwright_hard_wait",
			expected: []expectedFinding{{RuleID: "GUARD-E2E-01", Severity: sevWarning, Path: "checkout.spec.ts", Line: 5}}},

		// --- Kotlin / Swift ---
		{name: "Kotlin: coroutine advance", dir: "testdata/corpus/kt_coroutine_advance_positive",
			expected: []expectedFinding{{RuleID: "GUARD-KT-02", Severity: sevWarning, Path: "CoroutineAdvanceTest.kt", Line: 13}}},
		// F6 regression: the canonical trailing-lambda form must fire exactly
		// like runTest() while the provenance negatives stay silent.
		{name: "Kotlin: coroutine advance with trailing lambda", dir: "testdata/corpus/kt_coroutine_advance_lambda_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-KT-02", Severity: sevWarning, Path: "CoroutineLambdaTest.kt", Line: 11},
				{RuleID: "GUARD-KT-02", Severity: sevWarning, Path: "CoroutineLambdaTest.kt", Line: 19},
				{RuleID: "GUARD-KT-02", Severity: sevWarning, Path: "CoroutineLambdaTest.kt", Line: 29},
			},
			analyzedTests: 7},
		{name: "Kotlin: coroutine advance negative", dir: "testdata/corpus/kt_coroutine_advance_negative"},
		{name: "Kotlin: mock echo", dir: "testdata/corpus/kt_mock_echo_positive",
			expected: []expectedFinding{{RuleID: "GUARD-KT-01", Severity: sevWarning, Path: "MockEchoTest.kt", Line: 10}}},
		{name: "Kotlin: mock echo negative", dir: "testdata/corpus/kt_mock_echo_negative"},
		{name: "Kotlin: adapter assertions recognized", dir: "testdata/corpus/kt_adapter_positive",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "KotlinAdapterTest.kt", Line: 44}}},
		{name: "Kotlin: zero assertion", dir: "testdata/corpus/kt_adapter_zero_positive",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "KotlinAdapterTest.kt", Line: 7}}},
		{name: "Swift: timeout", dir: "testdata/corpus/swift_timeout_positive",
			expected: []expectedFinding{{RuleID: "GUARD-SWIFT-01", Severity: sevAdvisory, Path: "TimeoutTests.swift", Line: 4}}},
		{name: "Swift: timeout negative", dir: "testdata/corpus/swift_timeout_negative"},
		{name: "Swift: adapter assertions recognized", dir: "testdata/corpus/swift_adapter_positive",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-SKIP", Severity: sevWarning, Path: "AdapterTests.swift", Line: 21}}},
		{name: "Swift: zero assertions", dir: "testdata/corpus/swift_adapter_zero_positive",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "AdapterTests.swift", Line: 5},
				{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "AdapterTests.swift", Line: 10},
			}},
		{name: "Swift: performance measure counts as verification", dir: "testdata/corpus/swift_performance_measure"},
		{name: "Swift: empty performance block", dir: "testdata/corpus/swift_performance_empty",
			expected: []expectedFinding{{RuleID: "GUARD-UNIV-ZERO", Severity: sevBlocker, Path: "PerformanceTests.swift", Line: 4}}},

		// --- Discovery scope ---
		{name: "Scope: linked-worktree and vendor exclusions", dir: "testdata/corpus/scope_exclusions",
			excludedFiles: []excludedExpectation{
				{Path: "wt", Reason: "linked_worktree"},
				{Path: "vendor", Reason: "default_directory"},
			},
			analyzedTests: 1},
	}

	metrics := make(map[ruleLangKey]*metricCount)
	getMetric := func(ruleID, lang string) *metricCount {
		k := ruleLangKey{RuleID: ruleID, Language: lang}
		if metrics[k] == nil {
			metrics[k] = &metricCount{}
		}
		return metrics[k]
	}

	catalogByID := make(map[string]docs.RuleDoc)
	for _, doc := range docs.AllRules() {
		catalogByID[doc.ID] = doc
		for _, lang := range doc.Languages {
			getMetric(doc.ID, lang)
		}
	}

	eng := engine.New()
	totalFP := 0
	totalFN := 0

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Special fixture setup: synthesize untracked .git files for linked-worktree test
			if tc.dir == "testdata/corpus/scope_exclusions" {
				wtGit := filepath.Join(tc.dir, "wt", ".git")
				if _, err := os.Stat(wtGit); os.IsNotExist(err) {
					_ = os.WriteFile(wtGit, []byte("gitdir: /x/.git/worktrees/wt\n"), 0644)
					defer os.Remove(wtGit)
				}
			}

			res, err := eng.Run(engine.Options{
				TargetPaths: []string{tc.dir},
			})
			if err != nil {
				t.Fatalf("unexpected error running engine on %s: %v", tc.dir, err)
			}

			// Analysis-status contract: an ordinary case is a clean, complete
			// analysis of at least one test; an intentionally empty scope must
			// stay visibly incomplete instead of looking like a pass.
			analysis := res.Report.Analysis
			if tc.parseFailure {
				if analysis.FailedFiles != 1 {
					t.Errorf("expected 1 failed file, got %d", analysis.FailedFiles)
				}
				if analysis.Status == "complete" {
					t.Errorf("a malformed file must not report a complete analysis")
				}
				hasParseFailed := false
				for _, d := range res.Report.Diagnostics {
					if d.Code == "parse_failed" {
						hasParseFailed = true
					}
				}
				if !hasParseFailed {
					t.Errorf("expected a parse_failed diagnostic, got %v", res.Report.Diagnostics)
				}
				if res.Report.ExitCode != 2 {
					t.Errorf("expected exit code 2 for a parse failure, got %d", res.Report.ExitCode)
				}
				if len(res.Report.Findings) != 0 {
					t.Errorf("expected no findings from a malformed file, got %d", len(res.Report.Findings))
				}
				return
			}
			if analysis.FailedFiles != 0 {
				t.Errorf("expected 0 failed files, got %d", analysis.FailedFiles)
			}
			if tc.emptyScope {
				if analysis.Status != "incomplete" || analysis.Reason != "empty_scope" {
					t.Errorf("expected incomplete/empty_scope, got %s/%s", analysis.Status, analysis.Reason)
				}
				if analysis.AnalyzedTests != 0 {
					t.Errorf("expected 0 analysed tests in an empty scope, got %d", analysis.AnalyzedTests)
				}
				if res.Report.ExitCode != 2 {
					t.Errorf("expected exit code 2 for an empty scope, got %d", res.Report.ExitCode)
				}
			} else {
				if analysis.Status != "complete" || analysis.Reason != "analyzed" {
					t.Errorf("expected complete/analyzed, got %s/%s", analysis.Status, analysis.Reason)
				}
				if analysis.AnalyzedTests == 0 {
					t.Errorf("expected at least one analysed test; a case that discovers nothing cannot prove a negative")
				}
				if tc.analyzedTests > 0 && analysis.AnalyzedTests != tc.analyzedTests {
					t.Errorf("expected %d analysed tests, got %d", tc.analyzedTests, analysis.AnalyzedTests)
				}
			}

			// Every ordinary case with expected blockers exits 1, everything
			// else exits 0. This keeps finding/exit consistency in one contract.
			wantExit := 0
			for _, exp := range tc.expected {
				if exp.Severity == sevBlocker {
					wantExit = 1
				}
			}
			if tc.emptyScope {
				wantExit = 2
			}
			if res.Report.ExitCode != wantExit {
				t.Errorf("expected exit code %d, got %d", wantExit, res.Report.ExitCode)
			}

			analyzed := make(map[string]bool, len(res.Report.Scope.Files))
			for _, f := range res.Report.Scope.Files {
				analyzed[relativeToCase(tc.dir, f)] = true
			}
			excluded := make(map[string]string, len(res.Report.Scope.Excluded))
			for _, e := range res.Report.Scope.Excluded {
				excluded[relativeToCase(tc.dir, e.Path)] = e.Reason
			}

			for _, negFile := range tc.negativeFiles {
				if !analyzed[negFile] {
					t.Errorf("expected negative file %s to be analysed", negFile)
				}
				for _, f := range res.Report.Findings {
					if relativeToCase(tc.dir, f.FilePath) == negFile {
						t.Errorf("expected negative file %s to have no findings, got %s", negFile, f.RuleID)
						getMetric(f.RuleID, detectLanguageFromExt(negFile)).FP++
						totalFP++
					}
				}
			}

			for _, ex := range tc.excludedFiles {
				reason, ok := excluded[ex.Path]
				if !ok {
					t.Errorf("expected %s to be excluded, but it was scanned", ex.Path)
					continue
				}
				if reason != ex.Reason {
					t.Errorf("expected %s to be excluded as %s, got %s", ex.Path, ex.Reason, reason)
				}
			}

			// Catalog parity: every emitted rule and severity must be documented.
			for _, f := range res.Report.Findings {
				doc, known := catalogByID[f.RuleID]
				if !known {
					t.Errorf("rule %s is emitted by the engine but missing from the catalog", f.RuleID)
					continue
				}
				if f.Severity != doc.Severity {
					t.Errorf("rule %s emits severity %s but the catalog documents %s", f.RuleID, f.Severity, doc.Severity)
				}
			}

			// Every expected finding must point at a file the scan actually analysed.
			// An intentionally empty scope has no analysed file by definition.
			if !tc.emptyScope {
				for _, exp := range tc.expected {
					if !analyzed[exp.Path] {
						t.Errorf("expected finding file %s was not analysed", exp.Path)
					}
				}
			}

			matchedExpected := make([]bool, len(tc.expected))
			for _, f := range res.Report.Findings {
				rel := relativeToCase(tc.dir, f.FilePath)
				lang := detectLanguageFromExt(rel)
				found := false
				for i, exp := range tc.expected {
					if matchedExpected[i] {
						continue
					}
					if exp.RuleID == f.RuleID && exp.Severity == f.Severity && exp.Path == rel && exp.Line == f.LineNumber {
						matchedExpected[i] = true
						found = true
						getMetric(f.RuleID, lang).TP++
						break
					}
				}
				if !found {
					t.Errorf("unexpected false positive finding in %s: %s %s at %s:%d", tc.dir, f.RuleID, f.Severity, rel, f.LineNumber)
					getMetric(f.RuleID, lang).FP++
					totalFP++
				}
			}

			for i, matched := range matchedExpected {
				if !matched {
					exp := tc.expected[i]
					t.Errorf("missing expected finding in %s: %s %s at %s:%d", tc.dir, exp.RuleID, exp.Severity, exp.Path, exp.Line)
					getMetric(exp.RuleID, detectLanguageFromExt(exp.Path)).FN++
					totalFN++
				}
			}
		})
	}

	// Catalog coverage contract: every rule must be exercised for every language
	// its documentation claims, otherwise the catalog is claiming unverified reach.
	for _, doc := range docs.AllRules() {
		totalTP := 0
		for _, lang := range doc.Languages {
			m := metrics[ruleLangKey{RuleID: doc.ID, Language: lang}]
			if m == nil || m.TP == 0 {
				t.Errorf("catalog rule %s claims language %s but the corpus has no true positive for it", doc.ID, lang)
				continue
			}
			totalTP += m.TP
		}
		if totalTP == 0 {
			t.Errorf("catalog rule %s has no true-positive corpus case", doc.ID)
		}
	}

	// Format Precision/Recall Table
	var sortedKeys []ruleLangKey
	for k := range metrics {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		if sortedKeys[i].RuleID != sortedKeys[j].RuleID {
			return sortedKeys[i].RuleID < sortedKeys[j].RuleID
		}
		return sortedKeys[i].Language < sortedKeys[j].Language
	})

	var table strings.Builder
	table.WriteString("\n" + strings.Repeat("=", 80) + "\n")
	table.WriteString(fmt.Sprintf("%-28s %-12s %4s %4s %4s %10s %10s\n",
		"RULE ID", "LANGUAGE", "TP", "FP", "FN", "PRECISION", "RECALL"))
	table.WriteString(strings.Repeat("-", 80) + "\n")

	for _, k := range sortedKeys {
		m := metrics[k]
		precStr := "n/a"
		if m.TP+m.FP > 0 {
			precStr = fmt.Sprintf("%.1f%%", float64(m.TP)/float64(m.TP+m.FP)*100)
		}
		recStr := "n/a"
		if m.TP+m.FN > 0 {
			recStr = fmt.Sprintf("%.1f%%", float64(m.TP)/float64(m.TP+m.FN)*100)
		}

		table.WriteString(fmt.Sprintf("%-28s %-12s %4d %4d %4d %10s %10s\n",
			k.RuleID, k.Language, m.TP, m.FP, m.FN, precStr, recStr))
	}
	table.WriteString(strings.Repeat("=", 80) + "\n")

	t.Log(table.String())

	if totalFP > 0 || totalFN > 0 {
		t.Fatalf("Corpus precision failure: %d False Positive(s), %d False Negative(s)", totalFP, totalFN)
	}
}

// TestCatalogSeveritiesMatchEmittedSeverities pins the severity ladder for rules
// whose documentation changed during the accuracy overhaul.
func TestCatalogSeveritiesMatchEmittedSeverities(t *testing.T) {
	expected := map[string]models.Severity{
		"GUARD-UNIV-01":   models.SeverityWarning,
		"GUARD-GO-01":     models.SeverityAdvisory,
		"GUARD-NEST-02":   models.SeverityWarning,
		"GUARD-KT-02":     models.SeverityWarning,
		"GUARD-SWIFT-01":  models.SeverityAdvisory,
		"GUARD-TS-01":     models.SeverityAdvisory,
		"GUARD-UNIV-ZERO": models.SeverityBlocker,
	}
	for ruleID, want := range expected {
		doc, found := docs.Explain(ruleID)
		if !found {
			t.Fatalf("expected rule %s in the catalog", ruleID)
		}
		if doc.Severity != want {
			t.Errorf("expected %s severity %s, got %s", ruleID, want, doc.Severity)
		}
	}
}

// TestCatalogOmitsRulesWithoutDetectors guards against phantom catalog entries.
func TestCatalogOmitsRulesWithoutDetectors(t *testing.T) {
	if _, found := docs.Explain("GUARD-REACT-01"); found {
		t.Errorf("GUARD-REACT-01 has no detector and must not be documented")
	}
}
