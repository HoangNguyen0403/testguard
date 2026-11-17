package typescript

import (
	"fmt"
	"regexp"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	ts "github.com/HoangNguyen0403/testguard/pkg/parser/typescript"
)

var (
	uiExecutionRegex = regexp.MustCompile(`\b(render|renderHook|shallow|mount|screen\s*\.|fireEvent\s*\.|userEvent\s*\.)`)
	queryBuilderWithArgsRegex = regexp.MustCompile(`\b(?:build|create|generate)[A-Za-z0-9_]*(?:Query|Mutation|Subscription|Doc)\s*\(\s*[^)\s]`)
	staticGqlPrintAssignRegex = regexp.MustCompile(`(?:const|let|var)\s+([A-Za-z0-9_$]+)\s*=\s*(?:print\s*\(|[^;\n]+\.loc(?:\.source)?\.body)`)
	staticGqlSubjectRegex = regexp.MustCompile(`(?:\bprint\s*\(|\.loc(?:\.source)?\.body\b)`)
	staticI18nRegex = regexp.MustCompile(`Object\.keys\s*\(|typeof\s+[^)]+\[[^\]]+\]`)
)

// StaticPinningRule detects unit tests that verify only static query strings or
// translation JSON keys without executing or mounting runtime logic.
type StaticPinningRule struct{}

// NewStaticPinningRule creates an initialized StaticPinningRule.
func NewStaticPinningRule() *StaticPinningRule {
	return &StaticPinningRule{}
}

// Evaluate checks whether a test unit only asserts static GraphQL AST or JSON keys.
func (r *StaticPinningRule) Evaluate(unit parser.TestUnit) []models.Violation {
	if parser.IsE2EFile(unit.FilePath, []byte(unit.RawBody)) {
		return nil
	}

	// 1. Guardrail: UI component rendering or hook mounting executes real runtime logic
	if uiExecutionRegex.MatchString(unit.RawBody) {
		return nil
	}

	// 2. Guardrail: Dynamic query builders called with arguments test query construction logic
	if queryBuilderWithArgsRegex.MatchString(unit.RawBody) {
		return nil
	}

	tokens := lexTestBody(unit)
	assertions := ts.FindAssertions(unit.RawBody, tokens, 0, len(unit.RawBody), nil)
	if len(assertions) == 0 {
		return nil
	}

	// Collect identifiers assigned from print(GQL) or .loc.source.body
	pinnedVars := make(map[string]bool)
	matches := staticGqlPrintAssignRegex.FindAllStringSubmatch(unit.RawBody, -1)
	for _, m := range matches {
		if len(m) > 1 {
			pinnedVars[m[1]] = true
		}
	}

	isPatternA := len(pinnedVars) > 0 || staticGqlSubjectRegex.MatchString(unit.RawBody)
	isPatternB := staticI18nRegex.MatchString(unit.RawBody)

	if !isPatternA && !isPatternB {
		return nil
	}

	// Check if every assertion in the test is a static pinning assertion
	firstAssertLine := bodyStartLine(unit)
	for i, a := range assertions {
		if i == 0 && a.Line > 0 {
			firstAssertLine = bodyStartLine(unit) + (a.Line - 1)
		}

		isPinnedAssert := false
		if staticGqlSubjectRegex.MatchString(a.Raw) || staticGqlSubjectRegex.MatchString(a.Subject) {
			isPinnedAssert = true
		} else if pinnedVars[a.Subject] {
			isPinnedAssert = true
		} else if isPatternB && (staticI18nRegex.MatchString(a.Raw) || staticI18nRegex.MatchString(a.Subject) || regexp.MustCompile(`typeof\s+`).MatchString(a.Subject)) {
			isPinnedAssert = true
		}

		if !isPinnedAssert {
			// Test has other non-static assertions
			return nil
		}
	}

	return []models.Violation{
		{
			RuleID:     "GUARD-TS-STATIC-PINNING",
			Severity:   models.SeverityWarning,
			FilePath:   unit.FilePath,
			LineNumber: firstAssertLine,
			Column:     1,
			Message:    fmt.Sprintf("Static pinning test in '%s': assertions only verify static GraphQL AST or translation JSON schema without testing runtime logic", unit.Name),
			Suggestion: "Test the component, hook, or query-builder consuming the query/translations instead of asserting static document AST or JSON key schemas.",
		},
	}
}
