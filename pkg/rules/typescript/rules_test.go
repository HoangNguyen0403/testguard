package typescript_test

import (
	"os"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	ts "github.com/HoangNguyen0403/testguard/pkg/parser/typescript"
	"github.com/HoangNguyen0403/testguard/pkg/rules/typescript"
)

func TestNestJsControllerEchoRule(t *testing.T) {
	// Inline echo
	unit1 := parser.TestUnit{
		Name:      "findAll should return list",
		FilePath:  "src/users/users.controller.spec.ts",
		StartLine: 25,
		RawBody:   "jest.spyOn(service, 'findAll').mockResolvedValue(result);\nexpect(await controller.findAll()).toBe(result);",
	}

	checker := typescript.NewNestControllerEchoRule()
	violations1 := checker.Evaluate(unit1)
	if len(violations1) != 1 {
		t.Fatalf("expected 1 controller echo violation for inline, got %d", len(violations1))
	}
	if violations1[0].RuleID != "GUARD-NEST-01" {
		t.Errorf("expected GUARD-NEST-01, got %s", violations1[0].RuleID)
	}

	// Assigned result form (AC4)
	unit2 := parser.TestUnit{
		Name:      "echoes the stub through an assigned result",
		FilePath:  "src/orders/orders.controller.spec.ts",
		StartLine: 19,
		RawBody: `
    const orders = [{ id: 1 }];
    service.findAll.mockResolvedValue(orders);
    const result = await controller.findAll();
    expect(result).toEqual(orders);
`,
	}
	violations2 := checker.Evaluate(unit2)
	if len(violations2) != 1 {
		t.Fatalf("expected 1 controller echo violation for assigned result, got %d", len(violations2))
	}
	if violations2[0].RuleID != "GUARD-NEST-01" {
		t.Errorf("expected GUARD-NEST-01, got %s", violations2[0].RuleID)
	}
	if violations2[0].LineNumber != 19 {
		t.Errorf("expected LineNumber 19, got %d", violations2[0].LineNumber)
	}

	// Negative case: controller modifies or transforms result
	unit3 := parser.TestUnit{
		Name:      "transforms result",
		FilePath:  "src/orders/orders.controller.spec.ts",
		StartLine: 35,
		RawBody: `
    const orders = [{ id: 1 }];
    service.findAll.mockResolvedValue(orders);
    const result = await controller.findAll();
    expect(result).toEqual([{ id: 1, processed: true }]);
`,
	}
	if len(checker.Evaluate(unit3)) != 0 {
		t.Errorf("expected no violation when return value differs from mock")
	}
}

func TestNestJsUnmockedDbModuleRule(t *testing.T) {
	checker := typescript.NewNestUnmockedDbModuleRule()

	// Positive: orders.controller.spec.ts imports TypeOrmModule.forRoot at line 12
	contentOrders, err := os.ReadFile("testdata/orders.controller.spec.ts")
	if err != nil {
		t.Fatalf("read orders.controller.spec.ts: %v", err)
	}
	vOrders := checker.EvaluateFile("src/orders/orders.controller.spec.ts", string(contentOrders), nil)
	if len(vOrders) != 1 {
		t.Fatalf("expected 1 unmocked db violation in orders.controller.spec.ts, got %d", len(vOrders))
	}
	if vOrders[0].RuleID != "GUARD-NEST-02" {
		t.Errorf("expected GUARD-NEST-02, got %s", vOrders[0].RuleID)
	}
	if vOrders[0].Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", vOrders[0].Severity)
	}
	if vOrders[0].LineNumber != 12 {
		t.Errorf("expected LineNumber 12, got %d", vOrders[0].LineNumber)
	}

	// Negative: users.integration.spec.ts (integration file, :memory:, overrideProvider)
	contentUsers, err := os.ReadFile("testdata/users.integration.spec.ts")
	if err != nil {
		t.Fatalf("read users.integration.spec.ts: %v", err)
	}
	vUsers := checker.EvaluateFile("src/users/users.integration.spec.ts", string(contentUsers), nil)
	if len(vUsers) != 0 {
		t.Errorf("expected 0 violations for integration file, got %d", len(vUsers))
	}
}

func TestSnapshotOnlyRule(t *testing.T) {
	checker := typescript.NewSnapshotOnlyRule()

	// Positive: only snapshot matcher
	unitPos := parser.TestUnit{
		Name:       "renders profile",
		FilePath:   "src/Profile.test.tsx",
		StartLine:  10,
		RawBody:    "expect(render(<Profile />)).toMatchSnapshot();",
		Assertions: []string{"expect(render(<Profile />)).toMatchSnapshot()"},
	}
	vPos := checker.Evaluate(unitPos)
	if len(vPos) != 1 {
		t.Fatalf("expected 1 snapshot-only violation, got %d", len(vPos))
	}
	if vPos[0].RuleID != "GUARD-TS-01" {
		t.Errorf("expected GUARD-TS-01, got %s", vPos[0].RuleID)
	}
	if vPos[0].Severity != models.SeverityAdvisory {
		t.Errorf("expected Advisory severity, got %s", vPos[0].Severity)
	}

	// Negative: has snapshot and behavioral assertion
	unitNeg := parser.TestUnit{
		Name:       "renders profile",
		FilePath:   "src/Profile.test.tsx",
		StartLine:  10,
		RawBody:    "expect(screen.getByText('Alice')).toBeVisible();\nexpect(render(<Profile />)).toMatchSnapshot();",
		Assertions: []string{"toBeVisible", "toMatchSnapshot"},
	}
	if len(checker.Evaluate(unitNeg)) != 0 {
		t.Errorf("expected 0 violations when behavioral assertion is present")
	}

	// Negative: E2E file
	unitE2E := parser.TestUnit{
		Name:       "visual comparison in e2e",
		FilePath:   "e2e/home.e2e.ts",
		StartLine:  10,
		RawBody:    "expect(page).toMatchSnapshot();",
		Assertions: []string{"toMatchSnapshot"},
	}
	if len(checker.Evaluate(unitE2E)) != 0 {
		t.Errorf("expected 0 violations for E2E file")
	}
}

