package sample_test

import "testing"

func TestTableTRun(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{name: "one", in: 1, want: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.in*2 != tc.want {
				t.Errorf("got %d, want %d", tc.in*2, tc.want)
			}
		})
	}
}
