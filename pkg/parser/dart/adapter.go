package dart

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	blocTestHeaderRegex = regexp.MustCompile(`(?s)\bblocTest\s*(?:<[^>]+>)?\s*\(\s*(?:"((?:\\.|[^"])+)"|'((?:\\.|[^'])+)')\s*,`)
	unitTestHeaderRegex = regexp.MustCompile(`(?s)\b(?:test|testWidgets)\s*\(\s*(?:"((?:\\.|[^"])+)"|'((?:\\.|[^'])+)')\s*,\s*(\(.*?\)\s*(?:async\*?\s*)?\{)`)
	seedRegex           = regexp.MustCompile(`(?s)\bseed\s*:\s*\(\)\s*=>\s*([^,]+),`)
	actRegex            = regexp.MustCompile(`(?s)\bact\s*:\s*\([^)]*\)\s*(?:=>|\{)\s*(.*?)(?:,\s*\w+\s*:|\}\s*,|\);)`)
	expectRegex         = regexp.MustCompile(`(?s)\bexpect\s*:\s*\(\)\s*=>\s*(\[.*?\])\s*,`)
	assertionRegex      = regexp.MustCompile(`\b(expect|expectLater|verify|verifyNever|verifyInOrder)\s*\(`)
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
	assertingHelpers := findAssertingHelpers(src)
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
	blocMatches := blocTestHeaderRegex.FindAllStringSubmatchIndex(src, -1)
	for _, m := range blocMatches {
		var testName string
		if m[2] != -1 {
			testName = src[m[2]:m[3]]
		} else if m[4] != -1 {
			testName = src[m[4]:m[5]]
		}

		// m[0] is the start of "blocTest". Find the opening paren of the call.
		openParenRel := strings.Index(src[m[0]:], "(")
		if openParenRel == -1 {
			continue
		}
		openParen := m[0] + openParenRel
		closeParen := parser.FindMatchingParen(src, openParen)
		if closeParen == -1 {
			continue
		}

		// argsBody is from after the comma following the test name (m[1]) up to closeParen
		argsBody := strings.TrimSpace(src[m[1]:closeParen])

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
			return nil, fmt.Errorf("malformed test callback: unmatched brace in %s starting at line %d", filePath, offsetToLine(openBrace))
		}

		body := src[cbStart : closeBrace+1]

		var assertions []string
		// Scan body for direct assertions and calls to asserting local helpers
		for i := 0; i < len(body); {
			next := skipCommentsAndStrings(body, i)
			if next != i {
				i = next
				continue
			}
			if isIdentStart(body[i]) {
				start := i
				for i < len(body) && isIdentPart(body[i]) {
					i++
				}
				ident := body[start:i]
				after := skipWhitespaceAndComments(body, i)
				if after < len(body) && body[after] == '(' {
					if isDirectAssertion(ident) {
						assertions = append(assertions, ident)
					} else if assertingHelpers[ident] && ident != testName {
						shadowRe := regexp.MustCompile(`(?:\b(?:var|final|const)\s+` + regexp.QuoteMeta(ident) + `\b|\b` + regexp.QuoteMeta(ident) + `\s*=)`)
						if !shadowRe.MatchString(stripCommentsAndStrings(body)) {
							assertions = append(assertions, "helper:"+ident)
						}
					}
				}
				continue
			}
			i++
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

var forbiddenHelpers = map[string]bool{
	"if":            true,
	"for":           true,
	"while":         true,
	"switch":        true,
	"catch":         true,
	"else":          true,
	"return":        true,
	"case":          true,
	"default":       true,
	"break":         true,
	"continue":      true,
	"do":            true,
	"try":           true,
	"finally":       true,
	"throw":         true,
	"rethrow":       true,
	"assert":        true,
	"new":           true,
	"const":         true,
	"var":           true,
	"final":         true,
	"late":          true,
	"class":         true,
	"mixin":         true,
	"extension":     true,
	"enum":          true,
	"typedef":       true,
	"import":        true,
	"export":        true,
	"part":          true,
	"library":       true,
	"with":          true,
	"on":            true,
	"as":            true,
	"is":            true,
	"in":            true,
	"main":          true,
	"test":          true,
	"testWidgets":   true,
	"blocTest":      true,
	"group":         true,
	"setUp":         true,
	"setUpAll":      true,
	"tearDown":      true,
	"tearDownAll":   true,
	"expect":        true,
	"expectLater":   true,
	"verify":        true,
	"verifyNever":   true,
	"verifyInOrder": true,
}

func isDirectAssertion(name string) bool {
	switch name {
	case "expect", "expectLater", "verify", "verifyNever", "verifyInOrder":
		return true
	default:
		return false
	}
}

func isValidHelperName(name string) bool {
	return name != "" && !forbiddenHelpers[name]
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func skipCommentsAndStrings(src string, i int) int {
	n := len(src)
	for i < n {
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
			i += 2
			continue
		}
		if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
			quote := src[i+1]
			if i+3 < n && src[i+2] == quote && src[i+3] == quote {
				i += 4
				for i+2 < n && !(src[i] == quote && src[i+1] == quote && src[i+2] == quote) {
					i++
				}
				i += 3
				continue
			}
			i += 2
			for i < n && src[i] != quote && src[i] != '\n' {
				i++
			}
			if i < n && src[i] == quote {
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
					i += 3
					break
				}
				i++
			}
			continue
		}
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
					if i < n {
						i++
						continue
					}
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			continue
		}
		break
	}
	return i
}

