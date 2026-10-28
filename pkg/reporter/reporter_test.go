package reporter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/reporter"
)

type errWriter struct {
	failAfter int
	written   int
}

func (w *errWriter) Write(p []byte) (n int, err error) {
	if w.written >= w.failAfter {
		return 0, errors.New("simulated io error")
	}
	remaining := w.failAfter - w.written
	if len(p) > remaining {
		w.written += remaining
		return remaining, errors.New("simulated io error")
	}
	w.written += len(p)
	return len(p), nil
}

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

	report := models.NewScanReport()
	report.Analysis.Status = "complete"
	report.Analysis.Reason = "analyzed"
	report.Findings = violations
	report.Analysis.AnalyzedFiles = 10
	report.Policy.BlockerCount = 1
	report.Policy.Status = "blocker_found"
	report.ExitCode = 1

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	exitCode, err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	out := buf.String()
	if exitCode != 1 {
		t.Errorf("expected exit code 1 when Blocker exists, got %d", exitCode)
	}

	if !strings.Contains(out, "FAILED (complete/analyzed)") {
		t.Errorf("expected exact status FAILED (complete/analyzed) in output, got: %s", out)
	}
	if !strings.Contains(out, "1 Blocker") || !strings.Contains(out, "1 Warning") {
		t.Errorf("expected total counts in output")
	}
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

	report := models.NewScanReport()
	report.Analysis.Status = "complete"
	report.Analysis.Reason = "analyzed"
	report.Findings = violations
	report.Policy.BlockerCount = 1
	report.Policy.Status = "blocker_found"
	report.ExitCode = 1

	var buf bytes.Buffer
	r := reporter.NewAzureReporter(&buf)
	exitCode, err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	out := buf.String()
	if !strings.Contains(out, "TestGuard audit status: complete (analyzed)\n") {
		t.Errorf("expected audit status line in Azure output, got %q", out)
	}
	if !strings.Contains(out, "##vso[task.logissue type=error;sourcepath=test/math_test.dart;linenumber=15]Zero assertions found in test.\n") {
		t.Errorf("expected error logissue in Azure output, got %q", out)
	}
}

func TestAzureReporterCompleteNoFindingsNamesStatus(t *testing.T) {
	report := models.NewScanReport()
	report.Analysis.Status = "complete"
	report.Analysis.Reason = "analyzed"
	report.ExitCode = 0

	var buf bytes.Buffer
	exitCode, err := reporter.NewAzureReporter(&buf).Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	out := buf.String()
	if !strings.Contains(out, "TestGuard audit status: complete (analyzed)\n") {
		t.Errorf("expected audit status line for a clean complete scan, got %q", out)
	}
	if strings.Contains(out, "##vso[") {
		t.Errorf("clean complete scan must not emit annotations, got %q", out)
	}
}

