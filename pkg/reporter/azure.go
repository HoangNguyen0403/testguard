package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

// AzureReporter outputs Azure DevOps logging commands for PR inline annotations.
type AzureReporter struct {
	out io.Writer
}

// NewAzureReporter creates a new Azure DevOps reporter.
func NewAzureReporter(out io.Writer) *AzureReporter {
	return &AzureReporter{out: out}
}

// Render writes ##vso[task.logissue] commands for Azure Pipelines.
func (r *AzureReporter) Render(violations []models.Violation) {
	for _, v := range violations {
		issueType := "warning"
		if v.Severity == models.SeverityBlocker {
			issueType = "error"
		}
		fmt.Fprintf(r.out, "##vso[task.logissue type=%s;sourcepath=%s;linenumber=%d]%s\n",
			issueType, v.FilePath, v.LineNumber, v.Message)
	}
}

// WriteMarkdownReport writes a detailed markdown scorecard to .testguard/report.md.
func WriteMarkdownReport(filePath string, violations []models.Violation, scannedFiles, parseErrors, skipped int) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "# 🛡️ TestGuard Audit Report\n\n")
	fmt.Fprintf(f, "- **Scanned Files**: %d\n", scannedFiles)
	fmt.Fprintf(f, "- **Total Violations**: %d\n\n", len(violations))
	fmt.Fprintf(f, "| Severity | Rule ID | Location | Message |\n")
	fmt.Fprintf(f, "|---|---|---|---|\n")

	for _, v := range violations {
		badge := "💡 Advisory"
		switch v.Severity {
		case models.SeverityBlocker:
			badge = "🚨 Blocker"
		case models.SeverityWarning:
			badge = "⚠️ Warning"
		}
		fmt.Fprintf(f, "| %s | `%s` | `%s:%d` | %s |\n", badge, v.RuleID, v.FilePath, v.LineNumber, v.Message)
	}

	return nil
}

// WriteJSONReport writes the violations as formatted JSON to a writer.
func WriteJSONReport(out io.Writer, violations []models.Violation) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(violations)
}
