package sample_test

import "testing"

/*
func TestDisabledCommented(t *testing.T) {
	// This test was commented out and has no asserts
}
*/

func TestRealOne(t *testing.T) {
	if 1 != 1 {
		t.Fatal("math broke")
	}
}
