package parser

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	e2eSuffixes = []string{
		".e2e.ts",
		".e2e.tsx",
		".e2e.js",
		".e2e-spec.ts",
		".cy.ts",
		".cy.js",
	}

	tsjsImportRegex = regexp.MustCompile(`(?m)(?:` +
		`\bimport\s+['"` + "`" + `](?:@playwright/test|cypress)(?:/[^'"` + "`" + `]*)?['"` + "`" + `]` +
		`|` +
		`\bfrom\s*['"` + "`" + `](?:@playwright/test|cypress)(?:/[^'"` + "`" + `]*)?['"` + "`" + `]` +
		`|` +
		`\b(?:require|import)\s*\(\s*['"` + "`" + `](?:@playwright/test|cypress)(?:/[^'"` + "`" + `]*)?['"` + "`" + `]\s*\)` +
		`)`)

	dartIntegrationBinding = []byte("IntegrationTestWidgetsFlutterBinding")
)

// IsE2EFile reports whether a test file is an end-to-end / integration-UI test
// rather than a unit test.
func IsE2EFile(filePath string, src []byte) bool {
	// 1. Path segment (slash-normalized, case-sensitive) equals integration_test, e2e, playwright, or cypress
	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
	segments := strings.Split(normalizedPath, "/")
	for _, seg := range segments {
		switch seg {
		case "integration_test", "e2e", "playwright", "cypress":
			return true
		}
	}

	// 2. File name ends with .e2e.ts, .e2e.tsx, .e2e.js, .e2e-spec.ts, .cy.ts, or .cy.js
	fileName := filepath.Base(normalizedPath)
	for _, suffix := range e2eSuffixes {
		if strings.HasSuffix(fileName, suffix) {
			return true
		}
	}

	if len(src) == 0 {
		return false
	}

	// 3. (TS/JS) src imports @playwright/test or cypress
	if isTSJSFile(filePath) || filePath == "" {
		if tsjsImportRegex.Match(src) {
			return true
		}
	}

	// 4. (Dart) src contains IntegrationTestWidgetsFlutterBinding
	if isDartFile(filePath) || filePath == "" {
		if bytes.Contains(src, dartIntegrationBinding) {
			return true
		}
	}

	return false
}

func isTSJSFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func isDartFile(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".dart"
}
