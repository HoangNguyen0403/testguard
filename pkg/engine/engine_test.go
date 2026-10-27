package engine_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/models"
)

func TestEngineAuditDartBlocFile(t *testing.T) {
	tmpDir := t.TempDir()
	pubspecPath := filepath.Join(tmpDir, "pubspec.yaml")
	_ = os.WriteFile(pubspecPath, []byte("name: sample_app\ndev_dependencies:\n  flutter_test:\n    sdk: flutter\n"), 0644)

	testDir := filepath.Join(tmpDir, "test")
	_ = os.MkdirAll(testDir, 0755)

	testFile := filepath.Join(testDir, "customer_code_bloc_test.dart")
	content := []byte(`
void main() {
  blocTest<CustomerCodeBloc, CustomerCodeState>(
    'Fetch_MissingLoadingState',
    seed: () => baseState.copyWith(userInfo: fakeUser),
    build: () => CustomerCodeBloc(),
    act: (bloc) => bloc.add(const CustomerCodeEvent.fetch()),
    expect: () => [
      isA<CustomerCodeState>().having((s) => s.userInfo, 'userInfo', fakeUser),
    ],
  );
}
`)
	_ = os.WriteFile(testFile, content, 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		ScanAll:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error running engine: %v", err)
	}

	for _, v := range res.Violations {
		fmt.Printf("Detected violation: %+v\n", v)
	}

	hasUncoveredMutation := false
	hasSeedEcho := false
	for _, v := range res.Violations {
		if v.RuleID == "GUARD-DART-UNCOVERED-MUTATION" && v.Severity == models.SeverityBlocker {
			hasUncoveredMutation = true
		}
		if v.RuleID == "GUARD-DART-SEED-ECHO" && v.Severity == models.SeverityWarning {
			hasSeedEcho = true
		}
	}

	if !hasUncoveredMutation {
		t.Errorf("expected to detect GUARD-DART-UNCOVERED-MUTATION")
	}
	if !hasSeedEcho {
		t.Errorf("expected to detect GUARD-DART-SEED-ECHO")
	}
}

func TestEngineProgressCallback(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	_ = os.MkdirAll(testDir, 0755)

	file1 := filepath.Join(testDir, "a_test.go")
	_ = os.WriteFile(file1, []byte("package test\nimport \"testing\"\nfunc TestA(t *testing.T) { t.Fatal(\"x\") }\n"), 0644)

	file2 := filepath.Join(testDir, "b_test.go")
	_ = os.WriteFile(file2, []byte("package test\nimport \"testing\"\nfunc TestB(t *testing.T) { t.Fatal(\"x\") }\n"), 0644)

	type progressEvent struct {
		phase       string
		current     int
		total       int
		currentFile string
	}

	var events []progressEvent
	progressFn := func(phase string, current, total int, currentFile string) {
		events = append(events, progressEvent{
			phase:       phase,
			current:     current,
			total:       total,
			currentFile: currentFile,
		})
	}

	eng := engine.New()
	_, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		ScanAll:     true,
		ProgressFn:  progressFn,
	})
	if err != nil {
		t.Fatalf("unexpected error running engine: %v", err)
	}

	if len(events) < 3 {
		t.Fatalf("expected at least 3 progress events, got %d", len(events))
	}

	if events[0].phase != "discovering" {
		t.Errorf("expected first event phase 'discovering', got %q", events[0].phase)
	}

	if events[1].phase != "discovered" || events[1].total != 2 {
		t.Errorf("expected second event 'discovered' with total 2, got %+v", events[1])
	}

	scanningCount := 0
	for _, e := range events[2:] {
		if e.phase == "scanning" {
			scanningCount++
		}
	}
	if scanningCount != 2 {
		t.Errorf("expected 2 scanning events, got %d", scanningCount)
	}
}

func TestEngineTargetNotFound(t *testing.T) {
	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{"/nonexistent-directory-testguard-path-12345"},
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}
	if res.Report == nil {
		t.Fatalf("expected non-nil ScanReport")
	}
	if res.Report.Analysis.Status != "incomplete" {
		t.Errorf("expected status 'incomplete', got %q", res.Report.Analysis.Status)
	}
	if res.Report.Analysis.Reason != "input_failure" {
		t.Errorf("expected reason 'input_failure', got %q", res.Report.Analysis.Reason)
	}
	if res.Report.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", res.Report.ExitCode)
	}
	if len(res.Report.Diagnostics) == 0 || res.Report.Diagnostics[0].Code != "target_unreadable" {
		t.Errorf("expected target_unreadable diagnostic, got %+v", res.Report.Diagnostics)
	}
}

func TestEnginePathDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	_ = os.MkdirAll(testDir, 0755)

	file1 := filepath.Join(testDir, "sample_test.go")
	_ = os.WriteFile(file1, []byte("package test\nimport \"testing\"\nfunc TestSample(t *testing.T) {\n  if 1 != 1 { t.Fail() }\n}\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir, testDir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ScannedFiles != 1 {
		t.Errorf("expected exactly 1 scanned file due to deduplication, got %d", res.ScannedFiles)
	}
	if res.Report.Analysis.DiscoveredFiles != 1 {
		t.Errorf("expected DiscoveredFiles 1, got %d", res.Report.Analysis.DiscoveredFiles)
	}
	if res.Report.Analysis.AnalyzedFiles != 1 {
		t.Errorf("expected AnalyzedFiles 1, got %d", res.Report.Analysis.AnalyzedFiles)
	}
	if len(res.Report.Scope.Files) != 1 {
		t.Errorf("expected scope.files length 1, got %d", len(res.Report.Scope.Files))
	}
}

func TestEngineStableScopeAcrossPathSpellings(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("unable to resolve cwd: %v", err)
	}
	// In environments where tmpDir and cwd reside on different drives (e.g. Windows C: vs D:),
	// filepath.Rel cannot bridge drives. Use a local temp dir inside cwd.
	tmpDir, err := os.MkdirTemp(cwd, "testguard_spellings_*")
	if err != nil {
		t.Fatalf("unable to create local temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	testFile := filepath.Join(tmpDir, "sample_test.go")
	_ = os.WriteFile(testFile, []byte("package test\nimport \"testing\"\nfunc TestSample(t *testing.T) {\n  if 1 != 1 { t.Fatal(\"fail\") }\n}\n"), 0644)

	rel, err := filepath.Rel(cwd, testFile)
	if err != nil {
		t.Fatalf("unable to express fixture relative to cwd: %v", err)
	}
	wantFile := filepath.ToSlash(rel)
	spellings := []string{testFile, rel, "." + string(filepath.Separator) + rel}
	eng := engine.New()
	for _, spelling := range spellings {
		res, err := eng.Run(engine.Options{TargetPaths: []string{spelling}})
		if err != nil {
			t.Fatalf("unexpected error for spelling %q: %v", spelling, err)
		}
		report := res.Report
		if len(report.Scope.Files) != 1 || report.Scope.Files[0] != wantFile {
			t.Errorf("spelling %q: expected scope.files [%q], got %+v", spelling, wantFile, report.Scope.Files)
		}
		if len(report.Scope.RequestedPaths) != 1 || report.Scope.RequestedPaths[0] != spelling {
			t.Errorf("spelling %q: requested_paths must keep the original value, got %+v", spelling, report.Scope.RequestedPaths)
		}
		if report.Analysis.Status != "complete" || report.Analysis.AnalyzedTests != 1 || report.ExitCode != 0 {
			t.Errorf("spelling %q: expected complete/1 test/exit 0, got status=%q tests=%d exit=%d",
				spelling, report.Analysis.Status, report.Analysis.AnalyzedTests, report.ExitCode)
		}
	}
}

func TestEngineCompleteScanNoBlockers(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "ok_test.go")
	_ = os.WriteFile(file1, []byte("package test\nimport \"testing\"\nfunc TestOk(t *testing.T) {\n  if 1 != 1 { t.Fatal(\"fail\") }\n}\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	report := res.Report
	if report.SchemaVersion != 1 {
		t.Errorf("expected schema_version 1, got %d", report.SchemaVersion)
	}
	if report.Analysis.Status != "complete" {
		t.Errorf("expected complete status, got %q", report.Analysis.Status)
	}
	if report.Analysis.Reason != "analyzed" {
		t.Errorf("expected analyzed reason, got %q", report.Analysis.Reason)
	}
	if report.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", report.ExitCode)
	}
	if report.Policy.Status != "no_blocker" || report.Policy.BlockerCount != 0 {
		t.Errorf("expected no_blocker policy, got %+v", report.Policy)
	}
	if report.Analysis.AnalyzedFiles != 1 || report.Analysis.AnalyzedTests != 1 {
		t.Errorf("expected 1 file and 1 test analyzed, got files=%d, tests=%d", report.Analysis.AnalyzedFiles, report.Analysis.AnalyzedTests)
	}
}

func TestEngineParseFailureMalformedGo(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "syntax_error_test.go")
	_ = os.WriteFile(badFile, []byte("package test\nfunc TestMalformed(t *testing.T) {\n  var x = \n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	report := res.Report
	if report.Analysis.Status != "incomplete" {
		t.Errorf("expected incomplete status, got %q", report.Analysis.Status)
	}
	if report.Analysis.Reason != "input_failure" {
		t.Errorf("expected input_failure reason, got %q", report.Analysis.Reason)
	}
	if report.Analysis.FailedFiles != 1 {
		t.Errorf("expected 1 failed file, got %d", report.Analysis.FailedFiles)
	}
	if report.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", report.ExitCode)
	}
	hasParseDiag := false
	for _, d := range report.Diagnostics {
		if d.Code == "parse_failed" {
			hasParseDiag = true
		}
	}
	if !hasParseDiag {
		t.Errorf("expected parse_failed diagnostic in %+v", report.Diagnostics)
	}
}

func TestEngineIncompletePlusBlockerPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	// File 1: Blocker (zero assertions)
	fileBlocker := filepath.Join(tmpDir, "zero_assert_test.go")
	_ = os.WriteFile(fileBlocker, []byte("package test\nimport \"testing\"\nfunc TestZero(t *testing.T) {\n  _ = 1 + 1\n}\n"), 0644)

	// File 2: Syntax error
	fileBroken := filepath.Join(tmpDir, "broken_syntax_test.go")
	_ = os.WriteFile(fileBroken, []byte("package test\nfunc Broken( {\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	report := res.Report
	// Incomplete MUST take precedence over blocker (exit 2, not 1)
	if report.Analysis.Status != "incomplete" {
		t.Errorf("expected status incomplete, got %q", report.Analysis.Status)
	}
	if report.Analysis.Reason != "input_failure" {
		t.Errorf("expected reason input_failure, got %q", report.Analysis.Reason)
	}
	if report.ExitCode != 2 {
		t.Errorf("expected exit code 2 (incomplete takes precedence over blocker), got %d", report.ExitCode)
	}

	// Exact retained finding verification (R12)
	if len(report.Findings) != 1 {
		t.Fatalf("expected exactly 1 retained finding, got %d (%+v)", len(report.Findings), report.Findings)
	}
	finding := report.Findings[0]
	if finding.RuleID != "GUARD-UNIV-ZERO" {
		t.Errorf("expected rule GUARD-UNIV-ZERO, got %s", finding.RuleID)
	}
	if finding.Severity != models.SeverityBlocker {
		t.Errorf("expected SeverityBlocker, got %s", finding.Severity)
	}
	if !strings.HasSuffix(finding.FilePath, "zero_assert_test.go") {
		t.Errorf("expected finding in zero_assert_test.go, got %s", finding.FilePath)
	}
	if report.Policy.BlockerCount != 1 {
		t.Errorf("expected blocker count exactly 1, got %d", report.Policy.BlockerCount)
	}
	if report.Policy.Status != "blocker_found" {
		t.Errorf("expected policy status blocker_found, got %s", report.Policy.Status)
	}
}

func TestEngineExplicitUnsupportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	unsupportedFile := filepath.Join(tmpDir, "notes.txt")
	_ = os.WriteFile(unsupportedFile, []byte("some notes"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{unsupportedFile},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	report := res.Report
	if report.Analysis.Status != "incomplete" {
		t.Errorf("expected status incomplete, got %q", report.Analysis.Status)
	}
	if report.Analysis.Reason != "unsupported_scope" {
		t.Errorf("expected reason unsupported_scope, got %q", report.Analysis.Reason)
	}
	if report.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", report.ExitCode)
	}
	if len(report.Scope.Unsupported) != 1 {
		t.Errorf("expected 1 unsupported file in scope, got %+v", report.Scope.Unsupported)
	}
}

func TestEngineExplicitUnsupportedWithAllowEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	unsupportedFile := filepath.Join(tmpDir, "readme.md")
	_ = os.WriteFile(unsupportedFile, []byte("# Readme"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{unsupportedFile},
		AllowEmpty:  true, // --allow-empty cannot excuse unsupported files!
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	report := res.Report
	if report.Analysis.Status != "incomplete" || report.Analysis.Reason != "unsupported_scope" || report.ExitCode != 2 {
		t.Errorf("allow-empty must not excuse unsupported file: got status=%q, reason=%q, exit=%d",
			report.Analysis.Status, report.Analysis.Reason, report.ExitCode)
	}
}

func TestEngineEmptyScopeDisallowed(t *testing.T) {
	tmpDir := t.TempDir()

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		AllowEmpty:  false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	report := res.Report
	if report.Analysis.Status != "incomplete" {
		t.Errorf("expected status incomplete, got %q", report.Analysis.Status)
	}
	if report.Analysis.Reason != "empty_scope" {
		t.Errorf("expected reason empty_scope, got %q", report.Analysis.Reason)
	}
	if report.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", report.ExitCode)
	}
	hasEmptyDiag := false
	for _, d := range report.Diagnostics {
		if d.Code == "empty_scope" {
			hasEmptyDiag = true
		}
	}
	if !hasEmptyDiag {
		t.Errorf("expected empty_scope diagnostic, got %+v", report.Diagnostics)
	}

	// Verify empty array [] serialization (R2)
	data, _ := json.Marshal(report)
	var raw map[string]interface{}
	_ = json.Unmarshal(data, &raw)
	scope := raw["scope"].(map[string]interface{})
	if files, ok := scope["files"].([]interface{}); !ok || files == nil {
		t.Errorf("expected scope.files to be empty array [], got %v", scope["files"])
	}
}

func TestEngineEmptyScopeAllowed(t *testing.T) {
	tmpDir := t.TempDir()

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		AllowEmpty:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	report := res.Report
	if report.Analysis.Status != "not_applicable" {
		t.Errorf("expected status not_applicable, got %q", report.Analysis.Status)
	}
	if report.Analysis.Reason != "empty_allowed" {
		t.Errorf("expected reason empty_allowed, got %q", report.Analysis.Reason)
	}
	if report.ExitCode != 3 {
		t.Errorf("expected exit code 3 for not_applicable, got %d", report.ExitCode)
	}
	if len(report.Findings) != 0 {
		t.Errorf("expected empty findings [] for not_applicable, got %+v", report.Findings)
	}
	if len(report.Scope.Files) != 0 {
		t.Errorf("expected empty scope.files, got %+v", report.Scope.Files)
	}
}

func TestEngineDeterministicFaultInjection(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "fault_test.go")
	_ = os.WriteFile(testFile, []byte("package test\nfunc TestX(t *testing.T){}\n"), 0644)

	eng := engine.New()

	// 1. Injected Traversal Failure
	resTrav, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		FailTraversal: func(path string) error {
			return errors.New("simulated traversal denial")
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resTrav.Report.Analysis.Status != "incomplete" || resTrav.Report.Analysis.Reason != "input_failure" || resTrav.Report.ExitCode != 2 {
		t.Errorf("expected incomplete/input_failure/exit 2 on traversal fault, got %+v", resTrav.Report)
	}

	// 2. Injected Read Failure
	resRead, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		FailRead: func(path string) error {
			return errors.New("simulated read denial")
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resRead.Report.Analysis.Status != "incomplete" || resRead.Report.Analysis.Reason != "input_failure" || resRead.Report.ExitCode != 2 {
		t.Errorf("expected incomplete/input_failure/exit 2 on read fault, got %+v", resRead.Report)
	}
	if resRead.Report.Analysis.FailedFiles != 1 {
		t.Errorf("expected 1 failed file on read fault, got %d", resRead.Report.Analysis.FailedFiles)
	}
}

func TestEngineDiffModeRejected(t *testing.T) {
	eng := engine.New()
	res, err := eng.Run(engine.Options{
		DiffOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Analysis.Status != "error" || res.Report.Analysis.Reason != "invalid_usage" || res.Report.ExitCode != 2 {
		t.Errorf("expected error/invalid_usage/exit 2 for diff mode, got %+v", res.Report)
	}
}

func TestEngineGoASTFakeTestControls(t *testing.T) {
	tmpDir := t.TempDir()

	// File with only raw string containing test declaration, plus comments and helper function
	fixtureFile := filepath.Join(tmpDir, "fixture_test.go")
	fixtureContent := []byte(`package test
import "testing"

const fixture = ` + "`" + `
func TestFakeInString(t *testing.T) {
    t.Fatal("this is inside a string literal, not a real test declaration")
}
` + "`" + `

// func TestCommentedOut(t *testing.T) { t.Fail() }

func HelperFunction() string {
    return "helper"
}
`)
	_ = os.WriteFile(fixtureFile, fixtureContent, 0644)

	eng := engine.New()

	// 1. Without allow-empty: must report 0 analyzed tests and exit 2 empty_scope
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{fixtureFile},
		AllowEmpty:  false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Analysis.AnalyzedTests != 0 {
		t.Errorf("expected 0 analyzed tests for raw-string/comment fake tests, got %d", res.Report.Analysis.AnalyzedTests)
	}
	if res.Report.Analysis.Status != "incomplete" || res.Report.Analysis.Reason != "empty_scope" || res.Report.ExitCode != 2 {
		t.Errorf("expected incomplete/empty_scope/exit 2, got status=%q, reason=%q, exit=%d",
			res.Report.Analysis.Status, res.Report.Analysis.Reason, res.Report.ExitCode)
	}

	// 2. With allow-empty: must report 0 analyzed tests and exit 3 not_applicable
	resAllowed, err := eng.Run(engine.Options{
		TargetPaths: []string{fixtureFile},
		AllowEmpty:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resAllowed.Report.Analysis.AnalyzedTests != 0 {
		t.Errorf("expected 0 analyzed tests, got %d", resAllowed.Report.Analysis.AnalyzedTests)
	}
	if resAllowed.Report.Analysis.Status != "not_applicable" || resAllowed.Report.Analysis.Reason != "empty_allowed" || resAllowed.Report.ExitCode != 3 {
		t.Errorf("expected not_applicable/empty_allowed/exit 3, got status=%q, reason=%q, exit=%d",
			resAllowed.Report.Analysis.Status, resAllowed.Report.Analysis.Reason, resAllowed.Report.ExitCode)
	}

	// 3. Positive control: file with real executable Test declaration
	realFile := filepath.Join(tmpDir, "real_test.go")
	realContent := []byte(`package test
import "testing"
func TestRealExecutable(t *testing.T) {
    if 1 != 1 { t.Fatal("fail") }
}
`)
	_ = os.WriteFile(realFile, realContent, 0644)
	resReal, err := eng.Run(engine.Options{
		TargetPaths: []string{realFile},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resReal.Report.Analysis.AnalyzedTests != 1 {
		t.Errorf("expected 1 analyzed test for real executable function, got %d", resReal.Report.Analysis.AnalyzedTests)
	}
	if resReal.Report.Analysis.Status != "complete" || resReal.Report.ExitCode != 0 {
		t.Errorf("expected complete/exit 0 for real test, got status=%q, exit=%d",
			resReal.Report.Analysis.Status, resReal.Report.ExitCode)
	}
}

func TestEngineDartMalformedCallbackError(t *testing.T) {
	tmpDir := t.TempDir()
	brokenFile := filepath.Join(tmpDir, "broken_callback_test.dart")
	brokenContent := []byte(`
void main() {
  test('broken callback test', () {
    expect(1, 1);
    // unmatched callback brace
`)
	_ = os.WriteFile(brokenFile, brokenContent, 0644)

	eng := engine.New()

	// 1. Broken file alone must produce incomplete/input_failure/exit 2
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{brokenFile},
		AllowEmpty:  true, // allow-empty cannot excuse parser failure
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Analysis.Status != "incomplete" || res.Report.Analysis.Reason != "input_failure" || res.Report.ExitCode != 2 {
		t.Errorf("expected incomplete/input_failure/exit 2 for malformed Dart callback, got status=%q, reason=%q, exit=%d",
			res.Report.Analysis.Status, res.Report.Analysis.Reason, res.Report.ExitCode)
	}
	if res.Report.Analysis.FailedFiles != 1 {
		t.Errorf("expected 1 failed file, got %d", res.Report.Analysis.FailedFiles)
	}

	// 2. Broken file mixed with valid file with finding: finding must be retained
	goodFile := filepath.Join(tmpDir, "good_test.dart")
	goodContent := []byte(`
void main() {
  test('zero assert test', () {
    int x = 1 + 1;
  });
}
`)
	_ = os.WriteFile(goodFile, goodContent, 0644)
	resMixed, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resMixed.Report.Analysis.Status != "incomplete" || resMixed.Report.Analysis.Reason != "input_failure" || resMixed.Report.ExitCode != 2 {
		t.Errorf("expected incomplete/input_failure/exit 2 for mixed scope, got status=%q, exit=%d",
			resMixed.Report.Analysis.Status, resMixed.Report.ExitCode)
	}
	if resMixed.Report.Analysis.FailedFiles != 1 {
		t.Errorf("expected 1 failed file, got %d", resMixed.Report.Analysis.FailedFiles)
	}
	if resMixed.Report.Analysis.AnalyzedFiles != 1 {
		t.Errorf("expected only the good file to be analyzed, got %d", resMixed.Report.Analysis.AnalyzedFiles)
	}
	if resMixed.Report.Analysis.AnalyzedTests != 1 {
		t.Errorf("expected 1 located test declaration from good_test.dart, got %d", resMixed.Report.Analysis.AnalyzedTests)
	}
	if len(resMixed.Report.Findings) != 1 {
		t.Fatalf("expected exactly 1 retained finding from good_test.dart, got %d (%+v)",
			len(resMixed.Report.Findings), resMixed.Report.Findings)
	}
	finding := resMixed.Report.Findings[0]
	if finding.RuleID != "GUARD-UNIV-ZERO" || finding.Severity != models.SeverityBlocker {
		t.Errorf("expected retained GUARD-UNIV-ZERO blocker, got rule=%s severity=%s", finding.RuleID, finding.Severity)
	}
	if !strings.HasSuffix(finding.FilePath, "good_test.dart") {
		t.Errorf("expected finding located in good_test.dart, got %s", finding.FilePath)
	}
	if resMixed.Report.Policy.BlockerCount != 1 || resMixed.Report.Policy.Status != "blocker_found" {
		t.Errorf("expected policy blocker_count=1/blocker_found, got %+v", resMixed.Report.Policy)
	}
}

func TestEngineGoQuotedVsExecutableRules(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Quoted controls: bad patterns exist ONLY in strings and comments
	quotedDir := filepath.Join(tmpDir, "quoted")
	_ = os.MkdirAll(filepath.Join(quotedDir, "domain"), 0755)
	_ = os.MkdirAll(filepath.Join(quotedDir, "service"), 0755)

	domainQuotedFile := filepath.Join(quotedDir, "domain", "user_test.go")
	domainQuotedContent := []byte(`package domain_test
import "testing"

const mockSnippet = "ctrl := gomock.NewController(t); mockRepo := mocks.NewMockUserRepository(ctrl);"
// Comment mentioning gomock.NewController(t)
func TestDomainUser(t *testing.T) {
    const code = "mocks.NewMockOrder(ctrl)"
    if code == "" {
        t.Fatal("empty")
    }
}
`)
	_ = os.WriteFile(domainQuotedFile, domainQuotedContent, 0644)

	serviceQuotedFile := filepath.Join(quotedDir, "service", "order_test.go")
	serviceQuotedContent := []byte(`package service_test
import "testing"

const sqlmockCode = "db, mock, err := sqlmock.New();"
// gorm.Open(postgres.Open(dsn), &gorm.Config{})
func TestServiceOrder(t *testing.T) {
    msg := "pgxmock.NewPool() should not be flagged in string"
    tests := []struct{ in int; want int }{
        {in: 0, want: 0},
        {in: 1, want: 10},
    }
    if msg == "" || len(tests) == 0 {
        t.Fatal("invalid")
    }
}
`)
	_ = os.WriteFile(serviceQuotedFile, serviceQuotedContent, 0644)

	eng := engine.New()
	resQuoted, err := eng.Run(engine.Options{
		TargetPaths: []string{quotedDir},
	})
	if err != nil {
		t.Fatalf("unexpected error scanning quoted files: %v", err)
	}

	for _, f := range resQuoted.Report.Findings {
		if f.RuleID == "GUARD-GO-01" || f.RuleID == "GUARD-GO-02" || f.RuleID == "GUARD-GO-03" {
			t.Errorf("unexpected violation %s in quoted test: %+v", f.RuleID, f)
		}
	}

	// 2. Executable controls: real executable calls trigger correct rules with exact lines
	execDir := filepath.Join(tmpDir, "executable")
	_ = os.MkdirAll(filepath.Join(execDir, "domain"), 0755)
	_ = os.MkdirAll(filepath.Join(execDir, "service"), 0755)

	domainExecFile := filepath.Join(execDir, "domain", "order_test.go")
	domainExecContent := []byte(`package domain_test
import (
    "testing"
    "github.com/golang/mock/gomock"
)

func TestExecutableDomainMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    _ = ctrl
    t.Fatal("fail")
}
`)
	_ = os.WriteFile(domainExecFile, domainExecContent, 0644)

	serviceExecFile := filepath.Join(execDir, "service", "payment_test.go")
	serviceExecContent := []byte(`package service_test
import (
    "testing"
    "github.com/DATA-DOG/go-sqlmock"
)

func TestExecutableServiceSqlmock(t *testing.T) {
    db, mock, err := sqlmock.New()
    _ = db
    _ = mock
    _ = err

    tests := []struct{ in int; want int }{
        {in: 100, want: 10},
        {in: 100, want: 10},
    }
    _ = tests
    t.Fatal("fail")
}
`)
	_ = os.WriteFile(serviceExecFile, serviceExecContent, 0644)

	resExec, err := eng.Run(engine.Options{
		TargetPaths: []string{execDir},
	})
	if err != nil {
		t.Fatalf("unexpected error scanning executable files: %v", err)
	}

	foundRules := make(map[string]bool)
	for _, f := range resExec.Report.Findings {
		foundRules[f.RuleID] = true
		switch f.RuleID {
		case "GUARD-GO-01":
			if f.LineNumber != 8 {
				t.Errorf("expected GUARD-GO-01 at line 8, got %d", f.LineNumber)
			}
		case "GUARD-GO-02":
			if f.LineNumber != 8 {
				t.Errorf("expected GUARD-GO-02 at line 8, got %d", f.LineNumber)
			}
		case "GUARD-GO-03":
			if f.LineNumber != 15 {
				t.Errorf("expected GUARD-GO-03 at line 15, got %d", f.LineNumber)
			}
		}
	}

	if !foundRules["GUARD-GO-01"] {
		t.Errorf("expected GUARD-GO-01 to be detected in executable domain test")
	}
	if !foundRules["GUARD-GO-02"] {
		t.Errorf("expected GUARD-GO-02 to be detected in executable service test")
	}
	if !foundRules["GUARD-GO-03"] {
		t.Errorf("expected GUARD-GO-03 to be detected in executable table test")
	}

	// 3. Scan pkg/rules/golang/rules_test.go directly: must produce ZERO GUARD-GO-02 or GUARD-GO-03 false positives
	resRulesTest, err := eng.Run(engine.Options{
		TargetPaths: []string{"pkg/rules/golang/rules_test.go"},
	})
	if err != nil {
		t.Fatalf("unexpected error scanning rules_test.go: %v", err)
	}
	for _, f := range resRulesTest.Report.Findings {
		if f.RuleID == "GUARD-GO-02" || f.RuleID == "GUARD-GO-03" {
			t.Errorf("false positive: rules_test.go triggered %s: %+v", f.RuleID, f)
		}
	}
}

func TestEngineGoTestMainAndDAOExemption(t *testing.T) {
	tmpDir := t.TempDir()
	eng := engine.New()

	// 4. TestMain-only file: excluded from test units, leaving 0 analyzed tests and empty_scope (R4)
	testMainFile := filepath.Join(tmpDir, "main_only_test.go")
	_ = os.WriteFile(testMainFile, []byte(`package sample_test
import "testing"
func TestMain(m *testing.M) {
    m.Run()
}
`), 0644)
	resTestMain, err := eng.Run(engine.Options{
		TargetPaths: []string{testMainFile},
		AllowEmpty:  false,
	})
	if err != nil {
		t.Fatalf("unexpected error scanning TestMain file: %v", err)
	}
	if resTestMain.Report.Analysis.AnalyzedTests != 0 {
		t.Errorf("expected 0 analyzed tests for TestMain-only file, got %d", resTestMain.Report.Analysis.AnalyzedTests)
	}
	if resTestMain.Report.Analysis.Status != "incomplete" || resTestMain.Report.Analysis.Reason != "empty_scope" || resTestMain.Report.ExitCode != 2 {
		t.Errorf("expected incomplete/empty_scope/exit 2 for TestMain-only file, got status=%q, reason=%q, exit=%d",
			resTestMain.Report.Analysis.Status, resTestMain.Report.Analysis.Reason, resTestMain.Report.ExitCode)
	}

	// 5. DAO root-relative directory exemption (R3)
	daoDir := filepath.Join(tmpDir, "dao")
	_ = os.MkdirAll(daoDir, 0755)
	daoFile := filepath.Join(daoDir, "order_dao_test.go")
	_ = os.WriteFile(daoFile, []byte(`package dao_test
import (
    "testing"
    "github.com/DATA-DOG/go-sqlmock"
)
func TestOrderDAO(t *testing.T) {
    db, mock, err := sqlmock.New()
    _ = db
    _ = mock
    _ = err
    t.Fatal("fail")
}
`), 0644)
	resDAO, err := eng.Run(engine.Options{
		TargetPaths: []string{daoFile},
	})
	if err != nil {
		t.Fatalf("unexpected error scanning DAO file: %v", err)
	}
	for _, f := range resDAO.Report.Findings {
		if f.RuleID == "GUARD-GO-02" {
			t.Errorf("expected DAO file %s to be exempted from GUARD-GO-02, but violation was emitted", daoFile)
		}
	}
}

func TestEngineGoEntryPointProvenanceCounts(t *testing.T) {
	tmpDir := t.TempDir()
	eng := engine.New()

	emptyDir := filepath.Join(tmpDir, "emptyish")
	_ = os.MkdirAll(emptyDir, 0755)

	// Grouped parameters, helpers, and compile-only examples are not tests.
	_ = os.WriteFile(filepath.Join(emptyDir, "mixed_test.go"), []byte(`package sample

import "testing"

func TestGrouped(a, b *testing.T) {
	a.Fatal("fail")
}

func TestHelper() {}

func ExampleCompileOnly() {
	// no output marker: never executed by the go test runner
}
`), 0644)

	// A *testing.T signature without a testing import is not a valid entry point.
	_ = os.WriteFile(filepath.Join(emptyDir, "noimport_test.go"), []byte(`package sample

func TestNoProvenance(t *testing.T) {
	t.Fatal("fail")
}
`), 0644)

	resEmpty, err := eng.Run(engine.Options{
		TargetPaths: []string{emptyDir},
		AllowEmpty:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error scanning non-entry-point file: %v", err)
	}
	if resEmpty.Report.Analysis.AnalyzedTests != 0 {
		t.Errorf("expected 0 analyzed tests for non-entry-point declarations, got %d", resEmpty.Report.Analysis.AnalyzedTests)
	}
	if resEmpty.Report.Analysis.Status != "not_applicable" || resEmpty.Report.ExitCode != 3 {
		t.Errorf("expected not_applicable/exit 3, got status=%q, exit=%d", resEmpty.Report.Analysis.Status, resEmpty.Report.ExitCode)
	}

	validDir := filepath.Join(tmpDir, "valid")
	_ = os.MkdirAll(validDir, 0755)

	// Dot-imported testing remains recognizable.
	_ = os.WriteFile(filepath.Join(validDir, "dot_test.go"), []byte(`package sample

import . "testing"

func TestDotImported(t *T) {
	t.Fatal("fail")
}
`), 0644)

	// Examples with an output marker are executed.
	_ = os.WriteFile(filepath.Join(validDir, "example_test.go"), []byte(`package sample

import "fmt"

func ExampleRunnable() {
	fmt.Println("x")
	// Output: x
}
`), 0644)

	resValid, err := eng.Run(engine.Options{
		TargetPaths: []string{validDir},
	})
	if err != nil {
		t.Fatalf("unexpected error scanning valid entry points: %v", err)
	}
	if resValid.Report.Analysis.AnalyzedTests != 2 {
		t.Errorf("expected 2 analyzed tests, got %d", resValid.Report.Analysis.AnalyzedTests)
	}
	if resValid.Report.Analysis.Status != "complete" || resValid.Report.ExitCode != 0 {
		t.Errorf("expected complete/exit 0, got status=%q, exit=%d", resValid.Report.Analysis.Status, resValid.Report.ExitCode)
	}
}

func TestEngineGoIdenticalBodyCloneAcrossNames(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "service_test.go")
	_ = os.WriteFile(file, []byte(`package service

import "testing"

func TestAlpha(t *testing.T) {
	x := 42
	if x != 42 {
		t.Fatal("fail")
	}
}

func TestBeta(t *testing.T) {
	x := 42
	if x != 42 {
		t.Fatal("fail")
	}
}
`), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{file},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exactClones := 0
	for _, finding := range res.Report.Findings {
		if finding.RuleID == "GUARD-UNIV-01" && strings.Contains(finding.Message, "identical execution tokens") {
			exactClones++
		}
	}
	if exactClones != 1 {
		t.Errorf("expected 1 exact clone advisory across differently named Go tests, got %d (%+v)", exactClones, res.Report.Findings)
	}
}

func TestEngineGoRunnableExamplesOnly(t *testing.T) {
	tmpDir := t.TempDir()
	eng := engine.New()

	runnableDir := filepath.Join(tmpDir, "runnable")
	_ = os.MkdirAll(runnableDir, 0755)
	_ = os.WriteFile(filepath.Join(runnableDir, "example_test.go"), []byte(`package sample

import "fmt"

func ExampleMultiline() {
	fmt.Println("one")
	fmt.Println("two")
	// Output:
	// one
	// two
}

func ExampleEmpty() {
	// Output:
}
`), 0644)

	res, err := eng.Run(engine.Options{
		TargetPaths: []string{runnableDir},
	})
	if err != nil {
		t.Fatalf("unexpected error scanning runnable examples: %v", err)
	}
	if res.Report.Analysis.AnalyzedTests != 2 {
		t.Errorf("expected 2 runnable examples analyzed, got %d", res.Report.Analysis.AnalyzedTests)
	}
	if res.Report.Analysis.Status != "complete" || res.Report.ExitCode != 0 {
		t.Errorf("expected complete/exit 0, got status=%q, exit=%d", res.Report.Analysis.Status, res.Report.ExitCode)
	}

	docOnlyDir := filepath.Join(tmpDir, "doconly")
	_ = os.MkdirAll(docOnlyDir, 0755)
	_ = os.WriteFile(filepath.Join(docOnlyDir, "example_test.go"), []byte(`package sample

import "fmt"

// Output: documented
func ExampleDocumented() {
	fmt.Println("compile only")
}
`), 0644)

	resDoc, err := eng.Run(engine.Options{
		TargetPaths: []string{docOnlyDir},
	})
	if err != nil {
		t.Fatalf("unexpected error scanning documentation-only example: %v", err)
	}
	if resDoc.Report.Analysis.AnalyzedTests != 0 {
		t.Errorf("expected 0 analyzed tests for documentation-only example marker, got %d", resDoc.Report.Analysis.AnalyzedTests)
	}
	if resDoc.Report.Analysis.Status != "incomplete" || resDoc.Report.Analysis.Reason != "empty_scope" || resDoc.Report.ExitCode != 2 {
		t.Errorf("expected incomplete/empty_scope/exit 2, got status=%q, reason=%q, exit=%d",
			resDoc.Report.Analysis.Status, resDoc.Report.Analysis.Reason, resDoc.Report.ExitCode)
	}
}

func TestEngineGoAssertionReceiverScopeIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "receiver_scope_test.go")
	_ = os.WriteFile(file, []byte(`package sample_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisjointReceiverReuse(t *testing.T) {
	{
		check := assert.New(t)
		_ = check
	}
	check := errors.New("boom")
	_ = check.Error()
}
`), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{file},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Report.Findings) != 1 {
		t.Fatalf("expected exactly 1 zero-assertion finding, got %d (%+v)", len(res.Report.Findings), res.Report.Findings)
	}
	finding := res.Report.Findings[0]
	if finding.RuleID != "GUARD-UNIV-ZERO" || finding.Severity != models.SeverityBlocker {
		t.Errorf("expected GUARD-UNIV-ZERO blocker, got rule=%s severity=%s", finding.RuleID, finding.Severity)
	}
	if res.Report.Policy.BlockerCount != 1 || res.Report.ExitCode != 1 {
		t.Errorf("expected blocker_count 1/exit 1, got %+v exit=%d", res.Report.Policy, res.Report.ExitCode)
	}
}

func TestEngineGoUnaliasedPackageCallBeforeLocalShadow(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "service", "shadow_test.go")
	_ = os.MkdirAll(filepath.Dir(file), 0755)
	_ = os.WriteFile(file, []byte(`package service_test

import (
	"testing"
	"github.com/DATA-DOG/go-sqlmock"
)

func TestPackageCallBeforeShadow(t *testing.T) {
	db, mock, err := sqlmock.New()
	_ = db
	_ = mock
	_ = err
	{
		sqlmock := newFakeScope()
		res := sqlmock.New()
		_ = res
	}
	t.Fatal("control")
}
`), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{file},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var portFindings []models.Violation
	for _, finding := range res.Report.Findings {
		if finding.RuleID == "GUARD-GO-02" {
			portFindings = append(portFindings, finding)
		}
	}
	if len(portFindings) != 1 {
		t.Fatalf("expected exactly 1 GUARD-GO-02 for the genuine package call, got %d (%+v)", len(portFindings), res.Report.Findings)
	}
	if portFindings[0].LineNumber != 9 {
		t.Errorf("expected genuine package call at line 9, got %d", portFindings[0].LineNumber)
	}
}
