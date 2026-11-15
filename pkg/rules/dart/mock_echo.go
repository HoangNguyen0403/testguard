package dart

import (
	"fmt"
	"regexp"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	stubRegex   = regexp.MustCompile(`(?s)\b(?:thenAnswer\s*\(\s*(?:\([^)]*\)|_\w*)\s*(?:async\s*)?=>|thenReturn\s*\()\s*(Right|Left)\s*\(`)
	expectRegex = regexp.MustCompile(`(?s)\bexpect\s*\(`)
	sideRegex   = regexp.MustCompile(`(?s)\b(?:equals\s*\(\s*)?(Right|Left)\s*\(`)
)

type mockToken struct {
	side  string
	token string
}

// MockEchoRule detects 1:1 pass-through mock echoing in repository/service tests.
type MockEchoRule struct{}

// NewMockEchoRule creates a new mock echo rule.
func NewMockEchoRule() *MockEchoRule {
	return &MockEchoRule{}
}

func extractStubTokens(rawBody string) []mockToken {
	var stubs []mockToken
	indices := stubRegex.FindAllStringSubmatchIndex(rawBody, -1)
	for _, idx := range indices {
		if len(idx) < 4 {
			continue
		}
		side := rawBody[idx[2]:idx[3]]
		openParen := idx[1] - 1
		if openParen < len(rawBody) && rawBody[openParen] == '(' {
			closeParen := parser.FindMatchingParen(rawBody, openParen)
			if closeParen != -1 {
				token := normalizeWhitespace(rawBody[openParen+1 : closeParen])
				if token != "" {
					stubs = append(stubs, mockToken{side: side, token: token})
				}
			}
		}
	}
	return stubs
}

func extractExpectTokens(rawBody string) []mockToken {
	var expects []mockToken
	locs := expectRegex.FindAllStringIndex(rawBody, -1)
	for _, loc := range locs {
		openParen := loc[1] - 1
		closeParen := parser.FindMatchingParen(rawBody, openParen)
		if closeParen == -1 {
			continue
		}
		expectContent := rawBody[openParen+1 : closeParen]
		sideIndices := sideRegex.FindAllStringSubmatchIndex(expectContent, -1)
		for _, sIdx := range sideIndices {
			if len(sIdx) < 4 {
				continue
			}
			side := expectContent[sIdx[2]:sIdx[3]]
			sideOpenParen := sIdx[1] - 1
			if sideOpenParen < len(expectContent) && expectContent[sideOpenParen] == '(' {
				sideCloseParen := parser.FindMatchingParen(expectContent, sideOpenParen)
				if sideCloseParen != -1 {
					token := normalizeWhitespace(expectContent[sideOpenParen+1 : sideCloseParen])
					if token != "" {
						expects = append(expects, mockToken{side: side, token: token})
					}
				}
			}
		}
	}
	return expects
}

// Evaluate checks if the test passes through mock data with zero domain transformation.
func (r *MockEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation
	if unit.RawBody == "" {
		return violations
	}

	stubs := extractStubTokens(unit.RawBody)
	if len(stubs) == 0 {
		return violations
	}

	expects := extractExpectTokens(unit.RawBody)
	if len(expects) == 0 {
		return violations
	}

	for _, exp := range expects {
		for _, stub := range stubs {
			if exp.side == stub.side && exp.token == stub.token {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-DART-MOCK-ECHO",
					Severity:   models.SeverityWarning,
					FilePath:   unit.FilePath,
					LineNumber: unit.StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Pass-through mock echo in '%s': service merely returns mock '%s' with zero transformation.", unit.Name, exp.token),
					Suggestion: "Assert domain entity mapping, calculation outcomes, or error translation instead of 1:1 mock echoing.",
				})
				return violations
			}
		}
	}

	return violations
}
