package probe

import "testing"

func add(a, b int) int { return a + b }

func TestAddTwoNumbers(t *testing.T) {
	if got := add(1, 2); got != 3 {
		t.Fatalf("add(1, 2) = %d, want 3", got)
	}
}

func TestAddTwoNumbersAgain(t *testing.T) {
	if got := add(1, 2); got != 3 {
		t.Fatalf("add(1, 2) = %d, want 3", got)
	}
}
