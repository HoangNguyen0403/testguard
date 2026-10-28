package e2e

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	fakeTimersRegex     = regexp.MustCompile(`useFakeTimers\s*\(`)
	waitForTimeoutRegex = regexp.MustCompile(`\bwaitForTimeout\s*\(`)
	setTimeoutRegex     = regexp.MustCompile(`\bsetTimeout\s*\(`)
	mockCallRegex       = regexp.MustCompile(`\b(?:mockImplementation|mockImplementationOnce|mockReturnValue|mockReturnValueOnce|mockResolvedValue|mockRejectedValue)\s*\(`)
	classDeclRegex      = regexp.MustCompile(`\bclass\s+\w+`)
	futureDelayedRegex  = regexp.MustCompile(`\bFuture\.delayed\s*\(`)
	sleepRegex          = regexp.MustCompile(`\bsleep\s*\(`)
	dartMockStubRegex   = regexp.MustCompile(`\b(?:when|thenAnswer)\s*\(`)
)

// HardWaitRule traps arbitrary sleeps and hard waits in E2E tests (Playwright, Flutter).
type HardWaitRule struct{}

// NewHardWaitRule creates a new hard wait rule for E2E tests.
func NewHardWaitRule() *HardWaitRule {
	return &HardWaitRule{}
}

type candidateMatch struct {
	index    int
	callName string
}

// EvaluateFile reports hard waits in E2E test files (TS/JS and Dart).
func (r *HardWaitRule) EvaluateFile(units []parser.TestUnit, filePath string, fileSrc string) []models.Violation {
	var violations []models.Violation

	ext := strings.ToLower(filepath.Ext(filePath))
	isTS := ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx"
	isDart := ext == ".dart"

	if !isTS && !isDart {
		return nil
	}

	for _, unit := range units {
		var candidates []candidateMatch

		if isTS {
			hasFakeTimers := fakeTimersRegex.MatchString(fileSrc)

			// 1. waitForTimeout
			for _, m := range waitForTimeoutRegex.FindAllStringIndex(unit.RawBody, -1) {
				idx := m[0]
				if !isCodeIndex(unit.RawBody, idx) {
					continue
				}
				if isInsideMockCall(unit.RawBody, idx) || isInsideClassBody(unit.RawBody, idx) {
					continue
				}
				openParen := strings.Index(unit.RawBody[idx:], "(") + idx
				closeParen := findMatchingParen(unit.RawBody, openParen)
				if closeParen == -1 {
					continue
				}
				args := splitCallArgs(unit.RawBody[openParen+1 : closeParen])
				if len(args) >= 1 && isLiteralZero(args[0]) {
					continue
				}
				candidates = append(candidates, candidateMatch{index: idx, callName: "waitForTimeout"})
			}

			// 2. setTimeout
			if !hasFakeTimers {
				for _, m := range setTimeoutRegex.FindAllStringIndex(unit.RawBody, -1) {
					idx := m[0]
					if !isCodeIndex(unit.RawBody, idx) {
						continue
					}
					if isInsideMockCall(unit.RawBody, idx) || isInsideClassBody(unit.RawBody, idx) {
						continue
					}
					openParen := strings.Index(unit.RawBody[idx:], "(") + idx
					closeParen := findMatchingParen(unit.RawBody, openParen)
					if closeParen == -1 {
						continue
					}
					args := splitCallArgs(unit.RawBody[openParen+1 : closeParen])
					if len(args) < 2 {
						continue
					}
					if isLiteralZero(args[1]) {
						continue
					}
					candidates = append(candidates, candidateMatch{index: idx, callName: "setTimeout"})
				}
			}
		} else if isDart {
			// 1. Future.delayed
			for _, m := range futureDelayedRegex.FindAllStringIndex(unit.RawBody, -1) {
				idx := m[0]
				if !isCodeIndex(unit.RawBody, idx) {
					continue
				}
				if isInsideDartMock(unit.RawBody, idx) {
					continue
				}
				candidates = append(candidates, candidateMatch{index: idx, callName: "Future.delayed"})
			}

			// 2. sleep
			for _, m := range sleepRegex.FindAllStringIndex(unit.RawBody, -1) {
				idx := m[0]
				if !isCodeIndex(unit.RawBody, idx) {
					continue
				}
				if isInsideDartMock(unit.RawBody, idx) {
					continue
				}
				candidates = append(candidates, candidateMatch{index: idx, callName: "sleep"})
			}
		}

		if len(candidates) == 0 {
			continue
		}

		// Sort by index in RawBody so we report the first offending line
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].index < candidates[j].index
		})

		first := candidates[0]
		lineOffset := strings.Count(unit.RawBody[:first.index], "\n")
		line := unit.StartLine + lineOffset

		path := unit.FilePath
		if path == "" {
			path = filePath
		}

		violations = append(violations, models.Violation{
			RuleID:     "GUARD-E2E-01",
			Severity:   models.SeverityWarning,
			FilePath:   path,
			LineNumber: line,
			Column:     1,
			Message:    fmt.Sprintf("Hard wait in E2E test '%s': %s sleeps for a fixed time; wait for a UI condition instead (Playwright: await expect(locator).toBeVisible(); Flutter: pumpUntilFound / pumpAndSettle with a finder).", unit.Name, first.callName),
			Suggestion: "Wait for a UI condition instead of arbitrary sleep (Playwright: await expect(locator).toBeVisible(); Flutter: pumpUntilFound / pumpAndSettle with a finder).",
		})
	}

	return violations
}

