package weak_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorChecked(t *testing.T) {
	got, err := Build("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, "x", got.Name)
}
