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
	assertionRegex      = regexp.MustCompile(`\b(expect|expectLater|verify|verifyNever|verifyInOrder|verifyZeroInteractions|verifyNoMoreInteractions|fail|expectAsync[0-6]|expectAsyncUntil[0-6])\s*\(`)
)

// MetaBlocStateType is the TestUnit.Meta key holding the state type argument S from blocTest<B, S>.
const MetaBlocStateType = "bloc_state_type"

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
	masked := maskCommentsAndStrings(src)
	assertingHelpers := findAssertingHelpers(src)
	groups := findDartGroups(src)
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
	blocMatches := blocTestHeaderRegex.FindAllStringSubmatchIndex(masked, -1)
	for _, m := range blocMatches {
		var testName string
		if m[2] != -1 {
			testName = src[m[2]:m[3]]
		} else if m[4] != -1 {
			testName = src[m[4]:m[5]]
		}

		// m[0] is the start of "blocTest". Find the opening paren of the call.
		openParenRel := strings.Index(masked[m[0]:], "(")
		if openParenRel == -1 {
			continue
		}
		openParen := m[0] + openParenRel
		closeParen := findMatchingParen(src, openParen)
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
			Scope:      computeScope(groups, m[0]),
		}

		if stateType := extractBlocStateType(src, m[0]); stateType != "" {
			unit.Meta = map[string]string{
				MetaBlocStateType: stateType,
			}
		}

		if seedVal := extractNamedArg(argsBody, "seed"); seedVal != "" {
			unit.SeedBody = seedVal
		} else if seedM := seedRegex.FindStringSubmatch(argsBody); len(seedM) > 1 {
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
		loc := unitTestHeaderRegex.FindStringSubmatchIndex(masked[offset:])
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

		closeBrace := findMatchingBrace(src, openBrace)
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
			Scope:      computeScope(groups, offset+loc[0]),
		}

		units = append(units, unit)
		offset = closeBrace + 1
	}

	return units, nil
}

var forbiddenHelpers = map[string]bool{
	"if":                       true,
	"for":                      true,
	"while":                    true,
	"switch":                   true,
	"catch":                    true,
	"else":                     true,
	"return":                   true,
	"case":                     true,
	"default":                  true,
	"break":                    true,
	"continue":                 true,
	"do":                       true,
	"try":                      true,
	"finally":                  true,
	"throw":                    true,
	"rethrow":                  true,
	"assert":                   true,
	"new":                      true,
	"const":                    true,
	"var":                      true,
	"final":                    true,
	"late":                     true,
	"class":                    true,
	"mixin":                    true,
	"extension":                true,
	"enum":                     true,
	"typedef":                  true,
	"import":                   true,
	"export":                   true,
	"part":                     true,
	"library":                  true,
	"with":                     true,
	"on":                       true,
	"as":                       true,
	"is":                       true,
	"in":                       true,
	"main":                     true,
	"test":                     true,
	"testWidgets":              true,
	"blocTest":                 true,
	"group":                    true,
	"setUp":                    true,
	"setUpAll":                 true,
	"tearDown":                 true,
	"tearDownAll":              true,
	"expect":                   true,
	"expectLater":              true,
	"verify":                   true,
	"verifyNever":              true,
	"verifyInOrder":            true,
	"verifyZeroInteractions":   true,
	"verifyNoMoreInteractions": true,
	"fail":                     true,
	"expectAsync0":             true,
	"expectAsync1":             true,
	"expectAsync2":             true,
	"expectAsync3":             true,
	"expectAsync4":             true,
	"expectAsync5":             true,
	"expectAsync6":             true,
	"expectAsyncUntil0":        true,
	"expectAsyncUntil1":        true,
	"expectAsyncUntil2":        true,
	"expectAsyncUntil3":        true,
	"expectAsyncUntil4":        true,
	"expectAsyncUntil5":        true,
	"expectAsyncUntil6":        true,
}

