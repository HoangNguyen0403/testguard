package engine_test

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/models"
)

type expectedFinding struct {
	RuleID       string
	FileBasename string
	Language     string
}

type corpusCase struct {
	name          string
	dir           string
	language      string
	expected      []expectedFinding
	negativeFiles []string
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
	case ".ts", ".tsx", ".js", ".jsx":
		return "TypeScript"
	case ".dart":
		return "Dart"
	default:
		return "Universal"
	}
}

func TestPrecisionCorpus(t *testing.T) {
	cases := []corpusCase{
		// --- 1. Negative cases (zero findings expected / former false positives) ---
		{
			name:     "Go: t.Run table test",
			dir:      "testdata/corpus/go_trun_table",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: testify suite runner + asserting suite method",
			dir:      "testdata/corpus/go_testify_suite",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: same-file asserting helper",
			dir:      "testdata/corpus/go_asserting_helper",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: gomock EXPECT with NewController(t)",
			dir:      "testdata/corpus/go_gomock_expect",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: test inside block comment /* */",
			dir:      "testdata/corpus/go_commented_test",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: sqlmock in repository test",
			dir:      "testdata/corpus/go_sqlmock_repo",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: probe test with t.Run loop",
			dir:      "testdata/corpus/go_probe_table",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: table test with mock map nested params (GO-03 negative)",
			dir:      "testdata/corpus/go_table_mock_map",
			language: "Go",
			expected: nil,
		},
		{
			name:     "TS: helper expectCartEmpty()",
			dir:      "testdata/corpus/ts_asserting_helper",
			language: "TypeScript",
			expected: nil,
		},
		{
			name:     "TS: setTimeout(r, 0) flush",
			dir:      "testdata/corpus/ts_sleep_zero",
			language: "TypeScript",
			expected: nil,
		},
		{
			name:     "TS: mock latency in mockImplementation",
			dir:      "testdata/corpus/ts_mock_latency",
			language: "TypeScript",
			expected: nil,
		},
		{
			name:     "TS: fake timers with setTimeout",
			dir:      "testdata/corpus/ts_fake_timers",
			language: "TypeScript",
			expected: nil,
		},
		{
			name:     "Dart: asserting local helper",
			dir:      "testdata/corpus/dart_asserting_helper",
			language: "Dart",
			expected: nil,
		},
		{
			name:     "Dart: two blocTests differing only in seed",
			dir:      "testdata/corpus/dart_bloc_diff_seed",
			language: "Dart",
			expected: nil,
		},
		{
			name:     "Dart: two blocTests with setUp semicolon ending line",
			dir:      "testdata/corpus/dart_bloc_setup_semicolon",
			language: "Dart",
			expected: nil,
		},
		{
			name:          "Scope: linked-worktree subdir and vendor/ containing zero-assert test",
			dir:           "testdata/corpus/scope_exclusions",
			language:      "Go",
			expected:      nil,
			negativeFiles: []string{"worktree_test.go", "vendor_test.go"},
		},
		{
			name:     "Go: gomock in closure with package-qualified constructor and setup func (R1)",
			dir:      "testdata/corpus/go_gomock_closure_provenance",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Dart: helper inside group with named params and multiline signature (R2)",
			dir:      "testdata/corpus/dart_group_helper_named",
			language: "Dart",
			expected: nil,
		},
		{
			name:     "Go: table with positional strings not dropped as name (R3 / F10 negative)",
			dir:      "testdata/corpus/go_table_positional_strings",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: gomock with cross-file helper returning mock (Task 9 Shape 1)",
			dir:      "testdata/corpus/go_gomock_cross_file_helper",
			language: "Go",
			expected: nil,
		},
		{
			name:     "Go: pure delegating runner (Task 9 Shape 2)",
			dir:      "testdata/corpus/go_pure_delegating_runner",
			language: "Go",
			expected: nil,
		},
		{
			name:     "TS: multiline expression arrow helper (F2 negative)",
			dir:      "testdata/corpus/ts_multiline_arrow",
			language: "TypeScript",
			expected: nil,
		},
		{
			name:     "TS: commented-out helper declaration (F1 positive)",
			dir:      "testdata/corpus/ts_comment_helper_zero",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "comment_helper.test.ts", Language: "TypeScript"},
			},
		},

		// --- 2. Positive cases (exact findings expected / true positives) ---
		{
			name:     "TS: test with empty body",
			dir:      "testdata/corpus/ts_empty_body",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "empty.test.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "TS: render-only test without expect",
			dir:      "testdata/corpus/ts_render_only",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "render.test.tsx", Language: "TypeScript"},
			},
		},
		{
			name:     "TS: hollow text-only helper",
			dir:      "testdata/corpus/ts_hollow_helper",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "hollow.test.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "TS: hollow shadowed helper",
			dir:      "testdata/corpus/ts_hollow_shadow",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "shadow.test.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "TS: unit test with real timer sleep",
			dir:      "testdata/corpus/ts_real_sleep",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-REACT-02", FileBasename: "sleep.test.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "Playwright: spec importing @playwright/test with waitForTimeout",
			dir:      "testdata/corpus/playwright_hard_wait",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-E2E-01", FileBasename: "checkout.spec.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "Dart: Flutter integration_test with Future.delayed",
			dir:      "testdata/corpus/dart_e2e_future_delayed",
			language: "Dart",
			expected: []expectedFinding{
				{RuleID: "GUARD-E2E-01", FileBasename: "app_test.dart", Language: "Dart"},
			},
		},
		{
			name:     "Dart: zero assertions true positive",
			dir:      "testdata/corpus/dart_zero_assert",
			language: "Dart",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "empty_test.dart", Language: "Dart"},
			},
		},
		{
			name:     "Dart: hollow text-only helper",
			dir:      "testdata/corpus/dart_hollow_helper",
			language: "Dart",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "hollow_test.dart", Language: "Dart"},
			},
		},
		{
			name:     "Universal: expect(true).toBe(true) tautology",
			dir:      "testdata/corpus/tautology_expect_true",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-TAUTOLOGY", FileBasename: "tautology.test.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "Universal: it.skip committed skipped test",
			dir:      "testdata/corpus/skip_it_skip",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-SKIP", FileBasename: "skipped.test.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "Universal: 4 byte-identical TS tests -> 3x UNIV-01, 0x UNIV-02",
			dir:      "testdata/corpus/clones_four_identical",
			language: "TypeScript",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-01", FileBasename: "clones.test.ts", Language: "TypeScript"},
				{RuleID: "GUARD-UNIV-01", FileBasename: "clones.test.ts", Language: "TypeScript"},
				{RuleID: "GUARD-UNIV-01", FileBasename: "clones.test.ts", Language: "TypeScript"},
			},
		},
		{
			name:     "Go: test with no assertion at all",
			dir:      "testdata/corpus/go_zero_assert",
			language: "Go",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "no_assert_test.go", Language: "Go"},
			},
		},
		{
			name:     "Go: hollow shadowed helper",
			dir:      "testdata/corpus/go_hollow_shadow",
			language: "Go",
			expected: []expectedFinding{
				{RuleID: "GUARD-UNIV-ZERO", FileBasename: "shadow_test.go", Language: "Go"},
			},
		},
		{
			name:     "Go: table test with duplicate rows (GO-03 positive)",
			dir:      "testdata/corpus/go_table_duplicate",
			language: "Go",
			expected: []expectedFinding{
				{RuleID: "GUARD-GO-03", FileBasename: "table_duplicate_test.go", Language: "Go"},
			},
		},
	}

	metrics := make(map[ruleLangKey]*metricCount)
	getMetric := func(ruleID, lang string) *metricCount {
		k := ruleLangKey{RuleID: ruleID, Language: lang}
		if metrics[k] == nil {
			metrics[k] = &metricCount{}
		}
		return metrics[k]
	}

	// Register all rules under evaluation
	allTrackedKeys := []ruleLangKey{
		{RuleID: "GUARD-GO-01", Language: "Go"},
		{RuleID: "GUARD-GO-03", Language: "Go"},
		{RuleID: "GUARD-UNIV-ZERO", Language: "Go"},
		{RuleID: "GUARD-UNIV-ZERO", Language: "TypeScript"},
		{RuleID: "GUARD-UNIV-ZERO", Language: "Dart"},
		{RuleID: "GUARD-REACT-02", Language: "TypeScript"},
		{RuleID: "GUARD-E2E-01", Language: "TypeScript"},
		{RuleID: "GUARD-E2E-01", Language: "Dart"},
		{RuleID: "GUARD-UNIV-TAUTOLOGY", Language: "TypeScript"},
		{RuleID: "GUARD-UNIV-SKIP", Language: "TypeScript"},
		{RuleID: "GUARD-UNIV-01", Language: "TypeScript"},
		{RuleID: "GUARD-DART-BLOC-DUPLICATE", Language: "Dart"},
	}
	for _, k := range allTrackedKeys {
		getMetric(k.RuleID, k.Language)
	}

	eng := engine.New()
	totalFP := 0
	totalFN := 0

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := eng.Run(engine.Options{
				TargetPaths: []string{tc.dir},
			})
			if err != nil {
				t.Fatalf("unexpected error running engine on %s: %v", tc.dir, err)
			}

			// Check negative files first
			for _, negFile := range tc.negativeFiles {
				for _, f := range res.Report.Findings {
					if filepath.Base(f.FilePath) == negFile {
						t.Errorf("expected negative file %s to have no findings, got %s", negFile, f.RuleID)
						getMetric(f.RuleID, detectLanguageFromExt(f.FilePath)).FP++
						totalFP++
					}
				}
			}

			// Match actual against expected
			matchedExpected := make([]bool, len(tc.expected))
			for _, f := range res.Report.Findings {
				base := filepath.Base(f.FilePath)
				lang := detectLanguageFromExt(f.FilePath)
				found := false
				for i, exp := range tc.expected {
					if !matchedExpected[i] && exp.RuleID == f.RuleID && exp.FileBasename == base {
						matchedExpected[i] = true
						found = true
						getMetric(f.RuleID, lang).TP++
						break
					}
				}
				if !found {
					t.Errorf("unexpected false positive finding in %s: %s on %s", tc.dir, f.RuleID, base)
					getMetric(f.RuleID, lang).FP++
					totalFP++
				}
			}

			// Check for missing expected findings (FN)
			for i, matched := range matchedExpected {
				if !matched {
					exp := tc.expected[i]
					t.Errorf("missing expected finding in %s: %s on %s", tc.dir, exp.RuleID, exp.FileBasename)
					getMetric(exp.RuleID, exp.Language).FN++
					totalFN++
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
