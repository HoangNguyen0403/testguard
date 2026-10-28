package sample

import "testing"

type ParamMap map[string]any

func TestCasesWithNestedMaps(t *testing.T) {
	tests := []struct {
		name   string
		params ParamMap
		want   int
	}{
		{name: "case A", params: ParamMap{"key": "val"}, want: 1},
		{name: "case B", params: ParamMap{"key": "val"}, want: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.want <= 0 {
				t.Fatal("fail")
			}
		})
	}
}
