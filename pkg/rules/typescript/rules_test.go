package typescript_test

import (
	"strings"
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

func TestReactSleepRule_EvaluateFile(t *testing.T) {
	tests := []struct {
		name               string
		fileSrc            string
		units              []parser.TestUnit
		expectedViolations int
		expectedLine       int
		expectedSeverity   models.Severity
		expectedDelay      string
	}{
		{
			name:    "setTimeout with delay 0 is ignored",
			fileSrc: "describe('test', () => { it('flush', () => { setTimeout(r, 0); }); });",
			units: []parser.TestUnit{
				{
					Name:      "flush",
					FilePath:  "components/UserProfile.test.tsx",
					StartLine: 10,
					Kind:      "unit",
					RawBody:   "render(<UserProfile />);\nsetTimeout(r, 0);\nexpect(true).toBe(true);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:    "setTimeout without delay argument is ignored",
			fileSrc: "describe('test', () => { it('flush', () => { setTimeout(r); }); });",
			units: []parser.TestUnit{
				{
					Name:      "flush",
					FilePath:  "components/UserProfile.test.tsx",
					StartLine: 10,
					Kind:      "unit",
					RawBody:   "render(<UserProfile />);\nsetTimeout(r);\nexpect(true).toBe(true);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:    "mockImplementation latency is ignored",
			fileSrc: "describe('test', () => { it('fetches', () => { fetch.mockImplementation(() => new Promise(r => setTimeout(() => r(data), 100))); }); });",
			units: []parser.TestUnit{
				{
					Name:      "fetches",
					FilePath:  "components/UserProfile.test.tsx",
					StartLine: 20,
					Kind:      "unit",
					RawBody:   "fetchMock.mockImplementation(() => new Promise(r => setTimeout(() => r(data), 100)));\nexpect(true).toBe(true);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:    "fake timers in file suppresses setTimeout warnings",
			fileSrc: "beforeEach(() => { jest.useFakeTimers(); });\ndescribe('test', () => { it('waits', async () => { await new Promise(r => setTimeout(r, 500)); }); });",
			units: []parser.TestUnit{
				{
					Name:      "waits",
					FilePath:  "components/UserProfile.test.tsx",
					StartLine: 30,
					Kind:      "unit",
					RawBody:   "await new Promise(r => setTimeout(r, 500));\nexpect(true).toBe(true);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:    "setTimeout inside class body is ignored",
			fileSrc: "describe('test', () => { it('class mock', () => { class FakeTimer { schedule() { setTimeout(this.cb, 1000); } } }); });",
			units: []parser.TestUnit{
				{
					Name:      "class mock",
					FilePath:  "components/UserProfile.test.tsx",
					StartLine: 40,
					Kind:      "unit",
					RawBody:   "class FakeTimer {\n  schedule() {\n    setTimeout(this.cb, 1000);\n  }\n}\nexpect(true).toBe(true);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:    "real timer sleep with 500ms delay flags Warning at correct line",
			fileSrc: "describe('test', () => { it('renders', async () => { render(<UserProfile />); await new Promise(r => setTimeout(r, 500)); }); });",
			units: []parser.TestUnit{
				{
					Name:      "renders",
					FilePath:  "components/UserProfile.test.tsx",
					StartLine: 50,
					Kind:      "unit",
					RawBody:   "render(<UserProfile />);\nawait new Promise(r => setTimeout(r, 500));\nexpect(screen.getByText('User')).toBeInTheDocument();",
				},
			},
			expectedViolations: 1,
			expectedLine:       51,
			expectedSeverity:   models.SeverityWarning,
			expectedDelay:      "500",
		},
		{
			name:    "setTimeout with constant DEBOUNCE_MS flags Warning",
			fileSrc: "describe('test', () => { it('debounces', () => { setTimeout(r, DEBOUNCE_MS); }); });",
			units: []parser.TestUnit{
				{
					Name:      "debounces",
					FilePath:  "components/UserProfile.test.tsx",
					StartLine: 60,
					Kind:      "unit",
					RawBody:   "render(<UserProfile />);\nsetTimeout(r, DEBOUNCE_MS);\nexpect(true).toBe(true);",
				},
			},
			expectedViolations: 1,
			expectedLine:       61,
			expectedSeverity:   models.SeverityWarning,
			expectedDelay:      "DEBOUNCE_MS",
		},
	}

	checker := typescript.NewReactSleepRule()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations := checker.EvaluateFile(tc.units, tc.fileSrc)
			if len(violations) != tc.expectedViolations {
				t.Fatalf("expected %d violations, got %d", tc.expectedViolations, len(violations))
			}
			if tc.expectedViolations > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-REACT-02" {
					t.Errorf("expected GUARD-REACT-02, got %s", v.RuleID)
				}
				if v.Severity != tc.expectedSeverity {
					t.Errorf("expected severity %s, got %s", tc.expectedSeverity, v.Severity)
				}
				if v.LineNumber != tc.expectedLine {
					t.Errorf("expected line %d, got %d", tc.expectedLine, v.LineNumber)
				}
				if tc.expectedDelay != "" && !containsWord(v.Message, tc.expectedDelay) {
					t.Errorf("expected message to name delay %q, got: %s", tc.expectedDelay, v.Message)
				}
			}
		})
	}
}

func containsWord(s, substr string) bool {
	return strings.Contains(s, substr)
}
