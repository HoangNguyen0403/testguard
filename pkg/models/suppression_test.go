package models_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

func TestSuppressionParser(t *testing.T) {
	source := `
void main() {
  // testguard:ignore GUARD-DART-SEED-ECHO reason: legacy test
  test('echo test', () {
    expect(1, equals(1));
  });

  test('normal test', () {
    expect(2, equals(2));
  });
}
`

	suppressions := models.ParseSuppressions(source)
	if len(suppressions) != 1 {
		t.Fatalf("expected 1 suppression, got %d", len(suppressions))
	}

	if suppressions[0].RuleID != "GUARD-DART-SEED-ECHO" {
		t.Errorf("expected rule GUARD-DART-SEED-ECHO, got %s", suppressions[0].RuleID)
	}

	// Suppression on line 3 suppresses violations on line 3 and line 4 (the immediately following test)
	if !models.IsSuppressed(suppressions, "GUARD-DART-SEED-ECHO", 4) {
		t.Errorf("expected violation on line 4 to be suppressed")
	}

	if models.IsSuppressed(suppressions, "GUARD-DART-SEED-ECHO", 8) {
		t.Errorf("expected violation on line 8 NOT to be suppressed")
	}
}