func TestFloatingAssertRule(t *testing.T) {
	checker := typescript.NewFloatingAssertRule()

	// Positive: un-awaited resolves
	unitPos := parser.TestUnit{
		Name:      "floating async assertion",
		FilePath:  "weak.test.ts",
		StartLine: 22,
		RawBody:   "expect(fetchUser('a')).resolves.toEqual({ id: 'a' });",
	}
	vPos := checker.Evaluate(unitPos)
	if len(vPos) != 1 {
		t.Fatalf("expected 1 floating assert violation, got %d", len(vPos))
	}
	if vPos[0].RuleID != "GUARD-TS-FLOATING-ASSERT" {
		t.Errorf("expected GUARD-TS-FLOATING-ASSERT, got %s", vPos[0].RuleID)
	}
	if vPos[0].Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", vPos[0].Severity)
	}

	// Negative: awaited
	unitAwaited := parser.TestUnit{
		Name:      "awaited assertion",
		FilePath:  "valid.test.ts",
		StartLine: 22,
		RawBody:   "await expect(fetchUser('a')).resolves.toEqual({ id: 'a' });",
	}
	if len(checker.Evaluate(unitAwaited)) != 0 {
		t.Errorf("expected 0 violations when awaited")
	}

	// Negative: returned
	unitReturned := parser.TestUnit{
		Name:      "returned assertion",
		FilePath:  "valid.test.ts",
		StartLine: 22,
		RawBody:   "return expect(fetchUser('a')).resolves.toEqual({ id: 'a' });",
	}
	if len(checker.Evaluate(unitReturned)) != 0 {
		t.Errorf("expected 0 violations when returned")
	}
}

func TestCatchOnlyRule(t *testing.T) {
	checker := typescript.NewCatchOnlyRule()

	// Positive: all assertions inside catch, try has no fail()
	unitPos := parser.TestUnit{
		Name:      "swallowed error path",
		FilePath:  "weak.test.ts",
		StartLine: 14,
		RawBody: `
    try {
      await fetchUser('missing');
    } catch (e) {
      expect(e).toBeInstanceOf(Error);
    }
`,
	}
	vPos := checker.Evaluate(unitPos)
	if len(vPos) != 1 {
		t.Fatalf("expected 1 catch-only violation, got %d", len(vPos))
	}
	if vPos[0].RuleID != "GUARD-TS-CATCH-ONLY" {
		t.Errorf("expected GUARD-TS-CATCH-ONLY, got %s", vPos[0].RuleID)
	}

	// Negative: try has fail()
	unitFail := parser.TestUnit{
		Name:      "guarded error path",
		FilePath:  "valid.test.ts",
		StartLine: 14,
		RawBody: `
    try {
      await fetchUser('missing');
      fail('expected throw');
    } catch (e) {
      expect(e).toBeInstanceOf(Error);
    }
`,
	}
	if len(checker.Evaluate(unitFail)) != 0 {
		t.Errorf("expected 0 violations when try has fail()")
	}
}

func TestConditionalAssertRule(t *testing.T) {
	checker := typescript.NewConditionalAssertRule()

	// Positive: assertion only inside if
	unitPos := parser.TestUnit{
		Name:      "conditional assertion",
		FilePath:  "weak.test.ts",
		StartLine: 26,
		RawBody: `
    const r = parse('x');
    if (r) {
      expect(r.kind).toBe('x');
    }
`,
	}
	vPos := checker.Evaluate(unitPos)
	if len(vPos) != 1 {
		t.Fatalf("expected 1 conditional assert violation, got %d", len(vPos))
	}
	if vPos[0].RuleID != "GUARD-TS-CONDITIONAL-ASSERT" {
		t.Errorf("expected GUARD-TS-CONDITIONAL-ASSERT, got %s", vPos[0].RuleID)
	}

	// Negative: assertion outside condition
	unitNeg := parser.TestUnit{
		Name:      "unconditional assertion",
		FilePath:  "valid.test.ts",
		StartLine: 26,
		RawBody: `
    const r = parse('x');
    expect(r).toBeDefined();
    if (r) {
      expect(r.kind).toBe('x');
    }
`,
	}
	if len(checker.Evaluate(unitNeg)) != 0 {
		t.Errorf("expected 0 violations when unconditional assertion exists")
	}
}

