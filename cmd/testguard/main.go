package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/docs"
	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/reporter"
	"github.com/HoangNguyen0403/testguard/pkg/updater"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.4"
	commit  = "none"
	date    = "unknown"
	repo    = "HoangNguyen0403/testguard"
)

// classifyScanInvocation decides, from raw CLI arguments, whether the selected
// command is the scan command and whether it requested JSON output, while
// collecting the caller-supplied path operands. It relies on Cobra's command
// tree and registered flag metadata so option operands (including dash-prefixed
// ones) are never mistaken for commands or flags.
func classifyScanInvocation(root *cobra.Command, args []string) (isJSON bool, paths []string) {
	selected, _, err := root.Find(args)
	if err != nil || selected != root {
		return false, nil
	}

	flags := root.Flags()
	jsonFlag := false
	format := ""
	pastDashDash := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if pastDashDash {
			paths = append(paths, arg)
			continue
		}
		if arg == "--" {
			pastDashDash = true
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			paths = append(paths, arg)
			continue
		}

		longForm := strings.HasPrefix(arg, "--")
		if longForm {
			name := strings.TrimPrefix(arg, "--")
			value := ""
			hasValue := false
			if eq := strings.Index(name, "="); eq >= 0 {
				name, value, hasValue = name[:eq], name[eq+1:], true
			}
			f := flags.Lookup(name)
			if f == nil {
				continue
			}
			if !hasValue {
				switch {
				case f.NoOptDefVal != "":
					value = f.NoOptDefVal
				case i+1 < len(args):
					i++
					value = args[i]
				default:
					continue
				}
			}
			switch f.Name {
			case "json":
				if parsed, perr := strconv.ParseBool(value); perr == nil {
					jsonFlag = parsed
				} else {
					jsonFlag = true
				}
			case "format":
				format = value
			}
			continue
		}

		name := strings.TrimPrefix(arg, "-")
		for offset := 0; offset < len(name); offset++ {
			f := flags.ShorthandLookup(name[offset : offset+1])
			if f == nil {
				break
			}

			value := f.NoOptDefVal
			if f.NoOptDefVal != "" {
				if offset+1 < len(name) && name[offset+1] == '=' {
					value = name[offset+2:]
					offset = len(name)
				}
			} else if offset+1 < len(name) {
				value = strings.TrimPrefix(name[offset+1:], "=")
				offset = len(name)
			} else if i+1 < len(args) {
				i++
				value = args[i]
			} else {
				break
			}

			switch f.Name {
			case "json":
				if parsed, perr := strconv.ParseBool(value); perr == nil {
					jsonFlag = parsed
				} else {
					jsonFlag = true
				}
			case "format":
				format = value
			}
		}
	}

	if len(paths) == 0 {
		paths = []string{"."}
	}
	return jsonFlag || format == "json", paths
}

