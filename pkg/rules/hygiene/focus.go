package hygiene

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

var (
	tsFocusEachRegex = regexp.MustCompile(`^(?:it|test|describe|test\.describe)\.only\.each\s*` + "[(`]")
	tsFocusDeclRegex = regexp.MustCompile(`^(?:it\.only|test\.only|describe\.only|test\.describe\.only)\s*\(`)
	tsFocusBareRegex = regexp.MustCompile(`^(?:fit|fdescribe)\s*\(`)
	dartSoloTrue     = regexp.MustCompile(`^solo\s*:\s*true\b`)
)

// FocusRule detects committed focused test modifiers (e.g. it.only, fit, solo: true)
// that silently exclude the rest of the test suite from execution.
type FocusRule struct{}

// NewFocusRule creates a new instance of FocusRule.
func NewFocusRule() *FocusRule {
	return &FocusRule{}
}

// CheckFile scans a test file for focused test declarations or modifiers.
func (r *FocusRule) CheckFile(filePath string, src string) []models.Violation {
	if isTSFile(filePath) {
		return r.checkTSFocus(filePath, src)
	} else if isDartFile(filePath) {
		return r.checkDartFocus(filePath, src)
	}
	return nil
}

func (r *FocusRule) checkTSFocus(filePath string, src string) []models.Violation {
	var violations []models.Violation
	cleanSrc, _ := blankCommentsAndFindStrings(src, langTS)
	n := len(src)
	line := 1
	for i := 0; i < n; {
		if src[i] == '\n' {
			line++
			i++
			continue
		}

		// Single-line comment
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		// Multi-line block comment
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			i += 2
			continue
		}

		// Quoted strings
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					if i+1 < n && src[i+1] == '\n' {
						line++
					}
					i += 2
					continue
				}
				if src[i] == '\n' {
					line++
					if quote != '`' {
						break
					}
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			continue
		}

		if (i == 0 || !isIdentPart(src[i-1])) && !isMemberAccess(cleanSrc, i) {
			if match := tsFocusEachRegex.FindString(src[i:]); match != "" {
				snippet := strings.TrimSpace(match)
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-FOCUS",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Focused test: '%s' silently stops the rest of the file or test suite from running.", snippet),
					Suggestion: "Remove the focus modifier so the full test suite runs.",
				})
				i += len(match)
				continue
			}
			if match := tsFocusDeclRegex.FindString(src[i:]); match != "" {
				snippet := strings.TrimSpace(match)
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-FOCUS",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Focused test: '%s' silently stops the rest of the file or test suite from running.", snippet),
					Suggestion: "Remove the focus modifier so the full test suite runs.",
				})
				i += len(match)
				continue
			}
			if match := tsFocusBareRegex.FindString(src[i:]); match != "" {
				rem := src[i+len(match):]
				trimmed := strings.TrimLeft(rem, " \t\r\n")
				if len(trimmed) > 0 && (trimmed[0] == '\'' || trimmed[0] == '"' || trimmed[0] == '`') {
					snippet := strings.TrimSpace(match)
					violations = append(violations, models.Violation{
						RuleID:     "GUARD-UNIV-FOCUS",
						Severity:   models.SeverityWarning,
						FilePath:   filePath,
						LineNumber: line,
						Column:     1,
						Message:    fmt.Sprintf("Focused test: '%s' silently stops the rest of the file or test suite from running.", snippet),
						Suggestion: "Remove the focus modifier so the full test suite runs.",
					})
					i += len(match)
					continue
				}
			}
		}

		i++
	}

	return violations
}

func (r *FocusRule) checkDartFocus(filePath string, src string) []models.Violation {
	var violations []models.Violation
	cleanSrc, strRanges := blankCommentsAndFindStrings(src, langDart)
	n := len(cleanSrc)
	line := 1

	for i := 0; i < n; {
		if src[i] == '\n' {
			line++
			i++
			continue
		}

		if cleanSrc[i] == ' ' || cleanSrc[i] == '\t' || cleanSrc[i] == '\r' {
			i++
			continue
		}

		if (i == 0 || !isIdentPart(cleanSrc[i-1])) && !inStringRange(i, strRanges) {
			rem := cleanSrc[i:]
			if match := dartSoloTrue.FindString(rem); match != "" {
				enclosingCall := findEnclosingCallName(cleanSrc, i)
				if dartSkipCallNames[enclosingCall] {
					violations = append(violations, models.Violation{
						RuleID:     "GUARD-UNIV-FOCUS",
						Severity:   models.SeverityWarning,
						FilePath:   filePath,
						LineNumber: line,
						Column:     1,
						Message:    fmt.Sprintf("Focused test: '%s' silently stops the rest of the file or test suite from running.", "solo: true"),
						Suggestion: "Remove the focus modifier so the full test suite runs.",
					})
					i += len(match)
					continue
				}
			}
		}

		i++
	}

	return violations
}
