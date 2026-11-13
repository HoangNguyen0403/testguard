package dart

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var havingGetterRegex = regexp.MustCompile(`\.\s*(\w+)\s*$`)

// SeedEchoRule detects when expected state assertions echo values established in the seed baseline.
type SeedEchoRule struct{}

// NewSeedEchoRule creates a new seed echo rule.
func NewSeedEchoRule() *SeedEchoRule {
	return &SeedEchoRule{}
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
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

func parseSeedProps(seedBody string) map[string]string {
	seedProps := make(map[string]string)
	openParen := strings.Index(seedBody, "(")
	if openParen == -1 {
		return seedProps
	}

	n := len(seedBody)
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0

	for i := openParen + 1; i < n; {
		next := skipCommentsAndStrings(seedBody, i)
		if next != i {
			i = next
			continue
		}

		c := seedBody[i]
		if c == '(' {
			parenDepth++
			i++
			continue
		}
		if c == ')' {
			if parenDepth == 0 {
				break
			}
			parenDepth--
			i++
			continue
		}
		if c == '[' {
			bracketDepth++
			i++
			continue
		}
		if c == ']' {
			if bracketDepth > 0 {
				bracketDepth--
			}
			i++
			continue
		}
		if c == '{' {
			braceDepth++
			i++
			continue
		}
		if c == '}' {
			if braceDepth > 0 {
				braceDepth--
			}
			i++
			continue
		}

		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			if isIdentStart(c) {
				identStart := i
				for i < n && isIdentPart(seedBody[i]) {
					i++
				}
				name := seedBody[identStart:i]
				colonIdx := i
				for colonIdx < n && (seedBody[colonIdx] == ' ' || seedBody[colonIdx] == '\t' || seedBody[colonIdx] == '\n' || seedBody[colonIdx] == '\r') {
					colonIdx++
				}
				if colonIdx < n && seedBody[colonIdx] == ':' {
					valStart := colonIdx + 1
					for valStart < n && (seedBody[valStart] == ' ' || seedBody[valStart] == '\t' || seedBody[valStart] == '\n' || seedBody[valStart] == '\r') {
						valStart++
					}
					valEnd := valStart
					subParen := 0
					subBracket := 0
					subBrace := 0
					for valEnd < n {
						nextEnd := skipCommentsAndStrings(seedBody, valEnd)
						if nextEnd != valEnd {
							valEnd = nextEnd
							continue
						}
						sc := seedBody[valEnd]
						if sc == '(' {
							subParen++
						} else if sc == ')' {
							if subParen == 0 {
								break
							}
							subParen--
						} else if sc == '[' {
							subBracket++
						} else if sc == ']' {
							if subBracket > 0 {
								subBracket--
							}
						} else if sc == '{' {
							subBrace++
						} else if sc == '}' {
							if subBrace > 0 {
								subBrace--
							}
						} else if sc == ',' && subParen == 0 && subBracket == 0 && subBrace == 0 {
							break
						}
						valEnd++
					}
					rawVal := strings.TrimSpace(seedBody[valStart:valEnd])
					seedProps[name] = normalizeWhitespace(rawVal)
					i = valEnd
					continue
				}
			}
		}
		i++
	}

	return seedProps
}

type havingMatcher struct {
	propName string
	propVal  string
}

func parseHavingAssertions(expectBody string) []havingMatcher {
	var matchers []havingMatcher
	n := len(expectBody)

	for i := 0; i < n; {
		next := skipCommentsAndStrings(expectBody, i)
		if next != i {
			i = next
			continue
		}

		if strings.HasPrefix(expectBody[i:], ".having") {
			afterIdent := i + 7
			openParen := afterIdent
			for openParen < n && (expectBody[openParen] == ' ' || expectBody[openParen] == '\t' || expectBody[openParen] == '\n' || expectBody[openParen] == '\r') {
				openParen++
			}
			if openParen < n && expectBody[openParen] == '(' {
				closeParen := parser.FindMatchingParen(expectBody, openParen)
				if closeParen != -1 {
					argsStr := expectBody[openParen+1 : closeParen]
					var args []string
					argStart := 0
					pDepth := 0
					bDepth := 0
					brDepth := 0
					for j := 0; j < len(argsStr); {
						nextJ := skipCommentsAndStrings(argsStr, j)
						if nextJ != j {
							j = nextJ
							continue
						}
						switch argsStr[j] {
						case '(':
							pDepth++
						case ')':
							if pDepth > 0 {
								pDepth--
							}
						case '[':
							bDepth++
						case ']':
							if bDepth > 0 {
								bDepth--
							}
						case '{':
							brDepth++
						case '}':
							if brDepth > 0 {
								brDepth--
							}
						case ',':
							if pDepth == 0 && bDepth == 0 && brDepth == 0 {
								args = append(args, strings.TrimSpace(argsStr[argStart:j]))
								argStart = j + 1
							}
						}
						j++
					}
					if argStart < len(argsStr) {
						args = append(args, strings.TrimSpace(argsStr[argStart:]))
					}

					if len(args) >= 3 {
						prop := ""
						if m := havingGetterRegex.FindStringSubmatch(args[0]); len(m) > 1 {
							prop = m[1]
						} else {
							prop = strings.Trim(args[1], `'"`)
						}
						val := normalizeWhitespace(args[2])
						if strings.HasPrefix(val, "equals(") && strings.HasSuffix(val, ")") {
							val = normalizeWhitespace(val[7 : len(val)-1])
						}
						matchers = append(matchers, havingMatcher{
							propName: prop,
							propVal:  val,
						})
					}

					i = closeParen + 1
					continue
				}
			}
		}
		i++
	}

	return matchers
}

// Evaluate checks if a bloc test's expected states echo fields from the seed fixture.
func (r *SeedEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation
	if unit.Kind != "bloc" || unit.SeedBody == "" || unit.ExpectBody == "" {
		return violations
	}

	seedProps := parseSeedProps(unit.SeedBody)
	havings := parseHavingAssertions(unit.ExpectBody)
	if len(havings) == 0 {
		return violations
	}

	var echoedFields []string
	for _, h := range havings {
		seedVal, exists := seedProps[h.propName]
		if !exists {
			// Asserts a field the seed does not set -> report nothing
			return nil
		}
		if seedVal != h.propVal {
			// Asserts a value that differs from the seed -> report nothing
			return nil
		}
		echoedFields = append(echoedFields, h.propName)
	}

	var msg string
	if len(echoedFields) == 1 {
		msg = fmt.Sprintf("Seed echo assertion: expected state in '%s' merely echoes seed property '%s: %s' without mutation.", unit.Name, echoedFields[0], seedProps[echoedFields[0]])
	} else {
		msg = fmt.Sprintf("Seed echo assertion: expected state in '%s' merely echoes seed properties '%s' without mutation.", unit.Name, strings.Join(echoedFields, ", "))
	}

	violations = append(violations, models.Violation{
		RuleID:     "GUARD-DART-SEED-ECHO",
		Severity:   models.SeverityWarning,
		FilePath:   unit.FilePath,
		LineNumber: unit.StartLine,
		Column:     1,
		Message:    msg,
		Suggestion: "Assert the fields the event is supposed to change.",
	})

	return violations
}