func isDirectAssertion(name string) bool {
	switch name {
	case "expect", "expectLater", "verify", "verifyNever", "verifyInOrder",
		"verifyZeroInteractions", "verifyNoMoreInteractions", "fail",
		"expectAsync0", "expectAsync1", "expectAsync2", "expectAsync3", "expectAsync4", "expectAsync5", "expectAsync6",
		"expectAsyncUntil0", "expectAsyncUntil1", "expectAsyncUntil2", "expectAsyncUntil3", "expectAsyncUntil4", "expectAsyncUntil5", "expectAsyncUntil6":
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

// skipCommentsAndStrings advances past any run of comments and string
// literals starting at i and returns the first index that starts neither.
func skipCommentsAndStrings(src string, i int) int {
	n := len(src)
	for i < n {
		next := skipDartLexeme(src, i)
		if next == i {
			return i
		}
		i = next
	}
	return i
}

// skipDartLexeme returns the index just past the comment or string literal
// that starts at i, or i itself when src[i] opens neither. Dart string
// literals are interpolation-aware so a literal such as
// '${m['}']} end' stays a single lexeme.
func skipDartLexeme(src string, i int) int {
	n := len(src)
	if i >= n {
		return i
	}
	if i+1 < n && src[i] == '/' && src[i+1] == '/' {
		i += 2
		for i < n && src[i] != '\n' {
			i++
		}
		return i
	}
	if i+1 < n && src[i] == '/' && src[i+1] == '*' {
		return skipDartBlockComment(src, i)
	}
	if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
		return scanDartString(src, i+1, true)
	}
	if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
		return scanDartString(src, i, false)
	}
	return i
}

// skipDartBlockComment returns the index just past the block comment starting
// at start. Dart block comments nest.
func skipDartBlockComment(src string, start int) int {
	n := len(src)
	depth := 1
	i := start + 2
	for i < n {
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			depth++
			i += 2
			continue
		}
		if i+1 < n && src[i] == '*' && src[i+1] == '/' {
			depth--
			i += 2
			if depth == 0 {
				return i
			}
			continue
		}
		i++
	}
	return n
}

// scanDartString returns the index just past the string literal whose opening
// quote sits at quoteStart. Raw literals ignore escapes and interpolation;
// plain literals skip ${...} interpolations so quotes and braces nested inside
// them neither terminate the literal nor leak into the surrounding code.
// Unterminated single-line literals stop at the newline, matching Dart's rule
// that a non-triple literal cannot span lines.
func scanDartString(src string, quoteStart int, isRaw bool) int {
	n := len(src)
	if quoteStart >= n {
		return quoteStart
	}
	quote := src[quoteStart]
	delim := 1
	if quoteStart+2 < n && src[quoteStart+1] == quote && src[quoteStart+2] == quote {
		delim = 3
	}
	allowInterpolation := !isRaw && quote != '`'
	i := quoteStart + delim
	for i < n {
		if !isRaw && src[i] == '\\' {
			i += 2
			continue
		}
		if delim == 3 {
			if i+2 < n && src[i] == quote && src[i+1] == quote && src[i+2] == quote {
				return i + 3
			}
		} else if src[i] == quote {
			return i + 1
		}
		if allowInterpolation && src[i] == '$' && i+1 < n {
			if src[i+1] == '{' {
				i = skipDartInterpolation(src, i+1)
				continue
			}
			if isIdentStart(src[i+1]) {
				i += 2
				for i < n && isIdentPart(src[i]) {
					i++
				}
				continue
			}
		}
		if delim == 1 && src[i] == '\n' {
			return i
		}
		i++
	}
	return n
}

// skipDartInterpolation returns the index just past the brace that closes the
// ${...} interpolation opened at openBrace. Nested braces, comments and string
// literals inside the interpolation are skipped.
func skipDartInterpolation(src string, openBrace int) int {
	n := len(src)
	depth := 0
	for i := openBrace; i < n; {
		next := skipDartLexeme(src, i)
		if next != i {
			i = next
			continue
		}
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return n
}

// maskCommentsAndStrings returns src with every comment and string literal
// blanked while preserving byte offsets: comments become spaces, literal
// contents become filler that cannot form identifiers or delimiters, and line
// breaks survive. Callers locate declarations on the masked copy, then slice
// names and bodies out of the original source.
func maskCommentsAndStrings(src string) string {
	buf := []byte(src)
	n := len(src)
	for i := 0; i < n; {
		next := skipDartLexeme(src, i)
		if next == i {
			i++
			continue
		}
		blankDartLexeme(buf, src, i, next, ' ', 'x')
		i = next
	}
	return string(buf)
}

// blankDartLexeme overwrites the comment or string literal spanning
// src[start:end] inside buf, keeping quote delimiters, raw-string prefixes and
// line breaks intact so offsets and line numbers stay stable.
func blankDartLexeme(buf []byte, src string, start, end int, commentFiller, stringFiller byte) {
	if start < 0 || end > len(src) || start >= end {
		return
	}
	if src[start] == '/' {
		for k := start; k < end; k++ {
			if buf[k] != '\n' && buf[k] != '\r' {
				buf[k] = commentFiller
			}
		}
		return
	}
	open := start
	if buf[open] == 'r' || buf[open] == 'R' {
		open++
	}
	if open >= end {
		return
	}
	delim := 1
	if open+2 < end && buf[open+1] == buf[open] && buf[open+2] == buf[open] {
		delim = 3
	}
	contentStart := open + delim
	contentEnd := end
	if contentEnd-delim >= contentStart && buf[contentEnd-delim] == buf[open] {
		contentEnd -= delim
	}
	for k := contentStart; k < contentEnd; k++ {
		if buf[k] != '\n' && buf[k] != '\r' {
			buf[k] = stringFiller
		}
	}
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

func skipWhitespaceAndOnlyComments(src string, i int) int {
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
		if src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r' {
			i++
			continue
		}
		break
	}
	return i
}

