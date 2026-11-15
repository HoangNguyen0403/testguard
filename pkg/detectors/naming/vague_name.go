package naming

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	pureDigitsRegex = regexp.MustCompile(`^\d+$`)
	testPrefixRegex = regexp.MustCompile(`^(?:test|tests|testing)[\s_-]*\d+$`)

	placeholderWords = map[string]bool{
		"test":    true,
		"tests":   true,
		"testing": true,
		"sample":  true,
		"dummy":   true,
		"temp":    true,
		"foo":     true,
		"bar":     true,
		"baz":     true,
		"qwerty":  true,
		"asdf":    true,
	}
)

// Detector flags vague, hollow, or placeholder test descriptions.
type Detector struct{}

// New creates a new vague naming detector.
func New() *Detector {
	return &Detector{}
}

// Check evaluates test unit names and reports vague or placeholder names.
// Units whose adapter marked Meta["title_dynamic"]="true" carry the original
// title expression source rather than an evaluated runtime string, so an
// expression such as `it(foo, …)` may evaluate to a descriptive title and must
// not be judged on its source text. Literal titles keep the placeholder check.
func (d *Detector) Check(units []parser.TestUnit) []models.Violation {
	var violations []models.Violation

	for _, unit := range units {
		if unit.Meta["title_dynamic"] == "true" {
			continue
		}
		trimmed := strings.TrimSpace(unit.Name)
		lower := strings.ToLower(trimmed)

		isVague := false
		if trimmed == "" || pureDigitsRegex.MatchString(trimmed) || placeholderWords[lower] || testPrefixRegex.MatchString(lower) {
			isVague = true
		}

		if isVague {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-UNIV-NAME",
				Severity:   models.SeverityAdvisory,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Vague or placeholder test description '%s'.", unit.Name),
				Suggestion: "Provide a descriptive name explaining what scenario is being tested (e.g. Method_Scenario_ExpectedBehavior).",
			})
		}
	}

	return violations
}
