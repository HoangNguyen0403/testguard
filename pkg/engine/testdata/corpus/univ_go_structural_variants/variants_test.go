package probe

import "testing"

func add(a, b int) int { return a + b }

func TestAddOneAndOne(t *testing.T) {
	if got := add(1, 1); got != 2 {
		t.Fatalf("add(1, 1) = %d, want 2", got)
	}
}

func TestAddTwoAndTwo(t *testing.T) {
	if got := add(2, 2); got != 4 {
		t.Fatalf("add(2, 2) = %d, want 4", got)
	}
}

func TestAddThreeAndThree(t *testing.T) {
	if got := add(3, 3); got != 6 {
		t.Fatalf("add(3, 3) = %d, want 6", got)
	}
}

func TestAddFourAndFour(t *testing.T) {
	if got := add(4, 4); got != 8 {
		t.Fatalf("add(4, 4) = %d, want 8", got)
	}
}
