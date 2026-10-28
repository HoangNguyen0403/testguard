package e2e_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/e2e"
)

func TestHardWaitRule_EvaluateFile(t *testing.T) {
	tests := []struct {
		name               string
		filePath           string
		fileSrc            string
		units              []parser.TestUnit
		expectedViolations int
		expectedLine       int
		expectedCall       string
	}{
		{
			name:     "Playwright waitForTimeout flags Warning",
			filePath: "e2e/login.spec.ts",
			fileSrc:  "test('user login', async ({ page }) => { await page.waitForTimeout(3000); });",
			units: []parser.TestUnit{
				{
					Name:      "user login",
					FilePath:  "e2e/login.spec.ts",
					StartLine: 10,
					RawBody:   "await page.goto('/login');\nawait page.waitForTimeout(3000);\nawait expect(page.locator('#home')).toBeVisible();",
				},
			},
			expectedViolations: 1,
			expectedLine:       11,
			expectedCall:       "waitForTimeout",
		},
		{
			name:     "Playwright waitForCondition only produces no violation",
			filePath: "e2e/login.spec.ts",
			fileSrc:  "test('user login', async ({ page }) => { await expect(page.getByText('x')).toBeVisible(); });",
			units: []parser.TestUnit{
				{
					Name:      "user login",
					FilePath:  "e2e/login.spec.ts",
					StartLine: 20,
					RawBody:   "await page.goto('/login');\nawait expect(page.getByText('x')).toBeVisible();",
				},
			},
			expectedViolations: 0,
		},
		{
			name:     "TS E2E real-timer setTimeout flags Warning",
			filePath: "e2e/checkout.spec.ts",
			fileSrc:  "test('checkout', async () => { await new Promise(r => setTimeout(r, 1000)); });",
			units: []parser.TestUnit{
				{
					Name:      "checkout",
					FilePath:  "e2e/checkout.spec.ts",
					StartLine: 30,
					RawBody:   "await new Promise(r => setTimeout(r, 1000));",
				},
			},
			expectedViolations: 1,
			expectedLine:       30,
			expectedCall:       "setTimeout",
		},
		{
			name:     "TS E2E setTimeout with delay 0 is ignored",
			filePath: "e2e/flush.spec.ts",
			fileSrc:  "test('flush', async () => { setTimeout(r, 0); });",
			units: []parser.TestUnit{
				{
					Name:      "flush",
					FilePath:  "e2e/flush.spec.ts",
					StartLine: 40,
					RawBody:   "setTimeout(r, 0);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:     "TS E2E setTimeout with delay false is ignored",
			filePath: "e2e/false_delay.spec.ts",
			fileSrc:  "test('false delay', async () => { setTimeout(r, false); });",
			units: []parser.TestUnit{
				{
					Name:      "false delay",
					FilePath:  "e2e/false_delay.spec.ts",
					StartLine: 45,
					RawBody:   "setTimeout(r, false);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:     "TS E2E setTimeout with delay 0 and comments is ignored",
			filePath: "e2e/commented_zero.spec.ts",
			fileSrc:  "test('commented zero', async () => { setTimeout(r, 0 /* flush */); });",
			units: []parser.TestUnit{
				{
					Name:      "commented zero",
					FilePath:  "e2e/commented_zero.spec.ts",
					StartLine: 48,
					RawBody:   "setTimeout(r, 0 /* flush */);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:     "Dart Future.delayed in test body flags Warning",
			filePath: "integration_test/app_test.dart",
			fileSrc:  "testWidgets('app launches', (tester) async { await Future.delayed(const Duration(seconds: 2)); });",
			units: []parser.TestUnit{
				{
					Name:      "app launches",
					FilePath:  "integration_test/app_test.dart",
					StartLine: 50,
					RawBody:   "await tester.pumpWidget(const MyApp());\nawait Future.delayed(const Duration(seconds: 2));\nexpect(find.text('Ready'), findsOneWidget);",
				},
			},
			expectedViolations: 1,
			expectedLine:       51,
			expectedCall:       "Future.delayed",
		},
		{
			name:     "Dart sleep in test body flags Warning",
			filePath: "integration_test/sync_test.dart",
			fileSrc:  "testWidgets('syncs', (tester) async { sleep(const Duration(seconds: 1)); });",
			units: []parser.TestUnit{
				{
					Name:      "syncs",
					FilePath:  "integration_test/sync_test.dart",
					StartLine: 60,
					RawBody:   "sleep(const Duration(seconds: 1));",
				},
			},
			expectedViolations: 1,
			expectedLine:       60,
			expectedCall:       "sleep",
		},
		{
			name:     "Dart tester.pumpAndSettle is not flagged",
			filePath: "integration_test/app_test.dart",
			fileSrc:  "testWidgets('pumps', (tester) async { await tester.pumpAndSettle(); });",
			units: []parser.TestUnit{
				{
					Name:      "pumps",
					FilePath:  "integration_test/app_test.dart",
					StartLine: 70,
					RawBody:   "await tester.pumpWidget(const MyApp());\nawait tester.pumpAndSettle();\nexpect(find.byType(MyApp), findsOneWidget);",
				},
			},
			expectedViolations: 0,
		},
		{
			name:     "Dart tester.pump with duration is not flagged",
			filePath: "integration_test/app_test.dart",
			fileSrc:  "testWidgets('pump duration', (tester) async { await tester.pump(const Duration(milliseconds: 100)); });",
			units: []parser.TestUnit{
				{
					Name:      "pump duration",
					FilePath:  "integration_test/app_test.dart",
					StartLine: 80,
					RawBody:   "await tester.pump(const Duration(milliseconds: 100));",
				},
			},
			expectedViolations: 0,
		},
		{
			name:     "Dart Future.delayed inside when/thenAnswer is ignored",
			filePath: "integration_test/mock_test.dart",
			fileSrc:  "testWidgets('mocks delay', (tester) async { when(() => repo.get()).thenAnswer((_) => Future.delayed(d, () => x)); });",
			units: []parser.TestUnit{
				{
					Name:      "mocks delay",
					FilePath:  "integration_test/mock_test.dart",
					StartLine: 90,
					RawBody:   "when(() => repo.get()).thenAnswer((_) => Future.delayed(d, () => x));\nawait tester.pumpWidget(const MyApp());",
				},
			},
			expectedViolations: 0,
		},
	}

	checker := e2e.NewHardWaitRule()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations := checker.EvaluateFile(tc.units, tc.filePath, tc.fileSrc)
			if len(violations) != tc.expectedViolations {
				t.Fatalf("expected %d violations, got %d", tc.expectedViolations, len(violations))
			}
			if tc.expectedViolations > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-E2E-01" {
					t.Errorf("expected GUARD-E2E-01, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityWarning {
					t.Errorf("expected Warning severity, got %s", v.Severity)
				}
				if v.LineNumber != tc.expectedLine {
					t.Errorf("expected line %d, got %d", tc.expectedLine, v.LineNumber)
				}
				expectedMsg := "Hard wait in E2E test '" + tc.units[0].Name + "': " + tc.expectedCall + " sleeps for a fixed time; wait for a UI condition instead (Playwright: await expect(locator).toBeVisible(); Flutter: pumpUntilFound / pumpAndSettle with a finder)."
				if v.Message != expectedMsg {
					t.Errorf("unexpected message:\n  got:  %q\n  want: %q", v.Message, expectedMsg)
				}
			}
		})
	}
}