func TestAzureReporterEscapesDynamicContent(t *testing.T) {
	evilPath := "test/a_test.go\n##vso[task.complete result=Succeeded;]"
	evilMessage := "50%\r\n##vso[task.complete result=Succeeded;]"

	report := models.NewScanReport()
	report.Analysis.Status = "incomplete"
	report.Analysis.Reason = "input_failure"
	report.ExitCode = 2
	report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("target_unreadable", evilPath, evilMessage))
	report.Findings = append(report.Findings, models.Violation{
		RuleID:     "GUARD-UNIV-ZERO",
		Severity:   models.SeverityBlocker,
		FilePath:   evilPath,
		LineNumber: 3,
		Message:    evilMessage,
	})

	var buf bytes.Buffer
	exitCode, err := reporter.NewAzureReporter(&buf).Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	out := buf.String()
	if strings.Contains(out, "\r") || strings.Contains(out, "\n##vso[task.complete") {
		t.Errorf("line breaks in dynamic data must not start new command lines, got %q", out)
	}

	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	// status summary, audit-error annotation, diagnostic annotation, finding annotation
	if len(lines) != 4 {
		t.Errorf("expected 4 output lines, got %d (%q)", len(lines), out)
	}
	commands := 0
	for _, line := range lines {
		if !strings.HasPrefix(line, "##vso[") {
			continue
		}
		commands++
		if !strings.HasPrefix(line, "##vso[task.logissue type=") {
			t.Errorf("only logissue annotations may be emitted, got line %q", line)
		}
	}
	if commands != 3 {
		t.Errorf("expected 3 logissue commands, got %d (%q)", commands, out)
	}

	if !strings.Contains(out, "sourcepath=test/a_test.go%0A##vso[task.complete result=Succeeded%3B%5D]") {
		t.Errorf("expected escaped sourcepath property, got %q", out)
	}
	if !strings.Contains(out, "50%25%0D%0A##vso[task.complete result=Succeeded;]") {
		t.Errorf("expected escaped message body, got %q", out)
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
			RuleID:     "GUARD-UNIV-TAUTOLOGY",
			Severity:   models.SeverityBlocker,
			FilePath:   "datasource/ezrx/eZRx-Mobile/test/bloc_test.dart",
			LineNumber: 100,
			Message:    "Uncovered mutation.",
		},
	}

	report := models.NewScanReport()
	report.Analysis.Status = "complete"
	report.Analysis.Reason = "analyzed"
	report.Findings = violations
	report.Analysis.AnalyzedFiles = 100
	report.Policy.BlockerCount = 2
	report.Policy.Status = "blocker_found"
	report.ExitCode = 1

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	exitCode, err := r.RenderWithOptions(report, reporter.TerminalOptions{
		SummaryOnly: false,
		MaxDisplay:  10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestTerminalReporterCompleteCleanAndOutputFailure(t *testing.T) {
	cases := []struct {
		name   string
		status string
		reason string
		exit   int
		want   string
	}{
		{"complete-clean", "complete", "analyzed", 0, "PASSED (complete/analyzed)"},
		{"output-failure", "error", "output_failure", 2, "ERROR (error/output_failure)"},
	}

	for _, tc := range cases {
		report := models.NewScanReport()
		report.Analysis.Status = tc.status
		report.Analysis.Reason = tc.reason
		report.ExitCode = tc.exit

		var buf bytes.Buffer
		exitCode, err := reporter.NewTerminalReporter(&buf).Render(report)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if exitCode != tc.exit {
			t.Errorf("%s: expected exit code %d, got %d", tc.name, tc.exit, exitCode)
		}
		if !strings.Contains(buf.String(), tc.want) {
			t.Errorf("%s: expected exact status %q, got %s", tc.name, tc.want, buf.String())
		}
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

	report := models.NewScanReport()
	report.Analysis.Status = "complete"
	report.Analysis.Reason = "analyzed"
	report.Findings = violations
	report.Analysis.AnalyzedFiles = 10
	report.Policy.BlockerCount = 1
	report.Policy.Status = "blocker_found"
	report.ExitCode = 1

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	exitCode, err := r.RenderWithOptions(report, reporter.TerminalOptions{
		SummaryOnly: true,
		ReportPath:  ".testguard/custom-report.md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	out := buf.String()
	if !strings.Contains(out, ".testguard/custom-report.md") {
		t.Errorf("expected summary mode to reference the report path, got: %s", out)
	}
	if strings.Contains(out, "KEY FINDINGS") {
		t.Errorf("expected KEY FINDINGS to be omitted in SummaryOnly mode")
	}
}

func TestTerminalReporterNotApplicable(t *testing.T) {
	report := models.NewScanReport()
	report.Analysis.Status = "not_applicable"
	report.Analysis.Reason = "empty_allowed"
	report.ExitCode = 3

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	exitCode, err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 3 {
		t.Errorf("expected exit code 3 for not_applicable, got %d", exitCode)
	}

	out := buf.String()
	if !strings.Contains(out, "NOT APPLICABLE (not_applicable/empty_allowed)") {
		t.Errorf("expected Status: NOT APPLICABLE (not_applicable/empty_allowed), got %s", out)
	}
}

func TestTerminalReporterIncompleteDiagnostics(t *testing.T) {
	report := models.NewScanReport()
	report.Analysis.Status = "incomplete"
	report.Analysis.Reason = "input_failure"
	report.ExitCode = 2
	report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("parse_failed", "pkg/bad_test.go", "syntax error at line 5"))

	var buf bytes.Buffer
	r := reporter.NewTerminalReporter(&buf)
	exitCode, err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	out := buf.String()
	if !strings.Contains(out, "INCOMPLETE (incomplete/input_failure)") {
		t.Errorf("expected INCOMPLETE (incomplete/input_failure) status, got %s", out)
	}
	if !strings.Contains(out, "parse_failed") || !strings.Contains(out, "pkg/bad_test.go") {
		t.Errorf("expected diagnostic in output, got %s", out)
	}
}

func TestAzureReporterIncompleteAndNotApplicable(t *testing.T) {
	// 1. Incomplete emits error annotations
	reportInc := models.NewScanReport()
	reportInc.Analysis.Status = "incomplete"
	reportInc.Analysis.Reason = "input_failure"
	reportInc.ExitCode = 2
	reportInc.Diagnostics = append(reportInc.Diagnostics, models.NewDiagnostic("read_failed", "test/a_test.go", "permission denied"))

	var bufInc bytes.Buffer
	rInc := reporter.NewAzureReporter(&bufInc)
	exitInc, err := rInc.Render(reportInc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitInc != 2 {
		t.Errorf("expected exit code 2, got %d", exitInc)
	}
	outInc := bufInc.String()
	if !strings.Contains(outInc, "TestGuard audit status: incomplete (input_failure)\n") {
		t.Errorf("expected status summary in Azure output, got %s", outInc)
	}
	if !strings.Contains(outInc, "##vso[task.logissue type=error]TestGuard audit incomplete (input_failure)\n") {
		t.Errorf("expected error logissue in Azure output, got %s", outInc)
	}
	if !strings.Contains(outInc, "##vso[task.logissue type=error;sourcepath=test/a_test.go]read_failed: permission denied\n") {
		t.Errorf("expected diagnostic logissue in Azure output, got %s", outInc)
	}

	// 2. Not applicable emits summary and NEVER task-success commands
	reportNA := models.NewScanReport()
	reportNA.Analysis.Status = "not_applicable"
	reportNA.Analysis.Reason = "empty_allowed"
	reportNA.ExitCode = 3

	var bufNA bytes.Buffer
	rNA := reporter.NewAzureReporter(&bufNA)
	exitNA, err := rNA.Render(reportNA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitNA != 3 {
		t.Errorf("expected exit code 3, got %d", exitNA)
	}
	outNA := bufNA.String()
	if !strings.Contains(outNA, "TestGuard audit status: not_applicable (empty_allowed)\n") {
		t.Errorf("expected status line in Azure output, got %s", outNA)
	}
	if !strings.Contains(outNA, "TestGuard audit not applicable: empty_allowed (exit 3)\n") {
		t.Errorf("expected summary message, got %s", outNA)
	}
	if strings.Contains(outNA, "##vso[task.complete result=Succeeded;]") {
		t.Errorf("Azure reporter must NOT emit task-success commands")
	}
}

func TestTerminalReporterSuppressesArtifactClaimOnExportFailure(t *testing.T) {
	report := models.NewScanReport()
	report.Analysis.Status = "error"
	report.Analysis.Reason = "output_failure"
	report.ExitCode = 2
	report.Findings = []models.Violation{
		{RuleID: "GUARD-UNIV-ZERO", Severity: models.SeverityBlocker, FilePath: "a_test.go", LineNumber: 3, Message: "Zero assertions."},
		{RuleID: "GUARD-UNIV-ZERO", Severity: models.SeverityBlocker, FilePath: "b_test.go", LineNumber: 4, Message: "Zero assertions."},
	}
	report.Policy.BlockerCount = 2
	report.Policy.Status = "blocker_found"

	const reportPath = ".testguard/report.md"

	summaryBuf := &bytes.Buffer{}
	exitCode, err := reporter.NewTerminalReporter(summaryBuf).RenderWithOptions(report, reporter.TerminalOptions{
		SummaryOnly: true,
		ReportPath:  reportPath,
	})
	if err != nil {
		t.Fatalf("unexpected summary error: %v", err)
	}
	if exitCode != 2 {
		t.Errorf("summary: expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(summaryBuf.String(), "ERROR (error/output_failure)") {
		t.Errorf("summary: expected error/output_failure verdict, got %q", summaryBuf.String())
	}
	if strings.Contains(summaryBuf.String(), "written to "+reportPath) {
		t.Errorf("summary: failed export must not be advertised as a written artifact, got %q", summaryBuf.String())
	}

	truncatedBuf := &bytes.Buffer{}
	exitCode, err = reporter.NewTerminalReporter(truncatedBuf).RenderWithOptions(report, reporter.TerminalOptions{
		MaxDisplay: 1,
		ReportPath: reportPath,
	})
	if err != nil {
		t.Fatalf("unexpected truncated error: %v", err)
	}
	if exitCode != 2 {
		t.Errorf("truncated: expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(truncatedBuf.String(), "ERROR (error/output_failure)") {
		t.Errorf("truncated: expected error/output_failure verdict, got %q", truncatedBuf.String())
	}
	if !strings.Contains(truncatedBuf.String(), "a_test.go") {
		t.Errorf("truncated: expected retained finding to stay visible, got %q", truncatedBuf.String())
	}
	if strings.Contains(truncatedBuf.String(), "written to "+reportPath) {
		t.Errorf("truncated: failed export must not be advertised as a written artifact, got %q", truncatedBuf.String())
	}
}

func TestWriteJSONReportSchemaV1(t *testing.T) {
	report := models.NewScanReport()
	report.Tool = models.NewToolMeta("0.1.5", "none", "unknown")
	report.Scope.RequestedPaths = []string{"sample_test.go"}
	report.Scope.Files = []string{}
	report.Analysis.DiscoveredFiles = 0
	report.Analysis.AnalyzedFiles = 0
	report.Analysis.AnalyzedTests = 0
	report.ExitCode = 2

	var buf bytes.Buffer
	if err := reporter.WriteJSONReport(&buf, report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := buf.String()
	if !strings.HasSuffix(raw, "\n") {
		t.Fatalf("JSON output must end with trailing newline")
	}
	lines := strings.Split(raw, "\n")
	if len(lines) != 2 || lines[1] != "" {
		t.Fatalf("expected exactly 1 JSON line plus newline, got %d lines", len(lines)-1)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if parsed["schema_version"] != float64(1) {
		t.Errorf("expected schema_version: 1, got %v", parsed["schema_version"])
	}

	tool, ok := parsed["tool"].(map[string]interface{})
	if !ok || tool["name"] != "testguard" || tool["version"] != "0.1.5" {
		t.Errorf("unexpected tool payload: %+v", parsed["tool"])
	}
	if tool["commit"] != nil || tool["built_at"] != nil {
		t.Errorf("expected null commit and built_at for none/unknown, got commit=%v, built_at=%v", tool["commit"], tool["built_at"])
	}

	scope, ok := parsed["scope"].(map[string]interface{})
	if !ok || scope["effective_mode"] != "full" || scope["config_applied"] != false {
		t.Errorf("unexpected scope payload: %+v", parsed["scope"])
	}

	// Verify empty arrays are [] and never null
	if files, ok := scope["files"].([]interface{}); !ok || files == nil {
		t.Errorf("expected scope.files to be empty array [], got %v", scope["files"])
	}
	if findings, ok := parsed["findings"].([]interface{}); !ok || findings == nil {
		t.Errorf("expected findings to be empty array [], got %v", parsed["findings"])
	}
	if diags, ok := parsed["diagnostics"].([]interface{}); !ok || diags == nil {
		t.Errorf("expected diagnostics to be empty array [], got %v", parsed["diagnostics"])
	}
}

func TestWriteMarkdownReport(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.md")

	report := models.NewScanReport()
	report.Analysis.Status = "incomplete"
	report.Analysis.Reason = "input_failure"
	report.Analysis.AnalyzedFiles = 1
	report.Analysis.AnalyzedTests = 2
	report.Analysis.FailedFiles = 1
	report.ExitCode = 2
	report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("parse_failed", "broken_test.go", "syntax error"))
	report.Findings = append(report.Findings, models.Violation{
		RuleID:     "GUARD-GO-01",
		Severity:   models.SeverityBlocker,
		FilePath:   "sample_test.go",
		LineNumber: 12,
		Message:    "Port mocking violation",
	})

	if err := reporter.WriteMarkdownReport(reportPath, report); err != nil {
		t.Fatalf("unexpected error writing markdown report: %v", err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed reading written report: %v", err)
	}
	str := string(content)

	if !strings.Contains(str, "**Status**: incomplete (input_failure)") {
		t.Errorf("expected status in markdown, got: %s", str)
	}
	if !strings.Contains(str, "parse_failed") || !strings.Contains(str, "broken_test.go") {
		t.Errorf("expected diagnostic table in markdown, got: %s", str)
	}
	if !strings.Contains(str, "GUARD-GO-01") || !strings.Contains(str, "🚨 Blocker") {
		t.Errorf("expected violation table in markdown, got: %s", str)
	}
}

func TestWriterErrorPropagation(t *testing.T) {
	report := models.NewScanReport()
	report.Analysis.Status = "complete"
	report.Analysis.Reason = "analyzed"
	report.ExitCode = 0

	// 1. Terminal write error
	termWriter := &errWriter{failAfter: 5}
	rTerm := reporter.NewTerminalReporter(termWriter)
	exitTerm, err := rTerm.Render(report)
	if err == nil {
		t.Errorf("expected error from failing terminal writer, got nil")
	}
	if exitTerm != 2 {
		t.Errorf("expected exit code 2 on terminal write failure, got %d", exitTerm)
	}

	// 2. Azure write error
	azWriter := &errWriter{failAfter: 5}
	rAz := reporter.NewAzureReporter(azWriter)
	exitAz, err := rAz.Render(report)
	if err == nil {
		t.Errorf("expected error from failing azure writer, got nil")
	}
	if exitAz != 2 {
		t.Errorf("expected exit code 2 on azure write failure, got %d", exitAz)
	}

	// 3. JSON write error
	jsonWriter := &errWriter{failAfter: 5}
	if err := reporter.WriteJSONReport(jsonWriter, report); err == nil {
		t.Errorf("expected error from failing JSON writer, got nil")
	}
}
