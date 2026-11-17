package hygiene

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	calcKeywordRegex = regexp.MustCompile(`(?i)\b(net|total|subtotal|price|amount|rebate|discount|fee|tax|surcharge|mov|bonus|calc|compute)\b`)
	boundaryTitleRegex = regexp.MustCompile(`(?i)\b(zero|no[ _-](rebate|discount|fee|tax)|without|empty|free|exempt|waived|nil)\b`)
	operandZeroRegex = regexp.MustCompile(`(?i)\b(?:with[_-]?)?(rebate|discount|fee|tax|surcharge|mov_threshold|offset)\b\s*(?::=|=|:|\()\s*(?:0(?:\.0+)?|['"]0(?:\.0+)?['"]|\.empty\(\)|[A-Za-z0-9_]+\.empty\(\))`)
	operandNonZeroRegex = regexp.MustCompile(`(?i)\b(?:with[_-]?)?(rebate|discount|fee|tax|surcharge|mov_threshold|offset)\b\s*(?::=|=|:|\()\s*(?:[1-9][0-9]*(?:\.[0-9]+)?|['"][1-9][0-9]*(?:\.[0-9]+)?['"])`)
	assertZeroOutcomeRegex = regexp.MustCompile(`(?i)(?:toBe|toEqual|equals|toStrictEqual|Equal)\s*\(\s*(?:0(?:\.0+)?|['"]0(?:\.0+)?['"]|t,\s*0(?:\.0+)?)\s*\)|assert\.Zero\b`)
)

// IdentityFixtureRule detects calculation, pricing, discount, rebate, fee, or tax
// test cases where calculation operands are bound to zero or identity values while
// asserting a non-zero business outcome.
type IdentityFixtureRule struct{}

// NewIdentityFixtureRule creates an initialized IdentityFixtureRule.
func NewIdentityFixtureRule() *IdentityFixtureRule {
	return &IdentityFixtureRule{}
}

// Evaluate checks a test unit for identity math operand assignments.
func (r *IdentityFixtureRule) Evaluate(unit parser.TestUnit) []models.Violation {
	if !isDartFile(unit.FilePath) && !isTSFile(unit.FilePath) && !isGoFile(unit.FilePath) {
		return nil
	}

	// 1. Must match calculation keywords in title, scope, or body
	hasCalcKeyword := calcKeywordRegex.MatchString(unit.Name) ||
		calcKeywordRegex.MatchString(unit.Scope) ||
		calcKeywordRegex.MatchString(unit.RawBody)
	if !hasCalcKeyword {
		return nil
	}

	// 2. Exemption A: boundary titles (zero, no-discount, exempt, etc.)
	if isBoundaryTitle(unit.Name, unit.Scope) {
		return nil
	}

	// Blank comments to avoid false triggers in comments
	lang := langTS
	if isDartFile(unit.FilePath) {
		lang = langDart
	}
	cleanSrc, _ := blankCommentsAndFindStrings(unit.RawBody, lang)

	// 3. Must match an operand bound to zero / empty
	if !operandZeroRegex.MatchString(cleanSrc) {
		return nil
	}

	// 4. Exemption B: If opposing non-zero fixture exists and asserts zero result boundary
	if operandNonZeroRegex.MatchString(cleanSrc) && assertZeroOutcomeRegex.MatchString(cleanSrc) {
		return nil
	}

	// If the test exclusively asserts a zero outcome (boundary check), exempt it
	if assertZeroOutcomeRegex.MatchString(cleanSrc) && !hasNonZeroAssertion(cleanSrc) {
		return nil
	}

	// Locate the first matching line for the zero operand
	startLine := bodyStartLine(unit)
	lines := strings.Split(cleanSrc, "\n")
	matchedLine := startLine
	for i, l := range lines {
		if operandZeroRegex.MatchString(l) {
			matchedLine = startLine + i
			break
		}
	}

	return []models.Violation{
		{
			RuleID:     "GUARD-UNIV-IDENTITY-FIXTURE",
			Severity:   models.SeverityWarning,
			FilePath:   unit.FilePath,
			LineNumber: matchedLine,
			Message:    fmt.Sprintf("Calculation test '%s' binds calculation operand to identity value (0/empty) while asserting non-zero outcome", unit.Name),
			Suggestion: "Bind calculation operands (discounts, rebates, fees, taxes) to realistic non-zero values to ensure calculation logic is actively tested.",
		},
	}
}

var nonZeroNumRegex = regexp.MustCompile(`(?i)(?:toBe|toEqual|equals|toStrictEqual|Equal)\s*\(\s*(?:[1-9][0-9]*(?:\.[0-9]+)?|['"][1-9][0-9]*(?:\.[0-9]+)?['"]|t,\s*[1-9][0-9]*(?:\.[0-9]+)?)\s*\)`)

func hasNonZeroAssertion(s string) bool {
	return nonZeroNumRegex.MatchString(s)
}

func isBoundaryTitle(name, scope string) bool {
	combined := strings.ToLower(name + " " + scope)
	if strings.Contains(combined, "zero") {
		return true
	}
	return boundaryTitleRegex.MatchString(combined)
}
