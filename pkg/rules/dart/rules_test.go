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

func TestDetectUncoveredLoadingMutation(t *testing.T) {
	unitWithMissingLoading := parser.TestUnit{
		Name:       "test_missing_is_fetching",
		FilePath:   "customer_code_bloc_test.dart",
		StartLine:  50,
		Kind:       "bloc",
		ActBody:    "bloc.add(const CustomerCodeEvent.fetch())",
		ExpectBody: "[isA<CustomerCodeState>().having((s) => s.userInfo, 'userInfo', fakeUser)]", // Omitted isFetching
	}

	checker := dart.NewUncoveredMutationRule()
	violations := checker.Evaluate(unitWithMissingLoading)

	if len(violations) != 1 {
		t.Fatalf("expected 1 uncovered mutation violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-DART-UNCOVERED-MUTATION" {
		t.Errorf("expected GUARD-DART-UNCOVERED-MUTATION, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityBlocker {
		t.Errorf("expected Blocker severity, got %s", violations[0].Severity)
	}
}