func TestExistenceOnlyRule(t *testing.T) {
	checker := typescript.NewExistenceOnlyRule()

	// Positive: only toBeDefined()
	unitPos := parser.TestUnit{
		Name:      "existence only",
		FilePath:  "weak.test.ts",
		StartLine: 9,
		RawBody: `
    const r = add(1, 2);
    expect(r).toBeDefined();
`,
	}
	vPos := checker.Evaluate(unitPos)
	if len(vPos) != 1 {
		t.Fatalf("expected 1 existence-only violation, got %d", len(vPos))
	}
	if vPos[0].RuleID != "GUARD-TS-EXISTENCE-ONLY" {
		t.Errorf("expected GUARD-TS-EXISTENCE-ONLY, got %s", vPos[0].RuleID)
	}

	// Negative: value check
	unitNeg := parser.TestUnit{
		Name:      "value check",
		FilePath:  "valid.test.ts",
		StartLine: 9,
		RawBody: `
    const r = add(1, 2);
    expect(r).toBe(3);
`,
	}
	if len(checker.Evaluate(unitNeg)) != 0 {
		t.Errorf("expected 0 violations when value check is used")
	}
}

func TestSelfMockRule(t *testing.T) {
	checker := typescript.NewSelfMockRule()

	// Positive: asserts mock it just configured
	unitPos := parser.TestUnit{
		Name:      "asserts the mock it just configured",
		FilePath:  "weak.test.ts",
		StartLine: 33,
		RawBody: `
    const repo = { find: jest.fn().mockReturnValue(42) };
    expect(repo.find()).toBe(42);
`,
	}
	vPos := checker.Evaluate(unitPos)
	if len(vPos) != 1 {
		t.Fatalf("expected 1 self-mock violation, got %d", len(vPos))
	}
	if vPos[0].RuleID != "GUARD-TS-SELF-MOCK" {
		t.Errorf("expected GUARD-TS-SELF-MOCK, got %s", vPos[0].RuleID)
	}

	// Negative: mock passed to SUT
	unitNeg := parser.TestUnit{
		Name:      "mock passed to SUT",
		FilePath:  "valid.test.ts",
		StartLine: 33,
		RawBody: `
    const repo = { find: jest.fn().mockReturnValue(42) };
    const service = new UserService(repo);
    const user = service.getUser();
    expect(user.id).toBe(42);
`,
	}
	if len(checker.Evaluate(unitNeg)) != 0 {
		t.Errorf("expected 0 violations when SUT uses mock")
	}
}

func TestMockedSutRule(t *testing.T) {
	checker := typescript.NewMockedSutRule()

	// Positive: jest.mock('./userService') in userService.test.ts without 2nd arg
	srcPos := `
		import { UserService } from './userService';
		jest.mock('./userService');
	`
	vPos := checker.EvaluateFile("src/services/userService.test.ts", srcPos, nil)
	if len(vPos) != 1 {
		t.Fatalf("expected 1 mocked sut violation, got %d", len(vPos))
	}
	if vPos[0].RuleID != "GUARD-TS-MOCKED-SUT" {
		t.Errorf("expected GUARD-TS-MOCKED-SUT, got %s", vPos[0].RuleID)
	}

	// Negative: mock of external dependency
	srcNeg := `
		import { UserService } from './userService';
		jest.mock('./userRepository');
	`
	if len(checker.EvaluateFile("src/services/userService.test.ts", srcNeg, nil)) != 0 {
		t.Errorf("expected 0 violations when mocking dependency")
	}

	// Negative: mock with factory 2nd arg
	srcFactory := `
		import { UserService } from './userService';
		jest.mock('./userService', () => ({ UserService: jest.fn() }));
	`
	if len(checker.EvaluateFile("src/services/userService.test.ts", srcFactory, nil)) != 0 {
		t.Errorf("expected 0 violations when 2nd arg factory is provided")
	}
}

func TestWeakTestProbeIntegration(t *testing.T) {
	content, err := os.ReadFile("testdata/weak.test.ts")
	if err != nil {
		t.Fatalf("read weak.test.ts: %v", err)
	}

	adapter := ts.NewAdapter()
	units, err := adapter.LocateTests(content, "testdata/weak.test.ts")
	if err != nil {
		t.Fatalf("LocateTests: %v", err)
	}

	if len(units) != 10 {
		t.Fatalf("expected 10 units, got %d", len(units))
	}

	rules := []struct {
		eval func(parser.TestUnit) []models.Violation
	}{
		{eval: typescript.NewSnapshotOnlyRule().Evaluate},
		{eval: typescript.NewFloatingAssertRule().Evaluate},
		{eval: typescript.NewCatchOnlyRule().Evaluate},
		{eval: typescript.NewConditionalAssertRule().Evaluate},
		{eval: typescript.NewExistenceOnlyRule().Evaluate},
		{eval: typescript.NewSelfMockRule().Evaluate},
	}

	unitFindings := make(map[string][]string)
	for _, unit := range units {
		for _, r := range rules {
			for _, v := range r.eval(unit) {
				unitFindings[unit.Name] = append(unitFindings[unit.Name], v.RuleID)
			}
		}
	}

	expectedFindings := map[string][]string{
		"bare expect without matcher":         nil,
		"existence only":                      {"GUARD-TS-EXISTENCE-ONLY"},
		"swallowed error path":                {"GUARD-TS-CATCH-ONLY"},
		"floating async assertion":            {"GUARD-TS-FLOATING-ASSERT"},
		"conditional assertion":               {"GUARD-TS-CONDITIONAL-ASSERT"},
		"asserts the mock it just configured": {"GUARD-TS-SELF-MOCK"},
		"focused test left behind":            nil,
		"suppressed empty test":               nil,
		"parameterised %i":                    nil,
		"template literal name":               nil,
	}

	for name, expected := range expectedFindings {
		actual := unitFindings[name]
		if len(actual) != len(expected) {
			t.Errorf("unit %q expected findings %v, got %v", name, expected, actual)
			continue
		}
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("unit %q expected finding %s, got %s", name, expected[i], actual[i])
			}
		}
	}
}

