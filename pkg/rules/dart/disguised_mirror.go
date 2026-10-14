package dart

import (
	"fmt"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

// DisguisedMirrorRule flags isA<State>() matchers that chain 4 or more .having() calls
// where at least one property repeats a seeded value.
type DisguisedMirrorRule struct{}

// NewDisguisedMirrorRule creates a new disguised mirror rule.
func NewDisguisedMirrorRule() *DisguisedMirrorRule {
	return &DisguisedMirrorRule{}
}

// Evaluate checks if expect contains 4+ .having calls mirroring state properties.
func (r *DisguisedMirrorRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation
	if unit.Kind != "bloc" || unit.ExpectBody == "" {
		return violations
	}

	havingCount := strings.Count(unit.ExpectBody, ".having(")
	if havingCount >= 4 {
		violations = append(violations, models.Violation{
			RuleID:     "GUARD-DART-DISGUISED-MIRROR",
			Severity:   models.SeverityWarning,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Disguised state mirror in '%s': chaining %d .having() calls on state matcher.", unit.Name, havingCount),
			Suggestion: "Use a seeded base state fixture (baseState.copyWith(...)) and assert only mutated fields in state transitions.",
		})
	}

	return violations
}