type dartGroup struct {
	title string
	start int
	end   int
}

// readDartStringLiteral reads the string literal opening at start and returns
// its raw content together with the index just past the closing delimiter. It
// reports ("", start) when no terminated literal starts there, and understands
// raw and triple-quoted forms.
func readDartStringLiteral(src string, start int) (string, int) {
	n := len(src)
	if start >= n {
		return "", start
	}

	isRaw := false
	i := start
	if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
		isRaw = true
		i++
	}

	if i >= n || (src[i] != '\'' && src[i] != '"') {
		return "", start
	}

	quote := src[i]
	delim := 1
	if i+2 < n && src[i+1] == quote && src[i+2] == quote {
		delim = 3
	}
	end := scanDartString(src, i, isRaw)
	contentStart := i + delim
	contentEnd := end - delim
	if contentEnd < contentStart || contentEnd >= len(src) || src[contentEnd] != quote {
		return "", start
	}
	if delim == 3 && (contentEnd+2 >= len(src) || src[contentEnd+1] != quote || src[contentEnd+2] != quote) {
		return "", start
	}
	return src[contentStart:contentEnd], end
}

func extractNamedArg(argsBody, name string) string {
	n := len(argsBody)
	for i := 0; i < n; {
		next := skipCommentsAndStrings(argsBody, i)
		if next != i {
			i = next
			continue
		}
		if strings.HasPrefix(argsBody[i:], name) && (i == 0 || !isIdentPart(argsBody[i-1])) && (i+len(name) >= n || !isIdentPart(argsBody[i+len(name)])) {
			colon := skipWhitespaceAndOnlyComments(argsBody, i+len(name))
			if colon < n && argsBody[colon] == ':' {
				valStart := skipWhitespaceAndOnlyComments(argsBody, colon+1)
				if strings.HasPrefix(argsBody[valStart:], "()") {
					afterParen := skipWhitespaceAndOnlyComments(argsBody, valStart+2)
					if strings.HasPrefix(argsBody[afterParen:], "=>") {
						valStart = skipWhitespaceAndOnlyComments(argsBody, afterParen+2)
					}
				}
				parenDepth := 0
				bracketDepth := 0
				braceDepth := 0
				valEnd := valStart
				for valEnd < n {
					nextEnd := skipCommentsAndStrings(argsBody, valEnd)
					if nextEnd != valEnd {
						valEnd = nextEnd
						continue
					}
					c := argsBody[valEnd]
					if c == '(' {
						parenDepth++
					} else if c == ')' {
						if parenDepth > 0 {
							parenDepth--
						}
					} else if c == '[' {
						bracketDepth++
					} else if c == ']' {
						if bracketDepth > 0 {
							bracketDepth--
						}
					} else if c == '{' {
						braceDepth++
					} else if c == '}' {
						if braceDepth > 0 {
							braceDepth--
						}
					} else if c == ',' && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
						break
					}
					valEnd++
				}
				return strings.TrimSpace(argsBody[valStart:valEnd])
			}
		}
		i++
	}
	return ""
}

