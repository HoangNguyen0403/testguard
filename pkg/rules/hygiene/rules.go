package hygiene

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	numberRegex = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
	stringRegex = regexp.MustCompile(`^(?:"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|` + "`" + `[^` + "`" + `\\]*(?:\\.[^` + "`" + `\\]*)*` + "`" + `)$`)

	strLit    = `(?:"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|` + "`" + `[^` + "`" + `\\]*(?:\\.[^` + "`" + `\\]*)*` + "`" + `)`
	parenArg  = `(?:` + strLit + `|[^()]+?)`
	commaArg  = `(?:` + strLit + `|[^,()]+?)`

	// TS/JS tautology patterns
	tsExpectSameRegex = regexp.MustCompile(`expect\s*\(\s*(` + parenArg + `)\s*\)\s*\.\s*(?:toBe|toEqual|toStrictEqual)\s*\(\s*(` + parenArg + `)\s*\)`)
	tsTruthyTautology = regexp.MustCompile(`expect\s*\(\s*true\s*\)\s*\.\s*toBeTruthy\s*\(\s*\)`)
	tsFalsyTautology  = regexp.MustCompile(`expect\s*\(\s*false\s*\)\s*\.\s*toBeFalsy\s*\(\s*\)`)

	// Dart tautology patterns
	dartTrueIsTrue   = regexp.MustCompile(`expect\s*\(\s*true\s*,\s*isTrue\s*(?:,[^)]*)?\)`)
	dartFalseIsFalse = regexp.MustCompile(`expect\s*\(\s*false\s*,\s*isFalse\s*(?:,[^)]*)?\)`)
	dartEqualsTaut   = regexp.MustCompile(`expect\s*\(\s*(` + commaArg + `)\s*,\s*equals\s*\(\s*(` + parenArg + `)\s*\)\s*(?:,[^)]*)?\)`)
	dartExpectSame   = regexp.MustCompile(`expect\s*\(\s*(` + commaArg + `)\s*,\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)

	// Go tautology patterns
	goTrueTautology  = regexp.MustCompile(`(?:assert|require)\.True\s*\(\s*\w+\s*,\s*true\s*(?:,[^)]*)?\)`)
	goFalseTautology = regexp.MustCompile(`(?:assert|require)\.False\s*\(\s*\w+\s*,\s*false\s*(?:,[^)]*)?\)`)
	goEqualSame      = regexp.MustCompile(`(?:assert|require)\.Equal(?:Values|f)?\s*\(\s*\w+\s*,\s*(` + commaArg + `)\s*,\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)
)

func isIdentPart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func isLiteral(s string) bool {
	s = strings.TrimSpace(s)
	if s == "true" || s == "false" || s == "null" || s == "nil" {
		return true
	}
	if numberRegex.MatchString(s) {
		return true
	}
	if stringRegex.MatchString(s) {
		return true
	}
	return false
}

func isSameLiteral(l, r string) bool {
	l = strings.TrimSpace(l)
	r = strings.TrimSpace(r)
	if l != r {
		return false
	}
	return isLiteral(l)
}

func isTSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".mjs" || ext == ".cjs"
}

func isDartFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".dart")
}

func isGoFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".go")
}

type tautologyMatch struct {
	index int
}

type TautologyRule struct{}

func NewTautologyRule() *TautologyRule {
	return &TautologyRule{}
}

