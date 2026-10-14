package main

import (
	"fmt"
	"os"

	"github.com/HoangNguyen0403/testguard/pkg/docs"
	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/reporter"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var diffMode bool
	var allMode bool
	var format string
	var outputFile string

	rootCmd := &cobra.Command{
		Use:   "testguard [paths...]",
		Short: "Standalone polyglot test quality & architectural boundary guard CLI",
		Long: `TestGuard is a standalone, zero-config polyglot CLI that inspects test code,
detects duplicate tests, catches vanity test anti-patterns, and enforces architectural
boundaries across Dart/Flutter, Go, and TypeScript.`,
		Args:    cobra.ArbitraryArgs,
		Version: fmt.Sprintf("%s (commit: %s, built at: %s)", version, commit, date),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng := engine.New()

			opts := engine.Options{
				TargetPaths: args,
				DiffOnly:    diffMode,
				ScanAll:     allMode,
			}

			res, err := eng.Run(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(2)
			}

			// Render according to requested format
			switch format {
			case "json":
				return reporter.WriteJSONReport(os.Stdout, res.Violations)
			case "azure":
				r := reporter.NewAzureReporter(os.Stdout)
				r.Render(res.Violations)
			default:
				term := reporter.NewTerminalReporter(os.Stdout)
				exitCode := term.Render(res.Violations, res.ScannedFiles, res.ParseErrors, res.Skipped)

				// Auto-save markdown report
				outPath := outputFile
				if outPath == "" {
					outPath = ".testguard/report.md"
				}
				_ = reporter.WriteMarkdownReport(outPath, res.Violations, res.ScannedFiles, res.ParseErrors, res.Skipped)

				if exitCode != 0 {
					os.Exit(exitCode)
				}
			}

			return nil
		},
	}

	rootCmd.Flags().BoolVarP(&diffMode, "diff", "d", false, "Scan only modified test files against base branch")
	rootCmd.Flags().BoolVarP(&allMode, "all", "a", false, "Scan all test files in the workspace")
	rootCmd.Flags().StringVarP(&format, "format", "f", "console", "Output format (console, json, azure)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to export markdown scorecard (default: .testguard/report.md)")

	explainCmd := &cobra.Command{
		Use:   "explain <rule-id>",
		Short: "Display self-contained rationale, remediation, and code examples for a rule",
		Args:  cobra.ExactArgs(1),
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

	rulesCmd := &cobra.Command{
		Use:   "rules",
		Short: "List all built-in test quality and boundary rules",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available TestGuard Rules:")
			for _, r := range docs.AllRules() {
				fmt.Printf("  • %-30s [%-8s | %-10s] %s\n", r.ID, r.Severity, r.Language, r.Title)
			}
		},
	}

	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(rulesCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}