func TestOrdersControllerSpecProbe(t *testing.T) {
	content, err := os.ReadFile("testdata/orders.controller.spec.ts")
	if err != nil {
		t.Fatalf("read orders.controller.spec.ts: %v", err)
	}

	adapter := ts.NewAdapter()
	units, err := adapter.LocateTests(content, "testdata/orders.controller.spec.ts")
	if err != nil {
		t.Fatalf("LocateTests: %v", err)
	}

	// File-level GUARD-NEST-02
	dbChecker := typescript.NewNestUnmockedDbModuleRule()
	fileViolations := dbChecker.EvaluateFile("testdata/orders.controller.spec.ts", string(content), units)
	if len(fileViolations) != 1 {
		t.Fatalf("expected 1 GUARD-NEST-02 finding, got %d", len(fileViolations))
	}
	if fileViolations[0].RuleID != "GUARD-NEST-02" || fileViolations[0].LineNumber != 12 {
		t.Errorf("expected GUARD-NEST-02 at line 12, got %s at line %d", fileViolations[0].RuleID, fileViolations[0].LineNumber)
	}

	// Unit-level GUARD-NEST-01
	echoChecker := typescript.NewNestControllerEchoRule()
	var echoViolations []models.Violation
	for _, u := range units {
		echoViolations = append(echoViolations, echoChecker.Evaluate(u)...)
	}
	if len(echoViolations) != 2 {
		t.Fatalf("expected 2 GUARD-NEST-01 findings, got %d", len(echoViolations))
	}
	if echoViolations[0].RuleID != "GUARD-NEST-01" || echoViolations[0].LineNumber != 19 {
		t.Errorf("expected GUARD-NEST-01 at line 19, got %s at line %d", echoViolations[0].RuleID, echoViolations[0].LineNumber)
	}
	if echoViolations[1].RuleID != "GUARD-NEST-01" || echoViolations[1].LineNumber != 26 {
		t.Errorf("expected GUARD-NEST-01 at line 26, got %s at line %d", echoViolations[1].RuleID, echoViolations[1].LineNumber)
	}
}

func TestPrecisionProbeNoFindings(t *testing.T) {
	content, err := os.ReadFile("testdata/precision.test.tsx")
	if err != nil {
		t.Fatalf("read precision.test.tsx: %v", err)
	}

	adapter := ts.NewAdapter()
	units, err := adapter.LocateTests(content, "testdata/precision.test.tsx")
	if err != nil {
		t.Fatalf("LocateTests: %v", err)
	}

	if len(units) != 19 {
		t.Fatalf("expected 19 units, got %d", len(units))
	}

	// Verify JSX unit assertion count
	foundJsxUnit := false
	for _, u := range units {
		if u.Name == "jsx text with an apostrophe before the next test" {
			foundJsxUnit = true
			if len(u.Assertions) != 1 {
				t.Errorf("expected 1 assertion for jsx text unit, got %d: %v", len(u.Assertions), u.Assertions)
			}
		}
	}
	if !foundJsxUnit {
		t.Errorf("did not find 'jsx text with an apostrophe before the next test' unit")
	}

	// All unit-level rules
	unitRules := []struct {
		name string
		eval func(parser.TestUnit) []models.Violation
	}{
		{"GUARD-NEST-01", typescript.NewNestControllerEchoRule().Evaluate},
		{"GUARD-TS-01", typescript.NewSnapshotOnlyRule().Evaluate},
		{"GUARD-TS-FLOATING-ASSERT", typescript.NewFloatingAssertRule().Evaluate},
		{"GUARD-TS-CATCH-ONLY", typescript.NewCatchOnlyRule().Evaluate},
		{"GUARD-TS-CONDITIONAL-ASSERT", typescript.NewConditionalAssertRule().Evaluate},
		{"GUARD-TS-EXISTENCE-ONLY", typescript.NewExistenceOnlyRule().Evaluate},
		{"GUARD-TS-SELF-MOCK", typescript.NewSelfMockRule().Evaluate},
	}

	var allViolations []models.Violation
	for _, u := range units {
		for _, r := range unitRules {
			v := r.eval(u)
			if len(v) > 0 {
				t.Errorf("unexpected violation from %s on correct unit %q: %+v", r.name, u.Name, v)
				allViolations = append(allViolations, v...)
			}
		}
	}

	// File-level rules
	dbRule := typescript.NewNestUnmockedDbModuleRule()
	allViolations = append(allViolations, dbRule.EvaluateFile("testdata/precision.test.tsx", string(content), units)...)

	mockedSutRule := typescript.NewMockedSutRule()
	allViolations = append(allViolations, mockedSutRule.EvaluateFile("testdata/precision.test.tsx", string(content), units)...)

	sleepRule := typescript.NewReactSleepRule()
	allViolations = append(allViolations, sleepRule.EvaluateFile(units, string(content))...)

	if len(allViolations) != 0 {
		t.Fatalf("expected 0 findings on precision probe, got %d: %+v", len(allViolations), allViolations)
	}
}

