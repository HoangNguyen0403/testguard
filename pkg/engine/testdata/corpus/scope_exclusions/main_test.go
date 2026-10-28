package sample_test

import "testing"

func TestMainValid(t *testing.T) {
	if 1 != 1 {
		t.Fatal("fail")
	}
}
