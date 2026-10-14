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
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "build" {
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
		}

		return nil
	})

	return modules, err
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
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(d.Name(), "_test.dart") || strings.HasSuffix(d.Name(), "_test.go") || strings.HasSuffix(d.Name(), ".test.ts") || strings.HasSuffix(d.Name(), ".spec.ts") {
			// Check if generated
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
