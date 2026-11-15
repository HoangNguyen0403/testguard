package sample_test

import "testing"

func TestTableWholeRowPassed(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{{"one", "1", 1}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkRow(t, tt)
		})
	}
}
