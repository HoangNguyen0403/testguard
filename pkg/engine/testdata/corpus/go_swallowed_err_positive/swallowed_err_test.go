package weak_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorIgnored(t *testing.T) {
	got, err := Build("")
	if err != nil {
		return
	}
	assert.Equal(t, "x", got.Name)
}
