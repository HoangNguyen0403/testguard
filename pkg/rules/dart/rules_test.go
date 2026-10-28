package dart_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/dart"
)

func TestDetectSeedEcho(t *testing.T) {
	unitWithEcho := parser.TestUnit{
		Name:       "test_with_echo",
		FilePath:   "customer_code_bloc_test.dart",
		StartLine:  10,
		Kind:       "bloc",
		SeedBody:   "baseState.copyWith(userInfo: fakeUser)",
		ExpectBody: "[isA<CustomerCodeState>().having((s) => s.userInfo, 'userInfo', fakeUser)]",
	}

	checker := dart.NewSeedEchoRule()
	violations := checker.Evaluate(unitWithEcho)

	if len(violations) != 1 {
		t.Fatalf("expected 1 seed echo violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-DART-SEED-ECHO" {
		t.Errorf("expected GUARD-DART-SEED-ECHO, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", violations[0].Severity)
	}
}

func TestDetectMockEcho(t *testing.T) {
	unitWithMockEcho := parser.TestUnit{
		Name:      "test_mock_echo",
		FilePath:  "repo_test.dart",
		StartLine: 20,
		Kind:      "unit",
		RawBody:   "when(() => api.fetch()).thenAnswer((_) => Right(mockData));\nexpect(repo.fetch(), equals(Right(mockData)));",
	}

	checker := dart.NewMockEchoRule()
	violations := checker.Evaluate(unitWithMockEcho)

	if len(violations) != 1 {
		t.Fatalf("expected 1 mock echo violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-DART-MOCK-ECHO" {
		t.Errorf("expected GUARD-DART-MOCK-ECHO, got %s", violations[0].RuleID)
	}
}
