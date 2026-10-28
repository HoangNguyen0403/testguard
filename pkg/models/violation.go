package models

// Severity represents the impact level of a test quality violation.
type Severity string

const (
	// SeverityBlocker represents broken verification that fails CI immediately.
	SeverityBlocker Severity = "Blocker"
	// SeverityWarning represents high maintenance risk or architectural boundary violation.
	SeverityWarning Severity = "Warning"
	// SeverityAdvisory represents code smell, clone, or style advisory.
	SeverityAdvisory Severity = "Advisory"
)

// Violation represents an individual test quality or boundary violation found in a source file.
type Violation struct {
	RuleID     string   `json:"rule_id"`
	Severity   Severity `json:"severity"`
	FilePath   string   `json:"file_path"`
	LineNumber int      `json:"line_number"`
	Column     int      `json:"column"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion"`
}
