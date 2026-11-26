package engine_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/engine"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "TestUser")
	runGit(t, dir, "config", "user.email", "test@example.com")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\noutput: %s", args, dir, err, string(out))
	}
	return string(out)
}

func TestDiff_ResolveGitRepoRoot(t *testing.T) {
	repo := initGitRepo(t)
	subDir := filepath.Join(repo, "sub", "deep")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	root, err := engine.ResolveGitRepoRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error resolving repo root: %v", err)
	}

	evalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	evalRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	if evalRoot != evalRepo {
		t.Errorf("expected root %s, got %s", evalRepo, evalRoot)
	}

	// Non-git directory
	nonGit := t.TempDir()
	_, err = engine.ResolveGitRepoRoot(nonGit)
	if err == nil {
		t.Errorf("expected error resolving non-git dir, got nil")
	}
}

func TestDiff_CollectDiffFiles_BaseBranch(t *testing.T) {
	repo := initGitRepo(t)

	// Base commit on main branch
	file1 := filepath.Join(repo, "main_test.go")
	_ = os.WriteFile(file1, []byte("package main\n"), 0644)
	runGit(t, repo, "add", "main_test.go")
	runGit(t, repo, "commit", "-m", "initial commit")

	// Create and checkout branch
	runGit(t, repo, "checkout", "-b", "feature")

	file2 := filepath.Join(repo, "feature_test.go")
	_ = os.WriteFile(file2, []byte("package feature\n"), 0644)
	runGit(t, repo, "add", "feature_test.go")
	runGit(t, repo, "commit", "-m", "feature commit")

	diffFiles, err := engine.CollectDiffFiles(repo, "HEAD~1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffFiles) != 1 {
		t.Fatalf("expected 1 diff file, got %d (%v)", len(diffFiles), diffFiles)
	}
	evalDiff, _ := filepath.EvalSymlinks(diffFiles[0])
	evalF2, _ := filepath.EvalSymlinks(file2)
	if evalDiff != evalF2 {
		t.Errorf("expected %s, got %s", evalF2, evalDiff)
	}
}

func TestDiff_CollectDiffFiles_UncommittedAndUntracked(t *testing.T) {
	repo := initGitRepo(t)

	file1 := filepath.Join(repo, "tracked_test.go")
	_ = os.WriteFile(file1, []byte("package tracked\n"), 0644)
	runGit(t, repo, "add", "tracked_test.go")
	runGit(t, repo, "commit", "-m", "initial commit")

	// Modify tracked file without committing
	_ = os.WriteFile(file1, []byte("package tracked\n// modified\n"), 0644)

	// Create untracked file
	file2 := filepath.Join(repo, "untracked_test.go")
	_ = os.WriteFile(file2, []byte("package untracked\n"), 0644)

	diffFiles, err := engine.CollectDiffFiles(repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffFiles) != 2 {
		t.Fatalf("expected 2 diff files, got %d (%v)", len(diffFiles), diffFiles)
	}

	var evalDiffs []string
	for _, f := range diffFiles {
		ev, _ := filepath.EvalSymlinks(f)
		evalDiffs = append(evalDiffs, ev)
	}
	sort.Strings(evalDiffs)

	ev1, _ := filepath.EvalSymlinks(file1)
	ev2, _ := filepath.EvalSymlinks(file2)
	expected := []string{ev1, ev2}
	sort.Strings(expected)

	if evalDiffs[0] != expected[0] || evalDiffs[1] != expected[1] {
		t.Errorf("expected %v, got %v", expected, evalDiffs)
	}
}

func TestDiff_CollectDiffFiles_HeadFallback(t *testing.T) {
	repo := initGitRepo(t)

	file1 := filepath.Join(repo, "first_test.go")
	_ = os.WriteFile(file1, []byte("package first\n"), 0644)
	runGit(t, repo, "add", "first_test.go")
	runGit(t, repo, "commit", "-m", "first commit")

	file2 := filepath.Join(repo, "second_test.go")
	_ = os.WriteFile(file2, []byte("package second\n"), 0644)
	runGit(t, repo, "add", "second_test.go")
	runGit(t, repo, "commit", "-m", "second commit")

	// In clean repo with no baseRef specified, falls back to HEAD~1...HEAD
	diffFiles, err := engine.CollectDiffFiles(repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffFiles) != 1 {
		t.Fatalf("expected 1 diff file from HEAD~1 fallback, got %d (%v)", len(diffFiles), diffFiles)
	}
	evalDiff, _ := filepath.EvalSymlinks(diffFiles[0])
	evalF2, _ := filepath.EvalSymlinks(file2)
	if evalDiff != evalF2 {
		t.Errorf("expected %s, got %s", evalF2, evalDiff)
	}
}

