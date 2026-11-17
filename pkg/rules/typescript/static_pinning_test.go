package typescript_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/typescript"
)

func TestStaticPinningRule(t *testing.T) {
	rule := typescript.NewStaticPinningRule()

	tests := []struct {
		name     string
		unit     parser.TestUnit
		wantWarn bool
	}{
		{
			name: "print(GQL) string matching triggers warning",
			unit: parser.TestUnit{
				Name:      "orders query document contains required fields",
				FilePath:  "src/graphql/orders.test.ts",
				StartLine: 12,
				RawBody: `
const docStr = print(GET_ORDERS_QUERY);
expect(docStr).toContain("orderNumber");
expect(docStr).toContain("totalAmount");
`,
			},
			wantWarn: true,
		},
		{
			name: ".loc.source.body static matching triggers warning",
			unit: parser.TestUnit{
				Name:      "user query matches schema snapshot",
				FilePath:  "src/queries/user.test.ts",
				StartLine: 8,
				RawBody: `
expect(USER_QUERY.loc.source.body).toMatch(/userId/);
`,
			},
			wantWarn: true,
		},
		{
			name: "Object.keys translation json iteration triggers warning",
			unit: parser.TestUnit{
				Name:      "all translation keys are valid strings",
				FilePath:  "src/locales/i18n.test.ts",
				StartLine: 5,
				RawBody: `
Object.keys(enTranslations).forEach(key => {
    expect(typeof enTranslations[key]).toBe("string");
});
`,
			},
			wantWarn: true,
		},
		{
			name: "Dynamic query builder with arguments does not trigger",
			unit: parser.TestUnit{
				Name:      "builds marketplace query with discount filters",
				FilePath:  "src/queries/builder.test.ts",
				StartLine: 10,
				RawBody: `
const query = buildMarketplaceQuery({ market: "SG", withDiscounts: true });
expect(query).toContain("discountCode");
`,
			},
			wantWarn: false,
		},
		{
			name: "Component render with query execution does not trigger",
			unit: parser.TestUnit{
				Name:      "renders order list after query",
				FilePath:  "src/components/OrderList.test.tsx",
				StartLine: 15,
				RawBody: `
render(<OrderList query={GET_ORDERS_QUERY} />);
expect(screen.getByRole("table")).toBeInTheDocument();
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
				if v.RuleID != "GUARD-TS-STATIC-PINNING" {
					t.Errorf("expected RuleID GUARD-TS-STATIC-PINNING, got %s", v.RuleID)
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