func extractBlocStateType(src string, blocOffset int) string {
	i := blocOffset + 8 // len("blocTest")
	n := len(src)
	for i < n && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
		i++
	}
	if i < n && src[i] == '<' {
		closeAngle := findMatchingAngleBracket(src, i)
		if closeAngle != -1 {
			typeArgs := src[i+1 : closeAngle]
			depth := 0
			for j := range len(typeArgs) {
				if typeArgs[j] == '<' {
					depth++
				} else if typeArgs[j] == '>' {
					if depth > 0 {
						depth--
					}
				} else if typeArgs[j] == ',' && depth == 0 {
					return strings.TrimSpace(typeArgs[j+1:])
				}
			}
		}
	}
	return ""
}
func findDartGroups(src string) []dartGroup {
	var groups []dartGroup
	n := len(src)

	for i := 0; i < n; {
		next := skipCommentsAndStrings(src, i)
		if next != i {
			i = next
			continue
		}

		if strings.HasPrefix(src[i:], "group") && (i == 0 || !isIdentPart(src[i-1])) && (i+5 >= n || !isIdentPart(src[i+5])) {
			isMethodCall := false
			for p := i - 1; p >= 0; p-- {
				if src[p] == ' ' || src[p] == '\t' || src[p] == '\r' || src[p] == '\n' {
					continue
				}
				if src[p] == '.' {
					isMethodCall = true
				}
				break
			}
			if isMethodCall {
				i += 5
				continue
			}

			j := skipWhitespaceAndOnlyComments(src, i+5)
			if j < n && src[j] == '(' {
				k := skipWhitespaceAndOnlyComments(src, j+1)
				title, afterStr := readDartStringLiteral(src, k)
				if afterStr > k {
					p := skipWhitespaceAndOnlyComments(src, afterStr)
					if p < n && src[p] == ',' {
						cbStart := skipWhitespaceAndOnlyComments(src, p+1)
						if cbStart < n && src[cbStart] == '(' {
							parenClose := findMatchingParen(src, cbStart)
							if parenClose != -1 {
								afterParen := skipWhitespaceAndOnlyComments(src, parenClose+1)
								if strings.HasPrefix(src[afterParen:], "async*") {
									afterParen = skipWhitespaceAndOnlyComments(src, afterParen+6)
								} else if strings.HasPrefix(src[afterParen:], "async") && (afterParen+5 >= n || !isIdentPart(src[afterParen+5])) {
									afterParen = skipWhitespaceAndOnlyComments(src, afterParen+5)
								} else if strings.HasPrefix(src[afterParen:], "sync*") {
									afterParen = skipWhitespaceAndOnlyComments(src, afterParen+5)
								}

								if afterParen < n && src[afterParen] == '{' {
									closeBrace := findMatchingBrace(src, afterParen)
									if closeBrace != -1 {
										groups = append(groups, dartGroup{
											title: title,
											start: afterParen,
											end:   closeBrace,
										})
										i = afterParen + 1
										continue
									}
								}
							}
						}
					}
				}
			}
			i += 5
			continue
		}

		i++
	}

	return groups
}

func computeScope(groups []dartGroup, testOffset int) string {
	var enclosing []string
	for _, g := range groups {
		if g.start < testOffset && testOffset < g.end {
			enclosing = append(enclosing, g.title)
		}
	}
	return strings.Join(enclosing, " > ")
}

