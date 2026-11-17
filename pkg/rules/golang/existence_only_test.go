package golang_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/golang"
)

func TestExistenceOnlyRule(t *testing.T) {
	rule := golang.NewExistenceOnlyRule()

	tests := []struct {
		name     string
		unit     parser.TestUnit
		wantWarn bool
	}{
		{
			name: "Multi-return call asserting only NoError and NotNil triggers warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_Success",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_Success(t *testing.T) {
    user, err := svc.GetUser(ctx, 123)
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Multi-return call asserting only err == nil and res != nil triggers warning",
			unit: parser.TestUnit{
				Name:      "TestFindOrder_Success",
				FilePath:  "internal/repo/order_test.go",
				StartLine: 20,
				RawBody: `
func TestFindOrder_Success(t *testing.T) {
    order, err := repo.Find(ctx, "ORD-1")
    if err != nil {
        t.Fatal(err)
    }
    if order == nil {
        t.Fatal("expected order")
    }
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Inspecting res.Field does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_InspectsField",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_InspectsField(t *testing.T) {
    user, err := svc.GetUser(ctx, 123)
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "Alice", user.Name)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Error expectation test does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_NotFound",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_NotFound(t *testing.T) {
    user, err := svc.GetUser(ctx, 999)
    assert.Error(t, err)
    assert.Nil(t, user)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Passing res into assertion helper does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_HelperValidation",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_HelperValidation(t *testing.T) {
    user, err := svc.GetUser(ctx, 123)
    assert.NoError(t, err)
    assertValidUser(t, user)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Single-return error call does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestDeleteUser_Success",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestDeleteUser_Success(t *testing.T) {
    err := svc.DeleteUser(ctx, 123)
    assert.NoError(t, err)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Standard library fixture call does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestMarshalPayload",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestMarshalPayload(t *testing.T) {
    bytes, err := json.Marshal(map[string]string{"foo": "bar"})
    assert.NoError(t, err)
    assert.NotNil(t, bytes)
}
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
				if v.RuleID != "GUARD-GO-EXISTENCE-ONLY" {
					t.Errorf("expected RuleID GUARD-GO-EXISTENCE-ONLY, got %s", v.RuleID)
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
