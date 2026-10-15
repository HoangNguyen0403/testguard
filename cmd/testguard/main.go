package main

import (
	"fmt"
	"os"

	"github.com/HoangNguyen0403/testguard/pkg/docs"
	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/reporter"
	"github.com/HoangNguyen0403/testguard/pkg/updater"
	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.3"
	commit  = "none"
	date    = "unknown"
	repo    = "HoangNguyen0403/testguard"
)

func main() {
	var diffMode bool
	var allMode bool
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
		Args:    cobra.ArbitraryArgs,
		Version: fmt.Sprintf("%s (commit: %s, built at: %s)", version, commit, date),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng := engine.New()

			if format != "console" && format != "json" && format != "azure" {
				return fmt.Errorf("unsupported output format %q: must be console, json, or azure", format)
			}

			opts := engine.Options{
				TargetPaths: args,
				DiffOnly:    diffMode,
				ScanAll:     allMode,
			}

			var progress *reporter.ProgressReporter
			if !jsonMode && format != "json" {
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
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(2)
			}

			hasBlockers := false
			for _, v := range res.Violations {
				if v.Severity == models.SeverityBlocker {
					hasBlockers = true
					break
				}
			}

			// CLIG: Support --json direct flag or -f json
			if jsonMode || format == "json" {
				if err := reporter.WriteJSONReport(os.Stdout, res.Violations); err != nil {
					return err
				}
				if hasBlockers {
					os.Exit(1)
				}
				return nil
			}

			if format == "azure" {
				r := reporter.NewAzureReporter(os.Stdout)
				r.Render(res.Violations)
				if hasBlockers {
					os.Exit(1)
				}
				return nil
			}

			// Default terminal scorecard
			term := reporter.NewTerminalReporter(os.Stdout)
			outPath := outputFile
			if outPath == "" {
				outPath = ".testguard/report.md"
			}
			exitCode := term.RenderWithOptions(res.Violations, res.ScannedFiles, res.ParseErrors, res.Skipped, reporter.TerminalOptions{
				SummaryOnly: summaryMode,
				MaxDisplay:  topIssues,
				ReportPath:  outPath,
			})
			_ = reporter.WriteMarkdownReport(outPath, res.Violations, res.ScannedFiles, res.ParseErrors, res.Skipped)

			if exitCode != 0 {
				os.Exit(exitCode)
			}

			return nil
		},
	}

	// Flags
	rootCmd.Flags().BoolVarP(&diffMode, "diff", "d", false, "Scan only modified test files against base branch")
	rootCmd.Flags().BoolVarP(&allMode, "all", "a", false, "Scan all test files in the workspace")
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
		Use:   "rules",
		Short: "List all built-in test quality and boundary rules",
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
		Use:   "update",
		Short: "Check for and install the latest TestGuard release from GitHub",
		Example: `  testguard update`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updater.UpdateExecutable(version, repo)
		},
	}

	// Subcommand: init (Generate starter testguard.yaml)
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a starter testguard.yaml configuration file",
		Example: `  testguard init`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "testguard.yaml"
			if _, err := os.Stat(target); err == nil {
				fmt.Println("ℹ️ testguard.yaml already exists in this directory.")
				return nil
			}

			starterConfig := `# 🛡️ TestGuard Configuration File
# Documentation: https://github.com/HoangNguyen0403/testguard#readme

version: "1.0"

# Global CI / Execution Defaults
global:
  fail_on: blocker              # Exit 1 only on "blocker". Options: blocker, warning, none
  diff_base: origin/develop     # Default base branch for PR diffs
  exclude:
    - "**/*.g.dart"
    - "**/*.freezed.dart"
    - "**/mock_*.go"
    - "node_modules/**"
    - "vendor/**"

# In-Code Suppression Syntax
# e.g.: // testguard:ignore GUARD-DART-SEED-ECHO reason: legacy test refactoring deferred
suppression:
  enabled: true
  require_reason: true

# Language-Specific Rule Overrides
languages:
  dart:
    enabled: true
    test_globs: ["test/**/*_test.dart"]
  golang:
    enabled: true
    test_globs: ["**/*_test.go"]
  typescript:
    enabled: true
    test_globs: ["**/*.test.ts", "**/*.test.tsx", "**/*.spec.ts"]
`
			if err := os.WriteFile(target, []byte(starterConfig), 0644); err != nil {
				return fmt.Errorf("failed to write testguard.yaml: %w", err)
			}

			fmt.Println("✅ Generated starter testguard.yaml configuration file.")
			return nil
		},
	}

	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(rulesCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}