func (r *TautologyRule) Evaluate(unit parser.TestUnit) []models.Violation {
	body := unit.RawBody
	if strings.TrimSpace(body) == "" {
		return nil
	}

	var matches []tautologyMatch

	if isTSFile(unit.FilePath) {
		for _, loc := range tsTruthyTautology.FindAllStringIndex(body, -1) {
			matches = append(matches, tautologyMatch{index: loc[0]})
		}
		for _, loc := range tsFalsyTautology.FindAllStringIndex(body, -1) {
			matches = append(matches, tautologyMatch{index: loc[0]})
		}
		for _, loc := range tsExpectSameRegex.FindAllStringSubmatchIndex(body, -1) {
			if len(loc) >= 6 {
				l := body[loc[2]:loc[3]]
				r := body[loc[4]:loc[5]]
				if isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
	} else if isDartFile(unit.FilePath) {
		for _, loc := range dartTrueIsTrue.FindAllStringIndex(body, -1) {
			matches = append(matches, tautologyMatch{index: loc[0]})
		}
		for _, loc := range dartFalseIsFalse.FindAllStringIndex(body, -1) {
			matches = append(matches, tautologyMatch{index: loc[0]})
		}
		for _, loc := range dartEqualsTaut.FindAllStringSubmatchIndex(body, -1) {
			if len(loc) >= 6 {
				l := body[loc[2]:loc[3]]
				r := body[loc[4]:loc[5]]
				if isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
		for _, loc := range dartExpectSame.FindAllStringSubmatchIndex(body, -1) {
			if len(loc) >= 6 {
				l := body[loc[2]:loc[3]]
				r := body[loc[4]:loc[5]]
				if isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
	} else if isGoFile(unit.FilePath) {
		for _, loc := range goTrueTautology.FindAllStringIndex(body, -1) {
			matches = append(matches, tautologyMatch{index: loc[0]})
		}
		for _, loc := range goFalseTautology.FindAllStringIndex(body, -1) {
			matches = append(matches, tautologyMatch{index: loc[0]})
		}
		for _, loc := range goEqualSame.FindAllStringSubmatchIndex(body, -1) {
			if len(loc) >= 6 {
				l := body[loc[2]:loc[3]]
				r := body[loc[4]:loc[5]]
				if isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
	} else {
		return nil
	}

	if len(matches) == 0 {
		return nil
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].index < matches[j].index
	})

	firstIndex := matches[0].index
	line := unit.StartLine + strings.Count(body[:firstIndex], "\n")
	if line <= 0 {
		line = 1
	}

	return []models.Violation{
		{
			RuleID:     "GUARD-UNIV-TAUTOLOGY",
			Severity:   models.SeverityBlocker,
			FilePath:   unit.FilePath,
			LineNumber: line,
			Column:     1,
			Message:    fmt.Sprintf("Tautological assertion in '%s': assertion will always pass vacuously.", unit.Name),
			Suggestion: "Assert actual behavior or state produced by the system under test rather than comparing literals to themselves.",
		},
	}
}

type SkipRule struct{}

func NewSkipRule() *SkipRule {
	return &SkipRule{}
}

var (
	tsSkipRegex   = regexp.MustCompile(`^(?:it\.skip|test\.skip|describe\.skip|xit|xtest|xdescribe)\s*\(`)
	dartSkipTrue  = regexp.MustCompile(`^skip\s*:\s*true\b`)
	dartSkipStr   = regexp.MustCompile(`^skip\s*:\s*(?:'[^']*'|"[^"]*")`)
	goTestFuncRe  = regexp.MustCompile(`func\s+Test\w*\s*\(\s*(\w+)\s+\*testing\.T\s*\)`)
)

func (r *SkipRule) CheckFile(filePath string, src string) []models.Violation {
	if isTSFile(filePath) {
		return r.checkTSSkips(filePath, src)
	} else if isDartFile(filePath) {
		return r.checkDartSkips(filePath, src)
	} else if isGoFile(filePath) {
		return r.checkGoSkips(filePath, src)
	}
	return nil
}

func (r *SkipRule) checkTSSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
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

		// Standard quoted string literal: "..." or '...' or `...`
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

		// Check for TS/JS skip pattern
		if i == 0 || !isIdentPart(src[i-1]) {
			if match := tsSkipRegex.FindString(src[i:]); match != "" {
				snippet := strings.TrimSpace(match)
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-SKIP",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", snippet),
					Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
				})
				i += len(match)
				continue
			}
		}

		i++
	}

	return violations
}

func (r *SkipRule) checkDartSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
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

		// Raw string literal: r'...' or r"..."
		if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
			quote := src[i+1]
			if i+3 < n && src[i+2] == quote && src[i+3] == quote {
				i += 4
				for i+2 < n && !(src[i] == quote && src[i+1] == quote && src[i+2] == quote) {
					if src[i] == '\n' {
						line++
					}
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

		// Triple-quoted multi-line string literal: """...""" or '''...'''
		if (src[i] == '\'' || src[i] == '"') && i+2 < n && src[i+1] == src[i] && src[i+2] == src[i] {
			quote := src[i]
			i += 3
			for i+2 < n {
				if src[i] == '\\' {
					if src[i+1] == '\n' {
						line++
					}
					i += 2
					continue
				}
				if src[i] == '\n' {
					line++
				}
				if src[i] == quote && src[i+1] == quote && src[i+2] == quote {
					i += 3
					break
				}
				i++
			}
			continue
		}

		// Standard quoted string literal: "..." or '...'
		if src[i] == '\'' || src[i] == '"' {
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
					break
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			continue
		}

		// Check for Dart skip pattern outside strings and comments
		if i == 0 || !isIdentPart(src[i-1]) {
			rem := src[i:]
			var snippet string
			if match := dartSkipTrue.FindString(rem); match != "" {
				snippet = strings.TrimSpace(match)
			} else if match := dartSkipStr.FindString(rem); match != "" {
				snippet = strings.TrimSpace(match)
			}

			if snippet != "" {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-SKIP",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", snippet),
					Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
				})
				i += len(snippet)
				continue
			}
		}

		i++
	}

	return violations
}

