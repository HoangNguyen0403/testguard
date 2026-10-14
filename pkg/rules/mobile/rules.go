package mobile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	mockkEveryRegex     = regexp.MustCompile(`every\s*\{\s*[^}]+\s*\}\s*returns\s+([^\n;]+)`)
	kotlinAssertEqRegex = regexp.MustCompile(`assertEquals\s*\(\s*([^,\)]+)\s*,\s*viewModel\.\w+`)
	swiftTimeoutRegex   = regexp.MustCompile(`(?:wait\s*\(\s*for:\s*\[[^\]]+\],\s*timeout:\s*(\d+(?:\.\d+)?)|fulfillment\s*\(\s*of:\s*\[[^\]]+\],\s*timeout:\s*(\d+(?:\.\d+)?))`)
)

// KotlinCoroutineAdvanceRule checks that async/debounced ViewModel calls in Kotlin runTest call advanceUntilIdle().
type KotlinCoroutineAdvanceRule struct{}

func NewKotlinCoroutineAdvanceRule() *KotlinCoroutineAdvanceRule {
	return &KotlinCoroutineAdvanceRule{}
}

func (r *KotlinCoroutineAdvanceRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	lower := strings.ToLower(unit.RawBody)
	isAsyncAction := strings.Contains(lower, "search(") || strings.Contains(lower, "fetch(") || strings.Contains(lower, "debounce")

	if isAsyncAction {
		hasAdvance := strings.Contains(unit.RawBody, "advanceUntilIdle") || strings.Contains(unit.RawBody, "advanceTimeBy") || strings.Contains(unit.RawBody, "runCurrent")
		if !hasAdvance {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-KT-02",
				Severity:   models.SeverityBlocker,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Unadvanced coroutine virtual time in '%s': async suspend/debounce called without advanceUntilIdle().", unit.Name),
				Suggestion: "Call advanceUntilIdle() or testDispatcher.scheduler.advanceUntilIdle() before asserting StateFlow/LiveData state.",
			})
		}
	}

	return violations
}

// SwiftTimeoutRule checks that XCTest unit tests do not use excessive expectation timeouts (>2.0s).
type SwiftTimeoutRule struct{}

func NewSwiftTimeoutRule() *SwiftTimeoutRule {
	return &SwiftTimeoutRule{}
}

func (r *SwiftTimeoutRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	matches := swiftTimeoutRegex.FindAllStringSubmatch(unit.RawBody, -1)
	for _, m := range matches {
		timeoutStr := m[1]
		if timeoutStr == "" && len(m) > 2 {
			timeoutStr = m[2]
		}
		if timeoutStr != "" {
			timeoutVal, err := strconv.ParseFloat(timeoutStr, 64)
			if err == nil && timeoutVal > 2.0 {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-SWIFT-01",
					Severity:   models.SeverityWarning,
					FilePath:   unit.FilePath,
					LineNumber: unit.StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Excessive expectation timeout in '%s': timeout %.1fs exceeds 2.0s threshold for local unit test.", unit.Name, timeoutVal),
					Suggestion: "Use async/await or fulfill expectations within a tight sub-second timeout (<= 1.0s) to keep test feedback fast.",
				})
			}
		}
	}

	return violations
}