func isInsideDartMock(src string, targetIdx int) bool {
	matches := dartMockStubRegex.FindAllStringIndex(src, -1)
	for _, match := range matches {
		if !isCodeIndex(src, match[0]) {
			continue
		}
		openParen := strings.Index(src[match[0]:], "(") + match[0]
		closeParen := findMatchingParen(src, openParen)
		if closeParen != -1 && openParen < targetIdx && targetIdx < closeParen {
			return true
		}
	}
	return false
}

func isInsideMockCall(src string, targetIdx int) bool {
	matches := mockCallRegex.FindAllStringIndex(src, -1)
	for _, match := range matches {
		if !isCodeIndex(src, match[0]) {
			continue
		}
		openParen := strings.Index(src[match[0]:], "(") + match[0]
		closeParen := findMatchingParen(src, openParen)
		if closeParen != -1 && openParen < targetIdx && targetIdx < closeParen {
			return true
		}
	}
	return false
}

func isInsideClassBody(src string, targetIdx int) bool {
	matches := classDeclRegex.FindAllStringIndex(src, -1)
	for _, match := range matches {
		if !isCodeIndex(src, match[0]) {
			continue
		}
		openBrace := -1
		for i := match[1]; i < len(src); i++ {
			if !isCodeIndex(src, i) {
				continue
			}
			if src[i] == '{' {
				openBrace = i
				break
			}
		}
		if openBrace != -1 {
			closeBrace := parser.FindMatchingBrace(src, openBrace)
			if closeBrace != -1 && openBrace < targetIdx && targetIdx < closeBrace {
				return true
			}
		}
	}
	return false
}

func isLiteralZero(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" || s == "0.0" || s == "+0" || s == "-0" {
		return true
	}
	if val, err := strconv.ParseFloat(s, 64); err == nil && val == 0 {
		return true
	}
	return false
}

func isCodeIndex(src string, targetIdx int) bool {
	n := len(src)
	if targetIdx < 0 || targetIdx >= n {
		return false
	}

	for i := 0; i < n; {
		if i == targetIdx {
			return true
		}
		if i > targetIdx {
			return false
		}

		// Single-line comment
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			start := i
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		// Multi-line block comment
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			start := i
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		// Dart raw string literal: r'...' or r"..."
		if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
			quote := src[i+1]
			start := i
			if i+3 < n && src[i+2] == quote && src[i+3] == quote {
				i += 4
				for i+2 < n && !(src[i] == quote && src[i+1] == quote && src[i+2] == quote) {
					i++
				}
				i += 3
				if targetIdx >= start && targetIdx < i {
					return false
				}
				continue
			}
			i += 2
			for i < n && src[i] != quote && src[i] != '\n' {
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		// Triple-quoted multi-line string literal: """...""" or '''...'''
		if (src[i] == '\'' || src[i] == '"') && i+2 < n && src[i+1] == src[i] && src[i+2] == src[i] {
			quote := src[i]
			start := i
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
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		// Standard quoted string literal: "...", '...', or `...`
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			start := i
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		i++
	}

	return true
}

func findMatchingParen(src string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(src) || src[openIndex] != '(' {
		return -1
	}

	parenCount := 0
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

		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			continue
		}

		if src[i] == '(' {
			parenCount++
		} else if src[i] == ')' {
			parenCount--
			if parenCount == 0 {
				return i
			}
		}
	}

	return -1
}

func splitCallArgs(src string) []string {
	var args []string
	start := 0
	parenDepth := 0
	braceDepth := 0
	bracketDepth := 0
	n := len(src)

	for i := 0; i < n; i++ {
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
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			continue
		}

		switch src[i] {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ',':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				args = append(args, strings.TrimSpace(src[start:i]))
				start = i + 1
			}
		}
	}

	if start < n {
		trimmed := strings.TrimSpace(src[start:])
		if trimmed != "" {
			args = append(args, trimmed)
		}
	}

	return args
}
