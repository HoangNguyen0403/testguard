package golang

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	mockControllerRegex = regexp.MustCompile(`(?i)(?:gomock\.NewController|mocks\.NewMock|testify/mock)`)
	sqlmockRegex        = regexp.MustCompile(`(?i)(?:sqlmock\.New|pgxmock\.New|gorm\.Open)`)
	rowPatternRegex     = regexp.MustCompile(`(?m)^\s*\{([^}]+)\},?`)
	commentRegex        = regexp.MustCompile(`//.*`)

	repoOrInfraPatterns = []string{
		"/repository/",
		"_repository_test.go",
		"/infrastructure/",
		"/database/",
		"/db/",
		"/dao/",
		"/store/",
	}
)

func isRepositoryOrInfraLayer(filePath string) bool {
	normalized := "/" + strings.TrimPrefix(filepath.ToSlash(filePath), "/")
	for _, pattern := range repoOrInfraPatterns {
		if strings.Contains(normalized, pattern) || strings.Contains(filePath, pattern) {
			return true
		}
	}
	return false
}

// DomainPurityRule enforces that domain entity/model tests contain zero mocks.
type DomainPurityRule struct{}

func NewDomainPurityRule() *DomainPurityRule {
	return &DomainPurityRule{}
}

func (r *DomainPurityRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if !strings.Contains(unit.FilePath, "/domain/") && !strings.Contains(unit.FilePath, "domain_test.go") {
		return violations
	}

	if mockControllerRegex.MatchString(unit.RawBody) {
		violations = append(violations, models.Violation{
			RuleID:     "GUARD-GO-01",
			Severity:   models.SeverityBlocker,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Domain purity violation in '%s': domain tests must be pure and contain zero mocks.", unit.Name),
			Suggestion: "Remove mock controllers from domain tests. Test business logic using pure domain struct fixtures.",
		})
	}

	return violations
}

// PortMockingRule verifies that service/usecase tests mock interface ports, not concrete database drivers.
type PortMockingRule struct{}

func NewPortMockingRule() *PortMockingRule {
	return &PortMockingRule{}
}

func (r *PortMockingRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if isRepositoryOrInfraLayer(unit.FilePath) {
		return violations
	}

	if sqlmockRegex.MatchString(unit.RawBody) {
		violations = append(violations, models.Violation{
			RuleID:     "GUARD-GO-02",
			Severity:   models.SeverityWarning,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Port mocking violation in '%s': mocking concrete database connection instead of exported interface Port.", unit.Name),
			Suggestion: "Define and mock an abstract repository interface (e.g. domain.UserRepository) rather than concrete database connections.",
		})
	}

	return violations
}

// TableDuplicateRule detects identical duplicate rows in Go table-driven tests.
type TableDuplicateRule struct{}

func NewTableDuplicateRule() *TableDuplicateRule {
	return &TableDuplicateRule{}
}

func (r *TableDuplicateRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if !strings.Contains(unit.RawBody, "struct") {
		return violations
	}

	matches := rowPatternRegex.FindAllStringSubmatch(unit.RawBody, -1)
	seenRows := make(map[string]bool)

	for _, m := range matches {
		rowContent := commentRegex.ReplaceAllString(m[1], "")
		rowContent = strings.TrimSpace(rowContent)
		if len(rowContent) < 5 || strings.Contains(rowContent, ";") {
			continue
		}

		if seenRows[rowContent] {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-GO-03",
				Severity:   models.SeverityAdvisory,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Duplicate table test row in '%s': row {%s} is duplicated.", unit.Name, rowContent),
				Suggestion: "Ensure each table test case tests a distinct equivalence class, boundary condition, or error scenario.",
			})
			break
		}
		seenRows[rowContent] = true
	}

	return violations
}
