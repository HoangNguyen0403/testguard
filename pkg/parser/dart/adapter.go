package dart

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	blocTestRegex       = regexp.MustCompile(`(?s)blocTest\s*(?:<[^>]+>)?\s*\(\s*['"]([^'"]+)['"]\s*,\s*(.*?)\);\s*(?:\n|$)`)
	unitTestHeaderRegex = regexp.MustCompile(`(?s)\b(?:test|testWidgets)\s*\(\s*(?:"((?:\\.|[^"])+)"|'((?:\\.|[^'])+)')\s*,\s*(\(.*?\)\s*(?:async\*?\s*)?\{)`)
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
	lineOffsets := []int{0}
	for i := range len(src) {
		if src[i] == '\n' {
			lineOffsets = append(lineOffsets, i+1)
		}
	}
	offsetToLine := func(byteOffset int) int {
		low, high := 0, len(lineOffsets)-1
		line := 1
		for low <= high {
			mid := (low + high) / 2
			if lineOffsets[mid] <= byteOffset {
				line = mid + 1
				low = mid + 1
			} else {
				high = mid - 1
			}
		}
		return line
	}

	// 1. Locate blocTest invocations
	blocMatches := blocTestRegex.FindAllStringSubmatchIndex(src, -1)
	for _, m := range blocMatches {
		testName := src[m[2]:m[3]]
		argsBody := src[m[4]:m[5]]

		unit := parser.TestUnit{
			Name:       testName,
			FilePath:   filePath,
			StartLine:  offsetToLine(m[0]),
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

		var testName string
		if loc[2] != -1 {
			testName = src[offset+loc[2] : offset+loc[3]]
		} else {
			testName = src[offset+loc[4] : offset+loc[5]]
		}

		cbStart := offset + loc[6]
		openBrace := offset + loc[7] - 1

		closeBrace := parser.FindMatchingBrace(src, openBrace)
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
			StartLine:  offsetToLine(offset + loc[0]),
			Kind:       "unit",
			RawBody:    body,
			Assertions: assertions,
		}

		units = append(units, unit)
		offset = closeBrace + 1
	}

	return units, nil
}