func TestJsxApostropheRuleIntegration(t *testing.T) {
	content, err := os.ReadFile("testdata/jsx_apostrophe.test.tsx")
	if err != nil {
		t.Fatalf("read jsx_apostrophe.test.tsx: %v", err)
	}

	adapter := ts.NewAdapter()
	units, err := adapter.LocateTests(content, "testdata/jsx_apostrophe.test.tsx")
	if err != nil {
		t.Fatalf("LocateTests: %v", err)
	}

	if len(units) != 3 {
		t.Fatalf("expected 3 units, got %d", len(units))
	}

	if units[0].Name != "renders the signup prompt without asserting" || len(units[0].Assertions) != 0 {
		t.Errorf("unit[0] expected 0 assertions, got %d: %v", len(units[0].Assertions), units[0].Assertions)
	}
	if units[1].Name != "asserts the heading" || len(units[1].Assertions) != 1 {
		t.Errorf("unit[1] expected 1 assertion, got %d: %v", len(units[1].Assertions), units[1].Assertions)
	}
	if units[2].Name != "asserts the button" || len(units[2].Assertions) != 1 {
		t.Errorf("unit[2] expected 1 assertion, got %d: %v", len(units[2].Assertions), units[2].Assertions)
	}
}

func TestReactArbitrarySleepRule(t *testing.T) {
	unit := parser.TestUnit{
		Name:      "renders user details",
		FilePath:  "components/UserProfile.test.tsx",
		StartLine: 35,
		Kind:      "unit",
		RawBody:   "render(<UserProfile />);\nawait new Promise((r) => setTimeout(r, 1000));\nexpect(screen.getByText('User')).toBeInTheDocument();",
	}

	checker := typescript.NewReactSleepRule()
	violations := checker.EvaluateFile([]parser.TestUnit{unit}, unit.RawBody)

	if len(violations) != 1 {
		t.Fatalf("expected 1 arbitrary sleep violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-REACT-02" {
		t.Errorf("expected GUARD-REACT-02, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", violations[0].Severity)
	}
	if !containsWord(violations[0].Message, "setTimeout called with delay '1000'") {
		t.Errorf("expected message to mention delay 1000, got %s", violations[0].Message)
	}
}

func TestReactSleepRule_EvaluateFile(t *testing.T) {
	tests := []struct {
		name               string
		fileSrc            string
		units              []parser.TestUnit
		expectedViolations int
	}{
		{
			name: "single unit with positive setTimeout delay",
			fileSrc: `it('sleeps', async () => {
				await new Promise(r => setTimeout(r, 1000));
				expect(true).toBe(true);
			});`,
			units: []parser.TestUnit{
				{
					Name:      "sleeps",
					FilePath:  "sleep.test.ts",
					StartLine: 1,
					Kind:      "unit",
					RawBody: `it('sleeps', async () => {
				await new Promise(r => setTimeout(r, 1000));
				expect(true).toBe(true);
			});`,
				},
			},
			expectedViolations: 1,
		},
		{
			name: "setTimeout with 0 delay is ignored (event loop tick)",
			fileSrc: `it('ticks', async () => {
				await new Promise(r => setTimeout(r, 0));
				expect(true).toBe(true);
			});`,
			units: []parser.TestUnit{
				{
					Name:      "ticks",
					FilePath:  "tick.test.ts",
					StartLine: 1,
					Kind:      "unit",
					RawBody: `it('ticks', async () => {
				await new Promise(r => setTimeout(r, 0));
				expect(true).toBe(true);
			});`,
				},
			},
			expectedViolations: 0,
		},
		{
			name: "file using jest.useFakeTimers() is ignored",
			fileSrc: `jest.useFakeTimers();
			it('fakes', async () => {
				setTimeout(() => {}, 5000);
				expect(true).toBe(true);
			});`,
			units: []parser.TestUnit{
				{
					Name:      "fakes",
					FilePath:  "fake.test.ts",
					StartLine: 2,
					Kind:      "unit",
					RawBody: `it('fakes', async () => {
				setTimeout(() => {}, 5000);
				expect(true).toBe(true);
			});`,
				},
			},
			expectedViolations: 0,
		},
		{
			name: "setTimeout inside mock implementation is ignored",
			fileSrc: `it('mocks delay', () => {
				const mock = jest.fn().mockImplementation(() => {
					setTimeout(() => {}, 500);
				});
				expect(true).toBe(true);
			});`,
			units: []parser.TestUnit{
				{
					Name:      "mocks delay",
					FilePath:  "mock.test.ts",
					StartLine: 1,
					Kind:      "unit",
					RawBody: `it('mocks delay', () => {
				const mock = jest.fn().mockImplementation(() => {
					setTimeout(() => {}, 500);
				});
				expect(true).toBe(true);
			});`,
				},
			},
			expectedViolations: 0,
		},
		{
			name: "setTimeout inside class definition is ignored",
			fileSrc: `it('class timer', () => {
				class FakeService {
					delay() {
						setTimeout(() => {}, 300);
					}
				}
				expect(true).toBe(true);
			});`,
			units: []parser.TestUnit{
				{
					Name:      "class timer",
					FilePath:  "class.test.ts",
					StartLine: 1,
					Kind:      "unit",
					RawBody: `it('class timer', () => {
				class FakeService {
					delay() {
						setTimeout(() => {}, 300);
					}
				}
				expect(true).toBe(true);
			});`,
				},
			},
			expectedViolations: 0,
		},
	}

	checker := typescript.NewReactSleepRule()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations := checker.EvaluateFile(tc.units, tc.fileSrc)
			if len(violations) != tc.expectedViolations {
				t.Fatalf("expected %d violations, got %d", tc.expectedViolations, len(violations))
			}
		})
	}
}

