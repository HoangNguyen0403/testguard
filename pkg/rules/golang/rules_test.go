package golang_test

import (
	"os"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	goparser "github.com/HoangNguyen0403/testguard/pkg/parser/golang"
	"github.com/HoangNguyen0403/testguard/pkg/rules/golang"
)

func TestDomainPurityRule(t *testing.T) {
	unitWithMockInDomain := parser.TestUnit{
		Name:      "TestUserValidation",
		FilePath:  "internal/domain/user_test.go",
		StartLine: 12,
		RawBody:   "ctrl := gomock.NewController(t); mockRepo := mocks.NewMockUserRepository(ctrl);",
		FileImports: map[string]string{
			"gomock": "github.com/golang/mock/gomock",
			"mocks":  "example.com/app/internal/domain/mocks",
		},
	}

	checker := golang.NewDomainPurityRule()
	violations := checker.Evaluate(unitWithMockInDomain)

	if len(violations) != 1 {
		t.Fatalf("expected 1 domain purity violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-GO-01" {
		t.Errorf("expected GUARD-GO-01, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityAdvisory {
		t.Errorf("expected Advisory severity, got %s", violations[0].Severity)
	}
	if violations[0].Message != "Domain test 'TestUserValidation' uses mocks; consider pure fixtures if the team keeps domain tests mock-free." {
		t.Errorf("unexpected message: %s", violations[0].Message)
	}
	if violations[0].Suggestion != "Consider replacing mocks with pure domain fixtures if the team follows a mock-free domain test convention." {
		t.Errorf("unexpected suggestion: %s", violations[0].Suggestion)
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
		FileImports: map[string]string{
			"gomock": "github.com/golang/mock/gomock",
		},
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

// TestDomainPurityRule_NoImportCarrierIsNotGuessed pins the removal of the
// spelling-based fallback: without a resolved import map, a qualifier that
// happens to read "gomock" is not proof that the gomock package was imported.
func TestDomainPurityRule_NoImportCarrierIsNotGuessed(t *testing.T) {
	checker := golang.NewDomainPurityRule()

	unit := parser.TestUnit{
		Name:      "TestUnresolvedQualifier",
		FilePath:  "internal/domain/unresolved_test.go",
		StartLine: 5,
		RawBody: `func TestUnresolvedQualifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	_ = ctrl
}`,
	}

	if violations := checker.Evaluate(unit); len(violations) != 0 {
		t.Fatalf("expected no violation without a resolved import carrier, got %d", len(violations))
	}
}

func TestTableDuplicateRule(t *testing.T) {
	unitWithDuplicateRows := parser.TestUnit{
		Name:      "TestCalculateTax",
		FilePath:  "service/tax_test.go",
		StartLine: 15,
		RawBody: `func TestCalculateTax(t *testing.T) {
		tests := []struct{ in int; want int }{
			{in: 100, want: 10},
			{in: 100, want: 10}, // duplicate row!
		}
		for _, tc := range tests {
			t.Run("sub", func(t *testing.T) { _ = tc })
		}
		}`,
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
	for _, tc := range tests {
		t.Run("sub", func(t *testing.T) { _ = tc })
	}
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
		RawBody: `func TestBoundaryValues(t *testing.T) {
		tests := []struct{ in int; want int }{
			{in: 0, want: 0},
			{in: -1, want: 0},
			{in: 1, want: 1},
			{in: 100, want: 10},
			{in: 999999, want: 99999},
		}
		for _, tc := range tests {
			t.Run("sub", func(t *testing.T) { _ = tc })
		}
		}`,
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
	for _, tc := range tests {
		t.Run("sub", func(t *testing.T) { _ = tc })
	}
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
		RawBody: `func TestNamedRows(t *testing.T) {
		rows := []caseRow{{0}, {0}}
		for _, tc := range rows {
			t.Run("sub", func(t *testing.T) { _ = tc })
		}
		}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 duplicate row violation for named row type, got %d", len(violations))
	}
}

func TestTableDuplicateRule_Addendum2(t *testing.T) {
	checker := golang.NewTableDuplicateRule()

	t.Run("dedup-input fixture slice -> none", func(t *testing.T) {
		unit := parser.TestUnit{
			Name:      "TestDedupInputs",
			FilePath:  "service/dedup_test.go",
			StartLine: 10,
			RawBody: `func TestDedupInputs(t *testing.T) {
				inputs := []struct{ ID int }{{ID: 1}, {ID: 1}}
				_ = inputs
			}`,
		}
		if v := checker.Evaluate(unit); len(v) != 0 {
			t.Fatalf("expected 0 violations for dedup-input fixture slice, got %d: %+v", len(v), v)
		}
	})

	t.Run("mock Return slice with duplicate elements -> none", func(t *testing.T) {
		unit := parser.TestUnit{
			Name:      "TestMockReturnSlice",
			FilePath:  "service/repo_test.go",
			StartLine: 10,
			RawBody: `func TestMockReturnSlice(t *testing.T) {
				mock.EXPECT().Find().Return([]struct{ Code string }{{Code: "A"}, {Code: "A"}})
			}`,
		}
		if v := checker.Evaluate(unit); len(v) != 0 {
			t.Fatalf("expected 0 violations for mock Return slice, got %d: %+v", len(v), v)
		}
	})

	t.Run("ranged t.Run table with two rows identical except name -> 1", func(t *testing.T) {
		unit := parser.TestUnit{
			Name:      "TestRangedTableDuplicate",
			FilePath:  "service/calc_test.go",
			StartLine: 10,
			RawBody: `func TestRangedTableDuplicate(t *testing.T) {
				tests := []struct {
					name string
					in   int
					want int
				}{
					{name: "first case", in: 10, want: 20},
					{name: "second case", in: 10, want: 20},
				}
				for _, tc := range tests {
					t.Run(tc.name, func(t *testing.T) {
						if tc.in * 2 != tc.want { t.Fatal("fail") }
					})
				}
			}`,
		}
		if v := checker.Evaluate(unit); len(v) != 1 {
			t.Fatalf("expected 1 violation for identical rows except name, got %d: %+v", len(v), v)
		}
	})

	t.Run("ranged table with rows differing in want -> none", func(t *testing.T) {
		unit := parser.TestUnit{
			Name:      "TestRangedTableDistinct",
			FilePath:  "service/calc_test.go",
			StartLine: 10,
			RawBody: `func TestRangedTableDistinct(t *testing.T) {
				tests := []struct {
					name string
					in   int
					want int
				}{
					{name: "first case", in: 10, want: 20},
					{name: "second case", in: 10, want: 30},
				}
				for _, tc := range tests {
					t.Run(tc.name, func(t *testing.T) {
						if tc.in * 2 != tc.want { t.Fatal("fail") }
					})
				}
			}`,
		}
		if v := checker.Evaluate(unit); len(v) != 0 {
			t.Fatalf("expected 0 violations for rows differing in want, got %d: %+v", len(v), v)
		}
	})
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
func TestF10_NamedRowDistinctInputsNotDuplicate(t *testing.T) {
	checker := golang.NewTableDuplicateRule()
	unit := parser.TestUnit{
		Name:      "TestRows",
		FilePath:  "service/inputs_test.go",
		StartLine: 10,
		RawBody: `func TestRows(t *testing.T) {
		type row struct { input string; want int }
		cases := []row{{"first", 1}, {"second", 1}}
		for _, tc := range cases {
			t.Run(tc.input, func(t *testing.T) {
				if len(tc.input) != tc.want { t.Fatal(tc.input) }
			})
		}
	}`,
	}
	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for distinct input rows, got %d: %+v", len(violations), violations)
	}
}

func TestR3_PositionalStringAndBoolRowsNotDuplicate(t *testing.T) {
	checker := golang.NewTableDuplicateRule()
	unit := parser.TestUnit{
		Name:      "TestEnvRows",
		FilePath:  "service/env_test.go",
		StartLine: 10,
		RawBody: `func TestEnvRows(t *testing.T) {
		type envRow struct { env string; active bool }
		cases := []envRow{{"prd", true}, {"prod", true}}
		for _, tc := range cases {
			t.Run(tc.env, func(t *testing.T) {
				if !tc.active { t.Fatal(tc.env) }
			})
		}
	}`,
	}
	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for positional string and bool rows, got %d: %+v", len(violations), violations)
	}
}

func TestR3_RangeBodyConditionsOnNameFieldSkipped(t *testing.T) {
	checker := golang.NewTableDuplicateRule()
	unit := parser.TestUnit{
		Name:      "TestConditionsOnName",
		FilePath:  "service/cond_test.go",
		StartLine: 10,
		RawBody: `func TestConditionsOnName(t *testing.T) {
		cases := []struct { name string; want int }{
			{"case 1", 1},
			{"case 2", 1},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.name == "case 1" {
					// special logic
				}
			})
		}
	}`,
	}
	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations when condition branches on name field, got %d: %+v", len(violations), violations)
	}
}

func TestF11_UnrelatedRunReceiverNotTableTest(t *testing.T) {
	checker := golang.NewTableDuplicateRule()
	unit := parser.TestUnit{
		Name:      "TestData",
		FilePath:  "service/data_test.go",
		StartLine: 10,
		RawBody: `func TestData(t *testing.T) {
		type processor struct{}
		var p processor
		data := []struct{ n int }{{1}, {1}}
		for _, d := range data {
			p.Run(d.n)
		}
		if len(data) != 2 { t.Fatal(data) }
	}`,
	}
	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-test Run method, got %d: %+v", len(violations), violations)
	}
}

func TestF11_ShadowedTableNotRangedOver(t *testing.T) {
	checker := golang.NewTableDuplicateRule()
	unit := parser.TestUnit{
		Name:      "TestRows",
		FilePath:  "service/shadow_test.go",
		StartLine: 10,
		RawBody: `func TestRows(t *testing.T) {
		cases := []struct{ n int }{{1}, {1}}
		_ = cases
		{
			cases := []int{2, 3}
			for _, n := range cases {
				t.Run("unique", func(t *testing.T) {
					if n < 0 { t.Fatal(n) }
				})
			}
		}
	}`,
	}
	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for unused outer duplicate table, got %d: %+v", len(violations), violations)
	}
}

func TestN5_GO03NestedTestingParam(t *testing.T) {
	checker := golang.NewTableDuplicateRule()
	unit := parser.TestUnit{
		Name:      "TestRows",
		FilePath:  "service/nested_test.go",
		StartLine: 1,
		RawBody: `func TestRows(t *testing.T){
		cases:=[]struct{name string;n int}{{"a",1},{"b",1}};
		t.Run("parent",func(t *testing.T){
			for _,tc:=range cases{
				t.Run(tc.name,func(t *testing.T){
					if tc.n!=1{t.Fatal(tc.n)}
				})
			}
		})
	}`,
	}
	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for duplicate table row under nested t.Run param, got %d: %+v", len(violations), violations)
	}
	if violations[0].RuleID != "GUARD-GO-03" {
		t.Errorf("expected RuleID GUARD-GO-03, got %s", violations[0].RuleID)
	}
}

func TestEdgeE1_GO03TaglessSwitch(t *testing.T) {
	checker := golang.NewTableDuplicateRule()
	unit := parser.TestUnit{
		Name:      "TestRows",
		FilePath:  "service/tagless_test.go",
		StartLine: 1,
		RawBody: `func TestRows(t *testing.T){
		cases:=[]struct{name string;n int}{{"a",1},{"b",1}};
		for _,tc:=range cases{
			t.Run(tc.name,func(t *testing.T){
				switch{
				case tc.name=="a":
					if tc.n!=1{t.Fatal(tc.n)};
				default:
					if tc.n!=2{t.Fatal(tc.n)}
				}
			})
		}
	}`,
	}
	violations := checker.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations when tagless switch references name field, got %d: %+v", len(violations), violations)
	}
}

func TestSwallowedErrorRule_Positive(t *testing.T) {
	checker := golang.NewSwallowedErrorRule()

	unit := parser.TestUnit{
		Name:      "TestErrorIgnored",
		FilePath:  "weak_test.go",
		StartLine: 14,
		RawBody: `func TestErrorIgnored(t *testing.T) {
	got, err := Build("")
	if err != nil {
		return
	}
	assert.Equal(t, "x", got.Name)
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	v := violations[0]
	if v.RuleID != "GUARD-GO-SWALLOWED-ERR" {
		t.Errorf("expected GUARD-GO-SWALLOWED-ERR, got %s", v.RuleID)
	}
	if v.Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", v.Severity)
	}
	if v.LineNumber != 16 {
		t.Errorf("expected line 16, got %d", v.LineNumber)
	}
}

func TestSwallowedErrorRule_TRunClosure(t *testing.T) {
	checker := golang.NewSwallowedErrorRule()

	unit := parser.TestUnit{
		Name:      "TestInTRun",
		FilePath:  "weak_test.go",
		StartLine: 1,
		RawBody: `func TestInTRun(t *testing.T) {
	t.Run("sub", func(t *testing.T) {
		data, err := Fetch()
		if err != nil {
			return
		}
		_ = data
	})
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation in t.Run closure, got %d", len(violations))
	}
	if violations[0].LineNumber != 4 {
		t.Errorf("expected line 4, got %d", violations[0].LineNumber)
	}
}

func TestSwallowedErrorRule_NegativeCases(t *testing.T) {
	checker := golang.NewSwallowedErrorRule()

	cases := []struct {
		name string
		body string
	}{
		{
			name: "block has t.Fatal",
			body: `func TestFatal(t *testing.T) {
	got, err := Build("")
	if err != nil {
		t.Fatal(err)
		return
	}
	_ = got
}`,
		},
		{
			name: "block has t.Error",
			body: `func TestError(t *testing.T) {
	got, err := Build("")
	if err != nil {
		t.Error(err)
		return
	}
	_ = got
}`,
		},
		{
			name: "block has t.Skip",
			body: `func TestSkip(t *testing.T) {
	got, err := Build("")
	if err != nil {
		t.Skip(err)
		return
	}
	_ = got
}`,
		},
		{
			name: "block has panic",
			body: `func TestPanic(t *testing.T) {
	got, err := Build("")
	if err != nil {
		panic(err)
	}
	_ = got
}`,
		},
		{
			name: "block has log call",
			body: `func TestLog(t *testing.T) {
	got, err := Build("")
	if err != nil {
		log.Println(err)
		return
	}
	_ = got
}`,
		},
		{
			name: "inside defer func literal",
			body: `func TestDefer(t *testing.T) {
	defer func() {
		got, err := Build("")
		if err != nil {
			return
		}
		_ = got
	}()
}`,
		},
		{
			name: "inside goroutine func literal",
			body: `func TestGoroutine(t *testing.T) {
	go func() {
		got, err := Build("")
		if err != nil {
			return
		}
		_ = got
	}()
}`,
		},
		{
			name: "inside handler func literal",
			body: `func TestHandler(t *testing.T) {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := Build("")
		if err != nil {
			return
		}
		_ = got
	})
	_ = fn
}`,
		},
		{
			name: "not assigned from call",
			body: `func TestNotCall(t *testing.T) {
	var err error
	if err != nil {
		return
	}
}`,
		},
		{
			name: "if equality check",
			body: `func TestEq(t *testing.T) {
	got, err := Build("")
	if err == nil {
		return
	}
	_ = got
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      "TestX",
				FilePath:  "weak_test.go",
				StartLine: 1,
				RawBody:   tc.body,
			}
			violations := checker.Evaluate(unit)
			if len(violations) != 0 {
				t.Errorf("expected 0 violations for %s, got %d: %+v", tc.name, len(violations), violations)
			}
		})
	}
}

