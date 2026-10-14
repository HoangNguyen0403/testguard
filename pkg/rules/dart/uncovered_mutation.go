package dart

import (
	"fmt"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

// UncoveredMutationRule checks that when an asynchronous fetch event is dispatched,
// the expected states assert the loading state (e.g. isFetching: true).
type UncoveredMutationRule struct{}

// NewUncoveredMutationRule creates a new uncovered mutation rule.
func NewUncoveredMutationRule() *UncoveredMutationRule {
	return &UncoveredMutationRule{}
}

// Evaluate checks if an async fetch event omits asserting isFetching in expect.
func (r *UncoveredMutationRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation
	if unit.Kind != "bloc" || unit.ActBody == "" || unit.ExpectBody == "" {
		return violations
	}

	actLower := strings.ToLower(unit.ActBody)
	isAsyncFetch := strings.Contains(actLower, "fetch") ||
		strings.Contains(actLower, "search") ||
		strings.Contains(actLower, "load") ||
		strings.Contains(actLower, "submit")

	if isAsyncFetch {
		expectContainsFetching := strings.Contains(unit.ExpectBody, "isFetching")
		if !expectContainsFetching {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-DART-UNCOVERED-MUTATION",
				Severity:   models.SeverityBlocker,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Uncovered state mutation in '%s': async action dispatched but expect sequence omits asserting 'isFetching' state.", unit.Name),
				Suggestion: "Assert the intermediate loading transition (baseState.copyWith(isFetching: true)) before asserting success or failure.",
			})
		}
	}

	return violations
}
