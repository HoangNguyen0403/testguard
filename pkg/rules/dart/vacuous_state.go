package dart

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	dartparser "github.com/HoangNguyen0403/testguard/pkg/parser/dart"
)

// VacuousStateRule detects blocTest cases that assert only bare isA<S>() for the base state S without property assertions or verify blocks.
type VacuousStateRule struct{}

// NewVacuousStateRule creates a new vacuous state rule.
func NewVacuousStateRule() *VacuousStateRule {
	return &VacuousStateRule{}
}

func hasVerifyArg(rawBody string) bool {
	n := len(rawBody)
	for i := 0; i < n; {
		next := skipCommentsAndStrings(rawBody, i)
		if next != i {
			i = next
			continue
		}
		if strings.HasPrefix(rawBody[i:], "verify") && (i == 0 || !isIdentPart(rawBody[i-1])) && (i+6 >= n || !isIdentPart(rawBody[i+6])) {
			j := i + 6
			for j < n && (rawBody[j] == ' ' || rawBody[j] == '\t' || rawBody[j] == '\n' || rawBody[j] == '\r') {
				j++
			}
			if j < n && rawBody[j] == ':' {
				return true
			}
		}
		i++
	}
	return false
}

func parseListElements(body string) []string {
	openBracket := strings.Index(body, "[")
	if openBracket == -1 {
		return nil
	}
	closeBracket := strings.LastIndex(body, "]")
	if closeBracket <= openBracket {
		return nil
	}

	content := body[openBracket+1 : closeBracket]
	var elements []string
	start := 0
	pDepth := 0
	bDepth := 0
	brDepth := 0
	n := len(content)

	for i := 0; i < n; {
		next := skipCommentsAndStrings(content, i)
		if next != i {
			i = next
			continue
		}
		switch content[i] {
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
		case '<':
			// Angle brackets in generic type expressions
			pDepth++
		case '>':
			if pDepth > 0 {
				pDepth--
			}
		case ',':
			if pDepth == 0 && bDepth == 0 && brDepth == 0 {
				elem := strings.TrimSpace(content[start:i])
				if elem != "" {
					elements = append(elements, elem)
				}
				start = i + 1
			}
		}
		i++
	}
	if start < n {
		elem := strings.TrimSpace(content[start:])
		if elem != "" {
			elements = append(elements, elem)
		}
	}

	return elements
}

// Evaluate checks if a bloc test vacuously asserts only bare isA<S>() without property checks or verify blocks.
func (r *VacuousStateRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation
	if unit.Kind != "bloc" || unit.Meta == nil {
		return violations
	}

	stateType := unit.Meta[dartparser.MetaBlocStateType]
	if stateType == "" || unit.ExpectBody == "" {
		return violations
	}

	if hasVerifyArg(unit.RawBody) {
		return violations
	}

	elements := parseListElements(unit.ExpectBody)
	if len(elements) == 0 {
		return violations
	}

	barePattern := regexp.MustCompile(`^isA\s*<\s*` + regexp.QuoteMeta(stateType) + `\s*>\s*\(\s*\)$`)

	for _, elem := range elements {
		if strings.Contains(elem, ".having") {
			return violations
		}
		if !barePattern.MatchString(elem) {
			return violations
		}
	}

	violations = append(violations, models.Violation{
		RuleID:     "GUARD-DART-VACUOUS-STATE",
		Severity:   models.SeverityWarning,
		FilePath:   unit.FilePath,
		LineNumber: unit.StartLine,
		Column:     1,
		Message:    fmt.Sprintf("Vacuous state assertion: '%s' asserts only bare 'isA<%s>()' with no property matchers or verify block.", unit.Name, stateType),
		Suggestion: "Assert specific state properties with .having(...) or expect concrete state subclasses rather than bare isA.",
	})

	return violations
}
