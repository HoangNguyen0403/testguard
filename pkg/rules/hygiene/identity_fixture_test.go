package hygiene_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/hygiene"
)

func TestIdentityFixtureRule(t *testing.T) {
	rule := hygiene.NewIdentityFixtureRule()

	tests := []struct {
		name      string
		unit      parser.TestUnit
		wantWarn  bool
		wantLine  int
	}{
		{
			name: "TypeScript $0 rebate in pricing test triggers warning",
			unit: parser.TestUnit{
				Name:      "calculates net price with rebate",
				FilePath:  "src/pricing/calc.test.ts",
				StartLine: 10,
				RawBody: `
const rebate = 0;
const total = calculateNetPrice({ price: 100, rebate });
expect(total).toBe(100);
`,
			},
			wantWarn: true,
		},
		{
			name: "Dart 0.0 fee in amount calculation triggers warning",
			unit: parser.TestUnit{
				Name:      "computes total amount with fee",
				FilePath:  "test/checkout/total_test.dart",
				StartLine: 20,
				RawBody: `
final fee = 0.0;
final result = calculateTotal(price: 150.0, fee: fee);
expect(result, equals(150.0));
`,
			},
			wantWarn: true,
		},
		{
			name: "Go struct field with 0 discount in subtotal test triggers warning",
			unit: parser.TestUnit{
				Name:      "TestComputeSubtotal",
				FilePath:  "pkg/order/subtotal_test.go",
				StartLine: 15,
				RawBody: `
req := OrderCalcRequest{
    BasePrice: 200,
    Discount: 0,
}
res := ComputeSubtotal(req)
assert.Equal(t, 200, res.Subtotal)
`,
			},
			wantWarn: true,
		},
		{
			name: "Exemption: boundary title with zero rebate does not trigger",
			unit: parser.TestUnit{
				Name:      "returns base price when zero rebate applied",
				FilePath:  "src/pricing/calc.test.ts",
				StartLine: 10,
				RawBody: `
const rebate = 0;
const total = calculateNetPrice({ price: 100, rebate });
expect(total).toBe(100);
`,
			},
			wantWarn: false,
		},
		{
			name: "Exemption: boundary title with free or exempt does not trigger",
			unit: parser.TestUnit{
				Name:      "computes tax for exempt goods",
				FilePath:  "pkg/tax/tax_test.go",
				StartLine: 30,
				RawBody: `
tax := 0.0
total := CalculateTax(100.0, tax)
assert.Equal(t, 100.0, total)
`,
			},
			wantWarn: false,
		},
		{
			name: "Exemption: opposing non-zero fixture with non-zero operand does not trigger",
			unit: parser.TestUnit{
				Name:      "calculates net price with discount and surcharge",
				FilePath:  "src/pricing/calc.test.ts",
				StartLine: 40,
				RawBody: `
const discount = 15;
const surcharge = 5;
const total = calculateNetPrice({ price: 100, discount, surcharge });
expect(total).toBe(90);
`,
			},
			wantWarn: false,
		},
		{
			name: "Non-calculation test ignored",
			unit: parser.TestUnit{
				Name:      "renders user avatar",
				FilePath:  "src/components/avatar.test.tsx",
				StartLine: 5,
				RawBody: `
render(<Avatar name="Alice" offset={0} />);
expect(screen.getByText("Alice")).toBeInTheDocument();
`,
			},
			wantWarn: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations := rule.Evaluate(tc.unit)
			if tc.wantWarn {
				if len(violations) == 0 {
					t.Fatalf("expected violation, got 0")
				}
				v := violations[0]
				if v.RuleID != "GUARD-UNIV-IDENTITY-FIXTURE" {
					t.Errorf("expected RuleID GUARD-UNIV-IDENTITY-FIXTURE, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityWarning {
					t.Errorf("expected Severity Warning, got %s", v.Severity)
				}
			} else {
				if len(violations) != 0 {
					t.Fatalf("expected 0 violations, got %d: %+v", len(violations), violations)
				}
			}
		})
	}
}
