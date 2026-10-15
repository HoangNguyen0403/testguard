package engine

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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

// isTestFileName checks if a file name matches any supported test naming convention.
func isTestFileName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, "_test.dart") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".test.tsx") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".spec.tsx") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".test.jsx") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.HasSuffix(lower, ".spec.jsx") ||
		strings.HasSuffix(lower, "test.kt") ||
		strings.HasSuffix(lower, "tests.kt") ||
		strings.HasSuffix(lower, "test.swift") ||
		strings.HasSuffix(lower, "tests.swift")
}

// FindTestFiles searches for test files under a given directory for a language adapter.
func FindTestFiles(dir string, adapter parser.LanguageAdapter) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "build" || name == "dist") {
				return filepath.SkipDir
			}
			return nil
		}

		if isTestFileName(d.Name()) {
			header := make([]byte, 256)
			f, err := os.Open(path)
			if err == nil {
				_, _ = f.Read(header)
				f.Close()
			}
			if !adapter.IsGenerated(path, header) {
				files = append(files, path)
			}
		}

		return nil
	})

	return files, err
}
