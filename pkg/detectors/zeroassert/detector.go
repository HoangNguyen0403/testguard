package zeroassert

import (
	"fmt"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

// Detector flags test units that contain zero assertions.
type Detector struct{}

// New creates a new zero-assertion detector.
func New() *Detector {
	return &Detector{}
}

// Check evaluates test units and reports zero-assertion violations.
func (d *Detector) Check(units []parser.TestUnit) []models.Violation {
	var violations []models.Violation

	for _, unit := range units {
		// Only standard unit tests can have zero assertions; blocTest implicitly has expect:
		if unit.Kind == "unit" && len(unit.Assertions) == 0 {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-UNIV-ZERO",
				Severity:   models.SeverityBlocker,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Zero assertions found in test '%s'. Tests without outcome verification give false confidence.", unit.Name),
				Suggestion: "Add expect(), expectLater(), or verify() to assert observable state changes or business outcomes.",
			})
		}
	}

	return violations
}
