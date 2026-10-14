package dart

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	namedParamRegex = regexp.MustCompile(`(\w+)\s*:\s*([^,\)]+)`)
	havingPropRegex = regexp.MustCompile(`\.having\s*\(\s*\([^)]*\)\s*=>\s*[^.]*\.(\w+)\s*,\s*['"]\w+['"]\s*,\s*([^,\)]+)`)
)

// SeedEchoRule detects when expected state assertions echo values established in the seed baseline.
type SeedEchoRule struct{}

// NewSeedEchoRule creates a new seed echo rule.
func NewSeedEchoRule() *SeedEchoRule {
	return &SeedEchoRule{}
}

// Evaluate checks if a bloc test's expected states echo fields from the seed fixture.
func (r *SeedEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation
	if unit.Kind != "bloc" || unit.SeedBody == "" || unit.ExpectBody == "" {
		return violations
	}

	// 1. Extract properties assigned in seed
	seedProps := make(map[string]string)
	for _, m := range namedParamRegex.FindAllStringSubmatch(unit.SeedBody, -1) {
		seedProps[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
	}

	// 2. Check for .having matching properties and values
	for _, m := range havingPropRegex.FindAllStringSubmatch(unit.ExpectBody, -1) {
		propName := strings.TrimSpace(m[1])
		propVal := strings.TrimSpace(m[2])

		if seedVal, exists := seedProps[propName]; exists && seedVal == propVal {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-DART-SEED-ECHO",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Seed echo assertion: expected state in '%s' merely echoes seed property '%s: %s' without mutation.", unit.Name, propName, propVal),
				Suggestion: "Use a seeded base state fixture (baseState.copyWith(...)) and assert only mutated fields in state transitions.",
			})
		}
	}

	return violations
}
