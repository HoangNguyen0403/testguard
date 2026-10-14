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
	type scenarioKey struct {
		act    string
		expect string
	}
	seenScenarios := make(map[scenarioKey]parser.TestUnit)

	for _, unit := range units {
		if unit.Kind != "bloc" || unit.ActBody == "" || unit.ExpectBody == "" {
			continue
		}

		key := scenarioKey{
			act:    strings.TrimSpace(unit.ActBody),
			expect: strings.TrimSpace(unit.ExpectBody),
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
