package typescript_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/typescript"
)

func TestNestJsControllerEchoRule(t *testing.T) {
	unit := parser.TestUnit{
		Name:      "findAll should return list",
		FilePath:  "src/users/users.controller.spec.ts",
		StartLine: 25,
		RawBody:   "jest.spyOn(service, 'findAll').mockResolvedValue(result);\nexpect(await controller.findAll()).toBe(result);",
	}

	checker := typescript.NewNestControllerEchoRule()
	violations := checker.Evaluate(unit)

	if len(violations) != 1 {
		t.Fatalf("expected 1 controller echo violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-NEST-01" {
		t.Errorf("expected GUARD-NEST-01, got %s", violations[0].RuleID)
	}
}

func TestNestJsUnmockedDbModuleRule(t *testing.T) {
	unit := parser.TestUnit{
		Name:      "UsersService",
		FilePath:  "src/users/users.service.spec.ts",
		StartLine: 10,
		RawBody:   "Test.createTestingModule({ imports: [TypeOrmModule.forRoot(config)] });",
	}

	checker := typescript.NewNestUnmockedDbModuleRule()
	violations := checker.Evaluate(unit)

	if len(violations) != 1 {
		t.Fatalf("expected 1 unmocked db violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-NEST-02" {
		t.Errorf("expected GUARD-NEST-02, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityBlocker {
		t.Errorf("expected Blocker severity, got %s", violations[0].Severity)
	}
}

func TestReactArbitrarySleepRule(t *testing.T) {
	unit := parser.TestUnit{
		Name:      "renders user details",
		FilePath:  "components/UserProfile.test.tsx",
		StartLine: 35,
		RawBody:   "render(<UserProfile />);\nawait new Promise((r) => setTimeout(r, 1000));\nexpect(screen.getByText('User')).toBeInTheDocument();",
	}

	checker := typescript.NewReactSleepRule()
	violations := checker.Evaluate(unit)

	if len(violations) != 1 {
		t.Fatalf("expected 1 arbitrary sleep violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-REACT-02" {
		t.Errorf("expected GUARD-REACT-02, got %s", violations[0].RuleID)
	}
}