func containsWord(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestConditionalAssertBranchTotality pins branch-exhaustiveness handling: a
// conditional only hides an assertion when some reachable path asserts nothing.
func TestConditionalAssertBranchTotality(t *testing.T) {
	checker := typescript.NewConditionalAssertRule()

	tests := []struct {
		name     string
		body     string
		expected int
		line     int
	}{
		{
			name: "exhaustive if else both assert",
			body: `
    const result = parse(input);
    if (result.ok) {
      expect(result.value).toEqual(42);
    } else {
      expect(result.error.code).toBe('INVALID');
    }
`,
			expected: 0,
		},
		{
			name: "if else if else chain all assert",
			body: `
    const n = classify(input);
    if (n === 1) {
      expect(one(n)).toBe('one');
    } else if (n === 2) {
      expect(two(n)).toBe('two');
    } else {
      expect(other(n)).toBe('other');
    }
`,
			expected: 0,
		},
		{
			name: "switch with default every clause asserts",
			body: `
    switch (kind) {
      case 'a':
        expect(handle('a')).toBe(1);
        break;
      case 'b':
        expect(handle('b')).toBe(2);
        break;
      default:
        expect(handle('other')).toBe(0);
    }
`,
			expected: 0,
		},
		{
			name: "ternary with both branches asserting",
			body: `
    const label = ok ? expect(make('a')).toBe('a') : expect(make('b')).toBe('b');
`,
			expected: 0,
		},
		{
			name: "if else where else asserts nothing",
			body: `
    const r = parse('x');
    if (r) {
      expect(r.kind).toBe('x');
    } else {
      recordMissing('x');
    }
`,
			expected: 1,
			line:     7,
		},
		{
			name: "switch without default",
			body: `
    switch (kind) {
      case 'a':
        expect(handle('a')).toBe(1);
        break;
      case 'b':
        expect(handle('b')).toBe(2);
    }
`,
			expected: 1,
			line:     6,
		},
		{
			name: "ternary asserting one branch only",
			body: `
    const r = parse('x');
    r ? expect(r.kind).toBe('x') : undefined;
`,
			expected: 1,
			line:     7,
		},
		{
			name: "short circuit guarded assertion",
			body: `
    const r = parse('x');
    r && expect(r.kind).toBe('x');
`,
			expected: 1,
			line:     7,
		},
		{
			name: "single if without else",
			body: `
    const r = parse('x');
    if (r) {
      expect(r.kind).toBe('x');
    }
`,
			expected: 1,
			line:     7,
		},
		{
			name: "if else where else throws",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      throw new Error('expected ok');
    }
`,
			expected: 0,
		},
		{
			name: "if else where else calls fail",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      fail('expected ok');
    }
`,
			expected: 0,
		},
		{
			name: "if else where else calls expect.fail",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      expect.fail('expected ok');
    }
`,
			expected: 0,
		},
		{
			name: "if else where else calls assert.fail",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      assert.fail('expected ok');
    }
`,
			expected: 0,
		},
		{
			name: "if else where else calls expect.unreachable",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      expect.unreachable('expected ok');
    }
`,
			expected: 0,
		},
		{
			name: "switch default throws",
			body: `
    switch (kind) {
      case 'a':
        expect(handle('a')).toBe(1);
        break;
      case 'b':
        expect(handle('b')).toBe(2);
        break;
      default:
        throw new Error('unexpected kind');
    }
`,
			expected: 0,
		},
		{
			name: "ternary arms assert or fail",
			body: `
    const r = parse('x');
    r.ok ? expect(r.value).toBe('x') : fail('expected ok');
`,
			expected: 0,
		},
		{
			name: "nested conditional throw stays partial",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      if (r.retry) {
        throw new Error('retry');
      }
    }
`,
			expected: 1,
			line:     7,
		},
		{
			name: "nested exhaustive failure branches are total",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      if (r.kind === 'a') {
        throw new Error('kind a');
      } else {
        throw new Error('kind b');
      }
    }
