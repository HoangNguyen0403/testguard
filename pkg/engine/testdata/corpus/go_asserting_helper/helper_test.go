package sample_test

import "testing"

func verifyPositive(t *testing.T, n int) {
	if n <= 0 {
		t.Fatalf("expected positive, got %d", n)
	}
}

func TestWithHelper(t *testing.T) {
	verifyPositive(t, 42)
}
