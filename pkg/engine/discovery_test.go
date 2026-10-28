package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/parser/dart"
)

func TestDiscoverScopeSymlinkRejected(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real_test.go")
	_ = os.WriteFile(realFile, []byte("package test\nfunc TestReal(t *testing.T){}\n"), 0644)

	// 1. File symlink
	symlinkPath := filepath.Join(tmpDir, "symlink_test.go")
	err := os.Symlink(realFile, symlinkPath)
	if err != nil {
		t.Skip("symlinks not supported on this environment")
	}

	adapter := dart.NewAdapter()
	res := engine.DiscoverScope([]string{symlinkPath}, adapter, nil)

	if !res.InputFailure {
		t.Errorf("expected InputFailure to be true for explicit symlink target")
	}
	hasSymlinkDiag := false
	for _, d := range res.Diagnostics {
		if d.Code == "target_unreadable" {
			hasSymlinkDiag = true
		}
	}
	if !hasSymlinkDiag {
		t.Errorf("expected target_unreadable diagnostic for symlink target, got %+v", res.Diagnostics)
	}

	// 2. Directory symlink (with and without trailing slash)
	subDir := filepath.Join(tmpDir, "subdir")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "inner_test.go"), []byte("package test\nfunc TestInner(t *testing.T){}\n"), 0644)

	dirSymlink := filepath.Join(tmpDir, "dir_symlink")
	_ = os.Symlink(subDir, dirSymlink)

	// Without trailing slash
	resDir := engine.DiscoverScope([]string{dirSymlink}, adapter, nil)
	if !resDir.InputFailure {
		t.Errorf("expected InputFailure for directory symlink target")
	}

	// With trailing slash
	dirSymlinkTrailing := dirSymlink + string(filepath.Separator)
	resDirTrailing := engine.DiscoverScope([]string{dirSymlinkTrailing}, adapter, nil)
	if !resDirTrailing.InputFailure {
		t.Errorf("expected InputFailure for directory symlink with trailing slash")
	}
}

func TestDiscoverScopeDefaultDirectoryExclusions(t *testing.T) {
	tmpDir := t.TempDir()

	// Normal test file
	_ = os.WriteFile(filepath.Join(tmpDir, "normal_test.go"), []byte("package test\nfunc TestN(t *testing.T){}\n"), 0644)

	// Excluded directories
	dirs := []string{".git", ".testguard", "node_modules", "vendor", "build", "dist"}
	for _, d := range dirs {
		sub := filepath.Join(tmpDir, d)
		_ = os.MkdirAll(sub, 0755)
		_ = os.WriteFile(filepath.Join(sub, "hidden_test.go"), []byte("package test\n"), 0644)
	}

	adapter := dart.NewAdapter()
	res := engine.DiscoverScope([]string{tmpDir}, adapter, nil)

	if len(res.CandidateFiles) != 1 {
		t.Errorf("expected exactly 1 candidate file, got %d (%+v)", len(res.CandidateFiles), res.CandidateFiles)
	}

	// Verify all directories were recorded in Excluded with reason default_directory
	excludedCount := 0
	for _, ex := range res.Excluded {
		if ex.Reason == "default_directory" {
			excludedCount++
		}
	}
	if excludedCount < len(dirs) {
		t.Errorf("expected at least %d default_directory exclusions, got %d", len(dirs), excludedCount)
	}
}

func TestDiscoverScopeExplicitNonTestFile(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "server.go")
	_ = os.WriteFile(sourceFile, []byte("package main\nfunc main(){}\n"), 0644)

	adapter := dart.NewAdapter()
	res := engine.DiscoverScope([]string{sourceFile}, adapter, nil)

	if len(res.CandidateFiles) != 0 {
		t.Errorf("non-test file must not be in candidate files")
	}
	hasNonTest := false
	for _, ex := range res.Excluded {
		if ex.Reason == "non_test" {
			hasNonTest = true
		}
	}
	if !hasNonTest {
		t.Errorf("expected non_test exclusion for explicit non-test file, got %+v", res.Excluded)
	}
}

func TestDiscoverScopeExplicitUnsupportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	docFile := filepath.Join(tmpDir, "spec.pdf")
	_ = os.WriteFile(docFile, []byte("%PDF-1.4"), 0644)

	adapter := dart.NewAdapter()
	res := engine.DiscoverScope([]string{docFile}, adapter, nil)

	if len(res.Unsupported) != 1 {
		t.Errorf("expected 1 unsupported file, got %+v", res.Unsupported)
	}
	hasDiag := false
	for _, d := range res.Diagnostics {
		if d.Code == "unsupported_input" {
			hasDiag = true
		}
	}
	if !hasDiag {
		t.Errorf("expected unsupported_input diagnostic, got %+v", res.Diagnostics)
	}
}

func TestDiscoverScopeGeneratedDartFile(t *testing.T) {
	tmpDir := t.TempDir()
	genFile := filepath.Join(tmpDir, "model_test.dart")
	_ = os.WriteFile(genFile, []byte("// GENERATED CODE - DO NOT MODIFY BY HAND\nvoid main(){}\n"), 0644)

	adapter := dart.NewAdapter()
	res := engine.DiscoverScope([]string{tmpDir}, adapter, nil)

	if len(res.CandidateFiles) != 0 {
		t.Errorf("generated file must not be in candidate files")
	}
	if res.DiscoveredFiles != 1 {
		t.Errorf("discovered_files must count generated candidates before exclusion, got %d", res.DiscoveredFiles)
	}
	hasGenExcluded := false
	for _, ex := range res.Excluded {
		if ex.Reason == "generated" {
			hasGenExcluded = true
		}
	}
	if !hasGenExcluded {
		t.Errorf("expected generated exclusion, got %+v", res.Excluded)
	}
}

func TestDiscoverScopeOverlappingRootsDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sample_test.go")
	_ = os.WriteFile(testFile, []byte("package test\nfunc TestSample(t *testing.T){}\n"), 0644)

	adapter := dart.NewAdapter()

	// 1. Directory + Explicit file overlap
	resOverlap := engine.DiscoverScope([]string{tmpDir, testFile}, adapter, nil)
	if resOverlap.DiscoveredFiles != 1 {
		t.Errorf("expected exactly 1 discovered file across overlapping root and file, got %d", resOverlap.DiscoveredFiles)
	}
	if len(resOverlap.CandidateFiles) != 1 {
		t.Errorf("expected exactly 1 candidate file, got %d", len(resOverlap.CandidateFiles))
	}

	// 2. Relative + Absolute overlap of the same fixture.
	// In environments where tmpDir and cwd reside on different Windows drives (e.g. C: vs D:),
	// filepath.Rel cannot bridge drives; construct a fixture within cwd to test relative/absolute overlap.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("unable to resolve cwd: %v", err)
	}
	overlapDir, err := os.MkdirTemp(cwd, "testguard_overlap_*")
	if err != nil {
		t.Fatalf("unable to create local temp dir: %v", err)
	}
	defer os.RemoveAll(overlapDir)
	localTestFile := filepath.Join(overlapDir, "sample_test.go")
	if err := os.WriteFile(localTestFile, []byte("package test\n"), 0644); err != nil {
		t.Fatalf("unable to write local fixture: %v", err)
	}
	relTestFile, err := filepath.Rel(cwd, localTestFile)
	if err != nil {
		t.Fatalf("unable to express fixture relative to cwd: %v", err)
	}
	resMixed := engine.DiscoverScope([]string{localTestFile, relTestFile}, adapter, nil)
	if resMixed.InputFailure {
		t.Errorf("mixed spellings must resolve to one existing file, got diagnostics %+v", resMixed.Diagnostics)
	}
	if resMixed.DiscoveredFiles != 1 {
		t.Errorf("expected 1 discovered file for mixed absolute and relative spelling, got %d", resMixed.DiscoveredFiles)
	}
	if len(resMixed.CandidateFiles) != 1 {
		t.Errorf("expected 1 candidate file for mixed spelling, got %d", len(resMixed.CandidateFiles))
	}
}

