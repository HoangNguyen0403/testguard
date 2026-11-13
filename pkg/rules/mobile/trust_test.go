package mobile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/kotlin"
	"github.com/HoangNguyen0403/testguard/pkg/rules/mobile"
)

func evaluateKotlinFixture(t *testing.T, name string, evaluate func(parser.TestUnit) []models.Violation) []models.Violation {
	t.Helper()

	fixturePath := filepath.Join("testdata", name)
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}

	units, err := kotlin.NewAdapter().LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests(%s) error: %v", name, err)
	}

	var violations []models.Violation
	for _, u := range units {
		violations = append(violations, evaluate(u)...)
	}
	return violations
}

func assertFindings(t *testing.T, violations []models.Violation, wantIDs []string, wantLines []int, wantSeverity models.Severity) {
	t.Helper()

	if len(violations) != len(wantIDs) {
		t.Fatalf("expected %d findings, got %d: %+v", len(wantIDs), len(violations), violations)
	}
	for i, v := range violations {
		if v.RuleID != wantIDs[i] {
			t.Errorf("finding %d: expected rule %s, got %s", i, wantIDs[i], v.RuleID)
		}
		if v.LineNumber != wantLines[i] {
			t.Errorf("finding %d: expected line %d, got %d", i, wantLines[i], v.LineNumber)
		}
		if v.Severity != wantSeverity {
			t.Errorf("finding %d: expected severity %s, got %s", i, wantSeverity, v.Severity)
		}
	}
}

// TrustBoundaryTest.kt is the parent-reproduced false-positive file: a
// synchronous test under runTest, an unrelated stub, and a commented-out stub.
func TestKotlinCoroutineAdvanceRule_TrustBoundary(t *testing.T) {
	violations := evaluateKotlinFixture(t, "TrustBoundaryTest.kt", mobile.NewKotlinCoroutineAdvanceRule().Evaluate)
	assertFindings(t, violations, nil, nil, models.SeverityWarning)
}

func TestKotlinMockEchoRule_TrustBoundary(t *testing.T) {
	violations := evaluateKotlinFixture(t, "TrustBoundaryTest.kt", mobile.NewKotlinMockEchoRule().Evaluate)
	assertFindings(t, violations, nil, nil, models.SeverityWarning)
}

// AdversarialTest.kt mixes correct tests with the two hollow shapes each rule
// claims to catch; only those shapes may be reported.
func TestKotlinCoroutineAdvanceRule_Adversarial(t *testing.T) {
	violations := evaluateKotlinFixture(t, "AdversarialTest.kt", mobile.NewKotlinCoroutineAdvanceRule().Evaluate)
	assertFindings(t, violations, []string{"GUARD-KT-02"}, []int{104}, models.SeverityWarning)
}

func TestKotlinMockEchoRule_Adversarial(t *testing.T) {
	violations := evaluateKotlinFixture(t, "AdversarialTest.kt", mobile.NewKotlinMockEchoRule().Evaluate)
	assertFindings(t, violations, []string{"GUARD-KT-01"}, []int{96}, models.SeverityWarning)
}

// F6: `runTest { … }` is the canonical coroutine test form, so it must be
// analysed exactly like `runTest() { … }`. The three positives use a locally
// bound StandardTestDispatcher, inject it into the subject, call the subject
// and read its state without advancing; the four negatives are the same shape
// with one provenance link removed.
func TestKotlinCoroutineAdvanceRule_TrailingLambda(t *testing.T) {
	violations := evaluateKotlinFixture(t, "CoroutineLambdaTest.kt", mobile.NewKotlinCoroutineAdvanceRule().Evaluate)
	assertFindings(t, violations,
		[]string{"GUARD-KT-02", "GUARD-KT-02", "GUARD-KT-02"},
		[]int{11, 19, 29},
		models.SeverityWarning)
}

func TestKotlinCoroutineAdvanceRule_ProvenanceNegatives(t *testing.T) {
	violations := evaluateKotlinFixture(t, "CoroutineProvenanceNegativeTest.kt", mobile.NewKotlinCoroutineAdvanceRule().Evaluate)
	assertFindings(t, violations, nil, nil, models.SeverityWarning)
}

func TestKotlinCoroutineAdvanceRule_ProvenancePositives(t *testing.T) {
	violations := evaluateKotlinFixture(t, "CoroutineProvenancePositiveTest.kt", mobile.NewKotlinCoroutineAdvanceRule().Evaluate)
	assertFindings(t, violations, []string{"GUARD-KT-02", "GUARD-KT-02"}, []int{12, 19}, models.SeverityWarning)
}

func TestKotlinMockEchoRule_ProvenanceNegatives(t *testing.T) {
	violations := evaluateKotlinFixture(t, "EchoProvenanceNegativeTest.kt", mobile.NewKotlinMockEchoRule().Evaluate)
	assertFindings(t, violations, nil, nil, models.SeverityWarning)
}

func TestKotlinMockEchoRule_ProvenancePositives(t *testing.T) {
	violations := evaluateKotlinFixture(t, "EchoProvenancePositiveTest.kt", mobile.NewKotlinMockEchoRule().Evaluate)
	assertFindings(t, violations,
		[]string{"GUARD-KT-01", "GUARD-KT-01", "GUARD-KT-01", "GUARD-KT-01", "GUARD-KT-01"},
		[]int{12, 20, 28, 36, 44},
		models.SeverityWarning)
}