func (r *SkipRule) checkGoSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
	lines := strings.Split(src, "\n")

	inBlockComment := false
	inTestFunc := false
	lookingForFirstStmt := false
	var currentParam string

	for lineIdx, line := range lines {
		lineNum := lineIdx + 1
		trimmed := strings.TrimSpace(line)

		// Handle block comments across lines
		if inBlockComment {
			if endIdx := strings.Index(trimmed, "*/"); endIdx != -1 {
				inBlockComment = false
				trimmed = strings.TrimSpace(trimmed[endIdx+2:])
			} else {
				continue
			}
		}

		// Strip single line comments and inline block comments
		for {
			startIdx := strings.Index(trimmed, "/*")
			endIdx := strings.Index(trimmed, "*/")
			if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
				trimmed = strings.TrimSpace(trimmed[:startIdx] + " " + trimmed[endIdx+2:])
			} else if startIdx != -1 {
				inBlockComment = true
				trimmed = strings.TrimSpace(trimmed[:startIdx])
				break
			} else {
				break
			}
		}

		if commentIdx := strings.Index(trimmed, "//"); commentIdx != -1 {
			trimmed = strings.TrimSpace(trimmed[:commentIdx])
		}

		if trimmed == "" {
			continue
		}

		if !inTestFunc {
			if m := goTestFuncRe.FindStringSubmatch(line); len(m) > 1 {
				currentParam = m[1]
				inTestFunc = true
				if braceIdx := strings.Index(trimmed, "{"); braceIdx != -1 {
					afterBrace := strings.TrimSpace(trimmed[braceIdx+1:])
					if afterBrace != "" && afterBrace != "}" {
						// Statement on the same line as opening brace
						skipPattern := fmt.Sprintf(`^(?:%s\.(?:Skip|Skipf|SkipNow)|\w+\.(?:Skip|Skipf|SkipNow))\s*\(`, regexp.QuoteMeta(currentParam))
						skipRegex := regexp.MustCompile(skipPattern)
						if skipRegex.MatchString(afterBrace) {
							snippet := afterBrace
							if semiIdx := strings.Index(snippet, ";"); semiIdx != -1 {
								snippet = snippet[:semiIdx]
							}
							violations = append(violations, models.Violation{
								RuleID:     "GUARD-UNIV-SKIP",
								Severity:   models.SeverityWarning,
								FilePath:   filePath,
								LineNumber: lineNum,
								Column:     1,
								Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", strings.TrimSpace(snippet)),
								Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
							})
						}
						inTestFunc = false
						lookingForFirstStmt = false
						continue
					}
					lookingForFirstStmt = true
				}
			}
		} else if !lookingForFirstStmt {
			if braceIdx := strings.Index(trimmed, "{"); braceIdx != -1 {
				afterBrace := strings.TrimSpace(trimmed[braceIdx+1:])
				if afterBrace != "" && afterBrace != "}" {
					skipPattern := fmt.Sprintf(`^(?:%s\.(?:Skip|Skipf|SkipNow)|\w+\.(?:Skip|Skipf|SkipNow))\s*\(`, regexp.QuoteMeta(currentParam))
					skipRegex := regexp.MustCompile(skipPattern)
					if skipRegex.MatchString(afterBrace) {
						snippet := afterBrace
						if semiIdx := strings.Index(snippet, ";"); semiIdx != -1 {
							snippet = snippet[:semiIdx]
						}
						violations = append(violations, models.Violation{
							RuleID:     "GUARD-UNIV-SKIP",
							Severity:   models.SeverityWarning,
							FilePath:   filePath,
							LineNumber: lineNum,
							Column:     1,
							Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", strings.TrimSpace(snippet)),
							Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
						})
					}
					inTestFunc = false
					lookingForFirstStmt = false
					continue
				}
				lookingForFirstStmt = true
			}
		} else {
			// First non-blank, non-comment statement in test body
			skipPattern := fmt.Sprintf(`^(?:%s\.(?:Skip|Skipf|SkipNow)|\w+\.(?:Skip|Skipf|SkipNow))\s*\(`, regexp.QuoteMeta(currentParam))
			skipRegex := regexp.MustCompile(skipPattern)
			if skipRegex.MatchString(trimmed) {
				snippet := trimmed
				if semiIdx := strings.Index(snippet, ";"); semiIdx != -1 {
					snippet = snippet[:semiIdx]
				}
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-SKIP",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: lineNum,
					Column:     1,
					Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", strings.TrimSpace(snippet)),
					Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
				})
			}
			inTestFunc = false
			lookingForFirstStmt = false
		}
	}

	return violations
}
