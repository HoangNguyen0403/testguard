package sample

import "testing"

type row struct {
	input string
	want  int
}

func TestPositionalRows(t *testing.T) {
	cases := []row{
		{"first", 1},
		{"second", 1},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if len(tc.input) != tc.want {
				t.Fatal(tc.input)
			}
		})
	}
}