func skipWhitespaceAndComments(src string, i int) int {
	n := len(src)
	for i < n {
		next := skipCommentsAndStrings(src, i)
		if next != i {
			i = next
			continue
		}
		if src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r' {
			i++
			continue
		}
		break
	}
	return i
}

func findMatchingParen(src string, openParen int) int {
	if openParen < 0 || openParen >= len(src) || src[openParen] != '(' {
		return -1
	}
	depth := 0
	n := len(src)
	for i := openParen; i < n; {
		next := skipCommentsAndStrings(src, i)
		if next != i {
			i = next
			continue
		}
		if src[i] == '(' {
			depth++
		} else if src[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

func findMatchingAngleBracket(src string, openAngle int) int {
	if openAngle < 0 || openAngle >= len(src) || src[openAngle] != '<' {
		return -1
	}
	depth := 0
	n := len(src)
	for i := openAngle; i < n; {
		next := skipCommentsAndStrings(src, i)
		if next != i {
			i = next
			continue
		}
		if src[i] == '<' {
			depth++
		} else if src[i] == '>' {
			depth--
			if depth == 0 {
				return i
			}
		} else if src[i] == ';' || src[i] == '{' || src[i] == '}' || src[i] == '(' || src[i] == ')' {
			return -1
		}
		i++
	}
	return -1
}

func findSemicolon(src string, start int) int {
	n := len(src)
	for i := start; i < n; {
		next := skipCommentsAndStrings(src, i)
		if next != i {
			i = next
			continue
		}
		if src[i] == ';' {
			return i
		}
		i++
	}
	return -1
}

func stripComments(code string) string {
	var buf strings.Builder
	n := len(code)
	for i := 0; i < n; {
		if i+1 < n && code[i] == '/' && code[i+1] == '/' {
			i += 2
			for i < n && code[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < n && code[i] == '/' && code[i+1] == '*' {
			i += 2
			for i+1 < n && !(code[i] == '*' && code[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		if (code[i] == 'r' || code[i] == 'R') && i+1 < n && (code[i+1] == '\'' || code[i+1] == '"') {
			quote := code[i+1]
			if i+3 < n && code[i+2] == quote && code[i+3] == quote {
				buf.WriteString(code[i : i+4])
				i += 4
				for i+2 < n && !(code[i] == quote && code[i+1] == quote && code[i+2] == quote) {
					buf.WriteByte(code[i])
					i++
				}
				if i+2 < n {
					buf.WriteString(code[i : i+3])
					i += 3
				}
				continue
			}
			buf.WriteString(code[i : i+2])
			i += 2
			for i < n && code[i] != quote && code[i] != '\n' {
				buf.WriteByte(code[i])
				i++
			}
			if i < n && code[i] == quote {
				buf.WriteByte(code[i])
				i++
			}
			continue
		}
		if (code[i] == '\'' || code[i] == '"') && i+2 < n && code[i+1] == code[i] && code[i+2] == code[i] {
			quote := code[i]
			buf.WriteString(code[i : i+3])
			i += 3
			for i+2 < n {
				if code[i] == '\\' {
					buf.WriteByte(code[i])
					if i+1 < n {
						buf.WriteByte(code[i+1])
						i += 2
						continue
					}
				}
				if code[i] == quote && code[i+1] == quote && code[i+2] == quote {
					buf.WriteString(code[i : i+3])
					i += 3
					break
				}
				buf.WriteByte(code[i])
				i++
			}
			continue
		}
		if code[i] == '\'' || code[i] == '"' || code[i] == '`' {
			quote := code[i]
			buf.WriteByte(quote)
			i++
			for i < n && code[i] != quote {
				if code[i] == '\\' {
					buf.WriteByte(code[i])
					if i+1 < n {
						buf.WriteByte(code[i+1])
						i += 2
						continue
					}
				} else if code[i] == '\n' && quote != '`' {
					break
				}
				buf.WriteByte(code[i])
				i++
			}
			if i < n && code[i] == quote {
				buf.WriteByte(quote)
				i++
			}
			continue
		}
		buf.WriteByte(code[i])
		i++
	}
	return buf.String()
}

func stripCommentsAndStrings(code string) string {
	b := []byte(stripComments(code))
	n := len(b)
	for i := 0; i < n; {
		if b[i] == '\'' || b[i] == '"' {
			quote := b[i]
			b[i] = ' '
			i++
			for i < n {
				if b[i] == '\\' {
					b[i] = ' '
					if i+1 < n && b[i+1] != '\n' && b[i+1] != '\r' {
						b[i+1] = ' '
					}
					i += 2
					continue
				}
				if b[i] == quote {
					b[i] = ' '
					i++
					break
				}
				if b[i] != '\n' && b[i] != '\r' {
					b[i] = ' '
				}
				i++
			}
			continue
		}
		i++
	}
	return string(b)
}

func findAssertingHelpers(src string) map[string]bool {
	assertingHelpers := make(map[string]bool)
	n := len(src)
	for i := 0; i < n; {
		next := skipCommentsAndStrings(src, i)
		if next != i {
			i = next
			continue
		}
		if isIdentStart(src[i]) {
			identStart := i
			for i < n && isIdentPart(src[i]) {
				i++
			}
			name := src[identStart:i]

			j := skipWhitespaceAndComments(src, i)
			if j < n && src[j] == '<' {
				if k := findMatchingAngleBracket(src, j); k != -1 {
					j = skipWhitespaceAndComments(src, k+1)
				}
			}
			if j < n && src[j] == '(' {
				parenClose := findMatchingParen(src, j)
				if parenClose != -1 {
					k := skipWhitespaceAndComments(src, parenClose+1)
					if strings.HasPrefix(src[k:], "async*") {
						k = skipWhitespaceAndComments(src, k+6)
					} else if strings.HasPrefix(src[k:], "async") && (k+5 >= n || !isIdentPart(src[k+5])) {
						k = skipWhitespaceAndComments(src, k+5)
					} else if strings.HasPrefix(src[k:], "sync*") {
						k = skipWhitespaceAndComments(src, k+5)
					}

					if k < n && src[k] == '{' {
						openBrace := k
						closeBrace := parser.FindMatchingBrace(src, openBrace)
						if closeBrace != -1 {
							body := src[openBrace : closeBrace+1]
							if isValidHelperName(name) && assertionRegex.MatchString(stripCommentsAndStrings(body)) {
								assertingHelpers[name] = true
							}
							i = openBrace + 1
							continue
						}
					} else if k+1 < n && src[k] == '=' && src[k+1] == '>' {
						arrowStart := k + 2
						semiIdx := findSemicolon(src, arrowStart)
						if semiIdx != -1 {
							body := src[arrowStart:semiIdx]
							if isValidHelperName(name) && assertionRegex.MatchString(stripCommentsAndStrings(body)) {
								assertingHelpers[name] = true
							}
							i = semiIdx + 1
							continue
						}
					}
					// Matched call span - skip over it to maintain linear time
					i = parenClose + 1
					continue
				}
			}
			continue
		}
		i++
	}
	return assertingHelpers
}
