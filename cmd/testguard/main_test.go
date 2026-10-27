package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func helperProcess(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected process execution failure: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	os.Args = append([]string{"testguard"}, args...)
	main()
}

func TestCLIDiffModeRejected(t *testing.T) {
	stdout, stderr, exitCode := helperProcess(t, "--diff")
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for --diff, got %d (stderr: %s)", exitCode, stderr)
	}
	if !strings.Contains(stderr, "--diff mode is not supported; use full scan") {
		t.Errorf("expected rejection message in stderr, got %q", stderr)
	}
	if stdout != "" {
		t.Errorf("expected empty stdout for console error, got %q", stdout)
	}
}

func TestCLIJsonDiffModeEnvelope(t *testing.T) {
	stdout, _, exitCode := helperProcess(t, "--diff", "--json")
	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		t.Fatalf("stdout must be valid JSON: %v (raw: %s)", err, stdout)
	}

	if parsed["schema_version"] != float64(1) {
		t.Errorf("expected schema_version 1, got %v", parsed["schema_version"])
	}
	analysis, ok := parsed["analysis"].(map[string]interface{})
	if !ok || analysis["status"] != "error" || analysis["reason"] != "invalid_usage" {
		t.Errorf("expected analysis status=error, reason=invalid_usage, got %+v", parsed["analysis"])
	}
	if parsed["exit_code"] != float64(2) {
		t.Errorf("expected exit_code 2, got %v", parsed["exit_code"])
	}
}

func TestCLIJsonWithAzureRejected(t *testing.T) {
	stdout, _, exitCode := helperProcess(t, "--json", "-f", "azure")
	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		t.Fatalf("stdout must be valid JSON: %v", err)
	}
	if parsed["exit_code"] != float64(2) {
		t.Errorf("expected exit_code 2, got %v", parsed["exit_code"])
	}
}

func TestCLIOutputWithJsonRejected(t *testing.T) {
	// 1. Non-empty output
	_, _, exitCode := helperProcess(t, "--output", "report.md", "--json")
	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
	// 2. Explicitly empty output (R8)
	stdoutEmpty, _, exitCodeEmpty := helperProcess(t, "--output=", "--json")
	if exitCodeEmpty != 2 {
		t.Errorf("expected exit code 2 for --output=, got %d", exitCodeEmpty)
	}
	var parsedEmpty map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdoutEmpty)), &parsedEmpty); err != nil {
		t.Fatalf("stdout must be valid JSON: %v", err)
	}
	if parsedEmpty["exit_code"] != float64(2) {
		t.Errorf("expected exit_code 2, got %v", parsedEmpty["exit_code"])
	}

	// 3. Azure with short empty output (R8)
	_, stderrAzure, exitCodeAzure := helperProcess(t, "-f", "azure", "-o", "")
	if exitCodeAzure != 2 {
		t.Errorf("expected exit code 2 for azure with -o '', got %d", exitCodeAzure)
	}
	if !strings.Contains(stderrAzure, "--output is only supported with console format") {
		t.Errorf("expected output error in stderr for azure, got %q", stderrAzure)
	}
}

