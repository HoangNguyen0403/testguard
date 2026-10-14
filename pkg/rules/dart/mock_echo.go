package dart

import (
	"fmt"
	"regexp"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	thenAnswerRightRegex = regexp.MustCompile(`(?s)thenAnswer\s*\(\s*[^)]*\)\s*(?:async\s*)?=>\s*Right\s*\(\s*([^,\)]+)\)`)
	expectRightRegex     = regexp.MustCompile(`expect\s*\(\s*[^,]+,\s*(?:equals\s*\(\s*)?Right\s*\(\s*([^,\)]+)\)`)
)

// MockEchoRule detects 1:1 pass-through mock echoing in repository/service tests.
type MockEchoRule struct{}

// NewMockEchoRule creates a new mock echo rule.
func NewMockEchoRule() *MockEchoRule {
	return &MockEchoRule{}
}

// Evaluate checks if the test passes through mock data with zero domain transformation.
func (r *MockEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation
	if unit.RawBody == "" {
		return violations
	}

	thenMatch := thenAnswerRightRegex.FindStringSubmatch(unit.RawBody)
	expectMatch := expectRightRegex.FindStringSubmatch(unit.RawBody)

	if len(thenMatch) > 1 && len(expectMatch) > 1 {
		if thenMatch[1] == expectMatch[1] {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-DART-MOCK-ECHO",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Pass-through mock echo in '%s': service merely returns mock '%s' with zero transformation.", unit.Name, thenMatch[1]),
				Suggestion: "Assert domain entity mapping, calculation outcomes, or error translation instead of 1:1 mock echoing.",
			})
		}
	}

	return violations
}