func TestDiscoverScopeDuplicateRootsGeneratedCountedOnce(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "model_test.dart"), []byte("// GENERATED CODE - DO NOT MODIFY BY HAND\nvoid main(){}\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "real_test.go"), []byte("package test\nfunc TestReal(t *testing.T){}\n"), 0644)

	adapter := dart.NewAdapter()
	res := engine.DiscoverScope([]string{tmpDir, tmpDir}, adapter, nil)

	if res.DiscoveredFiles != 2 {
		t.Errorf("expected 2 unique discovered candidates across duplicate roots, got %d", res.DiscoveredFiles)
	}
	if len(res.CandidateFiles) != 1 {
		t.Errorf("expected only the non-generated candidate, got %+v", res.CandidateFiles)
	}
	generated := 0
	for _, ex := range res.Excluded {
		if ex.Reason == "generated" {
			generated++
		}
	}
	if len(res.Excluded) != 1 || generated != 1 {
		t.Errorf("expected exactly one generated exclusion, got %+v", res.Excluded)
	}
}

func TestDiscoverScope_LinkedWorktreeAndDefaultDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// root/a_test.go
	_ = os.WriteFile(filepath.Join(tmpDir, "a_test.go"), []byte("package test\nfunc TestA(t *testing.T){}\n"), 0644)

	// root/wt/.git (file: gitdir: /x/.git/worktrees/wt)
	wtDir := filepath.Join(tmpDir, "wt")
	_ = os.MkdirAll(wtDir, 0755)
	_ = os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: /x/.git/worktrees/wt\n"), 0644)
	_ = os.WriteFile(filepath.Join(wtDir, "b_test.go"), []byte("package test\nfunc TestB(t *testing.T){}\n"), 0644)

	// root/vendor/c_test.go
	vendorDir := filepath.Join(tmpDir, "vendor")
	_ = os.MkdirAll(vendorDir, 0755)
	_ = os.WriteFile(filepath.Join(vendorDir, "c_test.go"), []byte("package test\nfunc TestC(t *testing.T){}\n"), 0644)

	// root/third_party/d/e.test.ts
	tpDir := filepath.Join(tmpDir, "third_party", "d")
	_ = os.MkdirAll(tpDir, 0755)
	_ = os.WriteFile(filepath.Join(tpDir, "e.test.ts"), []byte("it('e', () => { expect(1).toBe(1); });\n"), 0644)

	adapter := dart.NewAdapter()

	// 1. Scanning root: only a_test.go is a candidate
	res := engine.DiscoverScope([]string{tmpDir}, adapter, nil)
	if len(res.CandidateFiles) != 1 || filepath.Base(res.CandidateFiles[0]) != "a_test.go" {
		t.Fatalf("expected candidate files to contain only a_test.go, got %+v", res.CandidateFiles)
	}

	var hasLinkedWorktree, hasVendor, hasThirdParty bool
	for _, ex := range res.Excluded {
		if ex.Reason == "linked_worktree" && filepath.Base(ex.Path) == "wt" {
			hasLinkedWorktree = true
		}
		if ex.Reason == "default_directory" && filepath.Base(ex.Path) == "vendor" {
			hasVendor = true
		}
		if ex.Reason == "default_directory" && filepath.Base(ex.Path) == "third_party" {
			hasThirdParty = true
		}
	}
	if !hasLinkedWorktree {
		t.Errorf("expected linked_worktree exclusion for wt, got exclusions: %+v", res.Excluded)
	}
	if !hasVendor {
		t.Errorf("expected default_directory exclusion for vendor, got exclusions: %+v", res.Excluded)
	}
	if !hasThirdParty {
		t.Errorf("expected default_directory exclusion for third_party, got exclusions: %+v", res.Excluded)
	}

	// 2. Scanning root/wt directly as the root still analyzes b_test.go
	resWT := engine.DiscoverScope([]string{wtDir}, adapter, nil)
	if len(resWT.CandidateFiles) != 1 || filepath.Base(resWT.CandidateFiles[0]) != "b_test.go" {
		t.Fatalf("expected candidate files for direct wt scan to be b_test.go, got %+v", resWT.CandidateFiles)
	}
}
