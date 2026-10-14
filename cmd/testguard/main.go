package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.1.0-dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var diffMode bool
	var allMode bool

	rootCmd := &cobra.Command{
		Use:   "testguard [paths...]",
		Short: "Standalone polyglot test quality & architectural boundary guard CLI",
		Long: `TestGuard is a standalone, zero-config polyglot CLI that inspects test code,
detects duplicate tests, catches vanity test anti-patterns, and enforces architectural
boundaries across Dart/Flutter, Go, and TypeScript.`,
		Version: fmt.Sprintf("%s (commit: %s, built at: %s)", version, commit, date),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := "default (diff on PR, all on main)"
			if diffMode {
				mode = "diff-only"
			} else if allMode {
				mode = "all (full suite)"
			}

			fmt.Printf("🛡️ TestGuard Polyglot Engine v%s [%s]\n", version, mode)
			fmt.Println("Status: Ready. Run 'testguard --help' for options.")
			return nil
		},
	}

	rootCmd.Flags().BoolVarP(&diffMode, "diff", "d", false, "Scan only modified test files against base branch")
	rootCmd.Flags().BoolVarP(&allMode, "all", "a", false, "Scan all test files in the workspace")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}
