package mobile_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/mobile"
)

func TestKotlinCoroutineAdvanceRule(t *testing.T) {
	unitMissingAdvance := parser.TestUnit{
		Name:      "testSearchDebounce",
		FilePath:  "viewmodel/SearchViewModelTest.kt",
		StartLine: 25,
		RawBody:   "viewModel.search(\"query\");\nassertEquals(expected, viewModel.state.value);",
	}

	checker := mobile.NewKotlinCoroutineAdvanceRule()
	violations := checker.Evaluate(unitMissingAdvance)

	if len(violations) != 1 {
		t.Fatalf("expected 1 coroutine advance violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-KT-02" {
		t.Errorf("expected GUARD-KT-02, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityBlocker {
		t.Errorf("expected Blocker severity, got %s", violations[0].Severity)
	}
}

func TestSwiftExcessiveTimeoutRule(t *testing.T) {
	unitWithLongTimeout := parser.TestUnit{
		Name:      "testFetchCart",
		FilePath:  "CartTests/CartViewModelTests.swift",
		StartLine: 40,
		RawBody:   "wait(for: [expectation], timeout: 10.0)",
	}

	checker := mobile.NewSwiftTimeoutRule()
	violations := checker.Evaluate(unitWithLongTimeout)

	if len(violations) != 1 {
		t.Fatalf("expected 1 swift timeout violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-SWIFT-01" {
		t.Errorf("expected GUARD-SWIFT-01, got %s", violations[0].RuleID)
	}
}
