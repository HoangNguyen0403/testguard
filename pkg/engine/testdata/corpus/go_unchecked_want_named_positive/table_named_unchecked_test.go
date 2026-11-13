package sample_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableNamedRowUnchecked(t *testing.T) {
	type row struct {
		name     string
		in       string
		want     int
		wantErr  bool
		expected string
	}
	tests := []row{{"one", "1", 1, false, "x"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := Parse(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
