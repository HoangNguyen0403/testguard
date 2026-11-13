package mobile_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/mobile"
)

func TestKotlinCoroutineAdvanceRule_UnconfinedDispatcher(t *testing.T) {
	unit := parser.TestUnit{
		Name:      "testUnconfined",
		FilePath:  "Test.kt",
		StartLine: 10,
		RawBody:   "runTest { val x = search(); assertEquals(1, x) }",
		Meta: map[string]string{
			"hasStandardDispatcher":   "true",
			"hasUnconfinedDispatcher": "true",
		},
		Assertions: []string{"assertEquals"},
	}
	rule := mobile.NewKotlinCoroutineAdvanceRule()
	violations := rule.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected no violation when UnconfinedTestDispatcher is present")
	}
}

func TestKotlinCoroutineAdvanceRule_AdvancementPresent(t *testing.T) {
	unit := parser.TestUnit{
		Name:      "testWithAdvance",
		FilePath:  "Test.kt",
		StartLine: 10,
		RawBody:   "runTest { val x = search(); advanceUntilIdle(); assertEquals(1, x) }",
		Meta: map[string]string{
			"hasStandardDispatcher": "true",
		},
		Assertions: []string{"assertEquals"},
	}
	rule := mobile.NewKotlinCoroutineAdvanceRule()
	violations := rule.Evaluate(unit)
	if len(violations) != 0 {
		t.Errorf("expected no violation when advanceUntilIdle is present")
	}
}

func TestKotlinMockEchoRule_Transformations(t *testing.T) {
	// Negative case: transformed value
	unitNeg := parser.TestUnit{
		Name:      "testTransformed",
		FilePath:  "Test.kt",
		StartLine: 10,
		RawBody: `
			every { repo.name() } returns "alice"
			val vm = ViewModel(repo)
			assertEquals("ALICE", vm.name.uppercase())
		`,
		Assertions: []string{"assertEquals"},
	}
	rule := mobile.NewKotlinMockEchoRule()
	violations := rule.Evaluate(unitNeg)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for transformed value, got %d", len(violations))
	}

	// Positive cases:
	// 1. obj.prop shouldBe X
	unitKotest := parser.TestUnit{
		Name:      "testKotestEcho",
		FilePath:  "Test.kt",
		StartLine: 20,
		RawBody: `
			val repo = mockk<Repo>()
			coEvery { repo.fetchId() } returns 123
			val vm = ViewModel(repo)
			vm.id shouldBe 123
		`,
		Assertions: []string{"shouldBe"},
	}
	vKotest := rule.Evaluate(unitKotest)
	if len(vKotest) != 1 || vKotest[0].RuleID != "GUARD-KT-01" {
		t.Errorf("expected GUARD-KT-01 violation for Kotest echo, got %+v", vKotest)
	}

	// 2. assertThat(obj.prop).isEqualTo(X)
	unitTruth := parser.TestUnit{
		Name:      "testTruthEcho",
		FilePath:  "Test.kt",
		StartLine: 30,
		RawBody: `
			val repo = mockk<Repo>()
			every { repo.status() } returns "ACTIVE"
			val vm = ViewModel(repo)
			assertThat(vm.status).isEqualTo("ACTIVE")
		`,
		Assertions: []string{"assertThat"},
	}
	vTruth := rule.Evaluate(unitTruth)
	if len(vTruth) != 1 || vTruth[0].RuleID != "GUARD-KT-01" {
		t.Errorf("expected GUARD-KT-01 violation for Truth echo, got %+v", vTruth)
	}
}

func TestSwiftTimeoutRule_Threshold(t *testing.T) {
	rule := mobile.NewSwiftTimeoutRule()

	// <= 2.0s: negative case
	unitOk := parser.TestUnit{
		Name:       "testOkTimeout",
		FilePath:   "Tests.swift",
		StartLine:  15,
		RawBody:    "wait(for: [exp], timeout: 2.0)",
		Assertions: []string{"wait(for:"},
	}
	if len(rule.Evaluate(unitOk)) != 0 {
		t.Errorf("expected no violation for 2.0s timeout")
	}

	// > 2.0s: positive case with fulfillment(of:
	unitLong := parser.TestUnit{
		Name:       "testLongFulfillment",
		FilePath:   "Tests.swift",
		StartLine:  25,
		RawBody:    "fulfillment(of: [exp], timeout: 5.0)",
		Assertions: []string{"fulfillment(of:"},
	}
	violations := rule.Evaluate(unitLong)
	if len(violations) != 1 || violations[0].Severity != models.SeverityAdvisory {
		t.Errorf("expected 1 Advisory violation for 5.0s timeout, got %+v", violations)
	}
}
