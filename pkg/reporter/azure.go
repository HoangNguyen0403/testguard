package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

var (
	// azurePropertyReplacer encodes values used inside a ##vso[...] property list.
	azurePropertyReplacer = strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
		";", "%3B",
		"]", "%5D",
	)
	// azureMessageReplacer encodes the message body of a logging command so
	// literal line breaks cannot start another command line.
	azureMessageReplacer = strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
	)
)

// escapeAzureProperty encodes Azure logging-command property content.
func escapeAzureProperty(value string) string { return azurePropertyReplacer.Replace(value) }

// escapeAzureMessage encodes Azure logging-command message content.
func escapeAzureMessage(value string) string { return azureMessageReplacer.Replace(value) }

// AzureReporter outputs Azure DevOps logging commands for PR inline annotations.
type AzureReporter struct {
	out io.Writer
}

// NewAzureReporter creates a new Azure DevOps reporter.
func NewAzureReporter(out io.Writer) *AzureReporter {
	return &AzureReporter{out: out}
}

// Render writes ##vso[task.logissue] commands for Azure Pipelines based on the ScanReport contract.
func (r *AzureReporter) Render(report *models.ScanReport) (int, error) {
	if report == nil {
		return 2, fmt.Errorf("scan report is nil")
	}

	var writeErr error
	printf := func(format string, a ...interface{}) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintf(r.out, format, a...)
	}

	// 1. Explicit status and reason summary for every verdict (R10)
	printf("TestGuard audit status: %s (%s)\n", report.Analysis.Status, report.Analysis.Reason)

	// emitIssue renders one logissue command; dynamic content is escaped so it
	// can never terminate the property list or start another command line.
	emitIssue := func(props, message string) {
		printf("##vso[task.logissue %s]%s\n", props, escapeAzureMessage(message))
	}

	// 2. Diagnostics as error annotations on incomplete or error
	if report.Analysis.Status == "incomplete" || report.Analysis.Status == "error" {
		emitIssue("type=error", fmt.Sprintf("TestGuard audit %s (%s)", report.Analysis.Status, report.Analysis.Reason))
		for _, d := range report.Diagnostics {
			props := "type=error"
			if d.Path != nil && *d.Path != "" {
				props += ";sourcepath=" + escapeAzureProperty(*d.Path)
			}
			emitIssue(props, d.Code+": "+d.Message)
		}
	} else if report.Analysis.Status == "not_applicable" {
		// Summary on not_applicable; MUST NOT emit task-success commands
		printf("TestGuard audit not applicable: %s (exit 3)\n", report.Analysis.Reason)
	}

	// 3. Findings
	for _, v := range report.Findings {
		issueType := "warning"
		if v.Severity == models.SeverityBlocker {
			issueType = "error"
		}
		props := fmt.Sprintf("type=%s;sourcepath=%s;linenumber=%d", issueType, escapeAzureProperty(v.FilePath), v.LineNumber)
		emitIssue(props, v.Message)
	}

	if writeErr != nil {
		return 2, writeErr
	}
	return report.ExitCode, nil
}

// createReportFile opens the Markdown report destination. Tests override it to
// inject deterministic open/write/close failures (same pattern as the engine
// fault-injection hooks) because permission tricks are not portable.
var createReportFile = func(filePath string) (io.WriteCloser, error) {
	return os.Create(filePath)
}

// WriteMarkdownReport writes a detailed markdown scorecard to .testguard/report.md.
func WriteMarkdownReport(filePath string, report *models.ScanReport) error {
	if report == nil {
		return fmt.Errorf("scan report is nil")
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := createReportFile(filePath)
	if err != nil {
		return err
	}
	writeErr := writeMarkdown(f, report)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// writeMarkdown renders the markdown scorecard, surfacing the first write error.
func writeMarkdown(out io.Writer, report *models.ScanReport) error {
	var writeErr error
	printf := func(format string, a ...interface{}) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintf(out, format, a...)
	}

	printf("# 🛡️ TestGuard Audit Report\n\n")
	printf("- **Status**: %s (%s)\n", report.Analysis.Status, report.Analysis.Reason)
	printf("- **Scanned Files**: %d\n", report.Analysis.AnalyzedFiles)
	printf("- **Analyzed Tests**: %d\n", report.Analysis.AnalyzedTests)
	printf("- **Failed Files**: %d\n", report.Analysis.FailedFiles)
	printf("- **Total Violations**: %d\n", len(report.Findings))
	printf("- **Exit Code**: %d\n\n", report.ExitCode)

	if len(report.Diagnostics) > 0 {
		printf("## Diagnostics\n\n")
		printf("| Code | Path | Message |\n")
		printf("|---|---|---|\n")
		for _, d := range report.Diagnostics {
			p := "-"
			if d.Path != nil && *d.Path != "" {
				p = fmt.Sprintf("`%s`", *d.Path)
			}
			printf("| `%s` | %s | %s |\n", d.Code, p, d.Message)
		}
		printf("\n")
	}

	if len(report.Findings) > 0 {
		printf("## Violations\n\n")
		printf("| Severity | Rule ID | Location | Message |\n")
		printf("|---|---|---|---|\n")

		for _, v := range report.Findings {
			badge := "💡 Advisory"
			switch v.Severity {
			case models.SeverityBlocker:
				badge = "🚨 Blocker"
			case models.SeverityWarning:
				badge = "⚠️ Warning"
			}
			printf("| %s | `%s` | `%s:%d` | %s |\n", badge, v.RuleID, v.FilePath, v.LineNumber, v.Message)
		}
	}

	if writeErr != nil {
		return writeErr
	}
	return nil
}

// WriteJSONReport writes the ScanReport as exactly one UTF-8 JSON object plus newline.
func WriteJSONReport(out io.Writer, report *models.ScanReport) error {
	if report == nil {
		report = models.NewScanReport()
	}
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if _, err := out.Write(data); err != nil {
		return err
	}
	_, err = out.Write([]byte("\n"))
	return err
}
