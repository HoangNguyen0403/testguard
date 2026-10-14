package dart

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	blocTestRegex = regexp.MustCompile(`(?s)blocTest\s*(?:<[^>]+>)?\s*\(\s*['"]([^'"]+)['"]\s*,\s*(.*?)\);\s*(?:\n|$)`)
	unitTestHeaderRegex = regexp.MustCompile(`(?s)\btest\s*\(\s*['"]((?:\\.|[^'"])+)['"]\s*,\s*(\(.*?\)\s*(?:async\*?\s*)?\{)`)
	seedRegex     = regexp.MustCompile(`(?s)\bseed\s*:\s*\(\)\s*=>\s*([^,]+),`)
	actRegex      = regexp.MustCompile(`(?s)\bact\s*:\s*\([^)]*\)\s*(?:=>|\{)\s*(.*?)(?:,\s*\w+\s*:|\}\s*,|\);)`)
	expectRegex   = regexp.MustCompile(`(?s)\bexpect\s*:\s*\(\)\s*=>\s*(\[.*?\])\s*,`)
	assertionRegex = regexp.MustCompile(`\b(expect|expectLater|verify|verifyNever|verifyInOrder)\s*\(`)
)

// Adapter implements the parser.LanguageAdapter for Dart.
type Adapter struct{}

// NewAdapter creates a new Dart LanguageAdapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() string {
	return "dart"
}

func (a *Adapter) FileExtensions() []string {
	return []string{".dart"}
}

func (a *Adapter) IsGenerated(filePath string, header []byte) bool {
	if strings.HasSuffix(filePath, ".g.dart") ||
		strings.HasSuffix(filePath, ".freezed.dart") ||
		strings.HasSuffix(filePath, ".gr.dart") {
		return true
	}
	if bytes.Contains(header, []byte("GENERATED CODE")) ||
		bytes.Contains(header, []byte("DO NOT MODIFY BY HAND")) {
		return true
	}
	return false
}

func (a *Adapter) LocateTests(source []byte, filePath string) ([]parser.TestUnit, error) {
	var units []parser.TestUnit
	src := string(source)
	lines := strings.Split(src, "\n")

	findLine := func(substr string) int {
		for i, l := range lines {
			if strings.Contains(l, substr) {
				return i + 1
			}
		}
		return 1
	}

	// 1. Locate blocTest invocations
	blocMatches := blocTestRegex.FindAllStringSubmatchIndex(src, -1)
	for _, m := range blocMatches {
		testName := src[m[2]:m[3]]
		argsBody := src[m[4]:m[5]]

		unit := parser.TestUnit{
			Name:       testName,
			FilePath:   filePath,
			StartLine:  findLine(testName),
			Kind:       "bloc",
			RawBody:    argsBody,
			Assertions: []string{"blocTest.expect"},
		}

		if seedM := seedRegex.FindStringSubmatch(argsBody); len(seedM) > 1 {
			unit.SeedBody = strings.TrimSpace(seedM[1])
		}
		if actM := actRegex.FindStringSubmatch(argsBody); len(actM) > 1 {
			unit.ActBody = strings.TrimSpace(actM[1])
		}
		if expM := expectRegex.FindStringSubmatch(argsBody); len(expM) > 1 {
			unit.ExpectBody = strings.TrimSpace(expM[1])
		}

		units = append(units, unit)
	}

	// Locate standard test() invocations
	for offset := 0; offset < len(src); {
		loc := unitTestHeaderRegex.FindStringSubmatchIndex(src[offset:])
		if loc == nil {
			break
		}

		testName := src[offset+loc[2] : offset+loc[3]]
		cbStart := offset + loc[4]
		openBrace := offset + loc[5] - 1

		closeBrace := findMatchingBrace(src, openBrace)
		if closeBrace == -1 {
			offset += loc[1]
			continue
		}

		body := src[cbStart : closeBrace+1]

		var assertions []string
		assertMatches := assertionRegex.FindAllString(body, -1)
		for _, a := range assertMatches {
			assertions = append(assertions, strings.TrimSuffix(a, "("))
		}

		unit := parser.TestUnit{
			Name:       testName,
			FilePath:   filePath,
			StartLine:  findLine(testName),
			Kind:       "unit",
			RawBody:    body,
			Assertions: assertions,
		}

		units = append(units, unit)
		offset = closeBrace + 1
	}

	return units, nil
}

// findMatchingBrace returns the index of the matching closing brace for the
// opening curly brace at openIndex, correctly skipping string literals and comments.
func findMatchingBrace(src string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(src) || src[openIndex] != '{' {
		return -1
	}

	braceCount := 0
	n := len(src)

	for i := openIndex; i < n; i++ {
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
			continue
		}

		if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
			quote := src[i+1]
			if i+3 < n && src[i+2] == quote && src[i+3] == quote {
				i += 4
				for i+2 < n && !(src[i] == quote && src[i+1] == quote && src[i+2] == quote) {
					i++
				}
				i += 2
				continue
			}
			i += 2
			for i < n && src[i] != quote && src[i] != '\n' {
				i++
			}
			continue
		}

		if (src[i] == '\'' || src[i] == '"') && i+2 < n && src[i+1] == src[i] && src[i+2] == src[i] {
			quote := src[i]
			i += 3
			for i+2 < n {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if src[i] == quote && src[i+1] == quote && src[i+2] == quote {
					i += 2
					break
				}
				i++
			}
			continue
		}

		if src[i] == '\'' || src[i] == '"' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' {
					break
				}
				i++
			}
			continue
		}

		if src[i] == '{' {
			braceCount++
		} else if src[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return i
			}
		}
	}

	return -1
}
