package sample

import "testing"

func check(t *testing.T) {
	t.Fatal("failure")
}

func TestHollow(t *testing.T) {
	check := func(t *testing.T) {}
	check(t)
}
