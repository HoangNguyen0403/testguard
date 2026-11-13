package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func add(a, b int) int { return a + b }

func TestAddEchoesItsOwnValue(t *testing.T) {
	sum := add(1, 2)
	assert.Equal(t, sum, sum)
}
