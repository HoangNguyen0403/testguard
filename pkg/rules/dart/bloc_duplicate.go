package dart

import (
	"fmt"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

// BlocDuplicateRule detects two blocTest cases within the same file that execute identical events and expect identical states.
type BlocDuplicateRule struct{}

// NewBlocDuplicateRule creates a new bloc duplicate rule.
func NewBlocDuplicateRule() *BlocDuplicateRule {
	return &BlocDuplicateRule{}
}

// Check evaluates all blocTest units within a file and flags identical execution pairs.
func (r *BlocDuplicateRule) Check(units []parser.TestUnit) []models.Violation {
	var violations []models.Violation
	seenScenarios := make(map[string]parser.TestUnit)

	for _, unit := range units {
		if unit.Kind != "bloc" || strings.TrimSpace(unit.RawBody) == "" {
			continue
		}

		key := normalizeScenarioBody(unit.RawBody, unit.Name)
		if key == "" {
			continue
		}

		if first, exists := seenScenarios[key]; exists {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-DART-BLOC-DUPLICATE",
				Severity:   models.SeverityAdvisory,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Duplicate BLoC scenario: '%s' has identical build/setUp/seed/act/expect to '%s' (line %d).", unit.Name, first.Name, first.StartLine),
				Suggestion: "Check whether this test exercises a distinct edge case or consolidate into a single parameterized test.",
			})
		} else {
			seenScenarios[key] = unit
		}
	}

	return violations
}

func normalizeScenarioBody(rawBody string, testName string) string {
	body := strings.TrimSpace(rawBody)
	if testName != "" {
		singleQuoted := "'" + testName + "'"
		doubleQuoted := `"` + testName + `"`
		if strings.HasPrefix(body, singleQuoted) {
			body = strings.TrimPrefix(body, singleQuoted)
			body = strings.TrimLeft(body, ", \t\r\n")
		} else if strings.HasPrefix(body, doubleQuoted) {
			body = strings.TrimPrefix(body, doubleQuoted)
			body = strings.TrimLeft(body, ", \t\r\n")
		}
	}
	return tokenizePreservingStrings(body)
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func tokenizePreservingStrings(src string) string {
	var tokens []string
	n := len(src)
	for i := 0; i < n; {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		// Single-line comment
		if i+1 < n && c == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment
		if i+1 < n && c == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		// Triple-quoted strings
		if (c == '\'' || c == '"') && i+2 < n && src[i+1] == c && src[i+2] == c {
			start := i
			i += 3
			for i+2 < n {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if src[i] == c && src[i+1] == c && src[i+2] == c {
					i += 3
					break
				}
				i++
			}
			tokens = append(tokens, src[start:i])
			continue
		}
		// Raw strings
		if (c == 'r' || c == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
			quote := src[i+1]
			start := i
			i += 2
			for i < n && src[i] != quote && src[i] != '\n' {
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			tokens = append(tokens, src[start:i])
			continue
		}
		// Regular quoted strings
		if c == '\'' || c == '"' {
			start := i
			quote := c
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if src[i] == '\n' {
					break
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			tokens = append(tokens, src[start:i])
			continue
		}
		// Identifiers and numbers
		if isIdentChar(c) {
			start := i
			for i < n && isIdentChar(src[i]) {
				i++
			}
			tokens = append(tokens, src[start:i])
			continue
		}
		// Multi-char operators
		if i+1 < n {
			two := src[i : i+2]
			if two == "=>" || two == "==" || two == "!=" || two == "<=" || two == ">=" ||
				two == "&&" || two == "||" || two == "++" || two == "--" || two == "??" || two == "?." {
				tokens = append(tokens, two)
				i += 2
				continue
			}
		}
		// Single punctuation / operator
		tokens = append(tokens, string(c))
		i++
	}
	return strings.Join(tokens, " ")
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
