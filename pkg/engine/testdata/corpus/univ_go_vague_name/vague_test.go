package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func add(a, b int) int { return a + b }

func Test1(t *testing.T) {
	assert.Equal(t, 3, add(1, 2))
}
