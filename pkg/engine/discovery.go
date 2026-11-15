package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

// DiscoverModules walks the root directory and locates modules containing test files.
func DiscoverModules(rootDir string) ([]parser.ModuleRoot, error) {
	var modules []parser.ModuleRoot

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if path != rootDir && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "build" || name == "dist") {
				return filepath.SkipDir
			}

			return nil
		}

		dir := filepath.Dir(path)
		switch d.Name() {
		case "pubspec.yaml":
			modules = append(modules, parser.ModuleRoot{
				Path:     dir,
				Language: "dart",
				HasTests: true,
			})
		case "go.mod":
			modules = append(modules, parser.ModuleRoot{
				Path:     dir,
				Language: "go",
				HasTests: true,
			})
		case "package.json":
			modules = append(modules, parser.ModuleRoot{
				Path:     dir,
				Language: "typescript",
				HasTests: true,
			})
		case "build.gradle", "build.gradle.kts":
			modules = append(modules, parser.ModuleRoot{
				Path:     dir,
				Language: "kotlin",
				HasTests: true,
			})
		case "Package.swift":
			modules = append(modules, parser.ModuleRoot{
				Path:     dir,
				Language: "swift",
				HasTests: true,
			})
		}

		return nil
	})

	return modules, err
}

// isLinkedGitWorktree checks if dir contains a .git regular file whose content starts with "gitdir:" and contains "/worktrees/".
func isLinkedGitWorktree(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	fi, err := os.Lstat(gitPath)
	if err != nil || fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		return false
	}
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return false
	}
	content := strings.TrimSpace(string(data))
	return strings.HasPrefix(content, "gitdir:") && strings.Contains(content, "/worktrees/")
}

// isGeneratedByAdapter reports whether the adapter that owns a file's extension
// considers it generated. Discovery therefore needs no language switch and no
// adapter can be skipped silently: an extension with no owner is never a
// candidate because isSupportedLanguageExtension rejects it first.
func isGeneratedByAdapter(adapters []parser.LanguageAdapter, filePath string, header []byte) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, adapter := range adapters {
		for _, declared := range adapter.FileExtensions() {
			if ext == declared {
				return adapter.IsGenerated(filePath, header)
			}
		}
	}
	return false
}

var (
	// testFileInfixes are the dot-delimited role markers that precede a test
	// file's extension (foo.test.ts, login.cy.ts, flow.e2e-spec.ts).
	testFileInfixes = []string{".test", ".spec", ".cy", ".e2e", ".e2e-spec"}

	// testFileExtensions are the script extensions that may follow an infix.
	testFileExtensions = []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".mts", ".cts"}
)

// isTestFileName checks if a file name matches any supported test naming convention.
func isTestFileName(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "_test.dart") || strings.HasSuffix(lower, "_test.go") {
		return true
	}
	if strings.HasSuffix(lower, "test.kt") || strings.HasSuffix(lower, "tests.kt") {
		return true
	}
	if strings.HasSuffix(lower, "test.swift") || strings.HasSuffix(lower, "tests.swift") {
		return true
	}
	for _, infix := range testFileInfixes {
		for _, ext := range testFileExtensions {
			if strings.HasSuffix(lower, infix+ext) {
				return true
			}
		}
	}
	return false
}

// isSupportedLanguageExtension returns true if the extension belongs to a supported language.
func isSupportedLanguageExtension(ext string) bool {
	lower := strings.ToLower(ext)
	switch lower {
	case ".dart", ".go", ".kt", ".swift":
		return true
	}
	for _, supported := range testFileExtensions {
		if lower == supported {
			return true
		}
	}
	return false
}

// normalizePath converts a path to a canonical cwd-relative slash path.
func normalizePath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// canonicalAbsPath returns the absolute cleaned path used for deduplication.
func canonicalAbsPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

// DiscoveryResult contains files and metadata from scope resolution.
type DiscoveryResult struct {
	CandidateFiles  []string
	Excluded        []models.ExcludedPath
	Unsupported     []string
	Diagnostics     []models.Diagnostic
	DiscoveredFiles int
	InputFailure    bool
}