func TestDiff_EmptyDiff(t *testing.T) {
	repo := initGitRepo(t)

	// Single initial commit with non-test file
	readme := filepath.Join(repo, "README.md")
	_ = os.WriteFile(readme, []byte("# Readme\n"), 0644)
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial")

	eng := engine.New()

	// 1. Without allow-empty: exits 2, incomplete, empty_scope
	res1, err := eng.Run(engine.Options{
		TargetPaths: []string{repo},
		DiffOnly:    true,
		AllowEmpty:  false,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}
	if res1.Report.ExitCode != 2 {
		t.Errorf("expected exit code 2 for empty diff without allow-empty, got %d", res1.Report.ExitCode)
	}
	if res1.Report.Analysis.Status != "incomplete" || res1.Report.Analysis.Reason != "empty_scope" {
		t.Errorf("expected incomplete/empty_scope, got %s/%s", res1.Report.Analysis.Status, res1.Report.Analysis.Reason)
	}
	if res1.Report.Scope.EffectiveMode != "diff" {
		t.Errorf("expected EffectiveMode diff, got %q", res1.Report.Scope.EffectiveMode)
	}

	// 2. With allow-empty: exits 3, not_applicable, empty_allowed
	res2, err := eng.Run(engine.Options{
		TargetPaths: []string{repo},
		DiffOnly:    true,
		AllowEmpty:  true,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}
	if res2.Report.ExitCode != 3 {
		t.Errorf("expected exit code 3 for empty diff with allow-empty, got %d", res2.Report.ExitCode)
	}
	if res2.Report.Analysis.Status != "not_applicable" || res2.Report.Analysis.Reason != "empty_allowed" {
		t.Errorf("expected not_applicable/empty_allowed, got %s/%s", res2.Report.Analysis.Status, res2.Report.Analysis.Reason)
	}
	if res2.Report.Scope.EffectiveMode != "diff" {
		t.Errorf("expected EffectiveMode diff, got %q", res2.Report.Scope.EffectiveMode)
	}
}

func TestDiff_MutualExclusionWithAll(t *testing.T) {
	eng := engine.New()
	res, err := eng.Run(engine.Options{
		ScanAll:  true,
		DiffOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}
	if res.Report.ExitCode != 2 {
		t.Errorf("expected exit code 2 for mutual exclusion, got %d", res.Report.ExitCode)
	}
	if res.Report.Analysis.Status != "error" || res.Report.Analysis.Reason != "invalid_usage" {
		t.Errorf("expected error/invalid_usage, got %s/%s", res.Report.Analysis.Status, res.Report.Analysis.Reason)
	}
	foundDiag := false
	for _, d := range res.Report.Diagnostics {
		if d.Code == "invalid_usage" {
			foundDiag = true
			break
		}
	}
	if !foundDiag {
		t.Errorf("expected invalid_usage diagnostic")
	}
}

func TestDiff_ScansModifiedTest(t *testing.T) {
	repo := initGitRepo(t)

	// Initial commit
	testFile := filepath.Join(repo, "sample_test.go")
	_ = os.WriteFile(testFile, []byte("package sample\nimport \"testing\"\nfunc TestValid(t *testing.T) {\n  if 1 != 1 { t.Fatal(\"err\") }\n}\n"), 0644)
	runGit(t, repo, "add", "sample_test.go")
	runGit(t, repo, "commit", "-m", "initial")

	// Untracked test file with hollow test (GUARD-UNIV-ZERO)
	hollowFile := filepath.Join(repo, "hollow_test.go")
	_ = os.WriteFile(hollowFile, []byte("package sample\nimport \"testing\"\nfunc TestHollow(t *testing.T) {\n  _ = 1 + 1\n}\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{repo},
		DiffOnly:    true,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}

	if res.Report.Scope.EffectiveMode != "diff" {
		t.Errorf("expected EffectiveMode 'diff', got %q", res.Report.Scope.EffectiveMode)
	}
	if res.Report.ExitCode != 1 {
		t.Errorf("expected exit code 1 (blocker found), got %d", res.Report.ExitCode)
	}
	if res.Report.Policy.Status != "blocker_found" {
		t.Errorf("expected policy status 'blocker_found', got %q", res.Report.Policy.Status)
	}
}

func TestDiff_Policy_FailOnWarning(t *testing.T) {
	tmpDir := t.TempDir()
	// Create test file with warning only (t.Skip -> GUARD-UNIV-SKIP)
	skipFile := filepath.Join(tmpDir, "skip_test.go")
	_ = os.WriteFile(skipFile, []byte(`package sample
import "testing"
func TestSkipped(t *testing.T) {
	t.Skip("skipping test")
	if 1 != 1 {
		t.Fatal("fail")
	}
}
`), 0644)

	eng := engine.New()

	// 1. Without FailOnWarning: exit code 0, status no_blocker
	res1, err := eng.Run(engine.Options{
		TargetPaths:   []string{tmpDir},
		FailOnWarning: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.Report.ExitCode != 0 {
		t.Errorf("expected exit code 0 without FailOnWarning, got %d", res1.Report.ExitCode)
	}
	if res1.Report.Policy.Status != "no_blocker" {
		t.Errorf("expected policy status 'no_blocker', got %q", res1.Report.Policy.Status)
	}

	// 2. With FailOnWarning: exit code 1, status warning_found
	res2, err := eng.Run(engine.Options{
		TargetPaths:   []string{tmpDir},
		FailOnWarning: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Report.ExitCode != 1 {
		t.Errorf("expected exit code 1 with FailOnWarning, got %d", res2.Report.ExitCode)
	}
	if res2.Report.Policy.Status != "warning_found" {
		t.Errorf("expected policy status 'warning_found', got %q", res2.Report.Policy.Status)
	}
}

func TestDiff_ExcludedDirectoryFiltering(t *testing.T) {
	repo := initGitRepo(t)

	// Initial commit
	initFile := filepath.Join(repo, "initial_test.go")
	_ = os.WriteFile(initFile, []byte("package sample\nimport \"testing\"\nfunc TestInit(t *testing.T) {\n  if 1 != 1 { t.Fatal(\"err\") }\n}\n"), 0644)
	runGit(t, repo, "add", "initial_test.go")
	runGit(t, repo, "commit", "-m", "init")

	// Create test files in excluded directories (node_modules, vendor, .hidden)
	nmDir := filepath.Join(repo, "node_modules", "pkg")
	_ = os.MkdirAll(nmDir, 0755)
	_ = os.WriteFile(filepath.Join(nmDir, "hollow_test.go"), []byte("package sample\nfunc TestBad(t *testing.T) {}\n"), 0644)

	hiddenDir := filepath.Join(repo, ".hidden")
	_ = os.MkdirAll(hiddenDir, 0755)
	_ = os.WriteFile(filepath.Join(hiddenDir, "hollow_test.go"), []byte("package sample\nfunc TestBad(t *testing.T) {}\n"), 0644)

	vendorDir := filepath.Join(repo, "vendor", "mod")
	_ = os.MkdirAll(vendorDir, 0755)
	_ = os.WriteFile(filepath.Join(vendorDir, "hollow_test.go"), []byte("package sample\nfunc TestBad(t *testing.T) {}\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{repo},
		DiffOnly:    true,
		AllowEmpty:  true,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}

	// Since all diffs are in excluded directories, no candidate test files remain
	if res.Report.Analysis.AnalyzedFiles != 0 {
		t.Errorf("expected 0 analyzed files, got %d (%v)", res.Report.Analysis.AnalyzedFiles, res.Report.Scope.Files)
	}
	if res.Report.ExitCode != 3 {
		t.Errorf("expected exit code 3 (empty_allowed), got %d", res.Report.ExitCode)
	}
	if len(res.Report.Scope.Excluded) < 3 {
		t.Errorf("expected at least 3 excluded directories, got %d (%v)", len(res.Report.Scope.Excluded), res.Report.Scope.Excluded)
	}
}

func TestDiff_TargetPathScoping(t *testing.T) {
	repo := initGitRepo(t)

	initFile := filepath.Join(repo, "root_test.go")
	_ = os.WriteFile(initFile, []byte("package sample\nimport \"testing\"\nfunc TestRoot(t *testing.T) {\n  if 1 != 1 { t.Fatal(\"err\") }\n}\n"), 0644)
	runGit(t, repo, "add", "root_test.go")
	runGit(t, repo, "commit", "-m", "init")

	subDir := filepath.Join(repo, "sub")
	_ = os.MkdirAll(subDir, 0755)
	subHollow := filepath.Join(subDir, "sub_hollow_test.go")
	_ = os.WriteFile(subHollow, []byte("package sub\nimport \"testing\"\nfunc TestSubHollow(t *testing.T) {\n  _ = 1 + 1\n}\n"), 0644)

	otherDir := filepath.Join(repo, "other")
	_ = os.MkdirAll(otherDir, 0755)
	otherHollow := filepath.Join(otherDir, "other_hollow_test.go")
	_ = os.WriteFile(otherHollow, []byte("package other\nimport \"testing\"\nfunc TestOtherHollow(t *testing.T) {\n  _ = 1 + 1\n}\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{subDir},
		DiffOnly:    true,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}

	if res.Report.Analysis.AnalyzedFiles != 1 {
		t.Fatalf("expected 1 analyzed file scoped to subDir, got %d (%v)", res.Report.Analysis.AnalyzedFiles, res.Report.Scope.Files)
	}
	if !strings.Contains(res.Report.Scope.Files[0], "sub_hollow_test.go") {
		t.Errorf("expected sub_hollow_test.go to be analyzed, got %s", res.Report.Scope.Files[0])
	}
}

func TestDiff_DiscoversHyphenTestFilesInTestsDirectory(t *testing.T) {
	repo := initGitRepo(t)

	initFile := filepath.Join(repo, "initial_test.go")
	_ = os.WriteFile(initFile, []byte("package sample\nimport \"testing\"\nfunc TestInit(t *testing.T) {\n  if 1 != 1 { t.Fatal(\"err\") }\n}\n"), 0644)
	runGit(t, repo, "add", "initial_test.go")
	runGit(t, repo, "commit", "-m", "init")

	testsDir := filepath.Join(repo, "__tests__")
	_ = os.MkdirAll(testsDir, 0755)
	testFile := filepath.Join(testsDir, "Foo-test.js")
	_ = os.WriteFile(testFile, []byte("it('adds', () => { expect(1 + 1).toBe(2); });\n"), 0644)

	// Root build script with the same hyphen suffix must stay out of the diff scope.
	buildScript := filepath.Join(repo, "build-and-test.js")
	_ = os.WriteFile(buildScript, []byte("module.exports = () => {};\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{repo},
		DiffOnly:    true,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}

	if res.Report.Scope.EffectiveMode != "diff" {
		t.Errorf("expected EffectiveMode 'diff', got %q", res.Report.Scope.EffectiveMode)
	}
	if res.Report.Analysis.AnalyzedFiles != 1 {
		t.Fatalf("expected 1 analyzed file, got %d (%v)", res.Report.Analysis.AnalyzedFiles, res.Report.Scope.Files)
	}
	if len(res.Report.Scope.Files) != 1 || filepath.Base(res.Report.Scope.Files[0]) != "Foo-test.js" {
		t.Errorf("expected __tests__/Foo-test.js to be the analyzed file, got %+v", res.Report.Scope.Files)
	}
	for _, file := range res.Report.Scope.Files {
		if filepath.Base(file) == "build-and-test.js" {
			t.Errorf("expected root build-and-test.js to stay out of the diff scope, got %+v", res.Report.Scope.Files)
		}
	}
}

func TestDiff_GitUnavailableDiagnostic(t *testing.T) {
	nonGitDir := t.TempDir()
	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{nonGitDir},
		DiffOnly:    true,
	})
	if err != nil {
		t.Fatalf("unexpected engine error: %v", err)
	}

	if res.Report.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", res.Report.ExitCode)
	}
	if res.Report.Analysis.Status != "incomplete" || res.Report.Analysis.Reason != "input_failure" {
		t.Errorf("expected incomplete/input_failure, got %s/%s", res.Report.Analysis.Status, res.Report.Analysis.Reason)
	}
	foundDiag := false
	for _, d := range res.Report.Diagnostics {
		if d.Code == "git_unavailable" {
			foundDiag = true
			break
		}
	}
	if !foundDiag {
		t.Errorf("expected diagnostic with code 'git_unavailable', got %+v", res.Report.Diagnostics)
	}
}
