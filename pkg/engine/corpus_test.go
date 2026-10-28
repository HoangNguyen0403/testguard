package engine_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/models"
)

type expectedFinding struct {
	RuleID       string
	FileBasename string
}

func TestPrecisionCorpus(t *testing.T) {
	cases := []struct {
		name     string
		dir      string
		expected []expectedFinding
	}{
		// --- 1. No finding expected (former false positives) ---
		{
			name:     "Go: t.Run table test",
			dir:      "testdata/corpus/go_trun_table",
			expected: nil,
		},
		{
			name:     "Go: testify suite runner + asserting suite method",
			dir:      "testdata/corpus/go_testify_suite",
			expected: nil,
		},
		{
			name:     "Go: same-file asserting helper",
			dir:      "testdata/corpus/go_asserting_helper",
			expected: nil,
		},
		{
			name:     "Go: gomock EXPECT with NewController(t)",
			dir:      "testdata/corpus/go_gomock_expect",
			expected: nil,
		},
		{
			name:     "Go: test inside block comment /* */",
			dir:      "testdata/corpus/go_commented_test",
			expected: nil,
		},
		{
			name:     "Go: sqlmock in repository test",
			dir:      "testdata/corpus/go_sqlmock_repo",
			expected: nil,
		},
		{
			name:     "Go: probe test with t.Run loop",
			dir:      "testdata/corpus/go_probe_table",
			expected: nil,
		},
		{
			name:     "TS: helper expectCartEmpty()",
			dir:      "testdata/corpus/ts_asserting_helper",
			expected: nil,
		},
		{
			name:     "TS: setTimeout(r, 0) flush",
			dir:      "testdata/corpus/ts_sleep_zero",
			expected: nil,
		},
		{
			name:     "TS: mock latency in mockImplementation",
			dir:      "testdata/corpus/ts_mock_latency",
			expected: nil,
		},
		{
			name:     "TS: fake timers with setTimeout",
			dir:      "testdata/corpus/ts_fake_timers",
			expected: nil,
		},
		{
			name:     "Dart: asserting local helper",
			dir:      "testdata/corpus/dart_asserting_helper",
			expected: nil,
		},
		{
			name:     "Dart: two blocTests differing only in seed",
			dir:      "testdata/corpus/dart_bloc_diff_seed",
			expected: nil,
		},
		{
			name:     "Dart: two blocTests with setUp semicolon ending line",
			dir:      "testdata/corpus/dart_bloc_setup_semicolon",
			expected: nil,
		},
		{
			name:     "Scope: linked-worktree subdir and vendor/ containing zero-assert test",
			dir:      "testdata/corpus/scope_exclusions",
			expected: nil,
		},

		// --- 2. Exact findings expected (true positives) ---
		{
			name: "TS: test with empty body",
			dir:  "testdata/corpus/ts_empty_body",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "empty.test.ts"},
			},
		},
		{
			name: "TS: render-only test without expect",
			dir:  "testdata/corpus/ts_render_only",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "render.test.tsx"},
			},
		},
		{
			name: "TS: unit test with real timer sleep",
			dir:  "testdata/corpus/ts_real_sleep",
			expected: []expectedFinding{
				{RuleID: "GUARD-REACT-02", FileBasename: "sleep.test.ts"},
			},
		},
		{
			name: "Playwright: spec importing @playwright/test with waitForTimeout",
			dir:  "testdata/corpus/playwright_hard_wait",
			expected: []expectedFinding{
				{RuleID: "GUARD-E2E-01", FileBasename: "checkout.spec.ts"},
			},
		},
		{
			name: "Dart: Flutter integration_test with Future.delayed",
			dir:  "testdata/corpus/dart_e2e_future_delayed",
			expected: []expectedFinding{
				{RuleID: "GUARD-E2E-01", FileBasename: "app_test.dart"},
			},
		},
		{
			name: "Universal: expect(true).toBe(true) tautology",
			dir:  "testdata/corpus/tautology_expect_true",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-TAUTOLOGY", FileBasename: "tautology.test.ts"},
			},
		},
		{
			name: "Universal: it.skip committed skipped test",
			dir:  "testdata/corpus/skip_it_skip",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-SKIP", FileBasename: "skipped.test.ts"},
			},
		},
		{
			name: "Universal: 4 byte-identical TS tests -> 3x UNIV-01, 0x UNIV-02",
			dir:  "testdata/corpus/clones_four_identical",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-01", FileBasename: "clones.test.ts"},
				{RuleID: "GUARD-UNIV-01", FileBasename: "clones.test.ts"},
				{RuleID: "GUARD-UNIV-01", FileBasename: "clones.test.ts"},
			},
		},
		{
			name: "Go: test with no assertion at all",
			dir:  "testdata/corpus/go_zero_assert",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "no_assert_test.go"},
			},
		},
	}

	eng := engine.New()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := eng.Run(engine.Options{
				TargetPaths: []string{tc.dir},
			})
			if err != nil {
				t.Fatalf("unexpected error running engine on %s: %v", tc.dir, err)
			}

			var actual []expectedFinding
			for _, f := range res.Report.Findings {
				actual = append(actual, expectedFinding{
					RuleID:       f.RuleID,
					FileBasename: filepath.Base(f.FilePath),
				})
			}

			// Sort both multisets for deterministic comparison
			sortFindings(actual)
			expected := make([]expectedFinding, len(tc.expected))
			copy(expected, tc.expected)
			sortFindings(expected)

			if len(actual) != len(expected) {
				t.Fatalf("finding count mismatch for %s: got %d (%+v), want %d (%+v)",
					tc.name, len(actual), actual, len(expected), expected)
			}

			for i := range actual {
				if actual[i] != expected[i] {
					t.Fatalf("finding mismatch for %s at [%d]: got %+v, want %+v",
						tc.name, i, actual[i], expected[i])
				}
			}

			// Specifically assert policy status / blocker consistency
			for _, f := range res.Report.Findings {
				if f.RuleID == "GUARD-REACT-02" && f.Severity != models.SeverityWarning {
					t.Errorf("GUARD-REACT-02 expected severity Warning, got %s", f.Severity)
				}
			}
		})
	}
}

func sortFindings(fs []expectedFinding) {
	sort.Slice(fs, func(i, j int) bool {
		if fs[i].RuleID != fs[j].RuleID {
			return fs[i].RuleID < fs[j].RuleID
		}
		return fs[i].FileBasename < fs[j].FileBasename
	})
}