func TestCLIPreRunEParserClassification(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantJSON  bool
		wantPaths []string
	}{
		{"json-first", []string{"--json", "--unknown-flag-123"}, true, []string{"."}},
		{"json-last", []string{"--unknown-flag-123", "--json"}, true, []string{"."}},
		{"format-value", []string{"-f", "json", "--top", "not-a-number"}, true, []string{"."}},
		{"format-attached", []string{"-fjson", "--top", "not-a-number"}, true, []string{"."}},
		{"format-equals-missing-value", []string{"--format=json", "--format"}, true, []string{"."}},
		{"invalid-json-bool", []string{"--json=invalid-bool"}, true, []string{"."}},
		{"json-path-preserved", []string{"--json", "fixtures/not_there", "--unknown-flag-123"}, true, []string{"fixtures/not_there"}},
		{"output-operand-is-not-a-command", []string{"--output", "rules", "--json", "--unknown-flag-123"}, true, []string{"."}},
		{"json-consumed-as-output-operand", []string{"--output", "--json", "--unknown-flag-123"}, false, nil},
		{"false-bool-word", []string{"--json=FALSE", "--unknown-flag-123"}, false, nil},
		{"false-bool-zero", []string{"--json=0", "--unknown-flag-123"}, false, nil},
		{"boolean-cluster-with-attached-value", []string{"-sfjson", "--top", "not-a-number"}, true, []string{"."}},
		{"boolean-cluster-value-operand-is-not-json", []string{"-so", "--json", "--unknown-flag-123"}, false, nil},
		{"boolean-cluster-with-equals-value", []string{"-sf=json", "--top", "not-a-number"}, true, []string{"."}},
		{"equals-attached-output-operand-is-not-json", []string{"-so=--json", "--unknown-flag-123"}, false, nil},
		{"boolean-equals-value", []string{"-s=false", "--unknown-flag-123"}, false, nil},
		{"standalone-shorthand-equals-value", []string{"-f=json", "--top", "not-a-number"}, true, []string{"."}},
		{"non-scan-command-selected", []string{"-f", "json", "explain", "NOT-A-RULE-ID"}, false, nil},
	}

	for _, tc := range cases {
		stdout, stderr, exitCode := helperProcess(t, tc.args...)
		if exitCode != 2 {
			t.Errorf("%s: expected exit code 2, got %d (stderr: %s)", tc.name, exitCode, stderr)
		}
		if !tc.wantJSON {
			if stdout != "" {
				t.Errorf("%s: expected stderr-only error, got stdout %q", tc.name, stdout)
			}
			if stderr == "" {
				t.Errorf("%s: expected a visible error on stderr", tc.name)
			}
			continue
		}
		if !strings.HasSuffix(stdout, "\n") || strings.Count(stdout, "\n") != 1 {
			t.Errorf("%s: expected exactly one newline-terminated object, got %q", tc.name, stdout)
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
			t.Errorf("%s: stdout must be valid JSON: %v (raw: %s)", tc.name, err, stdout)
			continue
		}
		if parsed["schema_version"] != float64(1) || parsed["exit_code"] != float64(2) {
			t.Errorf("%s: expected schema_version 1 and exit_code 2, got %v/%v", tc.name, parsed["schema_version"], parsed["exit_code"])
		}
		analysis, ok := parsed["analysis"].(map[string]interface{})
		if !ok || analysis["status"] != "error" || analysis["reason"] != "invalid_usage" {
			t.Errorf("%s: expected status=error, reason=invalid_usage, got %+v", tc.name, parsed["analysis"])
		}

		scope, ok := parsed["scope"].(map[string]interface{})
		if !ok {
			t.Errorf("%s: missing scope object in envelope", tc.name)
			continue
		}
		rawPaths, _ := scope["requested_paths"].([]interface{})
		gotPaths := make([]string, 0, len(rawPaths))
		for _, p := range rawPaths {
			s, _ := p.(string)
			gotPaths = append(gotPaths, s)
		}
		if len(gotPaths) != len(tc.wantPaths) {
			t.Errorf("%s: expected requested_paths %v, got %v", tc.name, tc.wantPaths, gotPaths)
			continue
		}
		for i := range gotPaths {
			if gotPaths[i] != tc.wantPaths[i] {
				t.Errorf("%s: expected requested_paths %v, got %v", tc.name, tc.wantPaths, gotPaths)
				break
			}
		}
	}
}

func TestCLIConsoleMarkdownWriteFailureExits2(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "sample_test.go"), []byte("package test\nimport \"testing\"\nfunc TestSample(t *testing.T) {\n  _ = 1 + 1\n}\n"), 0644)

	blockedPath := filepath.Join(tmpDir, "blocked")
	_ = os.WriteFile(blockedPath, []byte("not a directory"), 0644)
	outPath := filepath.Join(blockedPath, "report.md")

	stdout, stderr, exitCode := helperProcess(t, "--output", outPath, tmpDir)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 on markdown output failure, got %d (stderr: %s)", exitCode, stderr)
	}
	if !strings.Contains(stderr, "failed writing markdown report") {
		t.Errorf("expected markdown write failure on stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "ERROR (error/output_failure)") {
		t.Errorf("expected console to report error/output_failure instead of a pass, got %q", stdout)
	}
	if !strings.Contains(stdout, "1 Blocker(s)") {
		t.Errorf("expected collected blocker findings to survive the output failure, got %q", stdout)
	}
}

func TestCLIConsoleExportFailureSuppressesArtifactClaim(t *testing.T) {
	tmpDir := t.TempDir()
	for name, body := range map[string]string{
		"a_test.go": "package test\nimport \"testing\"\nfunc TestA(t *testing.T) {\n  _ = 1 + 1\n}\n",
		"b_test.go": "package test\nimport \"testing\"\nfunc TestB(t *testing.T) {\n  _ = 2 + 2\n}\n",
	} {
		_ = os.WriteFile(filepath.Join(tmpDir, name), []byte(body), 0644)
	}

	blockedPath := filepath.Join(tmpDir, "blocked")
	_ = os.WriteFile(blockedPath, []byte("not a directory"), 0644)
	outPath := filepath.Join(blockedPath, "report.md")

	cases := []struct {
		name string
		args []string
	}{
		{"summary", []string{"--summary", "--output", outPath, tmpDir}},
		{"truncated", []string{"--top", "1", "--output", outPath, tmpDir}},
	}
	for _, tc := range cases {
		stdout, stderr, exitCode := helperProcess(t, tc.args...)
		if exitCode != 2 {
			t.Errorf("%s: expected exit code 2, got %d (stderr: %s)", tc.name, exitCode, stderr)
		}
		if !strings.Contains(stderr, "failed writing markdown report") {
			t.Errorf("%s: expected markdown write failure on stderr, got %q", tc.name, stderr)
		}
		if !strings.Contains(stdout, "ERROR (error/output_failure)") {
			t.Errorf("%s: expected error/output_failure verdict, got %q", tc.name, stdout)
		}
		if !strings.Contains(stdout, "2 Blocker(s)") {
			t.Errorf("%s: expected retained blocker findings, got %q", tc.name, stdout)
		}
		if strings.Contains(stdout, "written to "+outPath) {
			t.Errorf("%s: failed export must not be advertised as a written artifact, got %q", tc.name, stdout)
		}
	}
}

func TestCLIAzureEscapesNewlineBearingTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing\n##vso[task.complete result=Succeeded;].go")

	stdout, stderr, exitCode := helperProcess(t, "-f", "azure", target)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for a missing target, got %d (stderr: %s)", exitCode, stderr)
	}
	if !strings.Contains(stdout, "%0A") || !strings.Contains(stdout, "%3B") || !strings.Contains(stdout, "%5D") {
		t.Errorf("expected encoded newline/semicolon/bracket in Azure output, got %q", stdout)
	}
	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		if !strings.HasPrefix(line, "##vso[") {
			continue
		}
		if !strings.HasPrefix(line, "##vso[task.logissue type=") {
			t.Errorf("only logissue annotations may be emitted, got line %q", line)
		}
	}
}

func TestCLINonScanCommandErrorPreserved(t *testing.T) {
	stdout, stderr, exitCode := helperProcess(t, "explain", "NOT-A-RULE-ID")
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for invalid rule ID")
	}
	if !strings.Contains(stderr, "unknown rule ID: NOT-A-RULE-ID") {
		t.Errorf("expected error message in stderr, got %q", stderr)
	}
	if strings.Contains(stdout, "schema_version") {
		t.Errorf("non-scan command must NOT emit JSON scan report")
	}

	// 2. explain without arguments
	_, stderrMissing, exitCodeMissing := helperProcess(t, "explain")
	if exitCodeMissing == 0 {
		t.Errorf("expected non-zero exit code for missing explain argument")
	}
	if stderrMissing == "" {
		t.Errorf("expected visible error in stderr for missing argument")
	}
}

func TestCLIEmptyScopeDisallowed(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, _, exitCode := helperProcess(t, "--json", tmpDir)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for empty scope without --allow-empty, got %d", exitCode)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		t.Fatalf("stdout must be valid JSON: %v", err)
	}
	analysis := parsed["analysis"].(map[string]interface{})
	if analysis["status"] != "incomplete" || analysis["reason"] != "empty_scope" {
		t.Errorf("expected incomplete/empty_scope, got %+v", analysis)
	}

	// Verify empty array [] (R2)
	scope := parsed["scope"].(map[string]interface{})
	if files, ok := scope["files"].([]interface{}); !ok || files == nil {
		t.Errorf("expected scope.files to be empty array [], got %v", scope["files"])
	}
}

func TestCLIEmptyScopeAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, _, exitCode := helperProcess(t, "--json", "--allow-empty", tmpDir)
	if exitCode != 3 {
		t.Errorf("expected exit code 3 for empty scope with --allow-empty, got %d (stdout: %s)", exitCode, stdout)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		t.Fatalf("stdout must be valid JSON: %v", err)
	}
	analysis := parsed["analysis"].(map[string]interface{})
	if analysis["status"] != "not_applicable" || analysis["reason"] != "empty_allowed" {
		t.Errorf("expected not_applicable/empty_allowed, got %+v", analysis)
	}
	if parsed["exit_code"] != float64(3) {
		t.Errorf("expected exit_code 3 in json, got %v", parsed["exit_code"])
	}

	// Verify empty array [] (R2)
	scope := parsed["scope"].(map[string]interface{})
	if files, ok := scope["files"].([]interface{}); !ok || files == nil {
		t.Errorf("expected scope.files to be empty array [], got %v", scope["files"])
	}
}

func TestCLICompleteScanJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sample_test.go")
	_ = os.WriteFile(testFile, []byte("package test\nimport \"testing\"\nfunc TestSample(t *testing.T){\n  if 1 != 1 { t.Fatal(\"err\") }\n}\n"), 0644)

	stdout, _, exitCode := helperProcess(t, "--json", tmpDir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stdout: %s)", exitCode, stdout)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		t.Fatalf("stdout must be valid JSON: %v", err)
	}
	if parsed["schema_version"] != float64(1) {
		t.Errorf("expected schema_version 1, got %v", parsed["schema_version"])
	}
	analysis := parsed["analysis"].(map[string]interface{})
	if analysis["status"] != "complete" || analysis["reason"] != "analyzed" {
		t.Errorf("expected complete/analyzed, got %+v", analysis)
	}
	policy := parsed["policy"].(map[string]interface{})
	if policy["status"] != "no_blocker" || policy["blocker_count"] != float64(0) {
		t.Errorf("expected no_blocker, got %+v", policy)
	}
}
