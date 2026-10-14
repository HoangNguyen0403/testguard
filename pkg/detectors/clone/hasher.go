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
	seenHashes := make(map[string]parser.TestUnit)

	for _, unit := range units {
		if len(unit.RawBody) < 20 {
			continue // Skip trivial 1-liners
		}

		h := HashTokenStream(unit.RawBody)
		if first, exists := seenHashes[h]; exists {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-UNIV-01",
				Severity:   models.SeverityAdvisory,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Duplicate test body: '%s' has identical execution tokens to '%s' (line %d).", unit.Name, first.Name, first.StartLine),
				Suggestion: "Check whether this test exercises a distinct edge case or consolidate into a table-driven parameterized test.",
			})
		} else {
			seenHashes[h] = unit
		}
	}

	return violations
}