func TestUncheckedWantRule_Positive(t *testing.T) {
	checker := golang.NewUncheckedWantRule()

	unit := parser.TestUnit{
		Name:      "TestTableUncheckedWant",
		FilePath:  "table_test.go",
		StartLine: 1,
		RawBody: `func TestTableUncheckedWant(t *testing.T) {
	tests := []struct {
		name    string
		in      int
		want    int
		wantErr bool
	}{
		{name: "ok", in: 1, want: 2, wantErr: false},
		{name: "fail", in: 2, want: 4, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in * 2; got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for unchecked wantErr, got %d: %+v", len(violations), violations)
	}
	v := violations[0]
	if v.RuleID != "GUARD-GO-UNCHECKED-WANT" {
		t.Errorf("expected GUARD-GO-UNCHECKED-WANT, got %s", v.RuleID)
	}
	if v.Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", v.Severity)
	}
	if v.LineNumber != 6 {
		t.Errorf("expected field declaration line 6, got %d", v.LineNumber)
	}
	if !strings.Contains(v.Message, "wantErr") {
		t.Errorf("expected message to mention 'wantErr', got %s", v.Message)
	}
}

func TestUncheckedWantRule_MultipleUncheckedFields(t *testing.T) {
	checker := golang.NewUncheckedWantRule()

	unit := parser.TestUnit{
		Name:      "TestMultiUnchecked",
		FilePath:  "table_test.go",
		StartLine: 1,
		RawBody: `func TestMultiUnchecked(t *testing.T) {
	tests := []struct {
		name         string
		expected     string
		expectedCode int
	}{
		{name: "a", expected: "ok", expectedCode: 200},
		{name: "b", expected: "fail", expectedCode: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.name
		})
	}
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d: %+v", len(violations), violations)
	}
	lines := []int{violations[0].LineNumber, violations[1].LineNumber}
	if lines[0] != 4 || lines[1] != 5 {
		t.Errorf("expected lines 4 and 5, got %v", lines)
	}
}

