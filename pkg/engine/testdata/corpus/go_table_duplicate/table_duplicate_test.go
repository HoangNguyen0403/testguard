package sample

import "testing"

func TestCalculate(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "first", in: 10, want: 20},
		{name: "second", in: 10, want: 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.in*2 != tc.want {
				t.Fatal("fail")
			}
		})
	}
}