`,
			expected: 0,
		},
		{
			name: "nested if with one failing path stays partial",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      if (r.kind === 'a') {
        throw new Error('kind a');
      } else {
        recordMissing(r);
      }
    }
`,
			expected: 1,
			line:     7,
		},
		{
			name: "throw inside try block stays partial",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      try {
        throw new Error('boom');
      } catch (e) {}
    }
`,
			expected: 1,
			line:     7,
		},
		{
			name: "fail inside call argument stays partial",
			body: `
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      [1].some((n) => fail(n));
    }
`,
			expected: 1,
			line:     7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      tc.name,
				FilePath:  "conditional.test.ts",
				StartLine: 5,
				RawBody:   tc.body,
			}
			violations := checker.Evaluate(unit)
			if len(violations) != tc.expected {
				t.Fatalf("expected %d violations, got %d: %+v", tc.expected, len(violations), violations)
			}
			for _, v := range violations {
				if v.RuleID != "GUARD-TS-CONDITIONAL-ASSERT" || v.Severity != models.SeverityWarning {
					t.Errorf("unexpected violation identity: %s %s", v.RuleID, v.Severity)
				}
				if v.LineNumber != tc.line {
					t.Errorf("expected line %d, got %d", tc.line, v.LineNumber)
				}
			}
		})
	}
}

// TestSelfMockRuleTokenAccuracy pins that comment text, string literals and
// shadowed parameters never count as a direct invocation of a mock.
func TestSelfMockRuleTokenAccuracy(t *testing.T) {
	checker := typescript.NewSelfMockRule()

	tests := []struct {
		name     string
		body     string
		expected int
	}{
		{
			name: "commented invocation is not a call",
			body: `
    const callback = vi.fn();
    // callback();
    emitter.subscribe(callback);
    emitter.emit('ready');
    expect(callback).toHaveBeenCalledWith('ready');
`,
			expected: 0,
		},
		{
			name: "quoted invocation is not a call",
			body: `
    const callback = vi.fn();
    const documentation = 'callback()';
    emitter.subscribe(callback);
    emitter.emit(documentation);
    expect(callback).toHaveBeenCalledWith(documentation);
`,
			expected: 0,
		},
		{
			name: "shadowed parameter is not the mock",
			body: `
    const callback = vi.fn();
    function invokeTwice(callback) {
      callback();
    }
    emitter.subscribe(callback);
    expect(callback).toHaveBeenCalledTimes(1);
`,
			expected: 0,
		},
		{
			name: "mock invoked only by the system under test",
			body: `
    const callback = vi.fn();
    const emitter = new Emitter(callback);
    emitter.emit('ready');
    expect(callback).toHaveBeenCalledWith('ready');
`,
			expected: 0,
		},
		{
			name: "direct invocation then call verification",
			body: `
    const save = jest.fn();
    save('x');
    expect(save).toHaveBeenCalledWith('x');
`,
			expected: 1,
		},
		{
			name: "assigned result of a direct mock call",
			body: `
    const compute = vi.fn(() => 42);
    const result = compute();
    expect(result).toBe(42);
`,
			expected: 1,
		},
		{
			// The adapter's RawBody includes the test callback wrapper.
			name: "direct invocation inside the callback wrapper",
			body: `() => {
    const save = jest.fn();
    save('x');
    expect(save).toHaveBeenCalledWith('x');
  }`,
			expected: 1,
		},
		{
			name: "object property mock called in the assertion",
			body: `
    const repo = { find: jest.fn().mockReturnValue(42) };
    expect(repo.find()).toBe(42);
`,
			expected: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      tc.name,
				FilePath:  "selfmock.test.ts",
				StartLine: 4,
				RawBody:   tc.body,
			}
			violations := checker.Evaluate(unit)
			if len(violations) != tc.expected {
				t.Fatalf("expected %d violations, got %d: %+v", tc.expected, len(violations), violations)
			}
			for _, v := range violations {
				if v.RuleID != "GUARD-TS-SELF-MOCK" || v.Severity != models.SeverityWarning {
					t.Errorf("unexpected violation identity: %s %s", v.RuleID, v.Severity)
				}
				if v.LineNumber != 4 {
					t.Errorf("expected line 4, got %d", v.LineNumber)
				}
			}
		})
	}
}

// TestRawBodyLineAttribution pins that statement-level diagnostics count lines
// from the first byte of RawBody. A declaration whose title or callback wrapper
// spans several lines shifts RawBody's origin below the declaration line, so
// using StartLine would misattribute every finding above its real line and
// break line-scoped 'testguard:ignore' comments; whole-test findings stay on
// the declaration line.
func TestRawBodyLineAttribution(t *testing.T) {
	// The declaration occupies lines 10-13 (multi-line title and callback
	// wrapper), so the first RawBody byte sits on line 14.
	meta := map[string]string{"body_start_line": "14"}

	conditional := parser.TestUnit{
		Name:      "multi-line titled conditional",
		FilePath:  "attribution.test.ts",
		StartLine: 10,
		Meta:      meta,
		RawBody: `
    const r = parse('x');
    if (r) {
      expect(r.kind).toBe('x');
    }
