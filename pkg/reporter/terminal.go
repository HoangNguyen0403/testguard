package reporter

import (
	"fmt"
	"io"
	"sort"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

// TerminalReporter renders user-friendly, non-annoying testguard audit reports.
type TerminalReporter struct {
	out io.Writer
}

// NewTerminalReporter creates a new terminal reporter.
func NewTerminalReporter(out io.Writer) *TerminalReporter {
	return &TerminalReporter{out: out}
}

// Render writes the formatted scorecard and findings to stdout and returns the exit code.
func (r *TerminalReporter) Render(violations []models.Violation, scannedFiles, parseErrors, skipped int) int {
	blockers := 0
	warnings := 0
	advisories := 0

	for _, v := range violations {
		switch v.Severity {
		case models.SeverityBlocker:
			blockers++
		case models.SeverityWarning:
			warnings++
		case models.SeverityAdvisory:
			advisories++
		}
	}

	status := "PASSED"
	if blockers > 0 {
		status = "FAILED"
	}

	fmt.Fprintf(r.out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(r.out, "🛡️ TestGuard Audit · %d files scanned · %d Parse Errors · %d Skipped\n", scannedFiles, parseErrors, skipped)
	fmt.Fprintf(r.out, "Status: %s · 🚨 %d Blocker(s) · ⚠️ %d Warning(s) · 💡 %d Advisory(ies)\n", status, blockers, warnings, advisories)
	fmt.Fprintf(r.out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(violations) == 0 {
		fmt.Fprintf(r.out, "✨ All tests adhere to boundary and quality standards!\n")
		return 0
	}

	// Sort: Blockers first, then Warnings, then Advisories
	sort.SliceStable(violations, func(i, j int) bool {
		severityWeight := func(s models.Severity) int {
			switch s {
			case models.SeverityBlocker:
				return 0
			case models.SeverityWarning:
				return 1
			default:
				return 2
			}
		}
		return severityWeight(violations[i].Severity) < severityWeight(violations[j].Severity)
	})

	// Print up to 10 findings, but never truncate Blockers
	maxDisplay := 10
	displayed := 0

	for i, v := range violations {
		if i >= maxDisplay && v.Severity != models.SeverityBlocker {
			fmt.Fprintf(r.out, "... and %d more advisory/warning issue(s) written to .testguard/report.md\n", len(violations)-i)
			break
		}

		icon := "💡"
		switch v.Severity {
		case models.SeverityBlocker:
			icon = "🚨"
		case models.SeverityWarning:
			icon = "⚠️"
		}

		fmt.Fprintf(r.out, "%s:%d:%d\n", v.FilePath, v.LineNumber, v.Column)
		fmt.Fprintf(r.out, "  %s [%s] %s\n", icon, v.RuleID, v.Message)
		if v.Suggestion != "" {
			fmt.Fprintf(r.out, "  💡 Fix: %s\n", v.Suggestion)
		}
		fmt.Fprintln(r.out)
		displayed++
	}

	if blockers > 0 {
		return 1
	}
	return 0
}
