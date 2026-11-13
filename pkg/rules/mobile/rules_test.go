package mobile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser/kotlin"
	"github.com/HoangNguyen0403/testguard/pkg/parser/swift"
	"github.com/HoangNguyen0403/testguard/pkg/rules/mobile"
)

func TestKotlinCoroutineAdvanceRule_StylesTest(t *testing.T) {
	fixturePath := filepath.Join("testdata", "StylesTest.kt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	rule := mobile.NewKotlinCoroutineAdvanceRule()
	var violations []models.Violation
	for _, u := range units {
		violations = append(violations, rule.Evaluate(u)...)
	}

	// Should fire ONLY on staleStateUnderStandardDispatcher
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation, got %d: %+v", len(violations), violations)
	}
	v := violations[0]
	if v.RuleID != "GUARD-KT-02" {
		t.Errorf("expected GUARD-KT-02, got %s", v.RuleID)
	}
	if v.Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", v.Severity)
	}
	if v.LineNumber != 68 {
		t.Errorf("expected line 68, got %d", v.LineNumber)
	}
}

func TestKotlinCoroutineAdvanceRule_CacheTest(t *testing.T) {
	fixturePath := filepath.Join("testdata", "CacheTest.kt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	rule := mobile.NewKotlinCoroutineAdvanceRule()
	var violations []models.Violation
	for _, u := range units {
		violations = append(violations, rule.Evaluate(u)...)
	}

	// Should not fire on synchronous fetch/search
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations for CacheTest, got %d: %+v", len(violations), violations)
	}
}

func TestKotlinMockEchoRule_StylesTest(t *testing.T) {
	fixturePath := filepath.Join("testdata", "StylesTest.kt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	rule := mobile.NewKotlinMockEchoRule()
	var violations []models.Violation
	for _, u := range units {
		violations = append(violations, rule.Evaluate(u)...)
	}

	// Should fire ONLY on mockEcho
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation, got %d: %+v", len(violations), violations)
	}
	v := violations[0]
	if v.RuleID != "GUARD-KT-01" {
		t.Errorf("expected GUARD-KT-01, got %s", v.RuleID)
	}
	if v.Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", v.Severity)
	}
	if v.LineNumber != 75 {
		t.Errorf("expected line 75, got %d", v.LineNumber)
	}
}

func TestSwiftTimeoutRule_StylesTests(t *testing.T) {
	fixturePath := filepath.Join("testdata", "StylesTests.swift")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	adapter := swift.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	rule := mobile.NewSwiftTimeoutRule()
	var violations []models.Violation
	for _, u := range units {
		violations = append(violations, rule.Evaluate(u)...)
	}

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(violations), violations)
	}
	v := violations[0]
	if v.RuleID != "GUARD-SWIFT-01" {
		t.Errorf("expected GUARD-SWIFT-01, got %s", v.RuleID)
	}
	if v.Severity != models.SeverityAdvisory {
		t.Errorf("expected Advisory severity, got %s", v.Severity)
	}
}
