package sample_test

import "testing"

func double(x int) int { return x * 2 }

var cases = []struct {
	name string
	in   int
	want int
}{
	{name: "two", in: 1, want: 2},
}

func TestDoubleTable(t *testing.T) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if double(tc.in) != tc.want {
				t.Errorf("x")
			}
		})
	}
}
