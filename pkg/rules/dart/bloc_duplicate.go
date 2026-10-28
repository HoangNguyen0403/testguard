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
				Message:    fmt.Sprintf("Duplicate BLoC scenario: '%s' has identical act and expect blocks to '%s' (line %d).", unit.Name, first.Name, first.StartLine),
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
	stripped := stripComments(body)
	words := strings.Fields(stripped)
	return strings.Join(words, " ")
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