func emitCLIError(isJSON bool, message string, args []string, version, commit, date string) {
	if isJSON {
		report := models.NewScanReport()
		report.Tool = models.NewToolMeta(version, commit, date)
		if len(args) > 0 {
			report.Scope.RequestedPaths = append([]string(nil), args...)
		} else {
			report.Scope.RequestedPaths = []string{"."}
		}
		report.Analysis.Status = "error"
		report.Analysis.Reason = "invalid_usage"
		report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("invalid_usage", "", message))
		report.ExitCode = 2
		if err := reporter.WriteJSONReport(os.Stdout, report); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON report: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
	os.Exit(2)
}
func main() {
	var diffMode bool
	var allMode bool
	var allowEmpty bool
	var jsonMode bool
	var format string
	var outputFile string
	var summaryMode bool
	var topIssues int
	rootCmd := &cobra.Command{
		Use:   "testguard [paths...]",
		Short: "Standalone polyglot test quality & architectural boundary guard CLI",
		Long: `TestGuard is a standalone, zero-config polyglot CLI that inspects test code,
detects duplicate tests, catches vanity test anti-patterns, and enforces architectural
boundaries across Dart/Flutter, Go, and TypeScript.

Website: https://github.com/HoangNguyen0403/testguard#readme
Issues:  https://github.com/HoangNguyen0403/testguard/issues`,
		Example: `  # 1. Day-to-day: Scan only files modified in your active branch/PR
  testguard --diff

  # 2. Full audit: Scan all test files across the entire monorepo
  testguard --all

  # 3. Target a specific project folder or single test file
  testguard path/to/project/test/

  # 4. Output machine-readable JSON for CI automation
  testguard --json

  # 5. Inspect why a rule exists and how to remediate it
  testguard explain GUARD-DART-SEED-ECHO`,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s (commit: %s, built at: %s)", version, commit, date),
		RunE: func(cmd *cobra.Command, args []string) error {
			isJSON := jsonMode || format == "json"

			if diffMode {
				emitCLIError(isJSON, "--diff mode is not supported; use full scan", args, version, commit, date)
			}

			if jsonMode && format == "azure" {
				emitCLIError(true, "--json cannot be combined with -f azure", args, version, commit, date)
			}
			if cmd.Flags().Changed("output") && (jsonMode || format == "json" || format == "azure") {
				emitCLIError(isJSON, "--output is only supported with console format", args, version, commit, date)
			}
			if format != "console" && format != "json" && format != "azure" {
				emitCLIError(isJSON, fmt.Sprintf("unsupported output format %q: must be console, json, or azure", format), args, version, commit, date)
			}

			eng := engine.New()

			opts := engine.Options{
				TargetPaths: args,
				DiffOnly:    diffMode,
				ScanAll:     allMode,
				AllowEmpty:  allowEmpty,
				Version:     version,
				Commit:      commit,
				Date:        date,
			}

			var progress *reporter.ProgressReporter
			if !isJSON {
				progress = reporter.NewProgressReporter(os.Stderr)
				opts.ProgressFn = progress.Update
			}
			defer func() {
				if progress != nil {
					progress.Finish()
				}
			}()

			res, err := eng.Run(opts)
			if progress != nil {
				progress.Finish()
			}
			if err != nil {
				emitCLIError(isJSON, err.Error(), args, version, commit, date)
			}

			report := res.Report

			if isJSON {
				if err := reporter.WriteJSONReport(os.Stdout, report); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing JSON report: %v\n", err)
					os.Exit(2)
				}
				os.Exit(report.ExitCode)
			}
			if format == "azure" {
				r := reporter.NewAzureReporter(os.Stdout)
				if _, err := r.Render(report); err != nil {
					fmt.Fprintf(os.Stderr, "Error: stdout write failure: %v\n", err)
					os.Exit(2)
				}
				os.Exit(report.ExitCode)
			}

			// Default terminal scorecard
			outPath := outputFile
			if outPath == "" {
				outPath = ".testguard/report.md"
			}
			if err := reporter.WriteMarkdownReport(outPath, report); err != nil {
				report.Analysis.Status = "error"
				report.Analysis.Reason = "output_failure"
				report.ExitCode = 2
				report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("output_failed", outPath, err.Error()))
				fmt.Fprintf(os.Stderr, "Error: failed writing markdown report: %v\n", err)
			}

			term := reporter.NewTerminalReporter(os.Stdout)
			if _, err := term.RenderWithOptions(report, reporter.TerminalOptions{
				SummaryOnly: summaryMode,
				MaxDisplay:  topIssues,
				ReportPath:  outPath,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Error: stdout write failure: %v\n", err)
				os.Exit(2)
			}

			os.Exit(report.ExitCode)
			return nil
		},
	}

	// Flags
	rootCmd.Flags().BoolVarP(&diffMode, "diff", "d", false, "Scan only modified test files against base branch (unsupported)")
	rootCmd.Flags().BoolVarP(&allMode, "all", "a", false, "Scan all test files in the workspace")
	rootCmd.Flags().BoolVar(&allowEmpty, "allow-empty", false, "Allow empty test scope (exit 3 not_applicable)")
	rootCmd.Flags().BoolVar(&jsonMode, "json", false, "Output machine-readable JSON (equivalent to -f json)")
	rootCmd.Flags().StringVarP(&format, "format", "f", "console", "Output format: console, json, azure")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to export markdown scorecard (default: .testguard/report.md)")
	rootCmd.Flags().BoolVarP(&summaryMode, "summary", "s", false, "Output high-level monorepo/module summary without file dumps")
	rootCmd.Flags().IntVar(&topIssues, "top", 10, "Maximum number of individual findings to display in terminal")
	// Subcommand: explain
	explainCmd := &cobra.Command{
		Use:   "explain <rule-id>",
		Short: "Display self-contained rationale, remediation, and code examples for a rule",
		Example: `  testguard explain GUARD-UNIV-ZERO
  testguard explain GUARD-DART-SEED-ECHO
  testguard explain GUARD-DART-UNCOVERED-MUTATION`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, found := docs.Explain(args[0])
			if !found {
				return fmt.Errorf("unknown rule ID: %s. Run 'testguard rules' to list all rules", args[0])
			}

			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("📖 Rule: %s — %s [%s | %s]\n", doc.ID, doc.Title, doc.Severity, doc.Language)
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
			fmt.Printf("📝 Description:\n  %s\n\n", doc.Description)
			fmt.Printf("💡 Why It Matters:\n  %s\n\n", doc.WhyItMatters)
			fmt.Printf("🛠️ Remediation:\n  %s\n\n", doc.Remediation)
			fmt.Printf("✅ Good Pattern:\n  %s\n\n", doc.GoodExample)
			fmt.Printf("❌ Bad Pattern:\n  %s\n", doc.BadExample)
			return nil
		},
	}

	// Subcommand: rules
	rulesCmd := &cobra.Command{
		Use:     "rules",
		Short:   "List all built-in test quality and boundary rules",
		Example: `  testguard rules`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available TestGuard Rules:")
			for _, r := range docs.AllRules() {
				fmt.Printf("  • %-30s [%-8s | %-10s] %s\n", r.ID, r.Severity, r.Language, r.Title)
			}
		},
	}

	// Subcommand: update (Self-Update via GitHub Releases)
	updateCmd := &cobra.Command{
		Use:     "update",
		Short:   "Check for and install the latest TestGuard release from GitHub",
		Example: `  testguard update`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updater.UpdateExecutable(version, repo)
		},
	}

	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(rulesCmd)
	rootCmd.AddCommand(updateCmd)
	if err := rootCmd.Execute(); err != nil {
		if jsonRequested, paths := classifyScanInvocation(rootCmd, os.Args[1:]); jsonRequested {
			emitCLIError(true, err.Error(), paths, version, commit, date)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(2)
		}
	}
}
