package clone

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	stringLiteralRegex = regexp.MustCompile(`'[^']*'|"[^"]*"`)
	numberLiteralRegex = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	whitespaceRegex    = regexp.MustCompile(`\s+`)
)

// Detector identifies duplicate tests and structural repetition.
type Detector struct{}

// New creates a new clone detector.
func New() *Detector {
	return &Detector{}
}

// Normalize replaces string and number literals with generic tokens.
func Normalize(code string) string {
	s := stringLiteralRegex.ReplaceAllString(code, "_LIT_")
	s = numberLiteralRegex.ReplaceAllString(s, "_NUM_")
	return strings.TrimSpace(whitespaceRegex.ReplaceAllString(s, " "))
}

// HashTokenStream produces a SHA256 hex string of the normalized code.
func HashTokenStream(code string) string {
	norm := Normalize(code)
	h := sha256.Sum256([]byte(norm))
	return fmt.Sprintf("%x", h)
}

// Check compares test units within the same file for duplicate execution bodies.
func (d *Detector) Check(units []parser.TestUnit) []models.Violation {
	var violations []models.Violation
	seenHashes := make(map[string][]parser.TestUnit)

	for _, unit := range units {
		if unit.Kind == "file" || len(unit.RawBody) < 20 {
			continue
		}

		h := HashTokenStream(unit.RawBody)
		seenHashes[h] = append(seenHashes[h], unit)
	}

	for _, group := range seenHashes {
		if len(group) == 2 || len(group) == 3 {
			// Exact clone between 2-3 tests
			for i := 1; i < len(group); i++ {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-01",
					Severity:   models.SeverityAdvisory,
					FilePath:   group[i].FilePath,
					LineNumber: group[i].StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Duplicate test body: '%s' has identical execution tokens to '%s' (line %d).", group[i].Name, group[0].Name, group[0].StartLine),
					Suggestion: "Check whether this test exercises a distinct edge case or consolidate into a table-driven parameterized test.",
				})
			}
		} else if len(group) >= 4 {
			// Structural repetition smell (>=4 tests)
			for i := 3; i < len(group); i++ {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-02",
					Severity:   models.SeverityAdvisory,
					FilePath:   group[i].FilePath,
					LineNumber: group[i].StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Structural repetition: '%s' shares identical AST structure with %d other tests in this file (first seen at '%s', line %d).", group[i].Name, len(group)-1, group[0].Name, group[0].StartLine),
					Suggestion: "Consolidate repetitive test structures into a parameterized table-driven test loop.",
				})
			}
		}
	}

	return violations
}