// DiscoverScope inspects target paths and resolves candidates, exclusions, and unsupported inputs.
// The adapter list supplies the per-language generated-file policy.
func DiscoverScope(paths []string, adapters []parser.LanguageAdapter, failTraversal func(path string) error) *DiscoveryResult {
	result := &DiscoveryResult{
		CandidateFiles: make([]string, 0),
		Excluded:       make([]models.ExcludedPath, 0),
		Unsupported:    make([]string, 0),
		Diagnostics:    make([]models.Diagnostic, 0),
	}

	seenDiscovered := make(map[string]bool)
	excludedMap := make(map[string]bool)
	unsupportedMap := make(map[string]bool)

	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, p := range paths {
		if failTraversal != nil {
			if ferr := failTraversal(p); ferr != nil {
				result.InputFailure = true
				result.Diagnostics = append(result.Diagnostics, models.NewDiagnostic("traversal_failed", p, ferr.Error()))
				continue
			}
		}

		trimmed := strings.TrimRight(p, string(filepath.Separator)+"/")
		if trimmed == "" {
			trimmed = p
		}
		fi, err := os.Lstat(trimmed)
		if err != nil {
			result.InputFailure = true
			result.Diagnostics = append(result.Diagnostics, models.NewDiagnostic("target_unreadable", p, err.Error()))
			continue
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			result.InputFailure = true
			result.Diagnostics = append(result.Diagnostics, models.NewDiagnostic("target_unreadable", p, fmt.Sprintf("target %q is a symlink; symlinks are not supported", p)))
			continue
		}

		if !fi.IsDir() {
			ext := filepath.Ext(trimmed)
			if !isSupportedLanguageExtension(ext) {
				norm := normalizePath(p)
				if !unsupportedMap[norm] {
					unsupportedMap[norm] = true
					result.Unsupported = append(result.Unsupported, norm)
				}
				result.Diagnostics = append(result.Diagnostics, models.NewDiagnostic("unsupported_input", norm, fmt.Sprintf("unsupported file extension %q for target %s", ext, p)))
				continue
			}

			if !isTestFileName(filepath.Base(trimmed)) {
				norm := normalizePath(p)
				if !excludedMap[norm] {
					excludedMap[norm] = true
					result.Excluded = append(result.Excluded, models.ExcludedPath{Path: norm, Reason: "non_test"})
				}
				continue
			}

			canon := canonicalAbsPath(trimmed)
			if !seenDiscovered[canon] {
				seenDiscovered[canon] = true
				result.DiscoveredFiles++

				header := make([]byte, 256)
				f, oerr := os.Open(trimmed)
				if oerr == nil {
					_, _ = f.Read(header)
					f.Close()
				}
				if isGeneratedByAdapter(adapters, trimmed, header) {
					norm := normalizePath(trimmed)
					if !excludedMap[norm] {
						excludedMap[norm] = true
						result.Excluded = append(result.Excluded, models.ExcludedPath{Path: norm, Reason: "generated"})
					}
				} else {
					result.CandidateFiles = append(result.CandidateFiles, trimmed)
				}
			}
			continue
		}

		// Directory traversal
		walkErr := filepath.WalkDir(trimmed, func(path string, d fs.DirEntry, werr error) error {
			if failTraversal != nil {
				if ferr := failTraversal(path); ferr != nil {
					result.InputFailure = true
					result.Diagnostics = append(result.Diagnostics, models.NewDiagnostic("traversal_failed", normalizePath(path), ferr.Error()))
					return nil
				}
			}
			if werr != nil {
				result.InputFailure = true
				result.Diagnostics = append(result.Diagnostics, models.NewDiagnostic("traversal_failed", normalizePath(path), werr.Error()))
				return nil
			}

			if d.IsDir() {
				if path != trimmed {
					name := d.Name()
					if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "third_party" || name == "build" || name == "dist" {
						norm := normalizePath(path)
						if !excludedMap[norm] {
							excludedMap[norm] = true
							result.Excluded = append(result.Excluded, models.ExcludedPath{Path: norm, Reason: "default_directory"})
						}
						return filepath.SkipDir
					}
					if isLinkedGitWorktree(path) {
						norm := normalizePath(path)
						if !excludedMap[norm] {
							excludedMap[norm] = true
							result.Excluded = append(result.Excluded, models.ExcludedPath{Path: norm, Reason: "linked_worktree"})
						}
						return filepath.SkipDir
					}
					if d.Type()&fs.ModeSymlink != 0 {
						return filepath.SkipDir
					}
				}
				return nil
			}

			if d.Type()&fs.ModeSymlink != 0 {
				return nil
			}

			if isTestFileName(d.Name()) {
				canon := canonicalAbsPath(path)
				if !seenDiscovered[canon] {
					seenDiscovered[canon] = true
					result.DiscoveredFiles++

					header := make([]byte, 256)
					f, oerr := os.Open(path)
					if oerr == nil {
						_, _ = f.Read(header)
						f.Close()
					}
					if isGeneratedByAdapter(adapters, path, header) {
						norm := normalizePath(path)
						if !excludedMap[norm] {
							excludedMap[norm] = true
							result.Excluded = append(result.Excluded, models.ExcludedPath{Path: norm, Reason: "generated"})
						}
					} else {
						result.CandidateFiles = append(result.CandidateFiles, path)
					}
				}
			}
			return nil
		})
		if walkErr != nil {
			result.InputFailure = true
			result.Diagnostics = append(result.Diagnostics, models.NewDiagnostic("traversal_failed", normalizePath(trimmed), walkErr.Error()))
		}
	}

	sort.Strings(result.Unsupported)
	sort.Slice(result.Excluded, func(i, j int) bool {
		return result.Excluded[i].Path < result.Excluded[j].Path
	})
	sort.Strings(result.CandidateFiles)

	return result
}
