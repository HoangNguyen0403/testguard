package typescript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/typescript"
)

func TestDualExtRule(t *testing.T) {
	tempDir := t.TempDir()

	tsFile := filepath.Join(tempDir, "calculator.test.ts")
	jsFile := filepath.Join(tempDir, "calculator.test.js")
	loneTsFile := filepath.Join(tempDir, "standalone.test.ts")

	if err := os.WriteFile(tsFile, []byte("describe('ts', () => {});\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsFile, []byte("describe('js', () => {});\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loneTsFile, []byte("describe('lone', () => {});\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rule := typescript.NewDualExtRule()

	t.Run("Full scan of twin js file triggers warning", func(t *testing.T) {
		rule.SetScannedFiles([]string{tsFile, jsFile})
		violations := rule.EvaluateFile(jsFile, "describe('js', () => {});\n", []parser.TestUnit{
			{FilePath: jsFile, StartLine: 1, Name: "js test"},
		})
		if len(violations) == 0 {
			t.Fatalf("expected dual ext violation on js file, got 0")
		}
		if violations[0].RuleID != "GUARD-TS-DUAL-EXT" {
			t.Errorf("expected GUARD-TS-DUAL-EXT, got %s", violations[0].RuleID)
		}
		if violations[0].Severity != models.SeverityWarning {
			t.Errorf("expected Warning, got %s", violations[0].Severity)
		}
		if violations[0].LineNumber != 1 {
			t.Errorf("expected LineNumber 1, got %d", violations[0].LineNumber)
		}
	})

	t.Run("Full scan of twin ts file when js is also scanned does not duplicate warning", func(t *testing.T) {
		rule.SetScannedFiles([]string{tsFile, jsFile})
		violations := rule.EvaluateFile(tsFile, "describe('ts', () => {});\n", []parser.TestUnit{
			{FilePath: tsFile, StartLine: 1, Name: "ts test"},
		})
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations on ts file when js is also scanned, got %d", len(violations))
		}
	})

	t.Run("Diff mode symmetry: scanning only newly added ts file when js exists on disk triggers warning", func(t *testing.T) {
		rule.SetScannedFiles([]string{tsFile})
		violations := rule.EvaluateFile(tsFile, "describe('ts', () => {});\n", []parser.TestUnit{
			{FilePath: tsFile, StartLine: 1, Name: "ts test"},
		})
		if len(violations) == 0 {
			t.Fatalf("expected dual ext violation on ts file in diff mode, got 0")
		}
		if violations[0].RuleID != "GUARD-TS-DUAL-EXT" {
			t.Errorf("expected GUARD-TS-DUAL-EXT, got %s", violations[0].RuleID)
		}
	})

	t.Run("Lone ts file without js counterpart does not trigger warning", func(t *testing.T) {
		rule.SetScannedFiles([]string{loneTsFile})
		violations := rule.EvaluateFile(loneTsFile, "describe('lone', () => {});\n", []parser.TestUnit{
			{FilePath: loneTsFile, StartLine: 1, Name: "lone test"},
		})
		if len(violations) != 0 {
			t.Fatalf("expected 0 violations for standalone ts file, got %d", len(violations))
		}
	})
}
