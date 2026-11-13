package sample_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableNamedRowChecked(t *testing.T) {
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
			got, err := Parse(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected err: %v", err)
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.expected, "x")
		})
	}
}
