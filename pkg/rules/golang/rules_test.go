package golang_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/golang"
)

func TestDomainPurityRule(t *testing.T) {
	unitWithMockInDomain := parser.TestUnit{
		Name:      "TestUserValidation",
		FilePath:  "internal/domain/user_test.go",
		StartLine: 12,
		RawBody:   "ctrl := gomock.NewController(t); mockRepo := mocks.NewMockUserRepository(ctrl);",
	}

	checker := golang.NewDomainPurityRule()
	violations := checker.Evaluate(unitWithMockInDomain)

	if len(violations) != 1 {
		t.Fatalf("expected 1 domain purity violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-GO-01" {
		t.Errorf("expected GUARD-GO-01, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityBlocker {
		t.Errorf("expected Blocker severity, got %s", violations[0].Severity)
	}
}

func TestDomainPurityRule_QuotedSourceInStringOrCommentIgnored(t *testing.T) {
	checker := golang.NewDomainPurityRule()

	unit := parser.TestUnit{
		Name:      "TestDomainFixtureGenerator",
		FilePath:  "internal/domain/fixture_test.go",
		StartLine: 10,
		RawBody: `
		const expectedSnippet = "ctrl := gomock.NewController(t); mockRepo := mocks.NewMockUserRepository(ctrl);"
		// We used to do: ctrl := gomock.NewController(t)
		// Now we use pure domain entities.
		user := "valid_domain_user"
		_ = user
		`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for quoted/commented mocks in domain, got %d", len(violations))
	}
}

func TestDomainPurityRule_ExecutableCallDetectedWithExactLine(t *testing.T) {
	checker := golang.NewDomainPurityRule()

	unit := parser.TestUnit{
		Name:      "TestDomainWithExecutableMock",
		FilePath:  "internal/domain/account_test.go",
		StartLine: 40,
		RawBody: `func TestDomainWithExecutableMock(t *testing.T) {
	entity := "pure"
	_ = entity
	ctrl := gomock.NewController(t)
	_ = ctrl
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 domain purity violation, got %d", len(violations))
	}
	expectedLine := 43
	if violations[0].LineNumber != expectedLine {
		t.Errorf("expected violation at line %d, got %d", expectedLine, violations[0].LineNumber)
	}
}

func TestDomainPurityRule_AliasedImportDetected(t *testing.T) {
	checker := golang.NewDomainPurityRule()

	unit := parser.TestUnit{
		Name:      "TestDomainAliasedGomock",
		FilePath:  "internal/domain/user_test.go",
		StartLine: 20,
		RawBody: `func TestDomainAliasedGomock(t *testing.T) {
	ctrl := gmock.NewController(t)
	_ = ctrl
}`,
		FileImports: map[string]string{
			"gmock": "github.com/golang/mock/gomock",
		},
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for aliased gomock import, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-GO-01" {
		t.Errorf("expected GUARD-GO-01, got %s", violations[0].RuleID)
	}
}

func TestTableDuplicateRule(t *testing.T) {
	unitWithDuplicateRows := parser.TestUnit{
		Name:      "TestCalculateTax",
		FilePath:  "service/tax_test.go",
		StartLine: 15,
		RawBody: `
		tests := []struct{ in int; want int }{
			{in: 100, want: 10},
			{in: 100, want: 10}, // duplicate row!
		}
		`,
	}

	checker := golang.NewTableDuplicateRule()
	violations := checker.Evaluate(unitWithDuplicateRows)

	if len(violations) != 1 {
		t.Fatalf("expected 1 table duplicate violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-GO-03" {
		t.Errorf("expected GUARD-GO-03, got %s", violations[0].RuleID)
	}
}

func TestTableDuplicateRule_QuotedSourceInStringOrCommentIgnored(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	unit := parser.TestUnit{
		Name:      "TestTableSyntaxChecker",
		FilePath:  "service/tax_test.go",
		StartLine: 10,
		RawBody: `
		const example = ` + "`" + `
		tests := []struct{ in int; want int }{
			{in: 100, want: 10},
			{in: 100, want: 10},
		}
		` + "`" + `
		// tests := []struct{ in int }{ {1}, {1} }
		v := 42
		_ = v
		`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for quoted table rows, got %d", len(violations))
	}
}

func TestTableDuplicateRule_ScalarSlicesAndNestedPayloadsIgnored(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	// 1. Repeated scalar string slices must NOT be treated as table tests (R6)
	unitStringSlice := parser.TestUnit{
		Name:      "TestStringFixture",
		FilePath:  "service/parser_test.go",
		StartLine: 10,
		RawBody: `
		repeatedStrings := []string{"repeat", "repeat"}
		_ = repeatedStrings
		`,
	}
	if len(checker.Evaluate(unitStringSlice)) != 0 {
		t.Errorf("expected 0 violations for scalar string slice with repeated items")
	}

	// 2. Repeated number slices must NOT be treated as table tests (R6)
	unitNumberSlice := parser.TestUnit{
		Name:      "TestNumberFixture",
		FilePath:  "service/calc_test.go",
		StartLine: 10,
		RawBody: `
		repeatedNumbers := []int{100, 100, 200}
		_ = repeatedNumbers
		`,
	}
	if len(checker.Evaluate(unitNumberSlice)) != 0 {
		t.Errorf("expected 0 violations for scalar int slice with repeated items")
	}
}

func TestTableDuplicateRule_ShortDuplicateRowsDetected(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	// Short duplicate struct rows like {0} must be detected and not skipped by arbitrary length (R6)
	unit := parser.TestUnit{
		Name:      "TestShortRows",
		FilePath:  "service/math_test.go",
		StartLine: 20,
		RawBody: `func TestShortRows(t *testing.T) {
	tests := []struct{ n int }{
		{0},
		{0},
	}
	_ = tests
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for duplicate short rows {0}, got %d", len(violations))
	}
	expectedLine := 23
	if violations[0].LineNumber != expectedLine {
		t.Errorf("expected violation at line %d, got %d", expectedLine, violations[0].LineNumber)
	}
}

func TestTableDuplicateRule_DistinctBoundaryValuesPreserved(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	unit := parser.TestUnit{
		Name:      "TestBoundaryValues",
		FilePath:  "service/calc_test.go",
		StartLine: 1,
		RawBody: `
		tests := []struct{ in int; want int }{
			{in: 0, want: 0},
			{in: -1, want: 0},
			{in: 1, want: 1},
			{in: 100, want: 10},
			{in: 999999, want: 99999},
		}
		`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for distinct boundary values in table, got %d", len(violations))
	}
}

func TestTableDuplicateRule_ExecutableDuplicateRowDetectedWithExactLine(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	unit := parser.TestUnit{
		Name:      "TestDuplicateTableRowsExecutable",
		FilePath:  "service/pricing_test.go",
		StartLine: 50,
		RawBody: `func TestDuplicateTableRowsExecutable(t *testing.T) {
	tests := []struct{ in int; want int }{
		{in: 100, want: 10},
		{in: 100, want: 10},
	}
	_ = tests
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 duplicate table violation, got %d", len(violations))
	}
	expectedLine := 53
	if violations[0].LineNumber != expectedLine {
		t.Errorf("expected violation at line %d, got %d", expectedLine, violations[0].LineNumber)
	}
}

func TestTableDuplicateRule_NestedArrayAndMapPayloadsIgnored(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	units := []parser.TestUnit{
		{
			Name:      "TestGridFixture",
			FilePath:  "service/grid_test.go",
			StartLine: 10,
			RawBody: `
		grid := [][]int{{1}, {1}}
		_ = grid
		`,
		},
		{
			Name:      "TestLookupFixture",
			FilePath:  "service/lookup_test.go",
			StartLine: 10,
			RawBody: `
		lookup := []map[string]int{{"a": 1}, {"a": 1}}
		_ = lookup
		`,
		},
	}

	for _, unit := range units {
		if violations := checker.Evaluate(unit); len(violations) != 0 {
			t.Errorf("expected 0 violations for nested payload fixture %s, got %d", unit.Name, len(violations))
		}
	}
}

func TestTableDuplicateRule_NamedRowTypeAmbiguityAccepted(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	// Named row types cannot be proven structs without a type checker; the
	// syntactic row shape is accepted and duplicate rows are still reported.
	unit := parser.TestUnit{
		Name:      "TestNamedRows",
		FilePath:  "service/named_test.go",
		StartLine: 12,
		RawBody: `
		rows := []caseRow{{0}, {0}}
		_ = rows
		`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 duplicate row violation for named row type, got %d", len(violations))
	}
}

func TestDomainPurityRule_RealPackageCallSurvivesLaterShadow(t *testing.T) {
	checker := golang.NewDomainPurityRule()

	unit := parser.TestUnit{
		Name:      "TestDomainPackageCallBeforeShadow",
		FilePath:  "internal/domain/order_test.go",
		StartLine: 10,
		RawBody: `func TestDomainPackageCallBeforeShadow(t *testing.T) {
	ctrl := gomock.NewController(t)
	_ = ctrl
	{
		gomock := newFakeScope()
		other := gomock.NewController(t)
		_ = other
	}
	t.Fatal("control")
}`,
		FileImports: map[string]string{
			"gomock": "github.com/golang/mock/gomock",
		},
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected only the genuine package call to be reported, got %d (%+v)", len(violations), violations)
	}
	if violations[0].RuleID != "GUARD-GO-01" {
		t.Errorf("expected GUARD-GO-01, got %s", violations[0].RuleID)
	}
	if violations[0].LineNumber != 11 {
		t.Errorf("expected genuine package call at line 11, got %d", violations[0].LineNumber)
	}
}

func TestDomainPurityRule_ImportedPackageShadowedLocallyIgnored(t *testing.T) {
	checker := golang.NewDomainPurityRule()

	unit := parser.TestUnit{
		Name:      "TestShadowedGomock",
		FilePath:  "internal/domain/shadow_test.go",
		StartLine: 25,
		RawBody: `func TestShadowedGomock(t *testing.T) {
	gomock := newFakeScope()
	ctrl := gomock.NewController(t)
	_ = ctrl
}`,
		FileImports: map[string]string{
			"gomock": "github.com/golang/mock/gomock",
		},
	}

	if violations := checker.Evaluate(unit); len(violations) != 0 {
		t.Errorf("expected 0 violations when the imported identifier is shadowed locally, got %d", len(violations))
	}
}
