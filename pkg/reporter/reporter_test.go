package reporter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/reporter"
)

func TestTerminalReporterTotalsFirst(t *testing.T) {
	violations := []models.Violation{
		{
			RuleID:     "GUARD-UNIV-ZERO",
			Severity:   models.SeverityBlocker,
			FilePath:   "test/math_test.dart",
			LineNumber: 15,
			Column:     5,
			Message:    "Zero assertions found in test.",
			Suggestion: "Add expect() to verify outcomes.",
		},
		{
			RuleID:     "GUARD-DART-SEED-ECHO",
			Severity:   models.SeverityWarning,
			FilePath:   "test/bloc_test.dart",
			LineNumber: 42,
			Column:     9,
			Message:    "Seed echo assertion.",
			Suggestion: "Assert state deltas only.",
		},
	}

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	exitCode := r.Render(violations, 10, 0, 0)

	out := buf.String()
	if exitCode != 1 {
		t.Errorf("expected exit code 1 when Blocker exists, got %d", exitCode)
	}

	// Totals must precede findings
	if !strings.Contains(out, "TestGuard Audit") {
		t.Errorf("expected header line with 'TestGuard Audit' in output")
	}
	if !strings.Contains(out, "1 Blocker") || !strings.Contains(out, "1 Warning") {
		t.Errorf("expected total counts in output")
	}

	// Must include path:line:col format
	if !strings.Contains(out, "test/math_test.dart:15:5") {
		t.Errorf("expected path:line:col format in output")
	}
}

func TestAzureDevOpsLogger(t *testing.T) {
	violations := []models.Violation{
		{
			RuleID:     "GUARD-UNIV-ZERO",
			Severity:   models.SeverityBlocker,
			FilePath:   "test/math_test.dart",
			LineNumber: 15,
			Message:    "Zero assertions found in test.",
		},
	}

	var buf bytes.Buffer
	r := reporter.NewAzureReporter(&buf)
	r.Render(violations)

	out := buf.String()
	expected := "##vso[task.logissue type=error;sourcepath=test/math_test.dart;linenumber=15]Zero assertions found in test.\n"
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestTerminalReporterMonorepoBreakdown(t *testing.T) {
	violations := []models.Violation{
		{
			RuleID:     "GUARD-REACT-02",
			Severity:   models.SeverityBlocker,
			FilePath:   "datasource/ezrx/frontend/src/app.test.tsx",
			LineNumber: 10,
			Message:    "Timer sleep fragility.",
		},
		{
			RuleID:     "GUARD-GO-03",
			Severity:   models.SeverityAdvisory,
			FilePath:   "datasource/ezrx/order-engine/order_test.go",
			LineNumber: 42,
			Message:    "Duplicate table row.",
		},
		{
			RuleID:     "GUARD-DART-UNCOVERED-MUTATION",
			Severity:   models.SeverityBlocker,
			FilePath:   "datasource/ezrx/eZRx-Mobile/test/bloc_test.dart",
			LineNumber: 100,
			Message:    "Uncovered mutation.",
		},
	}

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	exitCode := r.RenderWithOptions(violations, 100, 0, 0, reporter.TerminalOptions{
		SummaryOnly: false,
		MaxDisplay:  10,
	})

	out := buf.String()
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(out, "MODULE BREAKDOWN:") {
		t.Errorf("expected 'MODULE BREAKDOWN:' section in multi-module report")
	}
	if !strings.Contains(out, "ezrx/frontend") || !strings.Contains(out, "ezrx/order-engine") || !strings.Contains(out, "ezrx/eZRx-Mobile") {
		t.Errorf("expected module names in breakdown table, got: %s", out)
	}
	if !strings.Contains(out, "TOP RULE VIOLATIONS:") {
		t.Errorf("expected 'TOP RULE VIOLATIONS:' section in report")
	}
}

func TestTerminalReporterSummaryOnly(t *testing.T) {
	violations := []models.Violation{
		{
			RuleID:     "GUARD-REACT-02",
			Severity:   models.SeverityBlocker,
			FilePath:   "datasource/ezrx/frontend/src/app.test.tsx",
			LineNumber: 10,
			Message:    "Timer sleep fragility.",
		},
	}

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	_ = r.RenderWithOptions(violations, 10, 0, 0, reporter.TerminalOptions{
		SummaryOnly: true,
		ReportPath:  ".testguard/custom-report.md",
	})

	out := buf.String()
	if !strings.Contains(out, "Summary mode: Full detailed violations written to .testguard/custom-report.md") {
		t.Errorf("expected summary mode message, got: %s", out)
	}
	if strings.Contains(out, "KEY FINDINGS") {
		t.Errorf("expected KEY FINDINGS to be omitted in SummaryOnly mode")
	}
}