`,
	}
	violations := typescript.NewConditionalAssertRule().Evaluate(conditional)
	if len(violations) != 1 {
		t.Fatalf("expected 1 conditional assertion violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].RuleID != "GUARD-TS-CONDITIONAL-ASSERT" || violations[0].LineNumber != 16 {
		t.Fatalf("expected GUARD-TS-CONDITIONAL-ASSERT at line 16, got %s at %d", violations[0].RuleID, violations[0].LineNumber)
	}

	floating := parser.TestUnit{
		Name:      "multi-line titled floating assertion",
		FilePath:  "attribution.test.ts",
		StartLine: 10,
		Meta:      meta,
		RawBody: `
    expect(load('x')).resolves.toBe(1);
`,
	}
	violations = typescript.NewFloatingAssertRule().Evaluate(floating)
	if len(violations) != 1 {
		t.Fatalf("expected 1 floating assertion violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].RuleID != "GUARD-TS-FLOATING-ASSERT" || violations[0].LineNumber != 15 {
		t.Fatalf("expected GUARD-TS-FLOATING-ASSERT at line 15, got %s at %d", violations[0].RuleID, violations[0].LineNumber)
	}

	catchOnly := parser.TestUnit{
		Name:      "multi-line titled catch-only assertions",
		FilePath:  "attribution.test.ts",
		StartLine: 10,
		Meta:      meta,
		RawBody: `
    try {
      await load('x');
    } catch (e) {
      expect(e).toBeDefined();
    }
`,
	}
	violations = typescript.NewCatchOnlyRule().Evaluate(catchOnly)
	if len(violations) != 1 {
		t.Fatalf("expected 1 catch-only violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].RuleID != "GUARD-TS-CATCH-ONLY" || violations[0].LineNumber != 15 {
		t.Fatalf("expected GUARD-TS-CATCH-ONLY at line 15, got %s at %d", violations[0].RuleID, violations[0].LineNumber)
	}

	sleep := parser.TestUnit{
		Name:      "multi-line titled fixed sleep",
		FilePath:  "attribution.test.tsx",
		StartLine: 10,
		Kind:      "unit",
		Meta:      meta,
		RawBody: `
    await new Promise((r) => setTimeout(r, 1000));
`,
	}
	violations = typescript.NewReactSleepRule().EvaluateFile([]parser.TestUnit{sleep}, sleep.RawBody)
	if len(violations) != 1 {
		t.Fatalf("expected 1 arbitrary sleep violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].RuleID != "GUARD-REACT-02" || violations[0].LineNumber != 15 {
		t.Fatalf("expected GUARD-REACT-02 at line 15, got %s at %d", violations[0].RuleID, violations[0].LineNumber)
	}

	// Without the adapter fact the declaration line stays the origin, which is
	// the pre-existing behaviour every earlier fixture asserts.
	fallback := conditional
	fallback.Meta = nil
	violations = typescript.NewConditionalAssertRule().Evaluate(fallback)
	if len(violations) != 1 || violations[0].LineNumber != 12 {
		t.Fatalf("expected fallback origin to report line 12, got %+v", violations)
	}
}

// TestTrustBoundaryProbeNoFindings guards the parent's adversarial file: each
// test is correct, so every rule must stay silent.
func TestTrustBoundaryProbeNoFindings(t *testing.T) {
	content, err := os.ReadFile("testdata/trust_boundary.test.ts")
	if err != nil {
		t.Fatalf("read trust_boundary.test.ts: %v", err)
	}

	adapter := ts.NewAdapter()
	units, err := adapter.LocateTests(content, "testdata/trust_boundary.test.ts")
	if err != nil {
		t.Fatalf("LocateTests: %v", err)
	}
	if len(units) != 3 {
		t.Fatalf("expected 3 units, got %d", len(units))
	}

	rules := []struct {
		name string
		eval func(parser.TestUnit) []models.Violation
	}{
		{"GUARD-NEST-01", typescript.NewNestControllerEchoRule().Evaluate},
		{"GUARD-TS-01", typescript.NewSnapshotOnlyRule().Evaluate},
		{"GUARD-TS-FLOATING-ASSERT", typescript.NewFloatingAssertRule().Evaluate},
		{"GUARD-TS-CATCH-ONLY", typescript.NewCatchOnlyRule().Evaluate},
		{"GUARD-TS-CONDITIONAL-ASSERT", typescript.NewConditionalAssertRule().Evaluate},
		{"GUARD-TS-EXISTENCE-ONLY", typescript.NewExistenceOnlyRule().Evaluate},
		{"GUARD-TS-SELF-MOCK", typescript.NewSelfMockRule().Evaluate},
	}

	for _, u := range units {
		for _, r := range rules {
			if v := r.eval(u); len(v) > 0 {
				t.Errorf("unexpected %s on correct unit %q: %+v", r.name, u.Name, v)
			}
		}
	}
}
