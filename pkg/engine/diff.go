package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ResolveGitRepoRoot locates the root of the git repository containing dir.
// Returns the absolute root path, or an error if git is not installed or dir is not in a repository.
func ResolveGitRepoRoot(dir string) (string, error) {
	if dir == "" {
		dir = "."
	}
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git_unavailable: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func addGitLines(repoRoot string, gitArgs []string, out map[string]struct{}) error {
	args := append([]string{"-C", repoRoot}, gitArgs...)
	b, err := exec.Command("git", args...).Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(bytes.TrimSpace(b)), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out[trimmed] = struct{}{}
		}
	}
	return nil
}

// CollectDiffFiles discovers modified, newly added, and untracked files in repoRoot.
// When baseRef is provided, files are diffed against baseRef...HEAD alongside working tree and untracked changes.
// When baseRef is omitted, uncommitted changes are collected, falling back to HEAD~1...HEAD if the working tree is clean.
// Returns sorted absolute file paths, excluding directories and nonexistent files.
func CollectDiffFiles(repoRoot, baseRef string) ([]string, error) {
	fileSet := make(map[string]struct{})
	if baseRef != "" {
		if err := addGitLines(repoRoot, []string{"diff", "--name-only", "--diff-filter=ACMR", baseRef + "...HEAD"}, fileSet); err != nil {
			return nil, fmt.Errorf("git_diff_failed: %w", err)
		}
		_ = addGitLines(repoRoot, []string{"diff", "--name-only", "--diff-filter=ACMR", "HEAD"}, fileSet)
		_ = addGitLines(repoRoot, []string{"ls-files", "--others", "--exclude-standard"}, fileSet)
	} else {
		_ = addGitLines(repoRoot, []string{"diff", "--name-only", "--diff-filter=ACMR", "HEAD"}, fileSet)
		_ = addGitLines(repoRoot, []string{"ls-files", "--others", "--exclude-standard"}, fileSet)
		if len(fileSet) == 0 {
			_ = addGitLines(repoRoot, []string{"diff", "--name-only", "--diff-filter=ACMR", "HEAD~1...HEAD"}, fileSet)
		}
	}

	var result []string
	for relPath := range fileSet {
		absPath := filepath.Join(repoRoot, relPath)
		if fi, err := os.Stat(absPath); err == nil && !fi.IsDir() {
			result = append(result, absPath)
		}
	}
	sort.Strings(result)
	return result, nil
}
