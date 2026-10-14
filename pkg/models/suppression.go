package models

import (
	"regexp"
	"strings"
)

var (
	// Matches: // testguard:ignore RULE-ID reason: ... or /* testguard:ignore [RULE-ID] ... */ or # testguard:ignore RULE-ID
	suppressRegex = regexp.MustCompile(`(?i)(?://|#|/\*)\s*testguard:ignore\s*\[?([A-Z0-9_-]+)\]?\s*(?:reason:\s*(.*?))?(?:\*/|$)`)
)

// SuppressionRecord represents an in-code rule suppression directive.
type SuppressionRecord struct {
	RuleID     string
	LineNumber int
	Reason     string
}

// ParseSuppressions scans a source code string line-by-line and extracts all suppression comments.
func ParseSuppressions(source string) []SuppressionRecord {
	var records []SuppressionRecord
	lines := strings.Split(source, "\n")

	for i, line := range lines {
		matches := suppressRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			ruleID := strings.ToUpper(strings.TrimSpace(matches[1]))
			reason := ""
			if len(matches) > 2 {
				reason = strings.TrimSpace(matches[2])
			}

			records = append(records, SuppressionRecord{
				RuleID:     ruleID,
				LineNumber: i + 1,
				Reason:     reason,
			})
		}
	}

	return records
}

// IsSuppressed checks whether a given rule violation at a specific line is covered by an in-code suppression.
func IsSuppressed(records []SuppressionRecord, ruleID string, violationLine int) bool {
	upperRule := strings.ToUpper(ruleID)
	for _, rec := range records {
		if rec.RuleID == upperRule || rec.RuleID == "ALL" {
			// Directive suppresses current line or the immediately following test line
			if rec.LineNumber == violationLine || rec.LineNumber+1 == violationLine {
				return true
			}
		}
	}
	return false
}
