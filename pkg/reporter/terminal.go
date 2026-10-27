package reporter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

// TerminalOptions configures the terminal output presentation.
type TerminalOptions struct {
	SummaryOnly bool
	MaxDisplay  int    // Default: 10
	ReportPath  string // Default: .testguard/report.md
}

// TerminalReporter renders user-friendly, non-annoying testguard audit reports.
type TerminalReporter struct {
	out io.Writer
}

// NewTerminalReporter creates a new terminal reporter.
func NewTerminalReporter(out io.Writer) *TerminalReporter {
	return &TerminalReporter{out: out}
}

// Render writes the formatted scorecard and findings to stdout and returns the exit code and any write error.
func (r *TerminalReporter) Render(report *models.ScanReport) (int, error) {
	return r.RenderWithOptions(report, TerminalOptions{
		MaxDisplay: 10,
		ReportPath: ".testguard/report.md",
	})
}

// moduleStats tracks violations per module.
type moduleStats struct {
	name       string
	blockers   int
	warnings   int
	advisories int
}

func extractModule(filePath string) string {
	clean := filepath.ToSlash(filepath.Clean(filePath))

	// 1. Monorepo datasource path: e.g. datasource/ezrx/cart-service/... -> ezrx/cart-service
	if idx := strings.Index(clean, "datasource/"); idx != -1 {
		rest := clean[idx+len("datasource/"):]
		parts := strings.Split(rest, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		if len(parts) == 1 {
			return parts[0]
		}
	}

	// 2. Standard monorepo structures: packages/<pkg>, apps/<app>, services/<svc>, modules/<mod>
	prefixes := []string{"packages/", "apps/", "services/", "modules/", "submodules/"}
	for _, p := range prefixes {
		if idx := strings.Index(clean, p); idx != -1 {
			rest := clean[idx+len(p):]
			parts := strings.Split(rest, "/")
			if len(parts) >= 1 && parts[0] != "" {
				return p + parts[0]
			}
		}
	}

	// 3. Top-level directory if not a standard source/test directory
	parts := strings.Split(clean, "/")
	if len(parts) >= 2 && parts[0] != "." && parts[0] != "" && !strings.HasPrefix(parts[0], ".") {
		commonCodeDirs := map[string]bool{
			"test": true, "tests": true, "src": true, "pkg": true, "cmd": true, "internal": true, "app": true, "lib": true,
		}
		if !commonCodeDirs[parts[0]] {
			return parts[0]
		}
	}

	return "."
}

// RenderWithOptions renders the scorecard with monorepo module breakdown and summary controls.
func (r *TerminalReporter) RenderWithOptions(report *models.ScanReport, opts TerminalOptions) (int, error) {
	if report == nil {
		return 2, fmt.Errorf("scan report is nil")
	}
	if opts.MaxDisplay <= 0 {
		opts.MaxDisplay = 10
	}
	if opts.ReportPath == "" {
		opts.ReportPath = ".testguard/report.md"
	}
	// A prior Markdown export failure means the report path holds no artifact;
	// never advertise omitted details as written.
	exportFailed := report.Analysis.Reason == "output_failure"

	var writeErr error
	printf := func(format string, a ...interface{}) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintf(r.out, format, a...)
	}
	println := func(a ...interface{}) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintln(r.out, a...)
	}

	violations := report.Findings
	scannedFiles := report.Analysis.AnalyzedFiles
	parseErrors := report.Analysis.FailedFiles
	skipped := len(report.Scope.Excluded)

	blockers := 0
	warnings := 0
	advisories := 0

	moduleMap := make(map[string]*moduleStats)
	ruleCounts := make(map[string]int)

	for _, v := range violations {
		switch v.Severity {
		case models.SeverityBlocker:
			blockers++
		case models.SeverityWarning:
			warnings++
		case models.SeverityAdvisory:
			advisories++
		}

		mod := extractModule(v.FilePath)
		stats, exists := moduleMap[mod]
		if !exists {
			stats = &moduleStats{name: mod}
			moduleMap[mod] = stats
		}
		switch v.Severity {
		case models.SeverityBlocker:
			stats.blockers++
		case models.SeverityWarning:
			stats.warnings++
		case models.SeverityAdvisory:
			stats.advisories++
		}

		ruleCounts[v.RuleID]++
	}

	if report.Analysis.Status == "not_applicable" {
		printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		printf("🛡️ TestGuard Audit · Status: NOT APPLICABLE (%s/%s)\n", report.Analysis.Status, report.Analysis.Reason)
		printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
		printf("✨ Scope explicitly allows empty test set; no tests analyzed.\n")
		if writeErr != nil {
			return 2, writeErr
		}
		return report.ExitCode, nil
	}

	var status string
	switch report.Analysis.Status {
	case "complete":
		if blockers > 0 {
			status = "FAILED (complete/analyzed)"
		} else {
			status = "PASSED (complete/analyzed)"
		}
	case "incomplete":
		status = fmt.Sprintf("INCOMPLETE (%s/%s)", report.Analysis.Status, report.Analysis.Reason)
	case "error":
		status = fmt.Sprintf("ERROR (%s/%s)", report.Analysis.Status, report.Analysis.Reason)
	default:
		status = fmt.Sprintf("%s (%s)", strings.ToUpper(report.Analysis.Status), report.Analysis.Reason)
	}

	printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	if len(moduleMap) > 1 {
		printf("🛡️ TestGuard Monorepo Audit · %d files scanned across %d modules\n", scannedFiles, len(moduleMap))
	} else {
		printf("🛡️ TestGuard Audit · %d files scanned · %d Parse Errors · %d Skipped\n", scannedFiles, parseErrors, skipped)
	}
	printf("Status: %s · 🚨 %d Blocker(s) · ⚠️ %d Warning(s) · 💡 %d Advisory(ies)\n", status, blockers, warnings, advisories)
	printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(report.Diagnostics) > 0 {
		printf("DIAGNOSTICS:\n")
		for _, d := range report.Diagnostics {
			if d.Path != nil && *d.Path != "" {
				printf("  ❌ [%s] %s: %s\n", d.Code, *d.Path, d.Message)
			} else {
				printf("  ❌ [%s] %s\n", d.Code, d.Message)
			}
		}
		println()
	}

	if len(violations) == 0 {
		if report.Analysis.Status == "complete" {
			printf("✨ All tests adhere to boundary and quality standards!\n")
		}
		if writeErr != nil {
			return 2, writeErr
		}
		return report.ExitCode, nil
	}
	// 1. Render Module Breakdown if monorepo / multi-module
	if len(moduleMap) > 1 {
		var modules []*moduleStats
		for _, m := range moduleMap {
			modules = append(modules, m)
		}
		// Sort modules by blockers descending, then warnings descending, then advisories descending
		sort.Slice(modules, func(i, j int) bool {
			if modules[i].blockers != modules[j].blockers {
				return modules[i].blockers > modules[j].blockers
			}
			if modules[i].warnings != modules[j].warnings {
				return modules[i].warnings > modules[j].warnings
			}
			if modules[i].advisories != modules[j].advisories {
				return modules[i].advisories > modules[j].advisories
			}
			return modules[i].name < modules[j].name
		})

		printf("📦 MODULE BREAKDOWN:\n")
		printf("%-36s %10s %10s %12s   %s\n", "Module / Sub-Project", "Blockers", "Warnings", "Advisories", "Status")
		printf("─────────────────────────────────────────────────────────────────────────────\n")
		for _, m := range modules {
			modStatus := "✅ PASSED"
			if m.blockers > 0 {
				modStatus = "🚨 FAILED"
			} else if m.warnings > 0 {
				modStatus = "⚠️ WARNING"
			} else if m.advisories > 0 {
				modStatus = "💡 ADVISORY"
			}
			printf("%-36s %10d %10d %12d   %s\n", m.name, m.blockers, m.warnings, m.advisories, modStatus)
		}
		printf("─────────────────────────────────────────────────────────────────────────────\n\n")
	}

	// 2. Render Top Rule Violations
	type ruleStat struct {
		ruleID string
		count  int
	}
	var rules []ruleStat
	for rID, cnt := range ruleCounts {
		rules = append(rules, ruleStat{ruleID: rID, count: cnt})
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].count > rules[j].count
	})

	topRuleLimit := 5
	if len(rules) < topRuleLimit {
		topRuleLimit = len(rules)
	}
	printf("🔥 TOP RULE VIOLATIONS:\n")
	for i := range topRuleLimit {
		printf("  • %-30s %5d occurrence(s)\n", rules[i].ruleID, rules[i].count)
	}
	println()

	// If SummaryOnly mode, skip individual file listings
	if opts.SummaryOnly {
		if exportFailed {
			printf("✨ Summary mode: detailed violations were not exported because the report could not be written.\n")
		} else {
			printf("✨ Summary mode: Full detailed violations written to %s\n", opts.ReportPath)
		}
		if len(moduleMap) > 1 {
			printf("💡 Tip: Run 'testguard path/to/module' to inspect a specific service.\n")
		}
		if writeErr != nil {
			return 2, writeErr
		}
		return report.ExitCode, nil
	}

	// 3. Render Top Findings (capped to avoid terminal flooding)
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

	printf("🔍 KEY FINDINGS (Top %d):\n", opts.MaxDisplay)

	noColor := os.Getenv("NO_COLOR") != ""
	displayed := 0
	for _, v := range violations {
		if displayed >= opts.MaxDisplay {
			remaining := len(violations) - displayed
			if exportFailed {
				printf("... and %d more issue(s) not exported.\n", remaining)
			} else {
				printf("... and %d more issue(s) written to %s\n", remaining, opts.ReportPath)
			}
			if len(moduleMap) > 1 {
				printf("💡 Tip: Run 'testguard <module-path>' to inspect a specific service, or 'testguard --summary'.\n")
			}
			break
		}

		icon := "💡"
		if noColor {
			icon = "[ADVISORY]"
		}

		switch v.Severity {
		case models.SeverityBlocker:
			if noColor {
				icon = "[BLOCKER]"
			} else {
				icon = "🚨"
			}
		case models.SeverityWarning:
			if noColor {
				icon = "[WARNING]"
			} else {
				icon = "⚠️"
			}
		}

		printf("%s:%d:%d\n", v.FilePath, v.LineNumber, v.Column)
		printf("  %s [%s] %s\n", icon, v.RuleID, v.Message)
		if v.Suggestion != "" {
			fixLabel := "💡 Fix:"
			if noColor {
				fixLabel = "Fix:"
			}
			printf("  %s %s\n", fixLabel, v.Suggestion)
		}
		println()
		displayed++
	}

	if writeErr != nil {
		return 2, writeErr
	}
	return report.ExitCode, nil
}
