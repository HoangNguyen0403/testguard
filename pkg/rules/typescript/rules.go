package typescript

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	nestControllerMockRegex   = regexp.MustCompile(`mockResolvedValue\s*\(\s*([^,\)]+)\)`)
	nestControllerExpectRegex = regexp.MustCompile(`expect\s*\(\s*await\s+controller\.\w+\([^)]*\)\s*\)\.(?:resolves\.)?(?:toBe|toEqual)\s*\(\s*([^,\)]+)\)`)
	nestRealDbModuleRegex     = regexp.MustCompile(`(?i)(?:TypeOrmModule\.forRoot|MongooseModule\.forRoot|SequelizeModule\.forRoot|PrismaModule)`)
	fakeTimersRegex           = regexp.MustCompile(`useFakeTimers\s*\(`)
	setTimeoutRegex           = regexp.MustCompile(`\bsetTimeout\s*\(`)
	mockCallRegex             = regexp.MustCompile(`\b(?:mockImplementation|mockImplementationOnce|mockReturnValue|mockReturnValueOnce|mockResolvedValue|mockRejectedValue)\s*\(`)
	classDeclRegex            = regexp.MustCompile(`\bclass\s+\w+`)
)

// NestControllerEchoRule flags NestJS controller unit tests that do 1:1 pass-through mock echoing.
type NestControllerEchoRule struct{}

func NewNestControllerEchoRule() *NestControllerEchoRule {
	return &NestControllerEchoRule{}
}

func (r *NestControllerEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	mockMatch := nestControllerMockRegex.FindStringSubmatch(unit.RawBody)
	expectMatch := nestControllerExpectRegex.FindStringSubmatch(unit.RawBody)

	if len(mockMatch) > 1 && len(expectMatch) > 1 {
		if mockMatch[1] == expectMatch[1] {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-NEST-01",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Pass-through controller echo in '%s': controller test simply returns mock '%s' without validation or transformation.", unit.Name, mockMatch[1]),
				Suggestion: "Test validation pipes, guards, interceptors, or integration behavior via supertest rather than 1:1 controller pass-through mocks.",
			})
		}
	}

	return violations
}

// NestUnmockedDbModuleRule traps unmocked database modules inside NestJS unit tests.
type NestUnmockedDbModuleRule struct{}

func NewNestUnmockedDbModuleRule() *NestUnmockedDbModuleRule {
	return &NestUnmockedDbModuleRule{}
}

func (r *NestUnmockedDbModuleRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if nestRealDbModuleRegex.MatchString(unit.RawBody) {
		violations = append(violations, models.Violation{
			RuleID:     "GUARD-NEST-02",
			Severity:   models.SeverityBlocker,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Unmocked database module leak in '%s': unit test imports real database module.", unit.Name),
			Suggestion: "Use getRepositoryToken() or mock providers in Test.createTestingModule instead of importing real database connection modules.",
		})
	}

	return violations
}

// ReactSleepRule traps hardcoded setTimeout / sleep calls in React and frontend test bodies.
type ReactSleepRule struct{}

func NewReactSleepRule() *ReactSleepRule {
	return &ReactSleepRule{}
}

// EvaluateFile reports real-timer sleeps in unit tests of one file.
func (r *ReactSleepRule) EvaluateFile(units []parser.TestUnit, fileSrc string) []models.Violation {
	var violations []models.Violation

	cleanFileSrc := blankCommentsAndStrings(fileSrc)
	if fakeTimersRegex.MatchString(cleanFileSrc) {
		return nil
	}

	for _, unit := range units {
		if unit.Kind != "" && unit.Kind != "unit" {
			continue
		}

		matches := setTimeoutRegex.FindAllStringIndex(unit.RawBody, -1)
		for _, match := range matches {
			matchIdx := match[0]
			if !isCodeIndex(unit.RawBody, matchIdx) {
				continue
			}

			// Ignore if lexically inside mock argument list
			if isInsideMockCall(unit.RawBody, matchIdx) {
				continue
			}

			// Ignore if inside a class body
			if isInsideClassBody(unit.RawBody, matchIdx) {
				continue
			}

			openParen := strings.Index(unit.RawBody[matchIdx:], "(") + matchIdx
			closeParen := findMatchingParen(unit.RawBody, openParen)
			if closeParen == -1 {
				continue
			}

			args := splitCallArgs(unit.RawBody[openParen+1 : closeParen])
			if len(args) < 2 {
				// Delay argument is absent -> ignore
				continue
			}

			delay := args[1]
			if !isPositiveDelay(delay) {
				continue
			}

			lineOffset := strings.Count(unit.RawBody[:matchIdx], "\n")
			line := unit.StartLine + lineOffset

			violations = append(violations, models.Violation{
				RuleID:     "GUARD-REACT-02",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: line,
				Column:     1,
				Message:    fmt.Sprintf("Arbitrary sleep/setTimeout fragility in '%s': setTimeout called with delay '%s' in test.", unit.Name, delay),
				Suggestion: "Use React Testing Library's waitFor(...) or async findBy* queries instead of fixed setTimeout sleeps.",
			})

			// At most one violation per unit
			break
		}
	}

	return violations
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

		// Strings
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

var jsIdentRegex = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(?:\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)

func isPositiveDelay(s string) bool {
	s = blankCommentsAndStrings(s)
	s = strings.TrimSpace(s)
	if s == "" || s == "undefined" || s == "null" || s == "false" || s == "NaN" {
		return false
	}
	if s == "true" {
		return true
	}
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val > 0
	}
	return jsIdentRegex.MatchString(s)
}

func isLiteralZero(s string) bool {
	return !isPositiveDelay(s)
}

func blankCommentsAndStrings(src string) string {
	b := []byte(src)
	n := len(b)
	for i := 0; i < n; {
		if i+1 < n && b[i] == '/' && b[i+1] == '/' {
			b[i] = ' '
			b[i+1] = ' '
			i += 2
			for i < n && b[i] != '\n' {
				if b[i] != '\r' {
					b[i] = ' '
				}
				i++
			}
			continue
		}
		if i+1 < n && b[i] == '/' && b[i+1] == '*' {
			b[i] = ' '
			b[i+1] = ' '
			i += 2
			for i < n {
				if i+1 < n && b[i] == '*' && b[i+1] == '/' {
					b[i] = ' '
					b[i+1] = ' '
					i += 2
					break
				}
				if b[i] != '\n' && b[i] != '\r' {
					b[i] = ' '
				}
				i++
			}
			continue
		}
		if b[i] == '\'' || b[i] == '"' || b[i] == '`' {
			delim := b[i]
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
				if b[i] == delim {
					b[i] = ' '
					i++
					break
				}
				if (delim == '\'' || delim == '"') && b[i] == '\n' {
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