func TestUncheckedWantRule_NegativeCases(t *testing.T) {
	checker := golang.NewUncheckedWantRule()

	cases := []struct {
		name string
		body string
	}{
		{
			name: "all want fields read",
			body: `func TestAllRead(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{name: "a", want: 1, wantErr: false},
		{name: "b", want: 2, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.want
			_ = tt.wantErr
		})
	}
}`,
		},
		{
			name: "range variable passed whole to helper call",
			body: `func TestPassedWhole(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: "a", want: 1},
		{name: "b", want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, tt)
		})
	}
}`,
		},
		{
			name: "table not ranged with t.Run",
			body: `func TestNoTRun(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: "a", want: 1},
		{name: "b", want: 2},
	}
	for _, tt := range tests {
		_ = tt.name
	}
}`,
		},
		{
			name: "external unresolved row type",
			body: `func TestUnresolvedRowType(t *testing.T) {
	tests := []ExternalRow{
		{Name: "a", Want: 1},
		{Name: "b", Want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			_ = tt.Name
		})
	}
}`,
		},
		{
			name: "non-want field unread",
			body: `func TestNonWantUnread(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		comment string
	}{
		{name: "a", want: 1, comment: "note a"},
		{name: "b", want: 2, comment: "note b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.want
		})
	}
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      "TestX",
				FilePath:  "table_test.go",
				StartLine: 1,
				RawBody:   tc.body,
			}
			violations := checker.Evaluate(unit)
			if len(violations) != 0 {
				t.Errorf("expected 0 violations for %s, got %d: %+v", tc.name, len(violations), violations)
			}
		})
	}
}

func TestProof_OrderServiceTest_GuardGo01IsAdvisory(t *testing.T) {
	checker := golang.NewDomainPurityRule()

	unit := parser.TestUnit{
		Name:      "TestOrderServiceRejectsWhenStockMissing",
		FilePath:  "internal/domain/order_service_test.go",
		StartLine: 12,
		RawBody: `func TestOrderServiceRejectsWhenStockMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stock := mocks.NewMockStockPort(ctrl)
	stock.EXPECT().Available("sku-1").Return(0, nil)

	svc := NewOrderService(stock)
	err := svc.Place(Order{SKU: "sku-1", Qty: 2})

	assert.ErrorIs(t, err, ErrOutOfStock)
}`,
		FileImports: map[string]string{
			"gomock": "github.com/golang/mock/gomock",
			"mocks":  "example.com/app/internal/domain/mocks",
			"assert": "github.com/stretchr/testify/assert",
		},
	}

	violations := checker.Evaluate(unit)
	if len(violations) == 0 {
		t.Fatalf("expected GUARD-GO-01 violations on domain test with mocks, got 0")
	}
	for _, v := range violations {
		if v.RuleID != "GUARD-GO-01" {
			t.Errorf("expected rule GUARD-GO-01, got %s", v.RuleID)
		}
		if v.Severity != models.SeverityAdvisory {
			t.Errorf("expected Advisory severity, got %s", v.Severity)
		}
	}
}

func TestProof_WeakTest_SwallowedErrorAndLogsOnly(t *testing.T) {
	adapter := goparser.NewAdapter()
	source := []byte(`package weak

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExistenceOnly(t *testing.T) {
	got, _ := Build("x")
	assert.NotNil(t, got)
}

func TestErrorIgnored(t *testing.T) {
	got, err := Build("")
	if err != nil {
		return
	}
	assert.Equal(t, "x", got.Name)
}

func TestLogsOnly(t *testing.T) {
	got, err := Build("x")
	t.Logf("got=%v err=%v", got, err)
}

func TestNoErrorOnly(t *testing.T) {
	_, err := Build("x")
	assert.NoError(t, err)
}
`)
	units, err := adapter.LocateTests(source, "weak_test.go")
	if err != nil {
		t.Fatalf("unexpected error locating tests: %v", err)
	}

	unitMap := make(map[string]parser.TestUnit)
	for _, u := range units {
		unitMap[u.Name] = u
	}

	// TestLogsOnly has 0 assertions
	uLogs := unitMap["TestLogsOnly"]
	if len(uLogs.Assertions) != 0 {
		t.Errorf("expected TestLogsOnly to have 0 assertions, got %v", uLogs.Assertions)
	}

	// TestErrorIgnored triggers GUARD-GO-SWALLOWED-ERR at line 16
	swallowedChecker := golang.NewSwallowedErrorRule()
	violations := swallowedChecker.Evaluate(unitMap["TestErrorIgnored"])
	if len(violations) != 1 {
		t.Fatalf("expected 1 swallowed error violation for TestErrorIgnored, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-GO-SWALLOWED-ERR" {
		t.Errorf("expected GUARD-GO-SWALLOWED-ERR, got %s", violations[0].RuleID)
	}
	if violations[0].LineNumber != 16 {
		t.Errorf("expected line 16 for if err != nil, got %d", violations[0].LineNumber)
	}
}

func TestUncheckedWantRule_NamedRowType(t *testing.T) {
	checker := golang.NewUncheckedWantRule()

	unit := parser.TestUnit{
		Name:      "TestTableNamedRowUnchecked",
		FilePath:  "adv_test.go",
		StartLine: 1,
		RawBody: `func TestTableNamedRowUnchecked(t *testing.T) {
	type row struct {
		name     string
		in       string
		want     int
		wantErr  bool
		expected string
	}
	tests := []row{{"one", "1", 1, false, "x"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := Parse(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations for wantErr and expected, got %d: %+v", len(violations), violations)
	}
	if violations[0].LineNumber != 6 || !strings.Contains(violations[0].Message, "wantErr") {
		t.Errorf("expected violation 1 for wantErr at line 6, got %+v", violations[0])
	}
	if violations[1].LineNumber != 7 || !strings.Contains(violations[1].Message, "expected") {
		t.Errorf("expected violation 2 for expected at line 7, got %+v", violations[1])
	}
}

func TestUncheckedWantRule_PackageLevelNamedRowType(t *testing.T) {
	checker := golang.NewUncheckedWantRule()

	unit := parser.TestUnit{
		Name:      "TestTablePkgNamedRowUnchecked",
		FilePath:  "adv_pkg_test.go",
		StartLine: 8,
		RawBody: `type pkgRow struct {
	name     string
	in       string
	want     int
	wantErr  bool
}

func TestTablePkgNamedRowUnchecked(t *testing.T) {
	tests := []pkgRow{{"one", "1", 1, false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := Parse(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}`,
	}

	violations := checker.Evaluate(unit)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for package-level named row wantErr, got %d: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "wantErr") {
		t.Errorf("expected violation for wantErr, got %+v", violations[0])
	}
}

func TestProof_CorpusFixtures(t *testing.T) {
	adapter := goparser.NewAdapter()

	// 1. go_gomega_assert
	gomegaSrc, err := os.ReadFile("../../engine/testdata/corpus/go_gomega_assert/gomega_test.go")
	if err != nil {
		t.Fatalf("failed reading gomega fixture: %v", err)
	}
	gomegaUnits, err := adapter.LocateTests(gomegaSrc, "gomega_test.go")
	if err != nil || len(gomegaUnits) != 1 || len(gomegaUnits[0].Assertions) == 0 {
		t.Errorf("go_gomega_assert expected non-empty assertions, got %v", gomegaUnits)
	}

	// 2. go_quicktest_assert
	qtSrc, err := os.ReadFile("../../engine/testdata/corpus/go_quicktest_assert/quicktest_test.go")
	if err != nil {
		t.Fatalf("failed reading quicktest fixture: %v", err)
	}
	qtUnits, err := adapter.LocateTests(qtSrc, "quicktest_test.go")
	if err != nil || len(qtUnits) != 1 || len(qtUnits[0].Assertions) == 0 {
		t.Errorf("go_quicktest_assert expected non-empty assertions, got %v", qtUnits)
	}

	// 3. go_external_helper
	extSrc, err := os.ReadFile("../../engine/testdata/corpus/go_external_helper/external_helper_test.go")
	if err != nil {
		t.Fatalf("failed reading external helper fixture: %v", err)
	}
	extUnits, err := adapter.LocateTests(extSrc, "external_helper_test.go")
	if err != nil || len(extUnits) != 1 || len(extUnits[0].Assertions) != 1 || extUnits[0].Assertions[0] != "external-helper:testutil.AssertEqual" {
		t.Errorf("go_external_helper expected external-helper:testutil.AssertEqual, got %v", extUnits)
	}

	// 4. go_domain_purity_advisory (the retained domain/ copy; the directory-root
	// copy stays in place as a path-gate negative asserted by the corpus table)
	orderSrc, err := os.ReadFile("../../engine/testdata/corpus/go_domain_purity_advisory/domain/order_service_test.go")
	if err != nil {
		t.Fatalf("failed reading domain purity fixture: %v", err)
	}
	orderUnits, err := adapter.LocateTests(orderSrc, "internal/domain/order_service_test.go")
	if err != nil || len(orderUnits) != 1 {
		t.Fatalf("expected 1 unit in order_service_test.go, got %v", orderUnits)
	}
	purityRule := golang.NewDomainPurityRule()
	orderViolations := purityRule.Evaluate(orderUnits[0])
	if len(orderViolations) == 0 {
		t.Fatalf("expected domain purity violations on order_service_test.go")
	}
	for _, v := range orderViolations {
		if v.Severity != models.SeverityAdvisory {
			t.Errorf("expected Advisory severity, got %s", v.Severity)
		}
	}

	// 5. go_swallowed_err_positive
	swallowPosSrc, err := os.ReadFile("../../engine/testdata/corpus/go_swallowed_err_positive/swallowed_err_test.go")
	if err != nil {
		t.Fatalf("failed reading swallowed error positive fixture: %v", err)
	}
	swallowPosUnits, err := adapter.LocateTests(swallowPosSrc, "swallowed_err_test.go")
	if err != nil || len(swallowPosUnits) != 1 {
		t.Fatalf("expected 1 unit in swallowed_err_test.go, got %v", swallowPosUnits)
	}
	swallowRule := golang.NewSwallowedErrorRule()
	swallowPosViolations := swallowRule.Evaluate(swallowPosUnits[0])
	if len(swallowPosViolations) != 1 || swallowPosViolations[0].LineNumber != 11 {
		t.Errorf("expected 1 swallowed error violation at line 11, got %+v", swallowPosViolations)
	}

	// 6. go_swallowed_err_negative
	swallowNegSrc, err := os.ReadFile("../../engine/testdata/corpus/go_swallowed_err_negative/swallowed_err_neg_test.go")
	if err != nil {
		t.Fatalf("failed reading swallowed error negative fixture: %v", err)
	}
	swallowNegUnits, err := adapter.LocateTests(swallowNegSrc, "swallowed_err_neg_test.go")
	if err != nil || len(swallowNegUnits) != 1 {
		t.Fatalf("expected 1 unit in swallowed_err_neg_test.go, got %v", swallowNegUnits)
	}
	swallowNegViolations := swallowRule.Evaluate(swallowNegUnits[0])
	if len(swallowNegViolations) != 0 {
		t.Errorf("expected 0 violations for swallowed_err_neg_test.go, got %+v", swallowNegViolations)
	}

	// 7. go_unchecked_want_positive
	wantPosSrc, err := os.ReadFile("../../engine/testdata/corpus/go_unchecked_want_positive/table_unchecked_want_test.go")
	if err != nil {
		t.Fatalf("failed reading unchecked want positive fixture: %v", err)
	}
	wantPosUnits, err := adapter.LocateTests(wantPosSrc, "table_unchecked_want_test.go")
	if err != nil || len(wantPosUnits) != 1 {
		t.Fatalf("expected 1 unit in table_unchecked_want_test.go, got %v", wantPosUnits)
	}
	wantRule := golang.NewUncheckedWantRule()
	wantPosViolations := wantRule.Evaluate(wantPosUnits[0])
	if len(wantPosViolations) != 1 || wantPosViolations[0].LineNumber != 10 || !strings.Contains(wantPosViolations[0].Message, "wantErr") {
		t.Errorf("expected 1 unchecked want violation on wantErr at line 10, got %+v", wantPosViolations)
	}

	// 8. go_unchecked_want_negative
	wantNegSrc, err := os.ReadFile("../../engine/testdata/corpus/go_unchecked_want_negative/table_checked_want_test.go")
	if err != nil {
		t.Fatalf("failed reading unchecked want negative fixture: %v", err)
	}
	wantNegUnits, err := adapter.LocateTests(wantNegSrc, "table_checked_want_test.go")
	if err != nil || len(wantNegUnits) != 1 {
		t.Fatalf("expected 1 unit in table_checked_want_test.go, got %v", wantNegUnits)
	}
	wantNegViolations := wantRule.Evaluate(wantNegUnits[0])
	if len(wantNegViolations) != 0 {
		t.Errorf("expected 0 violations for table_checked_want_test.go, got %+v", wantNegViolations)
	}

	// 9. go_unchecked_want_named_positive
	namedPosSrc, err := os.ReadFile("../../engine/testdata/corpus/go_unchecked_want_named_positive/table_named_unchecked_test.go")
	if err != nil {
		t.Fatalf("failed reading named want positive fixture: %v", err)
	}
	namedPosUnits, err := adapter.LocateTests(namedPosSrc, "table_named_unchecked_test.go")
	if err != nil || len(namedPosUnits) != 1 {
		t.Fatalf("expected 1 unit in table_named_unchecked_test.go, got %v", namedPosUnits)
	}
	namedPosViolations := wantRule.Evaluate(namedPosUnits[0])
	if len(namedPosViolations) != 2 {
		t.Errorf("expected 2 violations for table_named_unchecked_test.go, got %+v", namedPosViolations)
	}

	// 10. go_unchecked_want_named_negative
	namedNegSrc, err := os.ReadFile("../../engine/testdata/corpus/go_unchecked_want_named_negative/table_named_checked_test.go")
	if err != nil {
		t.Fatalf("failed reading named want negative fixture: %v", err)
	}
	namedNegUnits, err := adapter.LocateTests(namedNegSrc, "table_named_checked_test.go")
	if err != nil || len(namedNegUnits) != 1 {
		t.Fatalf("expected 1 unit in table_named_checked_test.go, got %v", namedNegUnits)
	}
	namedNegViolations := wantRule.Evaluate(namedNegUnits[0])
	if len(namedNegViolations) != 0 {
		t.Errorf("expected 0 violations for table_named_checked_test.go, got %+v", namedNegViolations)
	}

	// 11. go_package_helper_assert
	pkgHelperSrc, err := os.ReadFile("../../engine/testdata/corpus/go_package_helper_assert/pkg_helper_test.go")
	if err != nil {
		t.Fatalf("failed reading package helper fixture: %v", err)
	}
	pkgHelperUnits, err := adapter.LocateTests(pkgHelperSrc, "pkg_helper_test.go")
	if err != nil || len(pkgHelperUnits) != 1 || len(pkgHelperUnits[0].Assertions) != 1 || pkgHelperUnits[0].Assertions[0] != "package-helper:checkRow" {
		t.Errorf("expected package-helper:checkRow for pkg_helper_test.go, got %v", pkgHelperUnits)
	}

	// 12. go_package_helper_negative
	pkgNegSrc, err := os.ReadFile("../../engine/testdata/corpus/go_package_helper_negative/pkg_setup_test.go")
	if err != nil {
		t.Fatalf("failed reading package helper negative fixture: %v", err)
	}
	pkgNegUnits, err := adapter.LocateTests(pkgNegSrc, "pkg_setup_test.go")
	if err != nil || len(pkgNegUnits) != 1 || len(pkgNegUnits[0].Assertions) != 0 {
		t.Errorf("expected 0 assertions for pkg_setup_test.go, got %v", pkgNegUnits)
	}
}

func TestProof_AdversarialSuite(t *testing.T) {
	adapter := goparser.NewAdapter()
	source := []byte(`package adv

import (
	"os"
	"reflect"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"example.com/app/internal/testutil"
)

func TestDeferredCloseReturn(t *testing.T) {
	f, err := os.CreateTemp("", "x")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			return
		}
	}()
	assert.NotEmpty(t, f.Name())
}

func TestHelperClosureReturn(t *testing.T) {
	parse := func(v string) int {
		n, err := Parse(v)
		if err != nil {
			return 0
		}
		return n
	}
	assert.Equal(t, 1, parse("1"))
}

func TestSkipsWithoutNetwork(t *testing.T) {
	conn, err := Dial()
	if err != nil {
		t.Skip("no network")
	}
	assert.NotNil(t, conn)
}

func TestLogsThenReturns(t *testing.T) {
	v, err := Parse("1")
	if err != nil {
		t.Logf("parse failed: %v", err)
		return
	}
	assert.Equal(t, 1, v)
}

func TestSubtestSwallows(t *testing.T) {
	t.Run("one", func(t *testing.T) {
		v, err := Parse("1")
		if err != nil {
			return
		}
		assert.Equal(t, 1, v)
	})
}

func TestTableWholeRowPassed(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{{"one", "1", 1}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkRow(t, tt)
		})
	}
}

func TestTableShadowedRow(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{{"one", "1", 1, false}}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v", err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTableUnrelatedExpPrefix(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		exports []string
		expiry  int
		want    int
	}{{"one", "1", nil, 0, 1}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := Parse(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTableNamedRowUnchecked(t *testing.T) {
	type row struct {
		name     string
		in       string
		want     int
		wantErr  bool
		expected string
	}
	tests := []row{{"one", "1", 1, false, "x"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := Parse(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExternalSetupOnly(t *testing.T) {
	testutil.Setup(t)
	Do()
}

func TestGomegaWithoutMatcher(t *testing.T) {
	g := NewWithT(t)
	g.Expect(Add(1, 2))
}

func TestDeepEqualResultDropped(t *testing.T) {
	reflect.DeepEqual(Add(1, 2), 3)
}

func TestGomegaPackageLevel(t *testing.T) {
	RegisterTestingT(t)
	Expect(Add(1, 2)).To(Equal(3))
}

func TestGoldenHelper(t *testing.T) {
	testutil.CompareGolden(t, "out.golden", Render())
}
`)

	units, err := adapter.LocateTests(source, "adversarial_test.go")
	if err != nil {
		t.Fatalf("unexpected error locating tests: %v", err)
	}

	unitMap := make(map[string]parser.TestUnit)
	for _, u := range units {
		unitMap[u.Name] = u
	}

	// 1. GUARD-GO-UNCHECKED-WANT exactly twice in TestTableNamedRowUnchecked; none in others
	wantRule := golang.NewUncheckedWantRule()
	namedViolations := wantRule.Evaluate(unitMap["TestTableNamedRowUnchecked"])
	if len(namedViolations) != 2 {
		t.Fatalf("expected exactly 2 violations for TestTableNamedRowUnchecked, got %d: %+v", len(namedViolations), namedViolations)
	}
	if !strings.Contains(namedViolations[0].Message, "wantErr") || !strings.Contains(namedViolations[1].Message, "expected") {
		t.Errorf("expected wantErr and expected violations, got %+v", namedViolations)
	}

	for _, name := range []string{"TestTableWholeRowPassed", "TestTableShadowedRow", "TestTableUnrelatedExpPrefix"} {
		if v := wantRule.Evaluate(unitMap[name]); len(v) != 0 {
			t.Errorf("expected 0 unchecked want violations for %s, got %+v", name, v)
		}
	}

	// 2. TestTableWholeRowPassed has assertion package-helper:checkRow
	uWhole := unitMap["TestTableWholeRowPassed"]
	if len(uWhole.Assertions) != 1 || uWhole.Assertions[0] != "package-helper:checkRow" {
		t.Errorf("expected [package-helper:checkRow] for TestTableWholeRowPassed, got %v", uWhole.Assertions)
	}

	// 3. Zero assertions remain for TestExternalSetupOnly, TestGomegaWithoutMatcher, TestDeepEqualResultDropped
	for _, name := range []string{"TestExternalSetupOnly", "TestGomegaWithoutMatcher", "TestDeepEqualResultDropped"} {
		if len(unitMap[name].Assertions) != 0 {
			t.Errorf("expected 0 assertions for %s, got %v", name, unitMap[name].Assertions)
		}
	}

	// 4. GUARD-GO-SWALLOWED-ERR exactly once, in TestSubtestSwallows
	swallowRule := golang.NewSwallowedErrorRule()
	subtestViolations := swallowRule.Evaluate(unitMap["TestSubtestSwallows"])
	if len(subtestViolations) != 1 || subtestViolations[0].RuleID != "GUARD-GO-SWALLOWED-ERR" {
		t.Fatalf("expected exactly 1 GUARD-GO-SWALLOWED-ERR for TestSubtestSwallows, got %+v", subtestViolations)
	}

	for _, name := range []string{"TestDeferredCloseReturn", "TestHelperClosureReturn", "TestSkipsWithoutNetwork", "TestLogsThenReturns"} {
		if v := swallowRule.Evaluate(unitMap[name]); len(v) != 0 {
			t.Errorf("expected 0 swallowed error violations for %s, got %+v", name, v)
		}
	}
}