// findMatchingParen returns the index of the closing paren matching openParen,
// skipping comments and interpolation-aware string literals.
func findMatchingParen(src string, openParen int) int {
	if openParen < 0 || openParen >= len(src) || src[openParen] != '(' {
		return -1
	}
	depth := 0
	n := len(src)
	for i := openParen; i < n; {
		next := skipDartLexeme(src, i)
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

// findMatchingBrace returns the index of the closing brace matching openBrace,
// skipping comments and interpolation-aware string literals.
func findMatchingBrace(src string, openBrace int) int {
	if openBrace < 0 || openBrace >= len(src) || src[openBrace] != '{' {
		return -1
	}
	depth := 0
	n := len(src)
	for i := openBrace; i < n; {
		next := skipDartLexeme(src, i)
		if next != i {
			i = next
			continue
		}
		switch src[i] {
		case '{':
			depth++
		case '}':
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

// stripComments removes comments from code while preserving string literals
// verbatim, so callers can search code without matching comment text.
func stripComments(code string) string {
	var buf strings.Builder
	buf.Grow(len(code))
	for i := 0; i < len(code); {
		next := skipDartLexeme(code, i)
		if next == i {
			buf.WriteByte(code[i])
			i++
			continue
		}
		if code[i] == '/' {
			i = next
			continue
		}
		buf.WriteString(code[i:next])
		i = next
	}
	return buf.String()
}

// stripCommentsAndStrings removes comments and blanks string literal contents
// with spaces so callers can detect identifiers and calls without matching
// text that only appears inside literals.
func stripCommentsAndStrings(code string) string {
	stripped := stripComments(code)
	buf := []byte(stripped)
	for i := 0; i < len(stripped); {
		next := skipDartLexeme(stripped, i)
		if next == i {
			i++
			continue
		}
		blankDartLexeme(buf, stripped, i, next, ' ', ' ')
		i = next
	}
	return string(buf)
}

func findAssertingHelpers(src string) map[string]bool {
	assertingHelpers := make(map[string]bool)
	n := len(src)
	if n == 0 {
		return assertingHelpers
	}

	closing := make([]int, n)
	for idx := range closing {
		closing[idx] = -1
	}
	var parenStack []int
	var braceStack []int
	for i := 0; i < n; {
		next := skipCommentsAndStrings(src, i)
		if next != i {
			i = next
			continue
		}
		switch src[i] {
		case '(':
			parenStack = append(parenStack, i)
		case ')':
			if len(parenStack) > 0 {
				open := parenStack[len(parenStack)-1]
				parenStack = parenStack[:len(parenStack)-1]
				closing[open] = i
			}
		case '{':
			braceStack = append(braceStack, i)
		case '}':
			if len(braceStack) > 0 {
				open := braceStack[len(braceStack)-1]
				braceStack = braceStack[:len(braceStack)-1]
				closing[open] = i
			}
		}
		i++
	}

	var scanRange func(start, end int)
	scanRange = func(start, end int) {
		for i := start; i < end; {
			next := skipCommentsAndStrings(src, i)
			if next != i {
				i = next
				continue
			}

			if src[i] == '{' {
				closeBrace := closing[i]
				if closeBrace != -1 && closeBrace < end {
					i = closeBrace + 1
					continue
				}
				i++
				continue
			}

			if src[i] == '(' {
				parenClose := closing[i]
				if parenClose != -1 && parenClose < end {
					k := skipWhitespaceAndComments(src, parenClose+1)
					if strings.HasPrefix(src[k:], "async*") {
						k = skipWhitespaceAndComments(src, k+6)
					} else if strings.HasPrefix(src[k:], "async") && (k+5 >= n || !isIdentPart(src[k+5])) {
						k = skipWhitespaceAndComments(src, k+5)
					} else if strings.HasPrefix(src[k:], "sync*") {
						k = skipWhitespaceAndComments(src, k+5)
					}
					if k < end && src[k] == '{' {
						openBrace := k
						closeBrace := closing[openBrace]
						if closeBrace != -1 && closeBrace < end {
							scanRange(openBrace+1, closeBrace)
							i = closeBrace + 1
							continue
						}
					}
				}
			}

			if isIdentStart(src[i]) {
				identStart := i
				for i < end && isIdentPart(src[i]) {
					i++
				}
				name := src[identStart:i]

				j := skipWhitespaceAndComments(src, i)
				if j < end && src[j] == '<' {
					if k := findMatchingAngleBracket(src, j); k != -1 && k < end {
						j = skipWhitespaceAndComments(src, k+1)
					}
				}
				if j < end && src[j] == '(' {
					parenClose := closing[j]
					if parenClose != -1 && parenClose < end {
						k := skipWhitespaceAndComments(src, parenClose+1)
						if strings.HasPrefix(src[k:], "async*") {
							k = skipWhitespaceAndComments(src, k+6)
						} else if strings.HasPrefix(src[k:], "async") && (k+5 >= n || !isIdentPart(src[k+5])) {
							k = skipWhitespaceAndComments(src, k+5)
						} else if strings.HasPrefix(src[k:], "sync*") {
							k = skipWhitespaceAndComments(src, k+5)
						}

						if k < end && src[k] == '{' {
							openBrace := k
							closeBrace := closing[openBrace]
							if closeBrace != -1 && closeBrace < end {
								body := src[openBrace : closeBrace+1]
								if isValidHelperName(name) && assertionRegex.MatchString(stripCommentsAndStrings(body)) {
									assertingHelpers[name] = true
								}
								scanRange(openBrace+1, closeBrace)
								i = closeBrace + 1
								continue
							}
						} else if k+1 < end && src[k] == '=' && src[k+1] == '>' {
							arrowStart := k + 2
							semiIdx := findSemicolon(src, arrowStart)
							if semiIdx != -1 && semiIdx < end {
								body := src[arrowStart:semiIdx]
								if isValidHelperName(name) && assertionRegex.MatchString(stripCommentsAndStrings(body)) {
									assertingHelpers[name] = true
								}
								i = semiIdx + 1
								continue
							}
						} else {
							scanRange(j+1, parenClose)
							i = parenClose + 1
							continue
						}
					}
				}
				continue
			}
			i++
		}
	}

	scanRange(0, n)
	return assertingHelpers
}
